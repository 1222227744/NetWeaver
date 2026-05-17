# NetWeaver API 接口文档 v1.3

## 1. 概述与全局规范

本文档定义了 NetWeaver 异地组网平台中，**控制平面（Controller）**与**管理平面（前端 Dashboard）**、**数据平面（Node）**之间的通信接口。

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
> Controller 收到 `link_metrics` 后，按 `(source_node_id, target_node_id)` 维度维护延迟时间序列，作为 `GET /api/v1/dashboard/metrics` 接口的真实数据来源。

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

------------------------------------------------------------------------

## 4. 辅助接口

以下接口为调试或健康检查用途，不属于正式业务 API。

| 路径      | 方法 | 说明                 |
|-----------|------|----------------------|
| `/`       | GET  | 健康检查，返回 pong  |
| `/ping`   | GET  | 健康检查，返回 pong  |

> 注：`GET /nodes` 为历史辅助路由，正式场景请使用 `GET /api/v1/dashboard/nodes`。


