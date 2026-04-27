## 1. 概述与全局规范

本文档定义了 NetWeaver 异地组网平台中，**控制平面（Controller）**与**管理平面（前端 Dashboard）**、**数据平面（Node）**之间的通信接口。

**全局基础响应结构（JSON）：**

JSON

    {
      "code": 200,          // 业务状态码（200为成功，400+为客户端错误，500+为服务端错误）
      "msg": "success",     // 状态描述或错误提示
      "data": {}            // 具体的业务数据负载
    }

------------------------------------------------------------------------

## 2. 前端 (Dashboard) \<-\> Controller 接口

该部分接口主要服务于基于 Vue 3 构建的 Web 管理控制台，用于呈现网络拓扑和管理节点。

### 2.1 获取全局网络状态统计

用于 Dashboard 首页的数据大屏展示。

- **请求方式 & 路径:** `GET /api/v1/dashboard/stats`

- **响应 JSON:**

JSON

    {
      "code": 200,
      "msg": "success",
      "data": {
        "total_nodes": 12,
        "online_nodes": 10,
        "total_traffic_gb": 45.2,
        "controller_uptime_sec": 86400
      }
    }

### 2.2 获取网络拓扑节点列表

拉取当前私有网络内的所有节点信息，以便前端使用图形库实时渲染网络拓扑图。

- **请求方式 & 路径:** `GET /api/v1/dashboard/nodes`

- **响应 JSON:**

JSON

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
            "connected_peers": 3
          },
          {
            "node_id": "nw-node-c3d4",
            "hostname": "macbook-pro",
            "virtual_ip": "10.0.0.3",
            "public_ip": "198.51.100.12",
            "nat_type": "Symmetric",
            "status": "online",
            "connected_peers": 3
          }
        ]
      }
    }

------------------------------------------------------------------------

## 3. Node \<-\> Controller 接口

该部分接口服务于数据平面节点。节点使用 Go/C++ 等底层程序调用，用于身份认证、NAT 穿透信令交换和心跳保活。

### 3.1 节点注册与接入 (Node Register)

节点首次启动时，向信令服务器汇报自身信息并请求分配虚拟内网 IP。

- **请求方式 & 路径:** `POST /api/v1/nodes/register`

- **请求 JSON:**

JSON

    {
      "machine_id": "e4:5f:01:aa:bb:cc",
      "hostname": "ubuntu-server-01",
      "os": "linux",
      "local_ip": "192.168.1.10"
    }

- **响应 JSON:**

JSON

    {
      "code": 200,
      "msg": "success",
      "data": {
        "node_id": "nw-node-a1b2",
        "virtual_ip": "10.0.0.2",     // Controller分配的TUN网卡IP
        "subnet_mask": "255.255.255.0",
        "keepalive_interval": 15      // 建议的心跳间隔(秒)
      }
    }

### 3.2 节点心跳与状态上报 (Heartbeat)

节点定期汇报自身存活状态及 NAT 探测结果，帮助 Controller 判断是否需要下发 UDP 打洞指令或回退到 Relay（中转）模式。

- **请求方式 & 路径:** `POST /api/v1/nodes/{node_id}/heartbeat`

- **请求 JSON:**

JSON

    {
      "nat_type": "Full Cone",      // STUN探测出的NAT类型
      "public_ip": "203.0.113.5",
      "public_port": 54321,         // 外部映射端口
      "current_rx_bytes": 1024560,
      "current_tx_bytes": 2048000
    }

- **响应 JSON:**

JSON

    {
      "code": 200,
      "msg": "success",
      "data": {
        "action_required": "none"   // 告知节点无需额外操作，或下发"sync_peers"指令
      }
    }

### 3.3 拉取对等节点路由表 (Get Peers)

节点获取局域网内其他所有在线节点的信息，用于在本地建立 P2P 连接（UDP 打洞）或建立转发隧道。

- **请求方式 & 路径:** `GET /api/v1/nodes/{node_id}/peers`

- **响应 JSON:**

JSON

    {
      "code": 200,
      "msg": "success",
      "data": {
        "peers": [
          {
            "target_virtual_ip": "10.0.0.3",
            "target_public_ip": "198.51.100.12",
            "target_public_port": 45678,
            "nat_type": "Symmetric",
            "recommend_mode": "relay"   // 因为目标是对称型NAT，Controller直接建议走Relay模式
          },
          {
            "target_virtual_ip": "10.0.0.4",
            "target_public_ip": "114.114.114.114",
            "target_public_port": 33445,
            "nat_type": "Full Cone",
            "recommend_mode": "p2p"     // 建议直接UDP打洞
          }
        ]
      }
    }

### 一些学习

### 1. 通信范式理论：Pull (拉) vs. Push (推)

RESTful 本质上是基于 HTTP 的**请求-响应（Request-Response）模型**，这是一种典型的 **Pull（拉取）** 机制。客户端不发请求，服务端就不能主动给客户端发数据。

- **理论冲突点：** 根据 NetWeaver 的架构设计，Controller 需要向 Node **下发** UDP 打洞指令。如果只用 RESTful，Node 就必须通过高频的心跳（轮询 Polling）去问 Controller："有我的新指令吗？" 这会极大地浪费带宽和 Controller 的并发性能。

- **补充知识体系：** \* **WebSocket / 长连接：** 允许全双工通信，Controller 可以随时主动把打洞指令 **Push** 给 Node。

  - **SSE (Server-Sent Events)：** 如果 Node 只需要单向接收指令，可以使用 SSE，它是基于 HTTP 的轻量级服务端推送技术。

  - **RPC (远程过程调用)：** 考虑到你们未来可能用 Rust、C++ 等系统级语言去写底层数据面处理 TUN/TAP 流量，节点内部的通信或严谨的信令交互，未来可以考虑升级为 gRPC，它基于 HTTP/2，不仅支持双向流，且序列化体积比 JSON 小得多。

### 2. 状态理论：Stateless (无状态) 与 Session (会话)

RESTful 架构的一个核心原则是**无状态（Stateless）**。这意味着服务器不会在两次请求之间保存客户端的状态，每一次请求都必须包含服务器理解该请求所需的所有信息。

- **业务应用：** 虽然 API 是无状态的，但你的 Controller 在业务逻辑上是**有状态**的（它必须知道哪些 Node 在线，NAT 类型是什么）。

- **设计启示：** 你在设计心跳接口（Heartbeat）时，就是在使用无状态的 API 去维持系统底层的状态机。这就要求心跳接口必须设计得极其轻量，且具有**容错性**（例如 Node 漏发了一次心跳，Controller 不能立刻判定其掉线，需要有超时剔除的机制）。

### 3. 接口演进与兼容性理论 (API Evolution)

在敏捷开发中，API 永远是在迭代的。

- **向后兼容 (Backward Compatibility)：** 当你作为 PM 决定在 v1.1 版本给接口新增一个字段时，绝对不能让还在运行 v1.0 代码的节点崩溃。

- **宽泛输入，严格输出 (Postel's Law / 鲁棒性原则)：** 设计 API 解析逻辑时，对 Node 或前端传来的冗余数据要有包容性（忽略不认识的 JSON 字段），但在 Controller 返回数据时，必须严格遵守契约，不多发、不乱发。

### 4. API 安全与鉴权理论 (Authentication & Authorization)

在这个私有网络平台中，随便一个设备调用接口就能接入网络是非常危险的。

- **JWT (JSON Web Token)：** 这是一种非常流行的无状态鉴权方案。Node 首次注册时提供机器码或预共享密钥，Controller 颁发一个 JWT。后续 Node 的每一次请求都在 HTTP Header 里带上这个 Token（`Authorization: Bearer <token>`），Controller 只需要校验 Token 的签名即可，无需查询数据库，性能极高。
