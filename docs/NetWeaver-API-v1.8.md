# NetWeaver API 接口文档 v1.8

## 1. 概述与全局规范

本文档定义了 NetWeaver 异地组网平台中，**控制平面（Controller）**与**管理平面（前端 Dashboard）**、**数据平面（Node）**之间的通信接口。

### 1.0 文档范围

**本文档覆盖以下全部协议**：

| 协议层              | 定义章节                     | 说明                                              |
|---------------------|------------------------------|---------------------------------------------------|
| Dashboard ↔ Controller | §2（HTTP/JSON）            | 管理控制台查询网络状态、拓扑、链路数据              |
| Node ↔ Controller      | §3（HTTP/JSON）            | 节点注册、心跳保活、Peer 列表拉取                  |
| 建链信令协议           | §5（Node 端状态机）        | P2P 打洞 / Relay 回退的完整流程与 Controller 协同  |
| Node ↔ Node 数据面     | §6（TUN 封装格式）         | P2P 直连时的报文封装、MTU、Keepalive               |
| Node ↔ Relay 数据面    | §6（TUN 封装格式）         | Relay 转发时的封装格式、会话标识与保活             |
| STUN 探测协议          | §11（STUN 交互规范）       | NAT 类型探测的 STUN Binding 请求/响应格式          |

> **阅读指引：** 本文档为 NetWeaver 控制面 + 数据面的完整接口规范。Dashboard 开发者只需阅读 §1-§4；Node 开发者需阅读全文。

### 1.1 统一响应结构

所有接口的响应体均采用统一的 JSON 结构：`code` / `msg` / `data`。

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

| 字段   | 类型          | 说明                                                   |
|--------|---------------|--------------------------------------------------------|
| code   | number        | 业务状态码，200 为成功，4xx 为客户端错误，5xx 为服务端错误 |
| msg    | string        | 状态描述或错误提示                                        |
| data   | object / null | 业务数据负载，无数据时为 `null`                           |

### 1.2 错误码约定

| 场景         | code | msg 规范                                  |
|--------------|------|-------------------------------------------|
| 正常成功     | 200  | `"success"` 或具体成功提示                  |
| 请求参数错误 | 400  | 明确提示缺少或错误的参数                     |
| 鉴权失败     | 401  | `"unauthorized: invalid or missing PSK"` 或 `"unauthorized: invalid or expired token"` |
| 资源不存在   | 404  | 明确提示 `"node not found"` 等              |
| 业务状态冲突 | 409  | 用于未来的冲突型场景                         |
| 资源不可用   | 503  | 依赖资源暂不可用，如 Relay 会话分配失败       |
| 服务端异常   | 500  | 简明错误描述，不泄露内部细节                  |

### 1.3 字段命名规范

- JSON 字段名统一使用 `snake_case`。
- 枚举值统一使用以下约定：

| 字段            | 可选值                                                                   |
|-----------------|--------------------------------------------------------------------------|
| status          | `"online"` / `"offline"`                                                 |
| type            | `"p2p"` / `"relay"`                                                      |
| recommend_mode  | `"p2p"` / `"relay"`                                                      |
| action_required | `"none"` / `"sync_peers"`                                                |
| nat_type        | `"Full Cone"` / `"Restricted Cone"` / `"Port Restricted Cone"` / `"Symmetric"` / `"Unknown"` |

### 1.4 时间与单位规范

| 字段类型           | 格式                  | 示例                            |
|--------------------|-----------------------|---------------------------------|
| 持续时长（运行时间） | 秒级整数              | `controller_uptime_sec: 86400`  |
| 折线图时间戳       | Unix 秒级整数         | `timestamp: 1714819200`         |
| 流量字节           | 无符号整数，单位 Byte | `total_rx_bytes: 123456789`     |
| 流量概览           | 浮点数，单位 GB       | `total_traffic_gb: 45.2`        |

### 1.5 URL 规范

- 正式 API 统一使用 `/api/v1/...` 前缀。
- `GET /` 和 `GET /ping` 为调试/健康检查接口，不在本文档正式接口范围内。
- `GET /nodes` 为历史辅助路由，当前正式接口请使用 `/api/v1/dashboard/nodes`。

### 1.6 接口设计原则

1. 接口语义必须真实且稳定，先修现有接口，不新增无关接口。
2. 先保证字段语义真实，再考虑字段丰富度。
3. 前端当前未使用的扩展字段不随意增加。
4. 内部结构可以复杂，对外 DTO 必须简单稳定。

### 1.7 接口认证总表

受保护接口必须按下表携带认证头。未携带 Header、Header 格式错误、凭证错误或 JWT 过期时，Controller 必须返回统一错误结构，`data` 固定为 `null`。

| 接口范围 / 路径 | 认证头 | 凭证类型 | 失败 HTTP code | 失败 msg |
|-----------------|--------|----------|----------------|----------|
| `POST /api/v1/auth/login` | 无 | 无，使用请求体用户名密码登录 | 400 / 401 | 参数错误返回明确 msg；用户名或密码错误返回 `"unauthorized: invalid username or password"` |
| `/api/v1/dashboard/*` | `Authorization: Bearer <jwt>` | Dashboard 登录接口签发的 JWT | 401 | 未携带、格式错误、签名无效或已过期均返回 `"unauthorized: invalid or expired token"` |
| `POST /api/v1/nodes/register` | `Authorization: Bearer <psk>` | Node 预共享密钥 PSK | 401 | 未携带、格式错误或 PSK 错误均返回 `"unauthorized: invalid or missing PSK"` |
| `POST /api/v1/nodes/{node_id}/heartbeat` | `Authorization: Bearer <psk>` | Node 预共享密钥 PSK | 401 | 未携带、格式错误或 PSK 错误均返回 `"unauthorized: invalid or missing PSK"` |
| `GET /api/v1/nodes/{node_id}/peers` | `Authorization: Bearer <psk>` | Node 预共享密钥 PSK | 401 | 未携带、格式错误或 PSK 错误均返回 `"unauthorized: invalid or missing PSK"` |

------------------------------------------------------------------------

## 2. 前端 (Dashboard) \<-\> Controller 接口

该部分接口主要服务于基于 Vue 3 构建的 Web 管理控制台，用于呈现网络拓扑和管理节点。

### 2.0 Dashboard 登录 (Auth Login)

Dashboard 在访问任何 `/api/v1/dashboard/*` 接口前，必须先通过本接口获取 JWT。

- **请求方式 & 路径:** `POST /api/v1/auth/login`
- **认证要求:** 无
- **Content-Type:** `application/json`
- **JWT 有效期:** 24 小时。过期后 Dashboard 必须重新登录获取新 Token。

**请求字段说明：**

| 字段名   | 类型   | 必填 | 说明                         |
|----------|--------|------|------------------------------|
| username | string | 是   | Dashboard 管理员用户名         |
| password | string | 是   | Dashboard 管理员密码，明文经 HTTPS 传输 |

**请求示例：**

```json
{
  "username": "admin",
  "password": "change-me"
}
```

**响应字段说明：**

| 字段名      | 类型   | 说明                                      |
|-------------|--------|-------------------------------------------|
| token       | string | JWT Token，后续请求放入 `Authorization` Header |
| token_type  | string | 固定为 `"Bearer"`                         |
| expires_in  | number | 有效期秒数，固定为 `86400`                |
| expires_at  | number | 过期时间，Unix 秒级时间戳                 |

**成功响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "token": "<jwt>",
    "token_type": "Bearer",
    "expires_in": 86400,
    "expires_at": 1714905600
  }
}
```

**失败响应示例：**

```json
{
  "code": 401,
  "msg": "unauthorized: invalid username or password",
  "data": null
}
```

### 2.1 获取全局网络状态统计

用于 Dashboard 首页的数据大屏展示（顶部统计卡片）。

- **请求方式 & 路径:** `GET /api/v1/dashboard/stats`
- **认证要求:** `Authorization: Bearer <jwt>`
- **请求参数:** 无

**字段说明：**

| 字段名                 | 类型   | 分类     | 说明                              |
|------------------------|--------|----------|-----------------------------------|
| total_nodes            | number | 核心字段 | 总节点数（含在线和离线）           |
| online_nodes           | number | 核心字段 | 当前在线节点数                     |
| total_traffic_gb       | number | 核心字段 | 总流量，单位 GB，保留两位小数      |
| controller_uptime_sec  | number | 核心字段 | 控制器运行时长，单位秒             |
| offline_nodes          | number | 扩展字段 | 当前离线节点数                     |
| total_edges            | number | 扩展字段 | 当前网络的边（连接关系）总数       |
| total_rx_bytes         | number | 扩展字段 | 所有节点累计接收字节数             |
| total_tx_bytes         | number | 扩展字段 | 所有节点累计发送字节数             |
| total_traffic_bytes    | number | 扩展字段 | 总流量字节数（rx + tx）           |
| last_updated_at        | number | 扩展字段 | 数据最后更新时间，Unix 秒级时间戳  |

**响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "total_nodes": 12,
    "online_nodes": 10,
    "total_traffic_gb": 45.2,
    "controller_uptime_sec": 86400,
    "offline_nodes": 2,
    "total_edges": 9,
    "total_rx_bytes": 123456789,
    "total_tx_bytes": 223456789,
    "total_traffic_bytes": 346913578,
    "last_updated_at": 1714819200
  }
}
```

### 2.2 获取网络拓扑节点列表

拉取当前私有网络内的所有节点信息，用于前端表格渲染、关系图节点渲染和悬浮详情卡片。

- **请求方式 & 路径:** `GET /api/v1/dashboard/nodes`
- **认证要求:** `Authorization: Bearer <jwt>`
- **请求参数:** 无

**核心字段说明（前端直接依赖）：**

| 字段名           | 类型            | 说明                                                         |
|------------------|-----------------|--------------------------------------------------------------|
| node_id          | string          | 节点全局唯一标识符，由 Controller 分配                         |
| hostname         | string          | 节点主机名称                                                   |
| virtual_ip       | string          | 分配给该节点的虚拟内网 IP                                      |
| public_ip        | string \| null  | 节点的公网出口 IP，未探测到时为空字符串或 `null`                |
| nat_type         | string          | NAT 类型，详见 1.3 节枚举值                                    |
| status           | string          | 节点在线状态：`"online"` 或 `"offline"`                        |
| connected_peers  | number          | 当前已建立真实连接的邻居节点数量，由后端根据真实连接状态表推导   |

> **注意：** `connected_peers` 的语义是"该节点当前真实已连接的邻居数"，来源于真实连接状态，而非在线节点总数。后端需根据心跳上报的 `connected_peer_ids` 或边状态表计算，确保数值真实可信。

**扩展字段说明（前端暂未使用，保留供后续扩展）：**

| 字段名            | 类型   | 说明                         |
|-------------------|--------|------------------------------|
| machine_id        | string | 机器唯一标识（如 MAC 地址）   |
| os                | string | 操作系统名称                 |
| local_ip          | string | 节点本地局域网 IP            |
| public_port       | number | 公网映射端口                 |
| current_rx_bytes  | number | 当前累计接收字节数            |
| current_tx_bytes  | number | 当前累计发送字节数            |
| last_seen         | number | 最后一次心跳时间，Unix 秒级   |

**响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "nodes": [
      {
        "node_id": "nw-node-a1b2",
        "hostname": "ubuntu-server-01",
        "virtual_ip": "10.0.0.2",
        "public_ip": "203.0.113.5",
        "nat_type": "Full Cone",
        "status": "online",
        "connected_peers": 1,
        "machine_id": "e4:5f:01:aa:bb:cc",
        "os": "linux",
        "local_ip": "192.168.1.10",
        "public_port": 54321,
        "current_rx_bytes": 1024560,
        "current_tx_bytes": 2048000,
        "last_seen": 1714819200
      },
      {
        "node_id": "nw-node-c3d4",
        "hostname": "linux-edge-02",
        "virtual_ip": "10.0.0.3",
        "public_ip": "198.51.100.12",
        "nat_type": "Symmetric",
        "status": "online",
        "connected_peers": 1,
        "machine_id": "aa:bb:cc:dd:ee:ff",
        "os": "linux",
        "local_ip": "192.168.1.20",
        "public_port": 45678,
        "current_rx_bytes": 512000,
        "current_tx_bytes": 1024000,
        "last_seen": 1714819200
      }
    ]
  }
}
```

### 2.3 获取节点关系边列表

获取当前网络中节点间的真实连接关系，用于前端渲染拓扑图的连线。

> **说明：** 关系边数据不直接并入 nodes 接口返回里，前端需调用此独立接口进行关系连线。边仅表示已建立真实连接的节点对，不是全连接。

- **请求方式 & 路径:** `GET /api/v1/dashboard/edges`
- **认证要求:** `Authorization: Bearer <jwt>`
- **请求参数:** 无

**字段说明：**

| 字段名 | 类型   | 说明                                        |
|--------|--------|---------------------------------------------|
| source | string | 源节点 node_id                               |
| target | string | 目标节点 node_id                             |
| type   | string | 连线类型：`"p2p"`（直连）或 `"relay"`（中转） |

**响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "edges": [
      {
        "source": "nw-node-a1b2",
        "target": "nw-node-c3d4",
        "type": "p2p"
      }
    ]
  }
}
```

### 2.4 获取节点链路监控数据

获取节点到指定目标节点之间的链路延迟历史数据，用于悬浮详情卡片的折线图渲染。

- **请求方式 & 路径:** `GET /api/v1/dashboard/nodes/{node_id}/metrics`
- **认证要求:** `Authorization: Bearer <jwt>`

**查询参数：**

| 参数名     | 类型   | 必填 | 说明                                              |
|------------|--------|------|---------------------------------------------------|
| target_id  | string | 是   | 目标节点 ID，指定要查询到哪个对端的链路延迟          |
| time_range | string | 否   | 查询时间范围，可选 `"1h"`、`"12h"`、`"24h"`，默认 `"1h"` |

**字段说明：**

| 字段名      | 类型   | 说明                              |
|-------------|--------|-----------------------------------|
| timestamp   | number | 数据点采集时间，Unix 秒级时间戳     |
| latency_ms  | number | 链路延迟数值，单位毫秒             |

**响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "metrics": [
      {
        "timestamp": 1714819200,
        "latency_ms": 45.5
      },
      {
        "timestamp": 1714819260,
        "latency_ms": 42.1
      }
    ]
  }
}
```

#### 2.4.1 完整语义定义

以下规定每种边界情况下该接口的**确切行为**，前后端必须以此为准：

| 边界场景                                     | HTTP code | msg                    | data                            |
|----------------------------------------------|-----------|------------------------|---------------------------------|
| `node_id` 不存在                              | 404       | `"node not found"`     | `null`                          |
| `target_id` 不存在                            | 404       | `"target not found"`   | `null`                          |
| `node_id == target_id`                         | 400       | `"target_id must be different from node_id"` | `null`       |
| 缺少必填参数 `target_id`                      | 400       | `"target_id is required"` | `null`                       |
| `time_range` 非法值（非 `1h`/`12h`/`24h`）    | 400       | `"invalid time_range"` | `null`                          |
| 两节点均存在，但从未建立过真实链路              | 200       | `"success"`            | `{ "metrics": [] }`             |
| 有真实链路，但当前时间范围内无采样点            | 200       | `"success"`            | `{ "metrics": [] }`             |
| 正常查询，有数据                              | 200       | `"success"`            | `{ "metrics": [...] }`          |

**数据排序与采样规则：**

| 规则项               | 规定值                                                                 |
|----------------------|------------------------------------------------------------------------|
| 返回结果排序         | 按 `timestamp` **升序**排列（旧数据在前，新数据在后）                    |
| 采样与保留策略       | 最近 **1 小时**保留 15 秒原始点；1 到 24 小时按 **1 分钟聚合点**保留并返回 |
| `time_range=1h`      | 返回最近 1 小时内的 15 秒原始点，最多约 240 个点                         |
| `time_range=12h` / `24h` | 最近 1 小时返回原始点，更早部分返回 1 分钟聚合点，整体按时间升序混合返回 |
| 延迟数据来源         | 节点间**主动探测**（ICMP ping 或 UDP probe），通过心跳的 `link_metrics` 字段上报至 Controller |
| `timestamp` 归属     | 由 Controller 在接收携带该 `link_metrics` 的心跳时生成，所有同一心跳内的 `link_metrics` 点共用同一个秒级 `timestamp` |

> **设计说明：** 当 Dashboard 请求两个存在但无真实链路的节点对的 metrics 时，返回 `200 + metrics: []` 而非 `404`。这是因为节点和链路是两个维度的资源——节点存在是事实，无链路数据只是当前无采样点。前端收到空数组时应展示"暂无数据"而非报错。`node_id == target_id` 不属于有效链路查询，因为链路延迟只定义在两个不同节点之间。

**Dashboard 鉴权失败示例（适用于所有 `/api/v1/dashboard/*` 接口）：**

```json
{
  "code": 401,
  "msg": "unauthorized: invalid or expired token",
  "data": null
}
```

### 2.5 节点与边的状态判定规则

以下规则定义了 Controller 如何判定节点在线/离线、边建立/删除，前后端与节点端必须统一遵循。

#### 2.5.1 节点状态判定

| 规则项             | 规定值 / 行为                                                              |
|--------------------|---------------------------------------------------------------------------|
| 心跳超时判定离线   | 连续 **3 个心跳周期**（默认 45 秒）未收到心跳，Controller 将节点标记为 `offline` |
| 离线后边处理       | 节点标记为 `offline` 后，**立即**从边列表中移除该节点关联的所有边           |
| 恢复在线后边处理   | 节点恢复在线后，**不会自动恢复**旧边。边需等待新心跳上报 `connected_peer_ids` 后重建 |
| `last_seen` 更新   | 每次成功心跳时更新为心跳到达时刻（UTC），不因 Dashboard 查询而更新          |

#### 2.5.2 边（Edges）状态判定

| 规则项                   | 规定值 / 行为                                                              |
|--------------------------|---------------------------------------------------------------------------|
| 边建立条件               | 当节点 A 和节点 B **均在心跳中互相上报对方**的 `node_id` 于 `connected_peer_ids` 中时，建立边 |
| 边删除条件               | 任一端连续 **3 个心跳周期**未在 `connected_peer_ids` 中包含对方，边删除    |
| 边是否无向去重           | **是无向边**。`(A, B)` 与 `(B, A)` 视为同一条边，在 edges 列表中仅出现一次，`source` 为 node_id 字典序较小的一方 |
| 同一对节点能否有多条边   | **不能**。一对节点之间最多存在一条边                                       |
| `edges.type = "p2p"`     | 当两端节点通过 UDP 打洞建立了**直接** P2P 通道时记为 `"p2p"`               |
| `edges.type = "relay"`   | 当两端节点无法打洞，数据通过 Relay 服务器转发时记为 `"relay"`              |
| `edges.type` 由谁决定    | 由心跳中 `link_metrics[].link_type` 字段决定，Controller 不推断 |
| `connected_peers` 与 `connected_peer_ids` 冲突 | 以必填的 `connected_peer_ids` 为准。`connected_peers` 数值仅作为辅助排查字段，不作为判定来源 |

#### 2.5.3 边类型与 `recommend_mode` 的区别

| 概念              | 含义                                                         | 来源                    |
|-------------------|--------------------------------------------------------------|-------------------------|
| `edges.type`      | 当前**真实**链路类型（已建链），取值 `"p2p"` 或 `"relay"`     | 节点上报的真实连接状态   |
| `recommend_mode`  | Controller 根据 NAT 类型给出的**建议**连接模式               | Controller 静态计算      |

> **关键区别：** `recommend_mode: "p2p"` 不保证已建立 P2P 连接。Dashboard 展示拓扑时应以 `edges.type` 为准渲染边的样式（如实线/虚线或颜色区分），`recommend_mode` 仅供 Node 建链决策参考。

------------------------------------------------------------------------

## 3. Node \<-\> Controller 接口

该部分接口服务于数据平面节点。节点程序用于身份注册、NAT 穿透信令交换和心跳保活。

### 3.1 节点注册与接入 (Node Register)

节点首次启动时，向信令服务器汇报自身信息并请求分配虚拟内网 IP。

- **请求方式 & 路径:** `POST /api/v1/nodes/register`
- **认证要求:** `Authorization: Bearer <psk>`

**请求字段说明：**

| 字段名      | 类型   | 必填 | 说明                        |
|-------------|--------|------|-----------------------------|
| machine_id  | string | 是   | 机器唯一标识（如 MAC 地址）  |
| hostname    | string | 是   | 节点主机名                   |
| os          | string | 是   | 操作系统名称                 |
| local_ip    | string | 是   | 节点本地局域网 IP            |

**请求示例：**

```json
{
  "machine_id": "e4:5f:01:aa:bb:cc",
  "hostname": "ubuntu-server-01",
  "os": "linux",
  "local_ip": "192.168.1.10"
}
```

**响应字段说明：**

| 字段名              | 类型   | 说明                            |
|---------------------|--------|---------------------------------|
| node_id             | string | Controller 分配的唯一节点 ID     |
| virtual_ip          | string | Controller 分配的 TUN 网卡 IP   |
| subnet_mask         | string | 子网掩码                        |
| keepalive_interval  | number | 建议的心跳间隔，单位秒           |

**响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "node_id": "nw-node-a1b2",
    "virtual_ip": "10.0.0.2",
    "subnet_mask": "255.255.0.0",
    "keepalive_interval": 15
  }
}
```

> **幂等性说明：** 同一个 `machine_id` 重复注册时，Controller **必须复用**原来已分配的 `node_id` 和 `virtual_ip`，不得重新分配新 ID 或新 IP。即：注册接口对同一 `machine_id` 是幂等的，返回值中 `node_id` 和 `virtual_ip` 应与首次注册时一致。节点重启或网络中断后重新注册时，依赖此语义恢复原有身份。

**鉴权失败响应示例：**

```json
{
  "code": 401,
  "msg": "unauthorized: invalid or missing PSK",
  "data": null
}
```

### 3.2 节点心跳与状态上报 (Heartbeat)

节点定期汇报自身存活状态、NAT 探测结果及已连接邻居信息。Controller 根据这些信息判断是否需要下发同步对端指令或回退到 Relay 模式。

- **请求方式 & 路径:** `POST /api/v1/nodes/{node_id}/heartbeat`
- **认证要求:** `Authorization: Bearer <psk>`

**请求字段说明：**

| 字段名              | 类型     | 必填 | 说明                                          |
|---------------------|----------|------|-----------------------------------------------|
| nat_type            | string   | 是   | STUN 探测出的 NAT 类型，详见 1.3 节枚举值       |
| public_ip           | string   | 是   | 公网出口 IP。STUN 成功时为 `XOR-MAPPED-ADDRESS` IP；STUN 全失败时允许为空字符串 |
| public_port         | number   | 是   | 公网映射端口。STUN 成功时为 `XOR-MAPPED-ADDRESS` Port；STUN 全失败时允许为 `0` |
| current_rx_bytes    | number   | 是   | Node 进程启动以来累计接收字节数                   |
| current_tx_bytes    | number   | 是   | Node 进程启动以来累计发送字节数                   |
| connected_peer_ids  | string[] | 是   | 节点当前已连接邻居 node_id 列表，用于构建真实边。无连接时必须传空数组 `[]` |
| connected_peers     | number   | 否   | 可选冗余字段，仅用于辅助排查，不作为 Controller 判定来源 |
| link_metrics        | object[] | 否   | 到各邻居的链路延迟指标，用于构建链路延迟时序数据  |

> **`public_ip` / `public_port` 上报策略：** 节点**必须携带** `public_ip` 和 `public_port` 字段，Controller **不负责推断**这两个字段。STUN 探测成功时二者必须为有效公网映射地址；STUN 全部失败时允许上报 `nat_type="Unknown"`、`public_ip=""`、`public_port=0`，Controller 必须接受该心跳，但将该节点标记为受限模式：不作为可 P2P 节点推荐，只能被推荐 Relay。除该受限场景外，`public_ip` 为空或 `public_port=0` 返回 `400`。

**请求示例：**

```json
{
  "nat_type": "Full Cone",
  "public_ip": "203.0.113.5",
  "public_port": 54321,
  "current_rx_bytes": 1024560,
  "current_tx_bytes": 2048000,
  "connected_peers": 2,
  "connected_peer_ids": ["nw-node-c3d4", "nw-node-e5f6"],
  "link_metrics": [
    { "target_node_id": "nw-node-c3d4", "latency_ms": 42.1, "link_type": "relay" },
    { "target_node_id": "nw-node-e5f6", "latency_ms": 18.7, "link_type": "p2p" }
  ]
}
```

> **`link_metrics` 子字段说明：**
> 
> | 字段名          | 类型   | 必填 | 说明                         |
> |-----------------|--------|------|------------------------------|
> | target_node_id  | string | 是   | 目标邻居节点 ID               |
> | latency_ms      | number | 是   | 到该邻居的延迟，单位毫秒       |
> | link_type       | string | 是   | 当前真实链路类型，只能为 `"p2p"` 或 `"relay"` |
> 
> Controller 收到 `link_metrics` 后，按 `(source_node_id, target_node_id)` 维度维护延迟时间序列，并用 `link_type` 更新真实边类型，作为 `GET /api/v1/dashboard/nodes/{node_id}/metrics` 和 `GET /api/v1/dashboard/edges` 的真实数据来源。延迟数据来自节点间**主动探测**（如 ICMP ping 或 UDP probe），而非心跳附带采集。`timestamp` 由 Controller 记录接收该心跳的时间生成，同一心跳中的多条 `link_metrics` 共用同一 `timestamp`。

**响应字段说明：**

| 字段名           | 类型   | 说明                                                         |
|------------------|--------|--------------------------------------------------------------|
| action_required  | string | `"none"` 表示无需操作，`"sync_peers"` 表示需同步对端列表      |
| connected_peers  | number | Controller 记录的该节点已连接邻居数                           |

**响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "action_required": "sync_peers",
    "connected_peers": 2
  }
}
```

#### 3.2.1 心跳上报规则

| 规则项               | 规定值 / 行为                                                                 |
|----------------------|-------------------------------------------------------------------------------|
| 心跳建议间隔         | 15 秒（以注册响应中 `keepalive_interval` 为准）                                |
| 心跳容忍超时         | 3 个心跳周期（45 秒）。超时后 Controller 将节点标记为 `offline`                |
| `connected_peer_ids` 上报 | 必填数组。当前无连接时必须传 `[]`，不得省略                                    |
| `connected_peers` 省略 | 允许省略。该字段仅用于辅助排查，不参与真实连接状态判定                         |
| `connected_peers` 与 `connected_peer_ids.length` 不一致 | 以 `connected_peer_ids.length` 为准，忽略 `connected_peers` 数值 |
| `link_metrics` 上报频率 | 每次心跳均可携带。建议与心跳同频上报，无需额外定时器                       |
| `link_metrics` 部分邻居不上报 | 允许。仅上报已探测到延迟数据的邻居，未上报的邻居不产生数据点            |
| Controller 历史采样保留 | 最近 1 小时保留 15 秒原始点；1 到 24 小时保留 1 分钟聚合点                 |
| 同一时间窗口重复上报 | 后上报的值**覆盖**先上报的值（按 `timestamp` 去重，保留最新）                |
| `connected_peer_ids` 含无效 ID | 忽略不存在的 peer ID，不计入 `connected_peers`，不建立边                    |
| 流量计数器复位语义 | `current_rx_bytes` / `current_tx_bytes` 为 Node **进程启动以来**累计值。Controller 发现任一计数小于上一次心跳值时，按 Node 进程重启或计数器复位处理：本次流量增量按 0 计，不报错，并从当前值重新累计后续增量 |
| STUN 失败受限模式 | 当 `nat_type="Unknown"` 且 `public_ip=""`、`public_port=0` 时，心跳成功；该节点不参与 P2P 推荐，其他节点获取它时 `recommend_mode` 必须为 `"relay"` |

**参数错误响应示例：**

```json
{
  "code": 400,
  "msg": "connected_peer_ids is required",
  "data": null
}
```

**鉴权失败响应示例：**

```json
{
  "code": 401,
  "msg": "unauthorized: invalid or missing PSK",
  "data": null
}
```

### 3.3 拉取对等节点路由表 (Get Peers)

节点获取局域网内其他所有在线节点的信息，用于在本地建立 P2P 连接（UDP 打洞）或建立转发隧道。

- **请求方式 & 路径:** `GET /api/v1/nodes/{node_id}/peers`
- **认证要求:** `Authorization: Bearer <psk>`

**响应字段说明：**

| 字段名             | 类型   | 说明                                              |
|--------------------|--------|---------------------------------------------------|
| target_node_id     | string | 目标节点的唯一 ID                                  |
| target_hostname    | string | 目标节点的主机名                                   |
| target_virtual_ip  | string | 目标节点的虚拟内网 IP                              |
| target_public_ip   | string | 目标节点的公网 IP                                  |
| target_public_port | number | 目标节点的公网映射端口                              |
| nat_type           | string | 目标节点的 NAT 类型，详见 1.3 节枚举值              |
| recommend_mode     | string | **建议**连接模式：`"p2p"`（UDP 打洞）或 `"relay"`（中转）。注意：此字段仅为 Controller 根据 NAT 类型给出的建议，**不等于**已建立的真实链路类型（真实链路类型见 `edges.type`） |

**响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "peers": [
      {
        "target_node_id": "nw-node-c3d4",
        "target_hostname": "linux-edge-02",
        "target_virtual_ip": "10.0.0.3",
        "target_public_ip": "198.51.100.12",
        "target_public_port": 45678,
        "nat_type": "Symmetric",
        "recommend_mode": "relay"
      },
      {
        "target_node_id": "nw-node-e5f6",
        "target_hostname": "raspberry-pi",
        "target_virtual_ip": "10.0.0.4",
        "target_public_ip": "114.114.114.114",
        "target_public_port": 33445,
        "nat_type": "Full Cone",
        "recommend_mode": "p2p"
      }
    ]
  }
}
```

#### 3.3.1 完整语义定义

| 边界场景 | HTTP code | msg | data |
|----------|-----------|-----|------|
| `node_id` 不存在 | 404 | `"node not found"` | `null` |
| 当前没有其他在线节点 | 200 | `"success"` | `{ "peers": [] }` |
| 存在其他节点但均离线 | 200 | `"success"` | `{ "peers": [] }` |
| 存在在线 peer | 200 | `"success"` | `{ "peers": [...] }` |
| 任一应返回 peer 被判定为 `recommend_mode="relay"`，但 Relay 会话资源分配失败 | 503 | `"relay resource unavailable"` | `null` |

**返回过滤规则：**

| 规则项 | 规定 |
|--------|------|
| 是否包含自己 | 返回结果**不得包含**请求路径中的 `node_id` 自己 |
| 是否返回离线节点 | 只返回 `status="online"` 的节点 |
| STUN 失败节点 | `nat_type="Unknown"` 且 `public_ip=""`、`public_port=0` 的在线节点可以返回，但 `recommend_mode` 必须为 `"relay"`，且必须携带 Relay 扩展字段 |
| Relay 资源失败策略 | 不跳过单个 peer。只要应返回结果中存在任一 peer 需要 Relay 但分配失败，整次请求返回 `503`，避免客户端拿到不完整路由表 |

**鉴权失败响应示例：**

```json
{
  "code": 401,
  "msg": "unauthorized: invalid or missing PSK",
  "data": null
}
```

#### 3.3.2 Relay 模式支撑说明

当 `recommend_mode = "relay"` 时，节点无法通过 UDP 打洞建立 P2P 直连，数据必须经由 Relay 服务器转发。以下规定 Relay 模式下的关键行为：

| 问题                                           | 规定                                                                                                              |
|------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| `recommend_mode=relay` 时节点连谁               | 节点**不直接连接对端节点**，而是将数据发往 Relay 服务器，由 Relay 转发至目标节点                                      |
| Relay 服务地址、端口、会话标识从哪来             | 通过 `peers` 接口**一并返回**。每个 peer 对象在 `recommend_mode=relay` 时，额外携带 `relay_addr`、`relay_port`、`relay_session_id` 字段（见下方扩展字段表） |
| Controller 只是"建议 relay"还是已分配 relay 资源 | Controller 在返回 `recommend_mode=relay` 时**已分配** Relay 会话资源，返回的 `relay_session_id` 即为已分配的会话凭证 |
| Controller 与 Relay 的关系 | Relay 是 Controller 的内部数据面模块，共享同一进程内会话表或同一内部存储，不定义额外外部 HTTP 接口。Controller 分配 `relay_session_id` 时，必须同步写入 Relay 可读取的会话映射 |

**`peers` 响应扩展字段（当 `recommend_mode=relay` 时出现）：**

| 字段名             | 类型   | 必填     | 说明                                                         |
|--------------------|--------|----------|--------------------------------------------------------------|
| relay_addr         | string | 条件必填 | Relay 服务器地址（IP 或域名）。仅 `recommend_mode=relay` 时返回 |
| relay_port         | number | 条件必填 | Relay 服务器 UDP 端口。仅 `recommend_mode=relay` 时返回         |
| relay_session_id   | uint32 | 条件必填 | Controller 分配的 Relay 会话标识，取值范围 `1` 到 `4294967295`。JSON 与二进制 Header 使用同一个无符号 32 位整数值 |

**Relay 模式 peers 响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "peers": [
      {
        "target_node_id": "nw-node-c3d4",
        "target_hostname": "linux-edge-02",
        "target_virtual_ip": "10.0.0.3",
        "target_public_ip": "198.51.100.12",
        "target_public_port": 45678,
        "nat_type": "Symmetric",
        "recommend_mode": "relay",
        "relay_addr": "<relay-server-host>",
        "relay_port": 9000,
        "relay_session_id": 2712847316
      }
    ]
  }
}
```

------------------------------------------------------------------------

## 4. 辅助接口

以下接口为调试或健康检查用途，不属于正式业务 API。

| 路径      | 方法 | 说明                 |
|-----------|------|----------------------|
| `/`       | GET  | 健康检查，返回 pong  |
| `/ping`   | GET  | 健康检查，返回 pong  |

> 注：`GET /nodes` 为历史辅助路由，正式场景请使用 `GET /api/v1/dashboard/nodes`。

------------------------------------------------------------------------

## 5. 建链协议

本章定义 Node 之间的建链流程，包括 STUN 探测、P2P 打洞、Relay 回退及状态同步。**Node 端实现必须遵循此状态机。**

### 5.1 建链时序

```
注册 → STUN 探测 → 拉取 Peers → 尝试 P2P 打洞 → 成功/失败 → Relay 回退 → 上报状态
```

| 步骤               | 触发时机                     | 说明                                                         |
|--------------------|------------------------------|--------------------------------------------------------------|
| 1. 注册            | Node 启动后立即执行          | 向 Controller 注册，获取 `node_id`、`virtual_ip`              |
| 2. STUN 探测       | 注册成功后立即执行            | 向 STUN 服务器发送 Binding Request，获取 `nat_type`、`public_ip`、`public_port` |
| 3. 拉取 Peers      | STUN 探测完成后立即执行       | 调用 `GET /api/v1/nodes/{node_id}/peers` 获取在线节点列表     |
| 4. P2P 打洞        | 拉取 Peers 后对每个 peer 执行 | 向对端 `(public_ip, public_port)` 发送 UDP 打洞包             |
| 5. 判定成功/失败   | 打洞重试结束后                | 见 5.2 节判定规则                                             |
| 6. Relay 回退      | P2P 失败后执行                | 连接 Relay 服务器，使用 `relay_session_id` 建立转发通道       |
| 7. 上报状态        | 建链完成后通过心跳上报        | 将 `connected_peer_ids` 和链路类型写入心跳                    |

### 5.2 P2P 打洞与 Relay 回退规则

| 规则项               | 规定值 / 行为                                                              |
|----------------------|---------------------------------------------------------------------------|
| STUN 探测发生时机     | **注册成功后**立即执行，早于拉取 Peers                                      |
| 打洞重试次数         | 每个 peer 最多重试 **3 次**                                                |
| 打洞超时             | 单次打洞等待响应 **2 秒**                                                   |
| 判定 P2P 失败条件    | 3 次重试均无响应，或收到 ICMP Port Unreachable                              |
| 自动回退 Relay 条件  | P2P 判定失败 + `peers` 返回中存在该 peer 的 `relay_session_id`               |
| 建链成功进入已连接   | 收到对端 `PUNCH_ACK`（P2P）或 `RELAY_CONFIRM`（Relay）后，将 peer 加入 `connected_peer_ids` |
| 断连后重新同步 peers | 检测到对端断连后，**立即**调用 `GET /api/v1/nodes/{node_id}/peers` 重新拉取列表，按最新 `recommend_mode` 重试建链 |
| 状态回写 Controller  | 建链/断链后，**下一次心跳**携带最新的 `connected_peer_ids` 和 `link_metrics` |

### 5.3 建链握手报文

建链握手报文使用 §6.2 定义的 NetWeaver Header，`Payload Length=0`，无 Inner IP Packet。解析端必须优先根据 `Message Type` 区分控制报文和普通数据报文。

| 报文类型 | Message Type | Session ID | 发送方向 | 用途 |
|----------|--------------|------------|----------|------|
| `PUNCH_PROBE` | `0x10` | `0x00000000` | Node A ↔ Node B，双方都向对端公网 `(public_ip, public_port)` 发送 | UDP 打洞探测包 |
| `PUNCH_ACK` | `0x11` | `0x00000000` | 收到 `PUNCH_PROBE` 的 Node 发回原发送方 | P2P 打洞确认包 |
| `RELAY_CONFIRM` | `0x20` | `relay_session_id` | Relay → Node A / Node B | Relay 已建立该会话转发表，节点可开始发送数据 |

**字段取值规则：**

| 字段 | `PUNCH_PROBE` | `PUNCH_ACK` | `RELAY_CONFIRM` |
|------|---------------|-------------|-----------------|
| `Src Virtual IP` | 发送方虚拟 IP | ACK 发送方虚拟 IP | Relay 可填 `0.0.0.0` |
| `Dst Virtual IP` | 目标 peer 虚拟 IP | 原 Probe 发送方虚拟 IP | 被确认节点虚拟 IP |
| `Payload Length` | `0` | `0` | `0` |
| `Checksum` | 按 §6.2 CRC 规则计算 | 按 §6.2 CRC 规则计算 | 按 §6.2 CRC 规则计算 |

**P2P 握手流程：**

```
1. 双方使用数据面 UDP socket，复用后续收发 TUN 数据的同一个本地源端口。
2. 双方各自向对端公网地址发送 PUNCH_PROBE，最多 3 次，每次间隔 2 秒。
3. 任一方收到 PUNCH_PROBE 后，立即向报文来源地址发送 PUNCH_ACK。
4. 节点收到对端 PUNCH_ACK 后，判定该 peer 的 P2P 建链成功。
5. 3 次 PUNCH_PROBE 后仍未收到 PUNCH_ACK，判定 P2P 失败并进入 Relay 回退。
```

**Relay 确认流程：**

```
1. Node 根据 peers 响应中的 relay_addr、relay_port、relay_session_id 向 Relay 发送 Message Type=KEEPALIVE 的空 Payload 报文。
2. Relay 根据 relay_session_id 查到该节点对会话后，向该节点返回 RELAY_CONFIRM。
3. Node 在 2 秒内收到 RELAY_CONFIRM 即判定 Relay 建链成功。
4. Node 最多重试 3 次；全部超时则该 peer 建链失败，不加入 connected_peer_ids。
```

### 5.4 建链状态机

```
                    ┌──────────┐
                    │  IDLE    │ 初始状态，未开始建链
                    └────┬─────┘
                         │ 拉取 peers 完成
                         ↓
                    ┌──────────┐
                    │  PROBING │ 正在 STUN 探测 / P2P 打洞中
                    └────┬─────┘
                         │
              ┌──────────┼──────────┐
              │ 成功                  │ 失败
              ↓                      ↓
        ┌──────────┐          ┌──────────┐
        │CONNECTED │          │  RELAY   │ 通过 Relay 转发
        │  (P2P)   │          └────┬─────┘
        └────┬─────┘               │ 心跳携带 connected_peer_ids
             │                     ↓
             │               ┌──────────┐
             └──────────────→│ SYNCED   │ 状态已同步至 Controller
                             └──────────┘
```

------------------------------------------------------------------------

## 6. 数据面封装协议

本章定义 TUN 报文在 Node ↔ Node（P2P）和 Node ↔ Relay 链路上的封装格式。

> **注意：** 压缩（`Flags.bit0`）与加密（`Flags.bit1`）当前均为保留位，暂不支持。生产部署时如需数据面加密，建议在上层使用 WireGuard 等成熟方案，而非自行实现。

### 6.1 封装概览

```
┌──────────────────────────────────────────────┐
│  IP Header (outer)                           │
├──────────────────────────────────────────────┤
│  UDP Header (src_port, dst_port)             │
├──────────────────────────────────────────────┤
│  NetWeaver Header (见 6.2)                    │
├──────────────────────────────────────────────┤
│  Inner IP Packet (原始 TUN 报文)              │
└──────────────────────────────────────────────┘
```

### 6.2 NetWeaver 报文头字段

| 字段          | 大小    | 说明                                                         |
|---------------|---------|--------------------------------------------------------------|
| Version       | 1 Byte  | 协议版本，当前为 `0x01`                                       |
| Message Type  | 1 Byte  | 报文类型，见下方类型表。解析端必须以该字段区分数据包、Keepalive 和握手控制包 |
| Flags         | 1 Byte  | bit0: 是否压缩（保留）, bit1: 是否加密（保留）, bit2: 保留兼容位。当前版本发送端必须置 0，接收端必须忽略 |
| Session ID    | 4 Bytes | 会话标识。P2P 模式为 `0x00000000`，Relay 模式为 `relay_session_id` 的 uint32 值 |
| Src Virtual IP| 4 Bytes | 源节点虚拟 IP（大端序）                                       |
| Dst Virtual IP| 4 Bytes | 目标节点虚拟 IP（大端序）                                     |
| Payload Length| 2 Bytes | 内层 IP 报文长度（不含 NetWeaver Header）                     |
| Checksum      | 2 Bytes | Header 校验和，使用 CRC-16/CCITT-FALSE                        |

**总计 Header 大小：19 Bytes。所有多字节整数均使用网络字节序（大端序）。**

**Message Type 取值：**

| 名称 | 值 | Payload | 说明 |
|------|----|---------|------|
| `DATA` | `0x01` | Inner IP Packet | 普通 TUN 数据报文 |
| `KEEPALIVE` | `0x02` | 空 | 数据面保活请求 |
| `KEEPALIVE_ACK` | `0x03` | 空 | 数据面保活响应 |
| `PUNCH_PROBE` | `0x10` | 空 | P2P UDP 打洞探测 |
| `PUNCH_ACK` | `0x11` | 空 | P2P UDP 打洞确认 |
| `RELAY_CONFIRM` | `0x20` | 空 | Relay 会话确认 |

**Checksum 算法：**

| 项目 | 规定 |
|------|------|
| 算法名 | `CRC-16/CCITT-FALSE` |
| 多项式 | `0x1021` |
| 初始值 | `0xFFFF` |
| 输入反射 | 否 |
| 输出反射 | 否 |
| 最终异或值 | `0x0000` |
| Checksum 字节序 | 网络字节序（大端序） |
| 计算范围 | 整个 NetWeaver Header，计算前先将 `Checksum` 字段置为 `0x0000`，不包含 Payload |

### 6.3 Node ↔ Node 数据面（P2P 直连）

当两个节点通过 UDP 打洞建立 P2P 直连后，TUN 报文按 6.1 节格式封装，经由 UDP 直接发送到对端公网地址。

| 项目               | 规定                                                         |
|--------------------|--------------------------------------------------------------|
| 封装格式           | `IP(outer) + UDP + NetWeaver Header + Inner IP Packet`       |
| Session ID         | `0x00000000`（表示 P2P 直连）                                 |
| Message Type       | 普通 TUN 数据使用 `DATA`；建链握手使用 `PUNCH_PROBE` / `PUNCH_ACK` |
| 目标地址           | 对端节点的 `(public_ip, public_port)`，由 peers 接口获取       |
| 发送端口           | Node 本地 UDP 监听端口（打洞时使用的同一端口）                 |

**P2P 发送流程：**

```
1. Node 从 TUN 设备读取 Inner IP Packet
2. 查找 Dst IP 对应的 peer node_id → (public_ip, public_port)
3. 构造 NetWeaver Header（Session ID=0, Src/Dst Virtual IP 填值）
4. 封装 UDP → 发送到对端 (public_ip, public_port)
```

**P2P 接收流程：**

```
1. Node UDP socket 收到报文
2. 校验 NetWeaver Header Checksum
3. 检查 Dst Virtual IP 是否为本节点虚拟 IP
4. 解出 Inner IP Packet，写入 TUN 设备
```

### 6.4 Node ↔ Relay 数据面（Relay 转发）

当 P2P 打洞失败，节点回退到 Relay 模式时，TUN 报文经由 Relay 服务器中转。

| 项目               | 规定                                                         |
|--------------------|--------------------------------------------------------------|
| 封装格式           | 与 P2P **完全相同**（`IP(outer) + UDP + NetWeaver Header + Inner IP Packet`） |
| Session ID         | `relay_session_id` 的 uint32 值（由 Controller 分配）           |
| Message Type       | 普通 TUN 数据使用 `DATA`；Relay 会话确认使用 `RELAY_CONFIRM`    |
| 目标地址           | Relay 服务器的 `(relay_addr, relay_port)`，由 peers 接口获取   |

**Relay 发送流程：**

```
1. Node 从 TUN 设备读取 Inner IP Packet
2. 查找 Dst IP 对应的 relay_session_id
3. 构造 NetWeaver Header（Session ID=relay_session_id, Src/Dst Virtual IP 填值）
4. 封装 UDP → 发送到 Relay (relay_addr, relay_port)
```

**Relay 转发流程：**

```
1. Relay 收到报文，解析 NetWeaver Header
2. 根据 Session ID 查表，找到目标节点的 (public_ip, public_port)
3. 保持封装不变，UDP 转发到目标节点
4. 目标节点收到后校验 Dst Virtual IP，解出 Inner IP Packet 写入 TUN
```

### 6.5 数据面公共约束

| 约束项             | 规定值 / 行为                                          |
|--------------------|--------------------------------------------------------|
| MTU 约束           | TUN 设备 MTU 建议 **1400 Bytes**（预留外层 IP(20)+UDP(8)+Header(19) = 47 Bytes 开销）。实际路径 MTU 由 PMTUD 动态发现 |
| Keepalive 包       | 数据面空闲超过 **30 秒**时，Node 向每个已连接 peer 发送 `Message Type=KEEPALIVE`、Payload 为空的报文 |
| Keepalive 响应     | 收到 Keepalive 包后立即回复 `Message Type=KEEPALIVE_ACK`、Payload 为空的报文 |
| 目标节点识别       | 由 NetWeaver Header 中的 `Dst Virtual IP` 字段识别。P2P 模式下 Node 自行判断，Relay 模式下由 Relay 服务器根据 Session ID 查表转发 |
| 分片策略           | NetWeaver 层不做分片。若 Inner IP Packet 超过路径 MTU，依赖内层 IP 层的分片机制 |
| 压缩               | 暂不支持（`Flags.bit0` 保留）。未来由独立的数据面扩展文档定义                     |
| 加密               | 暂不支持（`Flags.bit1` 保留）。生产部署建议在上层使用 WireGuard 等方案               |

------------------------------------------------------------------------

## 7. 运行配置与部署契约

### 7.1 Controller 配置项

| 配置项                 | 默认值              | 说明                                              |
|------------------------|---------------------|---------------------------------------------------|
| 监听地址               | `:8080`             | Controller HTTP 服务监听地址与端口                  |
| Overlay 网段           | `10.0.0.0/16`       | 虚拟内网 CIDR，Node 的 `virtual_ip` 从该网段分配     |
| 心跳间隔               | 15 秒               | 建议心跳间隔，通过注册响应 `keepalive_interval` 告知 Node |
| 心跳超时倍数           | 3x                  | 连续 3 个周期无心跳视为离线                          |
| 指标采样保留策略       | 1h 原始点 + 24h 聚合点 | 最近 1 小时保留 15 秒原始点；1 到 24 小时保留 1 分钟聚合点 |

### 7.2 Node 配置项

| 配置项                 | 格式                          | 说明                                              |
|------------------------|-------------------------------|---------------------------------------------------|
| Controller 地址        | `http://<host>:<port>`        | Node 连接 Controller 的完整 URL，如 `http://192.168.1.100:8080` |
| STUN 服务器地址列表    | `stun_servers[]`              | STUN 端点列表，每个元素为 `<host>:<port>`。正式 NAT 分类要求至少 **3 个条目**：`[S1P1, S1P2, S2P1]`，其中 `S1P1` 与 `S1P2` 为同一公网 STUN IP 的不同 UDP 端口，`S2P1` 为另一公网 STUN 端点 |
| Relay 服务器地址       | `<host>:<port>`               | Relay 服务器地址，也可由 `peers` 接口动态下发       |
| TUN 设备名             | `tuno`                        | 本地 TUN 虚拟网卡名称                               |

### 7.3 前端配置项

| 配置项 / 环境变量       | 说明                                                         |
|-------------------------|--------------------------------------------------------------|
| `VITE_PROXY_TARGET`     | Vite 开发服务器代理目标，默认 `http://127.0.0.1:8080`。开发时前端请求 `/api/*` 会被 Vite 代理转发至此地址 |
| `VITE_API_BASE_URL`     | 生产环境下的 API 基础 URL。生产部署时前端直接请求此地址         |

### 7.4 部署拓扑与联调约定

| 场景                       | 谁暴露服务                     | 谁来连接                                             |
|----------------------------|-------------------------------|------------------------------------------------------|
| 本机开发（前后端同机）      | Controller 监听 `127.0.0.1:8080` | Vite 代理 `127.0.0.1:8080`，Node 直连 `127.0.0.1:8080` |
| 局域网联调（多机）          | Controller 所在机器暴露 `0.0.0.0:8080` | 前端设 `VITE_PROXY_TARGET=http://<controller-lan-ip>:8080`，Node 设 Controller 地址为局域网 IP |
| 校园网 / 跨 NAT 联调        | Controller 需有公网可达地址（端口映射或云服务器） | Node 通过公网 IP 注册，前端通过公网 IP 或 VPN 访问 |

------------------------------------------------------------------------

## 8. 安全约定

### 8.1 鉴权机制

| 安全项                 | 规定                                                         |
|------------------------|--------------------------------------------------------------|
| 注册接口匿名性         | **不允许匿名注册**。Node 注册时需携带预共享密钥（Pre-Shared Key，以下简称 PSK） |
| 预共享密钥             | Controller 与所有 Node 预先配置同一个 PSK。PSK 通过 `Authorization: Bearer <psk>` Header 传递 |
| 鉴权失败返回           | HTTP `401 Unauthorized`，响应 `{ "code": 401, "msg": "unauthorized: invalid or missing PSK", "data": null }` |
| Dashboard 登录态       | **需要登录态**。Dashboard 通过 `POST /api/v1/auth/login` 获取 JWT Token，后续请求携带 `Authorization: Bearer <jwt>` |
| 管理接口权限体系       | Dashboard 接口与 Node 接口**分属不同权限体系**：Dashboard 使用 JWT（用户登录），Node 使用 PSK（机器身份） |
| Token 过期处理         | JWT 过期返回 `401`，Dashboard 跳转登录页；Node PSK 不设过期，重启后重新注册即可 |

### 8.2 安全边界

| 边界                   | 规定                                                         |
|------------------------|--------------------------------------------------------------|
| Node 间互信            | 同一 Controller 下的 Node 视为互信。跨 Controller 的 Node 互访不在本文档范围内 |
| 传输加密               | 控制面使用 HTTPS（生产环境强制）。数据面加密暂不支持（`Flags.bit1` 保留），见 §6.5 |
| PSK 泄露应对           | 更换 PSK 后，Controller 拒绝旧 PSK 的注册请求，已注册 Node 需重启并使用新 PSK |

------------------------------------------------------------------------

## 9. 平台与环境边界

| 支持项                 | 规定                                                         |
|------------------------|--------------------------------------------------------------|
| Linux                  | **唯一正式交付目标平台**。TUN 能力、Node 守护进程均以 Linux 为基准 |
| macOS                  | 开发调试兼容。TUN 能力通过 `/dev/tun` 或 `utun` 接口支持       |
| Windows                | **不在正式交付范围内**。Windows TUN 驱动（如 wintun）差异较大，暂不纳入开发计划 |
| 不同平台 TUN 能力      | 仅 Linux 要求完整支持 TUN 数据转发。macOS 仅用于 Controller / Dashboard 开发调试，不作为 Node 运行平台 |

> **注意：** 如果后续评审要求 Windows 支持，需单独评估 wintun 驱动集成工作量，并在数据面协议中增加 Windows 适配章节。

------------------------------------------------------------------------

## 10. 验收标准

以下为项目最终交付验收的**最小必须通过项**，全部通过方可视为"接口文档落地完成"。

| 序号 | 验收项                                           | 判定标准                                                                                                  |
|------|--------------------------------------------------|-----------------------------------------------------------------------------------------------------------|
| 1    | 跨网络注册                                       | 两个**不同局域网环境**的真实节点（如一台校园网、一台 4G 热点）能成功注册到同一个 Controller，Dashboard 可见两条记录             |
| 2    | Dashboard 显示真实节点                           | `GET /api/v1/dashboard/nodes` 返回至少 2 个 `status: "online"` 的节点，包含真实的 `public_ip`、`nat_type`，非模拟数据       |
| 3    | Dashboard 显示真实边                             | `GET /api/v1/dashboard/edges` 返回至少 1 条边，`type` 字段为 `"p2p"` 或 `"relay"`（非硬编码），能随建链/断链动态变化           |
| 4    | Dashboard 显示真实延迟曲线                       | `GET /api/v1/dashboard/nodes/{node_id}/metrics?target_id=xxx` 返回 `metrics` 数组，其中 `latency_ms` 为真实探测值，曲线点数 ≥ 10 |
| 5    | 虚拟 IP 互通                                     | 在节点 A 上 `ping <节点B的virtual_ip>` 能通（P2P 或 Relay 均可）；同一 1 分钟窗口内系统 ping 平均延迟与 Dashboard 延迟平均值误差不超过 `10ms` 或 `20%`（取较大容差） |
| 6    | 节点掉线后状态同步                               | 手动停止节点 B 的 Node 进程后，Dashboard 在 **60 秒内**显示节点 B `status: "offline"`，edges 中移除与 B 相关的边，`connected_peers` 归零 |
| 7    | 同一机器重复注册复用身份                         | 同一 `machine_id` 重启 Node 进程后，Controller 返回的 `node_id` 和 `virtual_ip` 与首次注册**完全一致**                         |
| 8    | 鉴权拦截                                         | 携带错误 PSK 注册时 Controller 返回 `401`，Dashboard 不带 JWT 访问时返回 `401`                                                |

------------------------------------------------------------------------

## 11. STUN 探测协议

本章定义 Node 与 STUN 服务器之间的交互规范，用于探测节点的 NAT 类型、公网 IP 和公网端口。Node 启动后必须完成 STUN 探测，将结果上报至 Controller。

### 11.1 协议概述

NetWeaver 使用标准的 **STUN (Session Traversal Utilities for NAT)** 协议进行 NAT 探测。正式 NAT 分类依赖 `stun_servers[]` 中的三个固定顺序端点，覆盖两类访问目标：同一 STUN IP 的不同端口、另一 STUN 端点。

| 项目             | 规定                                                         |
|------------------|--------------------------------------------------------------|
| 协议标准         | RFC 5389 (STUN)                                              |
| 传输层           | UDP                                                         |
| 默认端口         | 3478（标准 STUN 端口，部署时可通过配置修改）                  |
| 探测时机         | 节点注册成功后**立即执行**，早于拉取 Peers                     |
| 探测目的         | 获取 `nat_type`、`public_ip`、`public_port` 三个字段          |

**STUN 端点要求：**

| 名称 | 来源 | 要求 |
|------|------|------|
| `S1P1` | `stun_servers[0]` 主端口 | 必填，例如 `stun-a.example.com:3478` |
| `S1P2` | `stun_servers[1]` 备用端口 | 必填，必须与 `S1P1` 解析到同一公网 IP，但 UDP 端口不同，例如 `stun-a.example.com:3479` |
| `S2P1` | `stun_servers[2]` 主端口 | 必填，必须是另一公网 STUN 端点，例如 `stun-b.example.com:3478` |

### 11.2 STUN Binding 请求

Node 向 STUN 服务器发送 STUN Binding Request，格式遵循 RFC 5389。

**请求格式（RFC 5389 Binding Request）：**

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|0 0|     STUN Message Type = 0x0001 (Binding Request)         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Message Length                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Magic Cookie = 0x2112A442             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     Transaction ID (96 bits)                  |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### 11.3 STUN Binding 响应

**成功响应（Binding Success Response）：**

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|0 0|     STUN Message Type = 0x0101 (Binding Success)         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Message Length                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Magic Cookie                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     Transaction ID (96 bits)                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  XOR-MAPPED-ADDRESS Attribute (公网 IP + 端口)                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### 11.4 NAT 类型探测矩阵

Node 必须使用**同一个 UDP socket、同一个本地源端口**执行步骤 T1、T2、T3。不得在这些步骤之间新建 socket，否则端口映射结果不可比较。T4 仅用于端口受限验证，必须新建一个本地 UDP socket。

| 步骤 | 本地 socket | 目标 STUN 端点 | 目的 | 成功记录 |
|------|-------------|----------------|------|----------|
| T1 | Socket A | `S1P1` | 获取基准公网映射地址 | `mapped_1 = XOR-MAPPED-ADDRESS` |
| T2 | Socket A | `S1P2`（同一 STUN IP，不同端口） | 判断同 IP 不同端口下映射是否稳定 | `mapped_2 = XOR-MAPPED-ADDRESS` |
| T3 | Socket A | `S2P1`（另一 STUN 端点） | 判断跨 STUN 端点映射是否稳定 | `mapped_3 = XOR-MAPPED-ADDRESS` |
| T4 | Socket B（新建） | `S1P1` | 验证更换本地端口后的可达性和新映射 | `mapped_4 = XOR-MAPPED-ADDRESS` |

**判定规则按下表自上而下匹配，命中后停止：**

| 条件 | 判定 NAT 类型 |
|------|---------------|
| T1、T2、T3 均失败 | `"Unknown"` |
| T1 成功，且 `mapped_1` 与 Socket A 的本地 IP:Port 完全相同 | `"Full Cone"` |
| T1、T2、T3 均成功，且 `mapped_1 == mapped_2 == mapped_3` | `"Restricted Cone"` |
| T1、T2 成功，T3 成功但 `mapped_3 != mapped_1` | `"Symmetric"` |
| T1、T2 成功，T3 失败，T4 成功 | `"Port Restricted Cone"` |
| T1 成功，但 T2 或 T4 失败导致无法完成矩阵 | `"Unknown"` |

**上报字段选择：**

| 场景 | `public_ip` / `public_port` |
|------|-----------------------------|
| T1 成功 | 使用 `mapped_1` 的 IP 和 Port |
| T1 失败但后续步骤成功 | 使用第一个成功响应的 `XOR-MAPPED-ADDRESS` |
| 所有步骤失败 | `public_ip=""`，`public_port=0`，`nat_type="Unknown"` |

### 11.5 超时与重试

| 规则项               | 规定值 / 行为                                          |
|----------------------|--------------------------------------------------------|
| 单次 STUN 请求超时   | **3 秒**                                               |
| 最大重试次数         | **2 次**（共 3 次尝试）                                 |
| 全部重试失败后行为   | `nat_type` 设为 `"Unknown"`，`public_ip` 和 `public_port` 设为空字符串和 0，但仍可注册和心跳 |
| 探测结果有效期       | 无固定过期时间。NAT 类型变化时由下一次心跳自动更新      |

### 11.6 探测结果上报

STUN 探测完成后，Node 在**下一次心跳**中将结果写入对应字段：

```
POST /api/v1/nodes/{node_id}/heartbeat
{
  "nat_type": "Full Cone",           // ← STUN 探测结果
  "public_ip": "203.0.113.5",        // ← STUN XOR-MAPPED-ADDRESS IP
  "public_port": 54321,              // ← STUN XOR-MAPPED-ADDRESS Port
  ...
}
```

> **注意：** 如果 STUN 探测全部失败，仍必须发送心跳，`nat_type` 为 `"Unknown"`，`public_ip` 为空字符串，`public_port` 为 0。Controller 必须接受该心跳并将节点置为受限模式：只允许被推荐 Relay，不允许判定为可 P2P 节点。

------------------------------------------------------------------------

## 附录 A：版本历史

| 版本  | 日期       | 变更说明                                                                                   |
|-------|------------|-------------------------------------------------------------------------------------------|
| v1.1  | 2026-05    | 初始版本，定义 Dashboard/Node ↔ Controller 基础 HTTP 接口                                   |
| v1.2  | 2026-05    | 统一响应结构，修正字段命名，补全 nodes/stats 扩展字段                                        |
| v1.3  | 2026-05    | 明确注册幂等性、public_ip/public_port 主动上报、connected_peers 语义、recommend_mode 仅为建议 |
| v1.4  | 2026-05    | 文档自包含：补齐建链协议、数据面封装、STUN 探测协议、运行配置、安全约定、平台边界、验收标准 |
| v1.5  | 2026-05    | 清理外部文档引用，文档完全自包含；扩充数据面 P2P/Relay 双通道流程；STUN 探测独立成章 |
| v1.6  | 2026-05    | 新增 Dashboard 登录接口；link_metrics 补 link_type 字段；STUN 失败降级策略闭环；relay_session_id 类型规整；建链握手 ACK 协议定义 |
| v1.7  | 2026-05    | 认证总表；connected_peer_ids 必填；metrics 语义与状态规则收口；Relay 内部模块化；CRC-16/CCITT-FALSE；STUN 三端点矩阵；流量计数器复位语义；验收误差容差 |
| v1.8  | 2026-05    | 修正标题版本号；补充 metrics timestamp 生成语义；版本历史收口 |
