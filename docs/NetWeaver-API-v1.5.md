# NetWeaver API 接口文档 v1.5

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

------------------------------------------------------------------------

## 2. 前端 (Dashboard) \<-\> Controller 接口

该部分接口主要服务于基于 Vue 3 构建的 Web 管理控制台，用于呈现网络拓扑和管理节点。

### 2.1 获取全局网络状态统计

用于 Dashboard 首页的数据大屏展示（顶部统计卡片）。

- **请求方式 & 路径:** `GET /api/v1/dashboard/stats`
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
        "hostname": "macbook-pro",
        "virtual_ip": "10.0.0.3",
        "public_ip": "198.51.100.12",
        "nat_type": "Symmetric",
        "status": "online",
        "connected_peers": 1,
        "machine_id": "aa:bb:cc:dd:ee:ff",
        "os": "darwin",
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
| 缺少必填参数 `target_id`                      | 400       | `"target_id is required"` | `null`                       |
| `time_range` 非法值（非 `1h`/`12h`/`24h`）    | 400       | `"invalid time_range"` | `null`                          |
| 两节点均存在，但从未建立过真实链路              | 200       | `"success"`            | `{ "metrics": [] }`             |
| 有真实链路，但当前时间范围内无采样点            | 200       | `"success"`            | `{ "metrics": [] }`             |
| 正常查询，有数据                              | 200       | `"success"`            | `{ "metrics": [...] }`          |

**数据排序与采样规则：**

| 规则项               | 规定值                                                                 |
|----------------------|------------------------------------------------------------------------|
| 返回结果排序         | 按 `timestamp` **升序**排列（旧数据在前，新数据在后）                    |
| 采样周期             | 与心跳间隔一致，默认 **15 秒**一个采样点                                |
| 数据保留时长         | 每个 `(source, target)` 保留最近 **120 条**采样点（约 30 分钟数据）     |
| 延迟数据来源         | 节点间**主动探测**（ICMP ping 或 UDP probe），通过心跳的 `link_metrics` 字段上报至 Controller |

> **设计说明：** 当 Dashboard 请求两个存在但无真实链路的节点对的 metrics 时，返回 `200 + metrics: []` 而非 `404`。这是因为节点和链路是两个维度的资源——节点存在是事实，无链路数据只是当前无采样点。前端收到空数组时应展示"暂无数据"而非报错。

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
| `edges.type` 由谁决定    | 由心跳中 `link_metrics` 附带或独立上报的连接类型字段决定，Controller 不推断 |
| `connected_peers` 与 `connected_peer_ids` 冲突 | 以 `connected_peer_ids` 为准。`connected_peers` 数值仅作为辅助参考       |

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
    "subnet_mask": "255.255.255.0",
    "keepalive_interval": 15
  }
}
```

> **幂等性说明：** 同一个 `machine_id` 重复注册时，Controller **必须复用**原来已分配的 `node_id` 和 `virtual_ip`，不得重新分配新 ID 或新 IP。即：注册接口对同一 `machine_id` 是幂等的，返回值中 `node_id` 和 `virtual_ip` 应与首次注册时一致。节点重启或网络中断后重新注册时，依赖此语义恢复原有身份。

### 3.2 节点心跳与状态上报 (Heartbeat)

节点定期汇报自身存活状态、NAT 探测结果及已连接邻居信息。Controller 根据这些信息判断是否需要下发同步对端指令或回退到 Relay 模式。

- **请求方式 & 路径:** `POST /api/v1/nodes/{node_id}/heartbeat`

**请求字段说明：**

| 字段名              | 类型     | 必填 | 说明                                          |
|---------------------|----------|------|-----------------------------------------------|
| nat_type            | string   | 是   | STUN 探测出的 NAT 类型，详见 1.3 节枚举值       |
| public_ip           | string   | 是   | 公网出口 IP。节点必须主动上报，Controller 不推断 |
| public_port         | number   | 是   | 公网映射端口。节点必须主动上报，Controller 不推断 |
| current_rx_bytes    | number   | 是   | 当前累计接收字节数                              |
| current_tx_bytes    | number   | 是   | 当前累计发送字节数                              |
| connected_peers     | number   | 否   | 节点自报的已连接邻居数                          |
| connected_peer_ids  | string[] | 否   | 节点自报的已连接邻居 node_id 列表，用于构建真实边 |
| link_metrics        | object[] | 否   | 到各邻居的链路延迟指标，用于构建链路延迟时序数据  |

> **`public_ip` / `public_port` 上报策略：** 节点**必须主动上报** `public_ip` 和 `public_port`（通过 STUN 探测获得），Controller **不负责推断**这两个字段。若节点未上报或上报值为空，Controller 应返回 `400` 错误。此策略避免了前后端和节点端对"缺失时由谁补全"的理解分歧。

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
    { "target_node_id": "nw-node-c3d4", "latency_ms": 42.1 },
    { "target_node_id": "nw-node-e5f6", "latency_ms": 18.7 }
  ]
}
```

> **`link_metrics` 子字段说明：**
> 
> | 字段名          | 类型   | 必填 | 说明                         |
> |-----------------|--------|------|------------------------------|
> | target_node_id  | string | 是   | 目标邻居节点 ID               |
> | latency_ms      | number | 是   | 到该邻居的延迟，单位毫秒       |
> 
> Controller 收到 `link_metrics` 后，按 `(source_node_id, target_node_id)` 维度维护延迟时间序列，作为 `GET /api/v1/dashboard/nodes/{node_id}/metrics` 接口的真实数据来源。延迟数据来自节点间**主动探测**（如 ICMP ping 或 UDP probe），而非心跳附带采集。

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
| `connected_peers` 省略 | 允许省略（字段为 `否` 必填）。省略时 Controller 以 `connected_peer_ids.length` 为准 |
| `connected_peers` 与 `connected_peer_ids.length` 不一致 | 以 `connected_peer_ids.length` 为准，忽略 `connected_peers` 数值 |
| `link_metrics` 上报频率 | 每次心跳均可携带。建议与心跳同频上报，无需额外定时器                       |
| `link_metrics` 部分邻居不上报 | 允许。仅上报已探测到延迟数据的邻居，未上报的邻居不产生数据点            |
| Controller 历史采样保留 | 每个 `(source_node_id, target_node_id)` 保留最近 **120 条**采样点           |
| 同一时间窗口重复上报 | 后上报的值**覆盖**先上报的值（按 `timestamp` 去重，保留最新）                |
| `connected_peer_ids` 含无效 ID | 忽略不存在的 peer ID，不计入 `connected_peers`，不建立边                    |

### 3.3 拉取对等节点路由表 (Get Peers)

节点获取局域网内其他所有在线节点的信息，用于在本地建立 P2P 连接（UDP 打洞）或建立转发隧道。

- **请求方式 & 路径:** `GET /api/v1/nodes/{node_id}/peers`

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
        "target_hostname": "macbook-pro",
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

#### 3.3.1 Relay 模式支撑说明

当 `recommend_mode = "relay"` 时，节点无法通过 UDP 打洞建立 P2P 直连，数据必须经由 Relay 服务器转发。以下规定 Relay 模式下的关键行为：

| 问题                                           | 规定                                                                                                              |
|------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| `recommend_mode=relay` 时节点连谁               | 节点**不直接连接对端节点**，而是将数据发往 Relay 服务器，由 Relay 转发至目标节点                                      |
| Relay 服务地址、端口、会话标识从哪来             | 通过 `peers` 接口**一并返回**。每个 peer 对象在 `recommend_mode=relay` 时，额外携带 `relay_addr`、`relay_port`、`relay_session_id` 字段（见下方扩展字段表） |
| Controller 只是"建议 relay"还是已分配 relay 资源 | Controller 在返回 `recommend_mode=relay` 时**已同步分配** Relay 会话资源。即：Controller 已与 Relay 服务器协商好该节点对的转发通道，返回的 `relay_session_id` 即为已分配的会话凭证 |

**`peers` 响应扩展字段（当 `recommend_mode=relay` 时出现）：**

| 字段名             | 类型   | 必填     | 说明                                                         |
|--------------------|--------|----------|--------------------------------------------------------------|
| relay_addr         | string | 条件必填 | Relay 服务器地址（IP 或域名）。仅 `recommend_mode=relay` 时返回 |
| relay_port         | number | 条件必填 | Relay 服务器 UDP 端口。仅 `recommend_mode=relay` 时返回         |
| relay_session_id   | string | 条件必填 | Controller 分配的 Relay 会话标识，节点发往 Relay 时必须携带      |

**Relay 模式 peers 响应示例：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "peers": [
      {
        "target_node_id": "nw-node-c3d4",
        "target_hostname": "macbook-pro",
        "target_virtual_ip": "10.0.0.3",
        "target_public_ip": "198.51.100.12",
        "target_public_port": 45678,
        "nat_type": "Symmetric",
        "recommend_mode": "relay",
        "relay_addr": "<relay-server-host>",
        "relay_port": 9000,
        "relay_session_id": "sess-a1b2c3d4"
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
| 建链成功进入已连接   | 收到对端 UDP ACK（P2P）或 Relay 转发确认（Relay）后，将 peer 加入 `connected_peer_ids` |
| 断连后重新同步 peers | 检测到对端断连后，**立即**调用 `GET /api/v1/nodes/{node_id}/peers` 重新拉取列表，按最新 `recommend_mode` 重试建链 |
| 状态回写 Controller  | 建链/断链后，**下一次心跳**携带最新的 `connected_peer_ids` 和 `link_metrics` |

### 5.3 建链状态机

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
| Flags         | 1 Byte  | bit0: 是否压缩, bit1: 是否加密, bit2: 是否 Keepalive           |
| Session ID    | 4 Bytes | 会话标识。P2P 模式为 `0x00000000`，Relay 模式为 `relay_session_id` 的低 32 位 |
| Src Virtual IP| 4 Bytes | 源节点虚拟 IP（大端序）                                       |
| Dst Virtual IP| 4 Bytes | 目标节点虚拟 IP（大端序）                                     |
| Payload Length| 2 Bytes | 内层 IP 报文长度（不含 NetWeaver Header）                     |
| Checksum      | 2 Bytes | Header 校验和（CRC-16）                                       |

**总计 Header 大小：18 Bytes。**

### 6.3 Node ↔ Node 数据面（P2P 直连）

当两个节点通过 UDP 打洞建立 P2P 直连后，TUN 报文按 6.1 节格式封装，经由 UDP 直接发送到对端公网地址。

| 项目               | 规定                                                         |
|--------------------|--------------------------------------------------------------|
| 封装格式           | `IP(outer) + UDP + NetWeaver Header + Inner IP Packet`       |
| Session ID         | `0x00000000`（表示 P2P 直连）                                 |
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
| Session ID         | `relay_session_id` 的低 32 位（由 Controller 分配）            |
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
| MTU 约束           | TUN 设备 MTU 建议 **1400 Bytes**（预留外层 IP(20)+UDP(8)+Header(18) = 46 Bytes 开销）。实际路径 MTU 由 PMTUD 动态发现 |
| Keepalive 包       | 数据面空闲超过 **30 秒**时，Node 向每个已连接 peer 发送 Keepalive 包（`Flags.bit2=1`，Payload 为空） |
| Keepalive 响应     | 收到 Keepalive 包后立即回复一个空 Payload 的 ACK 包（`Flags.bit2=1`） |
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
| 指标采样保留数         | 120 条/链路          | 每条链路的延迟采样保留上限                           |

### 7.2 Node 配置项

| 配置项                 | 格式                          | 说明                                              |
|------------------------|-------------------------------|---------------------------------------------------|
| Controller 地址        | `http://<host>:<port>`        | Node 连接 Controller 的完整 URL，如 `http://192.168.1.100:8080` |
| STUN 服务器地址        | `<host>:<port>`               | STUN 服务器地址，如 `<stun-host>:3478`（具体地址由部署时配置） |
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
| Dashboard 登录态       | **需要登录态**。Dashboard 通过独立的登录接口获取 JWT Token，后续请求携带 `Authorization: Bearer <jwt>` |
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
| 5    | 虚拟 IP 互通                                     | 在节点 A 上 `ping <节点B的virtual_ip>` 能通（P2P 或 Relay 均可），ping 延迟与 Dashboard 显示的延迟一致                         |
| 6    | 节点掉线后状态同步                               | 手动停止节点 B 的 Node 进程后，Dashboard 在 **60 秒内**显示节点 B `status: "offline"`，edges 中移除与 B 相关的边，`connected_peers` 归零 |
| 7    | 同一机器重复注册复用身份                         | 同一 `machine_id` 重启 Node 进程后，Controller 返回的 `node_id` 和 `virtual_ip` 与首次注册**完全一致**                         |
| 8    | 鉴权拦截                                         | 携带错误 PSK 注册时 Controller 返回 `401`，Dashboard 不带 JWT 访问时返回 `401`                                                |

------------------------------------------------------------------------

## 11. STUN 探测协议

本章定义 Node 与 STUN 服务器之间的交互规范，用于探测节点的 NAT 类型、公网 IP 和公网端口。Node 启动后必须完成 STUN 探测，将结果上报至 Controller。

### 11.1 协议概述

NetWeaver 使用标准的 **STUN (Session Traversal Utilities for NAT)** 协议进行 NAT 探测：

| 项目             | 规定                                                         |
|------------------|--------------------------------------------------------------|
| 协议标准         | RFC 5389 (STUN)                                              |
| 传输层           | UDP                                                         |
| 默认端口         | 3478（标准 STUN 端口，部署时可通过配置修改）                  |
| 探测时机         | 节点注册成功后**立即执行**，早于拉取 Peers                     |
| 探测目的         | 获取 `nat_type`、`public_ip`、`public_port` 三个字段          |

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

**NAT 类型判定规则：**

| STUN 探测结果                                              | 判定 NAT 类型            |
|-----------------------------------------------------------|--------------------------|
| Binding Response 中 `XOR-MAPPED-ADDRESS` 与本地 socket 地址相同 | `"Full Cone"`            |
| 向另一 STUN 服务器 IP:Port 发送请求，收到响应但 mapped 不同   | `"Restricted Cone"`      |
| 同上，且需更换本地端口才能收到响应                            | `"Port Restricted Cone"` |
| 向另一 STUN 服务器发送请求，无响应                           | `"Symmetric"`            |
| 探测失败或超时                                             | `"Unknown"`              |

### 11.4 超时与重试

| 规则项               | 规定值 / 行为                                          |
|----------------------|--------------------------------------------------------|
| 单次 STUN 请求超时   | **3 秒**                                               |
| 最大重试次数         | **2 次**（共 3 次尝试）                                 |
| 全部重试失败后行为   | `nat_type` 设为 `"Unknown"`，`public_ip` 和 `public_port` 设为空字符串和 0，但仍可注册和心跳 |
| 探测结果有效期       | 无固定过期时间。NAT 类型变化时由下一次心跳自动更新      |

### 11.5 探测结果上报

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

> **注意：** 如果 STUN 探测全部失败，仍必须发送心跳，`nat_type` 为 `"Unknown"`，`public_ip` 为空字符串，`public_port` 为 0。Controller 收到后返回 `400` 要求重试（见 §3.2）。

------------------------------------------------------------------------

## 附录 A：版本历史

| 版本  | 日期       | 变更说明                                                                                   |
|-------|------------|-------------------------------------------------------------------------------------------|
| v1.1  | 2026-05    | 初始版本，定义 Dashboard/Node ↔ Controller 基础 HTTP 接口                                   |
| v1.2  | 2026-05    | 统一响应结构，修正字段命名，补全 nodes/stats 扩展字段                                        |
| v1.3  | 2026-05    | 明确注册幂等性、public_ip/public_port 主动上报、connected_peers 语义、recommend_mode 仅为建议 |
| v1.4  | 2026-05    | 文档自包含：补齐建链协议、数据面封装、STUN 探测协议、运行配置、安全约定、平台边界、验收标准 |
| v1.5  | 2026-05    | 清理外部文档引用，文档完全自包含；扩充数据面 P2P/Relay 双通道流程；STUN 探测独立成章 |


