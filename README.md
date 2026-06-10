# NetWeaver 仓库全量扫描记录

本文件不是宣传页，而是一次基于当前仓库实际文件的“项目现状盘点”。它的目标只有 4 个：

1. 说清楚当前仓库里到底有哪些文件、每个文件负责什么。
2. 说清楚哪些功能已经是代码落地，哪些还只是规划、演示或半闭环状态。
3. 说清楚前端、后端、文档之间目前哪里已经对齐，哪里还没有对齐。
4. 给后续继续开发的人一个统一、准确、可追溯的项目入口。

## 2026-05-24 最新进度补充

当前前端已经按 `docs/NetWeaver-API-v1.8.md` 进入收尾落地状态。本次前端分支为：

```text
feature/fronted-1222227744-前端收尾迭代-v1.8接口落地
```

本次前端已完成：

- 新增 Dashboard 登录流程：`POST /api/v1/auth/login`
- 登录成功后将 JWT 保存到浏览器本地，并在访问 `/api/v1/dashboard/*` 时自动添加 `Authorization: Bearer <jwt>`
- 统计卡片继续消费 `GET /api/v1/dashboard/stats`
- 在线节点表格和关系图节点继续消费 `GET /api/v1/dashboard/nodes`
- 关系图连线继续消费 `GET /api/v1/dashboard/edges`
- 链路延迟图改为悬停或点击真实连线后请求 `GET /api/v1/dashboard/nodes/{source}/metrics?target_id={target}`
- 已删除“当前节点查询自己”的 metrics 临时策略，前端不再发起 `node_id == target_id` 的错误请求
- 补齐登录过期、后端无响应、无在线节点、无真实边、无 metrics 采样点等页面状态

当前前端判断标准：

- 如果后端已经按 v1.8 实现登录、JWT、stats、nodes、edges、metrics，前端可以直接进入完整联调
- 如果页面登录失败，优先检查 `POST /api/v1/auth/login`
- 如果登录后控制台接口返回 401，优先检查后端 JWT 校验与前端请求头
- 如果图上没有连线，优先检查 `GET /api/v1/dashboard/edges` 是否返回真实 `{ source, target, type }`
- 如果连线有了但没有曲线，优先检查 metrics 是否使用 `source -> target` 查询并返回 `metrics` 数组

后续项目事实以 `docs/NetWeaver-API-v1.8.md` 为接口契约基准；较早文档和较早 README 段落中的 v1.1/v1.7 表述只作为历史记录。

本次扫描时间：`2026-05-12`  
本次扫描分支：`dev`  
本次扫描范围：仓库内 `backend/`、`frontend/`、`docs/`、根目录配置与说明文件  
本次扫描文件数：`43` 个非依赖目录文件  

说明：

- 本次已经逐个阅读全部源码、配置和 Markdown / TeX 说明文件。
- `frontend/node_modules/` 不属于项目源码，因此不纳入正文分析。
- `frontend/package-lock.json`、`backend/code/go.sum` 属于依赖锁定文件，已计入文件清单，但不逐行展开业务解释。
- `docs/Week1_Report.pdf` 属于编译后的二进制文档产物，功能说明以 `docs/Week1_Report.tex` 为准。

## 一句话结论

NetWeaver 当前已经不是“空仓库”或“只有想法”的状态。它已经具备：

- 前端管理后台骨架
- 真实 HTTP 请求层
- 统计卡片 / 在线节点表格 / 节点关系图 / 悬浮详情卡片
- Controller 基础服务
- Node 自动注册、心跳、拉取 peers 的基础闭环
- TUN 抓包实验
- UDP 双节点互发实验
- 协作规约、接口文档、阶段汇报与 AI 规划资料

但它还没有进入“完整成品”状态。当前最核心的问题不是“再补一页页面”，而是：

- 接口文档、后端实现、前端消费三者还没有完全对齐
- 真实关系边 `edges` 还不是由真实连接状态驱动
- 真实链路延迟 `metrics` 还没有做成文档定义的格式和语义
- `TUN -> P2P/Relay -> TUN` 的完整数据平面还没有真正串起来

换句话说，当前项目处于“控制平面基本成型，数据平面仍在实验和拼接阶段”的状态。

## 仓库总览

当前仓库顶层结构如下：

```text
NetWeaver/
├── backend/                # Go 后端代码与节点程序
├── docs/                   # 协作规约、接口文档、周报、AI 规划资料
├── frontend/               # Vue 3 管理后台
├── .gitignore              # 按前端 / 后端 / 文档分区管理的忽略规则
└── README.md               # 当前这份全量扫描记录
```

需要特别说明的结构事实：

- `docs/协作规约.md` 中提到的 `resources/`、`scripts/`、`config.json`、根目录 `.env` 目前并没有形成稳定的实际仓库结构。
- `backend/code/README.md` 中提到“根目录已配置 `go.work`”，但本次扫描没有在仓库中看到该文件。
- `frontend/README.md` 中有部分历史记录引用了已经不在当前仓库中的旧文档名，这些内容属于历史记录，不应再被当作当前真相。

## 文件清单

下面按目录把当前可见文件全部列出来，并说明用途与现状。

### 1. 根目录

| 文件 | 作用 | 当前状态 |
| --- | --- | --- |
| `.gitignore` | 管理全局、前端、后端、文档三部分的忽略规则 | 已按目录分区整理，规则清晰 |
| `README.md` | 项目总览与全量盘点 | 本次重写更新 |

### 2. docs 目录

| 文件 | 作用 | 当前状态 |
| --- | --- | --- |
| `docs/协作规约.md` | 约定分支、提交、目录、评审与协作方式 | 已成型，可作为团队协作准绳 |
| `docs/NetWeaver-API-v1.8.md` | 当前最新接口契约，覆盖 Dashboard / Node / 建链 / 数据面 / STUN / 验收标准 | 已作为当前项目落地基准 |
| `docs/Week3_PM_Report.md` | PM / 文档方向的周报，已包含 NAT 穿透流程与背景分析 | 内容较实，偏方案与论证 |
| `docs/Week1_Report.tex` | LaTeX 报告骨架 | 仅为 Week 1 骨架，不是完整论文 |
| `docs/Week1_Report.pdf` | `Week1_Report.tex` 编译产物 | 文档成品产物，不是源码 |
| `docs/ai对话内容.txt` | 大量 AI 规划记录与阶段安排素材 | 信息量大，适合做路线追溯，不适合作为唯一事实来源 |

文档侧当前结论：

- 协作规范和接口契约都已经有文档基础。
- PM 方向已经开始沉淀背景、意义、竞品痛点和 NAT 穿透流程。
- 论文材料处于“有骨架、有片段内容，但远未成稿”的阶段。

### 3. backend/code 目录

| 文件 | 作用 | 当前状态 |
| --- | --- | --- |
| `backend/code/go.mod` | Go 模块定义 | 已配置 Gin 与 water 等依赖 |
| `backend/code/go.sum` | Go 依赖锁定文件 | 锁文件，正常存在 |
| `backend/code/README.md` | 后端使用说明 | 有价值，但包含部分过时或环境依赖表述 |
| `backend/code/cmd/controller/main.go` | Controller 启动入口 | 已能读取监听地址并启动 Gin 服务 |
| `backend/code/cmd/node/main.go` | Node 程序总入口 | 已支持 `tun`、`run`、`p2p` 三种模式 |
| `backend/code/internal/controller/api/router.go` | Controller 路由与 handler | 已实现多条接口，但仍有文档不一致问题 |
| `backend/code/internal/controller/middleware/cors.go` | CORS 中间件 | 已实现基础跨域放行 |
| `backend/code/internal/controller/state/registry.go` | Controller 内存状态中心 | 已管理节点、心跳、统计、边与 metrics，但语义仍较粗糙 |
| `backend/code/internal/node/client/client.go` | Node 访问 Controller 的 HTTP 客户端 | 已封装 register / heartbeat / peers / metrics |
| `backend/code/internal/node/runtime/agent.go` | Node 运行时主逻辑 | 已实现注册、周期心跳、周期拉 peers |
| `backend/code/internal/node/p2p/peer.go` | 纯 UDP 双节点互发实验 | 已能互发字符串，不接 Controller |
| `backend/code/internal/node/tun/tun.go` | TUN 创建、抓包、自动回 ICMP Echo | 功能较完整，但仍是独立实验模块 |
| `backend/code/pkg/config/config.go` | 后端默认配置常量 | 已集中管理默认地址与超时等 |
| `backend/code/pkg/logger/logger.go` | 简单日志构造器 | 功能简单，可用但当前使用较少 |
| `backend/code/pkg/protocol/protocol.go` | 前后端 / 节点与 Controller 的协议结构定义 | 已集中定义 DTO，但部分字段设计与文档不一致 |

后端侧当前结论：

- 控制平面已经不是空白，Node 与 Controller 的基础交互链路也已经写出来了。
- 真实可用性最强的是 `register -> heartbeat -> peers` 这条基础链。
- `p2p` 和 `tun` 目前仍是两个相对独立的底层实验，没有在主线里打通。

### 4. frontend 目录

| 文件 | 作用 | 当前状态 |
| --- | --- | --- |
| `frontend/package.json` | 前端依赖与脚本入口 | 已配置 `dev` / `build` / `preview` |
| `frontend/package-lock.json` | npm 锁文件 | 锁文件，正常存在 |
| `frontend/README.md` | 前端历史开发记录 | 信息很详细，但有部分历史语境已过时 |
| `frontend/tsconfig.json` | TypeScript 配置 | 已开启严格检查并配置路径别名 |
| `frontend/vite.config.ts` | Vite 配置与代理规则 | 已支持局域网访问与代理转发 |
| `frontend/tailwind.config.js` | Tailwind 主题与扫描配置 | 已定义 `brand` 色板与 `shadow-panel` |
| `frontend/postcss.config.js` | PostCSS 配置 | 标准 Tailwind + autoprefixer 配置 |
| `frontend/index.html` | 前端 HTML 入口 | 标准单页应用入口 |
| `frontend/.env.example` | 环境变量模板 | 已支持 `VITE_API_BASE_URL` 与 `VITE_PROXY_TARGET` |
| `frontend/src/env.d.ts` | 环境变量类型声明 | 已同步环境变量类型 |
| `frontend/src/main.ts` | Vue 启动入口 | 已挂载 Element Plus 与全局样式 |
| `frontend/src/App.vue` | 应用总入口组件 | 当前直接渲染 Dashboard 页面 |
| `frontend/src/assets/main.css` | 全局样式与 Tailwind 组件类 | 已定义背景、字体、公共面板样式 |
| `frontend/src/api/dashboard.ts` | Dashboard 请求层与图数据转换层 | 已是前端最关键的接口定义文件 |
| `frontend/src/views/AdminDashboardPage.vue` | 当前页面总入口 | 负责拼装布局与业务主容器 |
| `frontend/src/components/layout/layout.types.ts` | 布局与菜单类型定义 | 结构清晰 |
| `frontend/src/components/layout/AdminShell.vue` | 页面整体骨架 | 已实现侧边栏 + 顶栏 + 内容区 |
| `frontend/src/components/layout/AdminSidebar.vue` | 侧边导航 | 已实现静态菜单与高亮切换 |
| `frontend/src/components/layout/AdminTopbar.vue` | 顶栏 | 已支持 actions 插槽与移动端按钮 |
| `frontend/src/components/dashboard/OnlineNodeTable.vue` | Dashboard 主业务容器 | 已负责拉取 stats / nodes / edges / metrics |
| `frontend/src/components/dashboard/NodeRelationGraph.vue` | 节点关系图 | 已支持 ECharts graph、拖拽、悬浮卡片联动 |
| `frontend/src/components/dashboard/LinkLatencyChart.vue` | 悬浮详情卡片与折线图 | 已支持真实 metrics props 渲染与空态 |

前端侧当前结论：

- 页面框架、交互和可视化组件已经比较完整。
- 前端不是“只有假界面”，而是已经按真实接口结构组织起来了。
- 当前前端最大的阻塞不在组件本身，而在于它依赖的 `edges` 和 `metrics` 后端契约还没有完全兑现。

## 当前功能到底做到哪一步

这一节只讲“已经落到代码里的事实”，不讲规划。

### 前端已经落地的事实

1. 前端已经使用 `Vue 3 + TypeScript + Vite + Element Plus + Tailwind CSS + Axios + ECharts`。
2. 页面启动链路完整：
   `index.html -> src/main.ts -> src/App.vue -> src/views/AdminDashboardPage.vue`
3. 后台页面骨架完整，且考虑了移动端抽屉侧边栏。
4. Dashboard 页面已经会并发请求：
   - `GET /api/v1/dashboard/stats`
   - `GET /api/v1/dashboard/nodes`
   - `GET /api/v1/dashboard/edges`
5. 节点悬浮时，页面还会请求：
   - `GET /api/v1/dashboard/nodes/{node_id}/metrics`
6. 页面上已经有 4 类真实业务展示位：
   - 顶部统计卡片
   - 节点关系图
   - 在线节点表格
   - 节点悬浮详情卡片与折线图
7. 当前前端地址切换方式已经成型：
   - 推荐改 `VITE_PROXY_TARGET`
   - 可选改 `VITE_API_BASE_URL`
8. 页面已经写了比较细的中文注释，适合 Vue 新手接手。

### 后端已经落地的事实

1. Controller 可以通过 `backend/code/cmd/controller/main.go` 启动。
2. Node 程序可以通过 `backend/code/cmd/node/main.go` 进入三种模式：
   - `tun`
   - `run`
   - `p2p`
3. Controller 当前已经注册了这些 handler：
   - `GET /`
   - `GET /ping`
   - `GET /nodes`
   - `GET /api/v1/dashboard/stats`
   - `GET /api/v1/dashboard/nodes`
   - `GET /api/v1/dashboard/edges`
   - `POST /api/v1/nodes/register`
   - `POST /api/v1/nodes/:node_id/heartbeat`
   - `GET /api/v1/nodes/:node_id/peers`
   - `GET /api/v1/nodes/:node_id/metrics`
4. Registry 已经有这些内存能力：
   - 节点注册
   - 机器 ID 到 Node ID 的复用
   - 虚拟 IP 分配
   - 在线 / 离线状态刷新
   - Dashboard 统计聚合
   - Dashboard 节点数组生成
   - Dashboard 边数组生成
   - peers 列表生成
   - 心跳衍生指标记录
5. Node runtime 已经有这些行为：
   - 启动时自动注册
   - 接收 `node_id` 与 `virtual_ip`
   - 按间隔发心跳
   - 按间隔拉 peers
   - 当 Controller 返回 `sync_peers` 时立即重拉 peers
6. `p2p` 模块已经可以让两个 UDP 节点定时互发消息。
7. `tun` 模块已经可以：
   - 创建或打开 TUN
   - 打印捕获的 IP 报文 Hex
   - 自动回 ICMP Echo Reply
   - 给出 Linux 权限与网络配置提示

### 文档已经落地的事实

1. 团队协作规约已经成文。
2. API 文档已经细化到请求路径、字段和 JSON 示例。
3. PM 周报已经开始沉淀技术背景与方案论证。
4. AI 对话内容已经形成大体路线和阶段目标素材。
5. LaTeX 周报骨架已经验证过基本可用。

## 代码与文档的对齐情况

这一节是本次扫描里最重要的部分。因为项目当前最大的风险，不是“没代码”，而是“文档、前端、后端各自都前进了，但没有完全走到同一条线上”。

### 接口对齐矩阵

| 接口 | 文档状态 | 后端当前实现 | 前端当前使用 | 当前结论 |
| --- | --- | --- | --- | --- |
| `GET /api/v1/dashboard/stats` | 已定义 | 已实现 | 已使用 | 基本可用。后端返回字段更多，前端当前只消费文档要求的核心字段 |
| `GET /api/v1/dashboard/nodes` | 已定义 | 已实现 | 已使用 | 基本可用。后端返回字段更多，前端当前只消费文档要求的核心字段 |
| `GET /api/v1/dashboard/edges` | 已定义 | 已实现 | 已使用 | 不对齐。后端当前返回 `source_node_id / target_node_id / recommend_mode` 语义，前端要求 `source / target / type` |
| `GET /api/v1/dashboard/nodes/{node_id}/metrics` | 已定义 | 未按文档路径实现 | 已使用 | 不对齐。前端请求的是 `dashboard/nodes/.../metrics`，后端实现的是 `/api/v1/nodes/:node_id/metrics` |
| `POST /api/v1/nodes/register` | 已定义 | 已实现 | Node runtime 已使用 | 基本可用 |
| `POST /api/v1/nodes/{node_id}/heartbeat` | 已定义 | 已实现 | Node runtime 已使用 | 基本可用，但 `connected_peers` 语义还是估算，不是真实连接数 |
| `GET /api/v1/nodes/{node_id}/peers` | 已定义 | 已实现 | Node runtime 已使用 | 基本可用，推荐模式根据 NAT 类型粗略推断 |

### 当前最关键的 4 个契约问题

#### 1. `edges` 字段结构不一致

接口文档和前端要求的 `edges` 是：

```json
{
  "source": "node-a",
  "target": "node-b",
  "type": "p2p"
}
```

后端当前的 `DashboardEdge` 却是：

- `edge_id`
- `source_node_id`
- `target_node_id`
- `source_hostname`
- `target_hostname`
- `source_virtual_ip`
- `target_virtual_ip`
- `source_public_ip`
- `target_public_ip`
- `recommend_mode`
- `status`
- `last_seen`

这会直接导致前端 `NodeRelationGraph` 中读取 `edge.source`、`edge.target`、`edge.type` 时拿不到预期字段。

#### 2. `metrics` 路径不一致

接口文档与前端都写的是：

`GET /api/v1/dashboard/nodes/{node_id}/metrics`

后端当前实现的是：

`GET /api/v1/nodes/{node_id}/metrics`

因此前端当前悬浮节点时，请求很可能直接命中 404。

#### 3. `metrics` 数据结构不一致

接口文档和前端要求的是：

```json
{
  "metrics": [
    {
      "timestamp": 1714819200,
      "latency_ms": 45.5
    }
  ]
}
```

后端当前返回的是：

```json
{
  "node_id": "xxx",
  "points": [
    {
      "timestamp": "2026-05-12T...",
      "current_rx_bytes": 123,
      "current_tx_bytes": 456,
      "connected_peers": 1,
      "status": "online"
    }
  ]
}
```

这意味着当前后端的 `metrics` 更像“节点心跳快照历史”，而不是“节点到目标节点的链路延迟历史”。

#### 4. `connected_peers` 与 `edges` 语义仍然是推导值，不是真值

当前 Registry 的处理方式是：

- 在线节点之间默认两两成边
- 每个在线节点的 `connected_peers = 在线节点总数 - 1`

这只能用来做早期演示，不能代表真实网络连接情况。

## 当前真实联调状态

综合前后端代码，本仓库当前最接近真实联调的部分是：

### 已形成基础闭环的部分

1. Node runtime 能启动并向 Controller 注册。
2. Controller 能为节点分配 `node_id` 和 `virtual_ip`。
3. Node runtime 能持续发送心跳。
4. Controller 能基于心跳维持在线状态。
5. Node runtime 能拉取其他在线节点的 peers 列表。
6. 前端能通过真实请求读取 `stats` 和 `nodes`。

### 仍然会卡住或失真的部分

1. 前端虽然请求了 `edges`，但后端 `edges` 字段结构不对，图上连线无法真正按预期渲染。
2. 前端虽然请求了 `metrics`，但当前路径和数据结构都对不上，悬浮卡片的真实折线图无法闭环。
3. 后端当前的 `p2p` 与 `tun` 还是两条分开的实验线，没有真正并进到 `runtime/agent.go` 主流程里。
4. 当前并没有实现“真实已连接对端列表”和“真实链路延迟采样”的统一上报机制。

## 运行方式与当前可验证性

这一节只记录本次扫描时在本机环境中核实到的事实，不做推测。

### 前端运行命令

进入前端目录：

```powershell
cd D:\Documents\WorkSpace\30-Playground\frontend\planA\NetWeaver\frontend
```

安装依赖：

```powershell
npm install
```

开发启动：

```powershell
npm run dev
```

类型检查：

```powershell
npx vue-tsc --noEmit
```

生产构建：

```powershell
npm run build
```

### 后端运行命令

Controller：

```powershell
go run ./backend/code/cmd/controller
```

Node runtime：

```powershell
go run ./backend/code/cmd/node run -controller http://127.0.0.1:8080
```

P2P 实验：

```powershell
go run ./backend/code/cmd/node p2p -profile A
go run ./backend/code/cmd/node p2p -profile B
```

TUN 实验：

```powershell
go run ./backend/code/cmd/node tun
```

### 本次环境下的实际验证结果

1. `frontend` 类型检查已通过：

```text
npx vue-tsc --noEmit
```

2. `frontend` 标准构建未通过，但原因不是源码报错，而是目标目录写入权限被拒绝：

```text
Could not create directory for output chunks: ...\frontend\dist\assets
拒绝访问。 (os error 5)
```

3. `backend` 的 `go test ./...` 在当前 PowerShell 环境下未能验证，因为本机当前 shell 没有识别到 `go` 命令。

因此，本次扫描可以确认：

- 前端 TypeScript 层面当前是可通过检查的。
- 前端标准生产构建受当前机器目录权限影响。
- 后端编译状态在本次 Windows PowerShell 环境里没有完成最终核实。

## 已发现的文档与仓库不一致点

这一节很重要，因为后续如果不修，会继续误导新成员。

### 1. 根 README 历史内容已经过时

本次重写前，根目录 `README.md` 仍停留在较早期状态，主要问题包括：

- 把后端描述成“只有 ping 和少量基础路由”
- 把前端描述成“只有骨架或演示阶段”
- 未反映当前已存在的 `stats / nodes / edges / metrics` 请求层与 Node runtime

### 2. `frontend/README.md` 有历史引用已失效

当前 `frontend/README.md` 中提到过这些旧文件名：

- `docs/前端后台骨架入门说明.md`
- `docs/NetWeaver v1.0 API接口文档.md`

但本次扫描没有在当前仓库看到这些文件。因此：

- `frontend/README.md` 更适合作为“前端开发历史记录”
- 不应再把其中所有历史引用当作当前仓库现状

### 3. `backend/code/README.md` 有环境假设偏旧

当前后端 README 存在这些偏差：

- 使用了具体 Linux 个人目录路径示例
- 声称根目录已配置 `go.work`
- 没有同步反映当前 Windows / WSL 混合开发背景

它仍然有使用价值，但需要后续补一次以当前团队环境为准的修订。

### 4. `docs/协作规约.md` 里的目录规划不完全等于现状

规约里写到的 `resources/`、`scripts/`、`config.json` 目前还没有变成真实稳定目录或文件。这不是错误，但说明：

- 规约中的目录结构部分，包含“理想规划”
- 当前仓库实际结构，还没有完全长成那个样子

## 现阶段最需要继续完成的事情

下面这部分不再讲“应该学什么”，而是讲“按照当前代码现状，接下来必须做什么”。

### A. 必须先收口接口契约

这是当前第一优先级。

必须统一的内容有：

1. `edges` 的对外返回字段
2. `metrics` 的请求路径
3. `metrics` 的返回结构
4. `connected_peers` 的真实语义
5. `edges` 的真实语义

如果这一步不先做，前端页面越做越多，后端越写越深，最终返工只会更大。

### B. 后端要把“推导值”改成“真实状态”

当前后端最大的问题不是“没有路由”，而是“有些数据虽然能返回，但不是实际网络状态”。

必须继续完成的后端工作：

1. 单独维护真实连接边状态，而不是所有在线节点两两连线。
2. 单独维护真实链路延迟采样，而不是复用心跳流量快照。
3. 让 Node 端把真实连接信息和链路采样上报给 Controller。
4. 将 `register / heartbeat / peers / edges / metrics` 串成统一状态机。

### C. 前端要等后端契约落地后完成最后闭环

前端组件本身已经基本到位，接下来主要不是重写页面，而是：

1. 改成完全消费最终版 `edges`
2. 改成完全消费最终版 `metrics`
3. 不再用“当前节点自己当 target_id”的临时占位策略
4. 补充更完整的错误态、空态与联调说明

### D. 数据平面要从“实验”走向“主线”

当前最本质的技术主线还没真正闭环：

1. `tun` 读到报文
2. 报文被封装进 `p2p` 或 `relay`
3. 对端节点解封装
4. 对端写回 `tun`
5. 最终通过虚拟 IP 互相 Ping 通

现在仓库里已经有 `tun.go` 和 `peer.go` 两个重要实验基础，但它们还没有被整合进 `runtime/agent.go` 主线。

### E. 文档、测试与部署还没有进入收口阶段

当前文档更多是“方案与过程沉淀”，还不是“最终交付包”。

后续必须继续补齐：

1. 联调记录
2. 部署说明
3. 测试记录
4. 弱网测试结论
5. 最终论文结构稿
6. 最终答辩 / 演示脚本

## 按角色拆分的继续开发清单

### 前端同学

当前已经完成：

- 后台骨架
- Dashboard 页面
- 真实请求层
- 节点表格
- 关系图
- 悬浮详情卡片
- 环境变量切换

接下来必须继续做：

1. 等后端确认最终 `edges` 结构后，改 `frontend/src/api/dashboard.ts` 与关系图消费逻辑。
2. 等后端确认最终 `metrics` 路径和结构后，改 `frontend/src/api/dashboard.ts`、`OnlineNodeTable.vue`、`LinkLatencyChart.vue`。
3. 增加“选择目标节点查看链路”的明确交互，而不是继续用当前的占位调用方式。
4. 继续完善联调失败时的页面提示。
5. 等契约稳定后，补生产部署说明与构建说明。

### 后端同学

当前已经完成：

- Controller 启动
- API 基础结构
- Registry
- Node runtime
- P2P 实验
- TUN 实验

接下来必须继续做：

1. 先按接口文档修正 `edges` 和 `metrics` 契约。
2. 在 Registry 中拆分真实边状态与真实 metrics 状态。
3. 让 Node runtime 不只上报流量和 NAT，还要上报真实连接与链路采样。
4. 把 `p2p` 实验逻辑逐步并入 `runtime/agent.go`。
5. 把 `tun` 与 `p2p/relay` 整合成真实数据通道。

### 文档 / PM / 测试同学

当前已经完成：

- 协作规约
- API v1.8
- PM 报告
- AI 路线资料
- LaTeX 骨架

接下来必须继续做：

1. 把接口文档作为唯一对外标准继续维护。
2. 针对当前代码现状补一份联调记录。
3. 把“接口文档与代码不一致点”列成表，跟进每次修复。
4. 补部署说明、测试记录、最终汇报材料。
5. 后期负责把弱网测试、加密、部署与成品演示材料组织起来。

## 当前项目阶段判断

如果把项目分成 4 个阶段：

1. 框架与协作基础
2. 控制平面成型
3. 数据平面闭环
4. 成品化与比赛交付

那么当前仓库的位置是：

- 第 1 阶段：基本完成
- 第 2 阶段：已完成大半，但接口契约仍需收口
- 第 3 阶段：已有底层实验基础，但主线闭环未完成
- 第 4 阶段：仅有局部规划和素材，未进入正式收口

因此，当前最准确的描述不是“项目还没开始”，也不是“项目已经做完”，而是：

**项目已经从框架搭建阶段进入控制平面与数据平面之间的关键拼接期。**

## 建议的阅读顺序

如果新成员要尽快看懂项目，建议按下面顺序读：

1. `docs/协作规约.md`
2. `docs/NetWeaver-API-v1.8.md`
3. 本 `README.md`
4. `frontend/README.md`
5. `frontend/src/api/dashboard.ts`
6. `frontend/src/components/dashboard/OnlineNodeTable.vue`
7. `backend/code/pkg/protocol/protocol.go`
8. `backend/code/internal/controller/api/router.go`
9. `backend/code/internal/controller/state/registry.go`
10. `backend/code/internal/node/runtime/agent.go`
11. `backend/code/internal/node/p2p/peer.go`
12. `backend/code/internal/node/tun/tun.go`

## 本次扫描后的结论

本仓库当前最有价值的地方，不是某一个单独文件，而是它已经同时具备了：

- 真实前端结构
- 真实 Controller 雏形
- 真实 Node runtime 雏形
- 真实底层实验代码
- 真实接口文档
- 真实阶段材料

它真正缺的不是“从 0 到 1”，而是“从多个 0.5 拼成一个完整的 1”。

当前后续开发的总原则应当非常明确：

1. 先以 `docs/NetWeaver-API-v1.8.md` 为唯一契约收口接口。
2. 再让后端把 `edges`、`metrics`、Dashboard JWT 和 Node PSK 做成真实语义。
3. 再让前后端围绕 v1.8 完成完整联调验收。
4. 最后把 `tun` 和 `p2p/relay` 并入统一主线，形成真正的异地组网闭环。

只要这 4 步走通，NetWeaver 就会从“前端、后端、文档都各自做了不少事”进入“真正能演示、能验收、能参赛”的状态。
