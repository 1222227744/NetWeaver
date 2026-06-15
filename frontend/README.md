# Frontend 开发记录

本文档用于记录 `frontend/` 目录内已经完成的开发内容、涉及文件和验证结果。目标是让前端侧改动可追踪、可回看、可对照。

## 当前范围

- 技术栈：`Vue 3 + TypeScript + Vite + Tailwind CSS + Element Plus`
- 当前主页面入口：[src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
- 当前布局骨架：[src/components/layout/AdminShell.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminShell.vue)

## 当前联调配置（请优先看这一节）

这部分写的是“现在这份前端代码应该怎么接真实后端”，优先级高于下面按时间顺序记录的历史日志。

当前已经不再保留 `test/` Mock 后端目录，也不再建议通过本地假接口联调。现在默认就是对真实后端联调。

### 1. 前端到底在哪里改后端地址

优先看这几个文件：

- [.env.development.local](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.development.local)
  - 这是你本机开发时最常改的地方
- [.env.example](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.example)
  - 这是环境变量模板，用来告诉别人应该填哪些项
- [vite.config.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/vite.config.ts)
  - 这里读取环境变量，并把 `/api` 代理到你配置的后端地址
- [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 这里定义了 axios 的 `baseURL` 和所有控制台接口

以后如果你想问“前端现在请求的是哪个后端”，先看 `.env.development.local`，不要先去源码里全局搜索 `axios`。

### 2. 最常用的联调方式：走 Vite 代理

这是当前最推荐的开发联调方式。优点是：

- 前端源码里继续写 `/api/v1/...`
- 浏览器不会直接跨域请求后端
- 只需要改环境变量，不需要改组件代码

请在 [.env.development.local](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.development.local) 里这样写：

```env
VITE_API_BASE_URL=
VITE_PROXY_TARGET=http://127.0.0.1:8080
```

含义是：

- `VITE_API_BASE_URL` 留空
  - 浏览器会继续请求相对路径 `/api/...`
- `VITE_PROXY_TARGET` 指向真实后端
  - Vite 开发服务器会把 `/api/...` 转发给这个地址

本机场景示例：

```env
VITE_API_BASE_URL=
VITE_PROXY_TARGET=http://127.0.0.1:8080
```

校园网 / 局域网联调示例：

```env
VITE_API_BASE_URL=
VITE_PROXY_TARGET=http://192.168.1.23:8080
```

这里的 `192.168.1.23:8080` 就替换成你队友机器上后端服务真实监听的 IP 和端口。

### 3. 另一种方式：浏览器直接请求后端

如果你明确需要让浏览器直接访问后端，而不是通过 Vite 代理，那么可以这样写：

```env
VITE_API_BASE_URL=http://192.168.1.23:8080
VITE_PROXY_TARGET=http://192.168.1.23:8080
```

这时：

- axios 会直接请求 `http://192.168.1.23:8080/api/v1/...`
- 是否能成功，取决于后端是否正确配置了 CORS

如果你只是普通开发联调，仍然建议优先走上一节的代理模式。

### 4. 修改环境变量后要做什么

`.env.*` 文件不是热更新的。也就是说：

1. 先修改 `.env.development.local`
2. 停掉当前的 `npm run dev`
3. 重新执行 `npm run dev`

只有这样新地址才会真正生效。

### 5. 当前联调需要后端具备哪些接口

前端当前严格按 `docs/NetWeaver-API-v1.8.md` 联调，已经实际使用到这些接口：

- `POST /api/v1/auth/login`
- `GET /api/v1/dashboard/stats`
- `GET /api/v1/dashboard/nodes`
- `GET /api/v1/dashboard/edges`
- `GET /api/v1/dashboard/nodes/{node_id}/metrics?target_id={target_id}&time_range=1h|12h|24h`

并且当前前端已经依赖这些规则：

- 登录成功后，后端必须返回 JWT
- 后续 `dashboard/*` 接口必须接受 `Authorization: Bearer <jwt>`
- `nodes` 返回的是全部节点，不只是在线节点
- `edges` 只返回真实已建链的边
- `metrics` 允许返回空数组 `[]`，前端会显示“暂无链路数据”

### 6. 当前最直接的运行与验证命令

首次安装依赖：

```powershell
cd frontend
npm install
```

开发联调：

```powershell
cd frontend
npm run dev
```

生产构建验证：

```powershell
cd frontend
npm run build
```

本地预览构建产物：

```powershell
cd frontend
npm run preview
```

### 7. 这几个文件各自负责什么

- [.env.development.local](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.development.local)
  - 改“你开发机现在要连哪个后端”
- [.env.example](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.example)
  - 给队友看的模板，不是日常直接运行文件
- [vite.config.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/vite.config.ts)
  - 定义 `/api` 代理规则，决定相对路径到底被转发到哪里
- [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 统一写接口路径、请求参数、响应类型、JWT 自动携带

## 阅读说明

下面的“变更记录”是按时间顺序写的。

- 越新的记录，优先级越高
- 较新的开发条目可能会覆盖较早时期的临时方案
- 如果历史记录里出现了旧的 Mock 流程，请以本节上面的“当前联调配置”为准

## 新手接手顺序

如果是刚学 Vue 的同学接手，建议不要一上来就随机点文件，而是按下面顺序读：

1. [index.html](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/index.html)
   - 先理解浏览器最开始打开的是什么
2. [src/main.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/main.ts)
   - 理解 Vue 应用是怎么启动的
3. [src/App.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/App.vue)
   - 理解当前最外层到底显示哪个页面
4. [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
   - 理解页面入口怎样把布局和业务组件拼起来
5. [src/components/layout/AdminShell.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminShell.vue)
   - 理解后台页面骨架
6. [src/components/layout/AdminSidebar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminSidebar.vue)
   - 理解侧边栏菜单数据怎么显示
7. [src/components/layout/AdminTopbar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminTopbar.vue)
   - 理解顶部栏、插槽和移动端抽屉按钮
8. [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
   - 这是当前前端业务主线，负责请求控制台数据并驱动页面
9. [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
   - 理解关系图怎么显示，悬停事件怎么往父组件回传
10. [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
    - 理解悬浮卡片和折线图怎么显示
11. [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
    - 理解前端到底请求了哪些接口，以及图数据如何转换
12. [vite.config.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/vite.config.ts)、[.env.example](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.example)、[src/env.d.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/env.d.ts)
    - 理解前端联调时到底向哪个地址请求后端
13. [tailwind.config.js](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/tailwind.config.js)、[postcss.config.js](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/postcss.config.js)、[src/assets/main.css](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/assets/main.css)
    - 理解样式系统从哪里来

## 关于“为什么有些文件不能直接写注释”

前端目录里大部分文件都已经补了详细注释，但有两个标准 JSON 文件不适合直接写注释：

- [package.json](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/package.json)
- [package-lock.json](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/package-lock.json)

原因：

- 标准 JSON 语法本身不支持注释
- 强行往里面塞伪注释字段，反而会让接手的人误以为它们是业务配置

阅读这两个文件时，建议这样理解：

- `package.json`
  - 看 `scripts`：知道怎么运行、构建、预览前端
  - 看 `dependencies`：知道项目运行时依赖哪些库
  - 看 `devDependencies`：知道项目开发和构建时依赖哪些工具
- `package-lock.json`
  - 主要用于锁定依赖版本
  - 日常阅读优先级很低，只有排查依赖版本冲突时才需要重点看

## 目录说明

```text
frontend/
├── src/
│   ├── api/                 # 前端请求层与数据转换入口
│   ├── assets/              # 全局样式与静态资源
│   ├── components/          # 公共组件
│   │   ├── dashboard/       # 业务组件（节点表格、关系图、悬停详情卡片）
│   │   └── layout/          # 布局组件（侧边栏、顶部栏、页面骨架）
│   └── views/               # 页面级组件
├── .env.example             # 前端环境变量示例
├── index.html               # 前端 HTML 入口
├── package.json             # 前端依赖与脚本
├── tailwind.config.js       # Tailwind 配置
├── vite.config.ts           # Vite 配置
└── README.md                # 前端开发记录
```

## 变更记录

### 2026-06-15 本次开发：控制台落地微调与部署说明对齐

目标：

- 把控制台从“偏展示页”进一步收敛成“更适合联调和演示的管理台”
- 修正容易误导联调人员的页面语义和后端默认配置说明
- 补足真实部署前最容易漏掉的默认值与环境变量说明

具体改动：

- 调整 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
  - 侧边栏菜单改成真正可用的“页内导航”，不再假装切多个页面
- 调整 [src/components/layout/AdminSidebar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminSidebar.vue)
  - 注释和菜单语义同步为“滚动到对应区块”
- 调整 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 去掉容易误导的箭头连线
  - 关系图高度不再固定写死，而是随节点数量扩展
  - 补充说明文字，让使用者知道图中链路是已经建立成功的真实链路
- 调整 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - 页面重新分成“概览、拓扑、节点列表”三个区块
  - 节点列表和拓扑图改成同一层的双栏布局，更接近控制台使用习惯
  - 表格默认把在线节点排在前面，便于联调时先关注当前活跃节点
- 调整 [src/App.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/App.vue) 和 [src/assets/main.css](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/assets/main.css)
  - 移除全局鼠标光晕，减少装饰性视觉干扰

这次改动后的页面理解方式：

- `控制台概览`：看总节点数、在线数、链路数、运行时长
- `链路拓扑`：看当前已建立成功的真实关系边，以及边的 `P2P / Relay` 类型
- `节点列表`：看节点明细，优先核对在线状态、最后心跳和公网地址

验证结果：

- 已执行前端构建检查，确保页面结构改动后仍可打包
- 这次没有引入新的前端依赖

### 2026-04-23 第一次开发：后台骨架初始化

目标：

- 建立可运行的前端工程
- 先把后台管理页面的基础框架搭起来
- 给后续继续开发预留清晰的扩展点

具体改动：

- 新建 [package.json](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/package.json)，初始化前端依赖和脚本
- 新建 [vite.config.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/vite.config.ts)，配置 Vite 与 `@` 路径别名
- 新建 [tailwind.config.js](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/tailwind.config.js) 和 [postcss.config.js](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/postcss.config.js)，完成 Tailwind CSS 基础配置
- 新建 [src/main.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/main.ts) 和 [src/App.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/App.vue)，建立 Vue 应用入口
- 新建 [src/assets/main.css](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/assets/main.css)，写入全局样式、背景和通用面板类
- 新建布局组件：
  - [src/components/layout/AdminShell.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminShell.vue)
  - [src/components/layout/AdminSidebar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminSidebar.vue)
  - [src/components/layout/AdminTopbar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminTopbar.vue)
  - [src/components/layout/layout.types.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/layout.types.ts)
- 新建页面组件 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)，初版内容以骨架卡片和占位区块为主
- 新建 [docs/前端后台骨架入门说明.md](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/docs/前端后台骨架入门说明.md)，补充新手可读的结构说明、命令说明和开发入口说明

当时页面状态：

- 已具备后台页面基础布局
- 已具备侧边栏、顶部栏、内容区
- 已具备响应式布局和移动端抽屉侧边栏
- 页面主体仍以占位内容为主，尚未接入具体业务列表

验证结果：

- `npm run build` 在当时已通过

### 2026-04-27 第二次开发：在线节点列表（Mock 数据）

目标：

- 依照 API 文档的 `GET /api/v1/dashboard/nodes`
- 写死一段 mock JSON
- 在页面的 `Element Plus Table` 中渲染在线节点列表
- 在代码里补充必要注释，明确 `ref`、`reactive`、axios 请求层和数据流

参考文档：

- [NetWeaver v1.0 API接口文档.md](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/docs/NetWeaver%20v1.0%20API接口文档.md)

具体改动：

- 新建 [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 按接口文档定义 `ApiResponse<T>`、`DashboardNode`、`DashboardNodesData`
  - 写入本地 mock JSON，字段与接口文档保持一致
  - 提供 `getDashboardNodes()` 方法
  - 使用 axios 请求层，并通过自定义 adapter 返回 mock 数据
  - 维持 `response.data` 读取结构，和真实请求层的返回习惯保持一致
- 新建 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - 使用 `ref<DashboardNode[]>` 保存表格数据
  - 使用 `reactive(...)` 保存页面状态：`loading`、统计数字、刷新时间
  - 在 `onMounted()` 中触发一次数据读取
  - 过滤 `status === 'online'` 的节点后再渲染 `el-table`
  - 增加统计卡片：总节点数、在线节点数、已连接对等节点数、Full Cone 节点数
- 修改 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
  - 保留原有 `AdminShell` 布局结构
  - 页面标题从骨架展示切换为在线节点列表
  - 页面主体替换为 `OnlineNodeTable`
  - 头部标记改为接口路径与 Mock JSON 标识
- 修改根目录 [.gitignore](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/.gitignore)
  - 新增 `verify-dist/` 忽略规则

当前页面状态：

- 页面主体已经从占位内容切换成真实业务列表
- 在线节点列表已通过 `el-table` 渲染
- 节点字段来自 mock JSON：
  - `node_id`
  - `hostname`
  - `virtual_ip`
  - `public_ip`
  - `nat_type`
  - `status`
  - `connected_peers`
- 页面会在请求层返回后只显示 `status === "online"` 的节点

关于请求层的当前实现：

- 当前请求层已经使用 `axios`
- 当前仍然返回 mock 数据，但 mock 数据是通过 axios 自定义 adapter 返回
- 页面侧按标准 axios 返回结构读取 `response.data`

验证结果：

- `npx vue-tsc --noEmit` 已通过
- `npx vite build --configLoader native --outDir verify-dist` 已通过
- 默认 `npm run build` 因当前机器对 `frontend/dist/assets` 的目录写入限制失败，问题出在输出目录权限，不在源码本身

## 本次涉及的关键文件

- [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
- [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
- [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
- [.gitignore](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/.gitignore)

## 运行与验证命令记录

进入前端目录：

```powershell
cd d:\Documents\WorkSpace\30-Playground\frontend\planA\NetWeaver\frontend
```

开发预览：

```powershell
npm run dev
```

首次安装或补装依赖：

```powershell
npm install
```

如果后续新增某个前端依赖，例如：

```powershell
npm install axios
npm install echarts
```

类型检查：

```powershell
npx vue-tsc --noEmit
```

打包验证：

```powershell
npx vite build --configLoader native --outDir verify-dist
```

## 以后遇到 npm 权限问题时的处理流程

目标：

- 保持源码仍然采用标准 npm 依赖方式
- 不把 CDN 脚本或长期本地绕过方案留在业务代码里

处理顺序：

1. 先确认缺少的是哪个 npm 包
2. 在 `frontend/` 目录执行对应安装命令
3. 安装完成后检查这三处是否同步更新：
   - [package.json](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/package.json)
   - [package-lock.json](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/package-lock.json)
   - `frontend/node_modules/`
4. 如果某次为了临时验证使用了本地脚本或全局变量方式，安装成功后要立刻删掉临时方案，恢复为 npm 导入
5. 再执行类型检查和打包验证

本项目当前已通过 npm 管理的新增依赖：

- `axios`
- `echarts`

## 维护约定

- 以后前端每完成一轮可识别功能，就在这个文件继续追加一节开发记录
- 每一节至少记录：日期、目标、修改文件、页面结果、验证结果
- 如果开发过程受环境限制影响，也直接记录为事实，避免后续重复排查

### 2026-05-01 第三次开发：npm 依赖收口与临时目录清理

目标：

- 将前端依赖彻底收口到 npm 管理方式
- 清理历史排障和临时验证过程中产生的无用目录
- 明确以后标准的运行、构建、验证和清理流程

本次确认的无用历史目录：

- `frontend/cache-test-dir`
- `frontend/writable-cache`
- `frontend/public/vendor`
- `frontend/verify-dist`
- `frontend/verify-dist-graph`
- `frontend/verify-dist-graph-2`
- `frontend/verify-dist-npm`

这些目录的来源说明：

- `cache-test-dir`
  仅用于测试当前工作区是否允许创建目录。
- `writable-cache`
  仅用于测试 npm 缓存能否改到其他本地路径。
- `public/vendor`
  曾尝试放本地脚本资源，后来已经改回 npm 依赖导入方式。
- `verify-dist*`
  都是某次 `vite build --outDir ...` 产生的临时验证产物目录，不属于源码，也不属于标准部署目录。

本次处理结果：

- 前端源码已完全恢复为标准 npm 依赖方式：
  - `axios`
  - `echarts`
- `.gitignore` 中已经移除了 `verify-dist`、`verify-dist-graph`、`verify-dist-graph-2` 这些一次性目录规则
- 这些历史目录应在验证完成后直接删除，不应继续保留，也不应继续写进忽略规则

执行环境事实记录：

- 当前会话下，删除上述目录的命令被本地执行策略拦截，未能由自动化命令直接完成
- 这不影响源码本身的运行逻辑，但会影响工作区整洁度

以后前端的标准验证流程：

1. 安装依赖

```powershell
cd d:\Documents\WorkSpace\30-Playground\frontend\planA\NetWeaver\frontend
npm install
```

2. 启动开发环境

```powershell
npm run dev
```

3. 类型检查

```powershell
npx vue-tsc --noEmit
```

4. 标准生产构建

```powershell
npm run build
```

5. 本地预览打包结果

```powershell
npm run preview
```

标准目录使用规则：

- 日常开发只修改 `src/`、`index.html`、配置文件和 `README.md`
- 正式构建结果只认 `frontend/dist`
- 若为了排障临时使用了自定义 `--outDir`，验证完成后必须立即删除对应目录
- 不再保留 `verify-dist*`、测试缓存目录或其他一次性目录

### 2026-05-03 第四次开发：移除 Mock，切到真实接口请求

目标：

- 删除 `dashboard` 请求层里原来写死的 mock 数据
- 改成通过 axios 请求真实接口
- 保持节点表格、概览卡片、节点关系图继续复用同一份接口数据
- 记录当前前端到底已经接入了 API 文档里的哪些接口

本次具体改动：

- 修改 [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 删除原来的 axios `adapter` mock 实现
  - 删除写死在文件里的本地节点 JSON
  - 新增 `dashboardHttp` 请求实例
  - `getDashboardNodes()` 改为真正请求 `GET /api/v1/dashboard/nodes`
- 修改 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - `loadOnlineNodes()` 改为直接消费真实接口返回
  - 增加接口失败时的错误提示、状态清空和说明注释
  - 删除页面里“当前还是 mock 响应”的说明
- 修改 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
  - 页面头部标记从 `Mock JSON` 改为 `Real API`
  - 页面说明改为“通过 axios 请求真实接口”
- 修改 [vite.config.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/vite.config.ts)
  - 新增 `/api` 代理到 `http://127.0.0.1:8080`
  - `npm run dev` 和 `npm run preview` 都会按这个代理规则转发接口请求
- 修改 [src/env.d.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/env.d.ts)
  - 补充 `VITE_API_BASE_URL` 类型声明

当前前端已经实际使用到的接口：

- `GET /api/v1/dashboard/nodes`
  - 用途：在线节点表格
  - 用途：页面顶部 4 个统计卡片的数据来源
  - 用途：ECharts 节点关系图的原始节点数组来源

当前前端还没有实际使用到的控制台接口：

- `GET /api/v1/dashboard/stats`

与后端现状有关的事实记录：

- 当前仓库里的 Go 控制器代码还没有实现 `GET /api/v1/dashboard/nodes`
- 当前后端只看得到根路径 `/` 的 `Ping` 接口
- 所以前端源码虽然已经切到真实请求模式，但如果后端还没补这个路由，页面会显示接口失败提示，而不会再回退到 mock 数据

### 2026-05-03 第五次开发：节点悬停详情卡片与可切换后端地址

目标：

- 不再预设一个并不存在的链路延迟接口
- 保留折线图画图逻辑，但先只作为本地展示预览
- 把折线图从页面固定区块改为“节点悬停时，在鼠标附近显示”
- 保持本机联调和局域网联调都能通过环境变量切换后端地址

本次具体改动：

- 修改 [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 删除先前自行约定的链路延迟接口类型和请求方法
  - 当前请求层重新只保留 `GET /api/v1/dashboard/nodes`
- 重写 [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
  - 组件职责从“实时请求延迟接口”改为“节点悬停详情卡片”
  - 组件接收当前悬停节点与屏幕坐标
  - 卡片中显示节点基本信息和一张本地生成的折线图
  - 每个节点的折线图都依据自身字段生成，因此不同节点会呈现不同走势
- 修改 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 监听 ECharts 节点的 `mouseover`、`mousemove`、`mouseout`
  - 鼠标悬停在节点上时，在鼠标附近显示详情卡片
  - 鼠标移出节点后隐藏卡片
  - 节点原生 tooltip 不再显示，避免和自定义悬浮卡片重叠
- 修改 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - 删除页面中固定放置的链路延迟图区域
- 修改 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
  - 页面标题更新为“在线节点与关系预览”
  - 页面描述同步改为“节点悬停时的详细信息预览”
- 修改 [vite.config.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/vite.config.ts)
  - 代理地址改为从环境变量读取
- 修改 [src/env.d.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/env.d.ts)
  - 当前只保留 `VITE_API_BASE_URL` 类型声明
- 新建 [.env.example](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.example)
  - 提供本机联调和局域网联调的配置模板

当前前端已经实际使用到的接口：

- `GET /api/v1/dashboard/nodes`

当前折线图的性质说明：

- 折线图还没有接真实后端接口
- 当前只是节点详情卡片中的本地演示图
- 它的作用是先把悬浮展示结构、ECharts 画图逻辑和交互方式定下来
- 等后端以后补充真实链路延迟接口后，再把本地演示数据替换为真实数据

如何修改网络请求网址：

1. 复制环境变量模板

开发联调时使用：

```powershell
cd d:\Documents\WorkSpace\30-Playground\frontend\planA\NetWeaver\frontend
Copy-Item .env.example .env.development.local
```

如果你后面要验证 `npm run build` 或 `npm run preview`，则再复制一份：

```powershell
Copy-Item .env.example .env.production.local
```

2. 本机联调时，保持：

```env
VITE_PROXY_TARGET=http://127.0.0.1:8080
```

3. 切到局域网联调时，把它改成队友机器的局域网地址，例如：

```env
VITE_PROXY_TARGET=http://192.168.1.23:8080
```

4. 修改完成后，必须重新启动对应模式的前端服务：

```powershell
npm run dev
```

如果你在验证生产预览，则重启：

```powershell
npm run preview
```

关于 `VITE_API_BASE_URL` 和 `VITE_PROXY_TARGET` 的区别：

- `VITE_PROXY_TARGET`
  - 给 Vite 开发代理使用
  - 浏览器仍然请求 `/api/...`
  - 由 Vite 转发到真实后端
  - 本地开发和局域网联调时，优先推荐改这个，通常更省事
- `VITE_API_BASE_URL`
  - 给前端浏览器端 axios 直接使用
  - 会让浏览器直接请求完整后端地址
  - 只有在你明确需要绕过 Vite 代理时才改它

### 2026-05-03 第六次开发：悬浮卡片层级与节点交互修正

目标：

- 修正悬浮详情卡片会被下方在线节点列表遮挡的问题
- 恢复节点拖拽能力
- 修正悬浮卡片总是在第一个节点附近展开的问题

本次具体改动：

- 修改 [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
  - 使用 `Teleport` 将悬浮详情卡片直接渲染到 `body`
  - 提高悬浮卡片层级，避免被页面后续区块遮挡
- 修改 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 把 ECharts 节点配置从 `draggable: false` 改为 `draggable: true`
  - 悬浮定位不再使用不稳定的 `offsetX`、`offsetY`
  - 改为读取原生鼠标事件的 `clientX`、`clientY`
  - 节点拖拽过程中也同步更新悬浮卡片位置

本次页面结果：

- 节点现在可以直接拖拽
- 悬浮详情卡片会出现在当前鼠标所在节点附近
- 悬浮卡片不会再被下方在线节点列表遮挡

验证结果：

- `npx vue-tsc --noEmit` 已通过

### 2026-05-10 第七次开发：控制台真实接口闭环（前端本周任务）

目标：

- 把请求层从“只接一个 nodes 接口”扩展到完整的 `stats / nodes / edges / metrics`
- 让顶部统计卡片改为消费真实 `stats` 接口
- 让关系图改为消费真实 `edges` 接口，不再由前端本地猜测连线
- 让链路折线图组件只负责渲染真实 metrics 数据或显示空态
- 继续保持开发环境与局域网联调地址通过环境变量切换

本次具体改动：

- 修改 [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 新增 `DashboardStatsData`
  - 新增 `DashboardEdge`、`DashboardEdgesData`
  - 新增 `DashboardMetricPoint`、`DashboardNodeMetricsData`
  - 新增 `getDashboardStats()`
  - 新增 `getDashboardEdges()`
  - 新增 `getNodeMetrics()`
  - `buildDashboardGraphData()` 改为同时接收 `nodes` 与 `edges`
  - 删除原来“只截前两个节点、前端自行推导一条边”的图数据构造方式
- 修改 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - `loadOnlineNodes()` 改为 `loadDashboard()`
  - 通过 `Promise.all()` 并发请求 `stats / nodes / edges`
  - 顶部四张卡片改为显示：
    - `total_nodes`
    - `online_nodes`
    - `total_traffic_gb`
    - `controller_uptime_sec`
  - 新增 5 秒轮询刷新
  - 新增 `metricsLoading`
  - 悬停关系图节点时会发起 `metrics` 请求
- 修改 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - props 新增 `edges`、`metrics`
  - 图数据改为消费真实边数组
  - 图布局从 `none` 改为 `force`
  - 新增 `node-hover` 事件，把当前悬停节点抛给父组件，由父组件统一决定是否去请求 metrics
- 修改 [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
  - 删除本地随机生成折线图数据的逻辑
  - 改为通过 `metrics` props 渲染真实时间序列
  - 如果当前没有监控数据，则显示“暂无链路数据”空态图
- 修改 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
  - 页面标题改为“控制台真实接口总览”
  - 页面描述改为展示 `dashboard/*` 系列真实接口
- 修改 [src/env.d.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/env.d.ts)
  - 增加 `VITE_PROXY_TARGET` 类型声明
- 修改 [.env.example](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.example)
  - 明确说明本周默认通过 `VITE_PROXY_TARGET` 切换联调地址

本次页面结果：

- 顶部统计卡片不再由前端根据节点表格自行计算
- 节点关系图不再只显示“两节点单连线演示图”
- 页面现在会主动请求真实 `stats / nodes / edges`
- 当后端 `metrics` 还没有返回有效数据时，悬浮卡片中的折线图会显示空态，而不是继续展示前端随机生成的演示曲线

当前已实际接入的控制台接口：

- `GET /api/v1/dashboard/stats`
- `GET /api/v1/dashboard/nodes`
- `GET /api/v1/dashboard/edges`
- `GET /api/v1/dashboard/nodes/{node_id}/metrics`

当前仍然保留的事实说明：

- 这一节记录的是 2026-05-10 当时的阶段性做法
- 2026-05-24 的第八次开发已经删除“当前节点查询自己”的临时策略，改为悬停或点击真实链路后按 `source -> target` 请求 metrics

### 2026-05-24 第八次开发：按 API v1.8 完成前端收尾闭环

目标：

- 按 `docs/NetWeaver-API-v1.8.md` 收口 Dashboard 前端
- 增加登录流程，先调用 `POST /api/v1/auth/login` 获取 JWT
- 让后续 `GET /api/v1/dashboard/*` 请求自动携带 `Authorization: Bearer <jwt>`
- 删除 metrics 的临时自查调用，改为悬停或点击真实连线后查询 `source -> target`
- 保留复杂拓扑能力：节点可以继续拖拽，关系图可以展示多节点、多边、P2P/Relay 混合链路
- 补齐登录过期、后端离线、无在线节点、无真实边、无 metrics 采样点等页面状态

本次具体改动：

- 修改 [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 新增 `DashboardLoginRequest`、`DashboardLoginData`、`DashboardAuthStorage`
  - 新增 `loginDashboard()`
  - 新增 `setDashboardAuth()`、`getStoredDashboardAuth()`、`hasValidDashboardAuth()`、`clearDashboardAuth()`
  - 新增 axios 请求拦截器：访问 `/api/v1/dashboard/*` 时自动加 JWT
  - 新增 axios 响应拦截器：Dashboard 接口返回 `401` 时清空本地登录态
  - 将 `ApiResponse<T>` 的 `data` 改为 `T | null`，对齐接口文档中的失败结构
  - 将 `DashboardStatsData`、`DashboardNode` 扩展到 v1.8 字段
  - 新增 `DashboardMetricsTimeRange = '1h' | '12h' | '24h'`
  - `getNodeMetrics()` 现在会在前端阻止 `nodeId === targetId` 的错误调用
  - `GraphLinkItem` 新增 `id`、`edgeType`，方便关系图事件把具体链路传回父组件
- 修改 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - 新增 Dashboard 登录表单
  - 新增 JWT 登录态判断、退出登录、登录过期处理
  - 登录成功后再请求 `stats / nodes / edges`
  - 保留 5 秒轮询刷新，但轮询时不再强制打断页面主要加载状态
  - 新增链路时间范围选择：`1h`、`12h`、`24h`
  - 关系图连线悬停或点击后，请求 `GET /api/v1/dashboard/nodes/{source}/metrics?target_id={target}`
  - 请求失败时区分 401、超时、无响应、普通接口错误
  - 表格继续只展示在线节点，离线节点只参与统计卡片
- 修改 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 同时支持节点悬停和边悬停
  - 节点悬停只展示节点基本信息，不请求 metrics
  - 边悬停或点击会触发 `link-hover` 事件，把 `source / target / edgeType` 回传父组件
  - 保留 ECharts `graph`、`force` 布局、节点拖拽、缩放和平移
  - 拓扑图支持多节点、多真实边、P2P/Relay 文字标签
- 修改 [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
  - 悬浮卡片从“节点延迟”改为“节点详情 + 链路详情”
  - 节点卡片提示用户悬停连线查看真实链路延迟
  - 链路卡片展示源节点、目标节点、链路类型、时间范围和 ECharts 折线图
  - metrics 为空时显示“暂无链路数据”，不伪造曲线
  - metrics 加载中时显示“链路数据加载中”
- 修改 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
  - 页面标题和描述更新为 API v1.8、JWT、真实链路 metrics 的闭环说明

当前前端已经实际使用到的接口：

- `POST /api/v1/auth/login`
- `GET /api/v1/dashboard/stats`
- `GET /api/v1/dashboard/nodes`
- `GET /api/v1/dashboard/edges`
- `GET /api/v1/dashboard/nodes/{node_id}/metrics?target_id={target_id}&time_range=1h|12h|24h`

当前页面状态：

- 未登录时只显示登录卡片，不会请求 Dashboard 受保护接口
- 登录成功后展示统计卡片、控制栏、节点关系图和在线节点表格
- JWT 过期或无效时自动清空登录态，并提示重新登录
- 无在线节点时显示空态说明
- 有在线节点但无真实链路时，关系图显示节点并说明等待后端 edges 数据
- 悬停节点时显示节点详情
- 悬停或点击连线时显示链路详情和真实 metrics 折线图

后端登录接口未完成时的本地测试办法：

- 当时曾启动根目录 `test/mock-backend.js` 做临时联调（该文件现已删除）
- 前端保持正常登录流程，不再通过环境变量绕开登录
- 模拟账号为 `admin / admin123`
- 模拟后端会返回 JWT，并校验 Dashboard 接口的 `Authorization` 请求头

验证结果：

- `npx vue-tsc --noEmit` 已通过

- `npm run build` 的 TypeScript 与 Vite transform 阶段已通过，但当前机器无法写入历史遗留的 `frontend/dist/assets`，报错为 `拒绝访问`
- `npx vite build --configLoader native --outDir .codex-build-check` 已通过，用于确认源码可以生产构建
- `.codex-build-check` 为一次性验证目录，验证后已立即删除，没有作为遗留目录保留

本次之后的联调重点：

- 后端需要实现并返回 `POST /api/v1/auth/login`
- Dashboard 接口需要校验并接受 `Authorization: Bearer <jwt>`
- `GET /api/v1/dashboard/edges` 必须返回 `{ source, target, type }`
- `GET /api/v1/dashboard/nodes/{node_id}/metrics` 必须要求 `target_id`，且不能把 `node_id == target_id` 当作有效请求
- metrics 为空数组是正常空态，前端会展示空图；有数据时会按 `timestamp` 升序画真实延迟曲线

### 2026-05-24 第十次开发：新增 Mock 后端并移除登录绕过开关（历史方案，相关目录已删除）

说明：

- 这一阶段曾临时使用根目录 `test/` 做本地模拟联调
- `test/` 目录已经在后续清理中删除
- 下面这段记录只用于追踪当时做过什么，不再作为当前联调方式

目标：

- 根目录新增 `test/` 模拟后端，覆盖前端当前需要的全部接口
- 删除前端中临时绕过 Dashboard 登录的环境变量和判断逻辑
- 本地测试也必须走登录、JWT 保存、Dashboard 请求自动携带认证头这一完整流程

本次具体改动：

- 历史上曾新增 `test/mock-backend.js`（现已删除）
  - 使用 Node.js 原生 `http` 模块
  - 不需要安装依赖
  - 不做持久化
  - 写死节点、链路、统计和延迟曲线数据
  - 支持 `POST /api/v1/auth/login`
  - 支持 `GET /api/v1/dashboard/stats`
  - 支持 `GET /api/v1/dashboard/nodes`
  - 支持 `GET /api/v1/dashboard/edges`
  - 支持 `GET /api/v1/dashboard/nodes/{node_id}/metrics`
  - Dashboard 接口会校验 `Authorization: Bearer <token>`
- 历史上曾新增 `test/README.md`（现已删除）
  - 记录启动命令、登录账号和前端联调方法
- 修改 [src/api/dashboard.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/api/dashboard.ts)
  - 删除 `VITE_ENABLE_DASHBOARD_AUTH` 相关逻辑
  - Dashboard 受保护接口始终按登录态携带 JWT
  - Dashboard 接口返回 `401` 时始终清空本地登录态
- 修改 [.env.example](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/.env.example)
  - 删除 `VITE_ENABLE_DASHBOARD_AUTH`
- 修改 [src/env.d.ts](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/env.d.ts)
  - 删除 `VITE_ENABLE_DASHBOARD_AUTH` 类型声明
- 修改 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - 删除开发预览模式分支
  - 退出按钮始终走清空登录态流程

当时的本地测试命令（现已废弃）：

```powershell
node test/mock-backend.js
cd frontend
npm run dev
```

测试账号：

```text
username: admin
password: admin123
```

### 2026-05-24 第九次开发：清理用户界面中的开发说明

目标：

- 清除页面上不应展示给普通用户的开发说明、接口说明、版本标签和占位文案
- 保留代码注释，继续方便团队成员学习和维护
- 让管理后台界面更接近实际产品，而不是开发调试页

本次具体改动：

- 精简 [src/components/layout/AdminTopbar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminTopbar.vue)
  - 顶部栏只保留页面标题和必要插槽
  - 删除面包屑、默认消息按钮、新建按钮、占位用户信息和过长说明
- 精简 [src/components/layout/AdminSidebar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminSidebar.vue)
  - 删除品牌区说明文字
  - 删除底部环境/开发状态提示块
- 精简 [src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
  - 删除顶部 `JWT`、`API`、`v1.8` 等开发标签
  - 侧边栏菜单收敛为实际当前页面相关入口
- 精简 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - 删除统计卡片说明文字
  - 删除控制区大段说明
  - 删除表格上方解释性文字和重复状态标签
  - 保留刷新按钮、最近刷新时间、时间范围选择、错误提示、统计数字和表格
- 精简 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 删除拓扑图说明段落和关系摘要卡片
  - 保留图标题、链路数量和图本体
- 精简 [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
  - 删除悬浮卡片中的说明性摘要
  - 悬浮卡片只展示节点/链路字段和折线图

验证结果：

- `npx vue-tsc --noEmit` 已通过

### 2026-05-24 第十一次开发：节点关系图懒刷新

目标：

- 关系图不再因为后端 5 秒轮询而强制重绘
- 只有节点或链路拓扑真的变化时，才重新渲染 ECharts 关系图
- 保留统计卡片、在线节点表格、悬浮卡片数据的正常刷新

本次具体改动：

- 修改 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 新增 `graphTopologySignature`
  - 不再深度监听完整 `graphData`
  - 改为只监听会影响图形结构和外观的字段
  - 节点 `last_seen`、流量字段、metrics 采样变化不会再触发整张关系图重新布局

懒刷新规则：

- 会触发关系图重绘：
  - 在线节点新增或消失
  - 节点名称、虚拟 IP、NAT 类型、状态、真实邻居数变化
  - 连线的 `source`、`target`、`type` 变化
- 不会触发关系图重绘：
  - 节点最后心跳时间变化
  - 节点实时收发流量变化
  - 悬浮卡片里的链路延迟折线图数据变化

验证结果：

- `npx vue-tsc --noEmit` 已通过

### 2026-05-24 第十二次开发：主题色、明暗模式与鼠标光晕

目标：

- 解决浅色卡片区域里鼠标不明显的问题
- 在顶部栏增加连续主题色滑条，不限制成几个固定颜色档位
- 增加灯光式明暗按钮，让同一套主题色可以在亮色和暗色氛围之间切换
- 让卡片、背景、Element Plus 按钮、表格、输入框等基础控件尽量跟随主题变化

本次具体改动：

- 修改 [src/App.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/App.vue)
  - 新增全局鼠标光晕容器
  - 通过 `pointermove` 把鼠标坐标写入 CSS 变量
  - 不隐藏浏览器原生鼠标，避免影响输入框、按钮和表格操作
- 修改 [src/components/layout/AdminTopbar.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminTopbar.vue)
  - 新增主题色连续滑条
  - 新增亮色/暗色灯光切换按钮
  - 使用 `localStorage` 保存用户选中的主题色和亮暗模式
- 修改 [src/components/layout/AdminShell.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminShell.vue)
  - 给后台骨架增加 `app-theme-shell` 类，方便全局主题样式精准作用在后台区域
- 修改 [src/assets/main.css](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/assets/main.css)
  - 新增 `--theme-hue`、`--app-primary`、`--app-panel-bg` 等全局主题变量
  - 新增 `data-theme-mode="light|dim"` 两套亮暗变量
  - 将 `.panel-surface` 改为读取主题变量
  - 补充 Element Plus 常用变量，使按钮、表格、输入框跟随主题色
  - 新增小尺寸鼠标光晕和鼠标描边效果
- 修改 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 主题切换时重新读取 CSS 变量并更新 ECharts 关系图文字、标签和 tooltip 配色
  - 主题滑条连续拖动时使用 `requestAnimationFrame` 合并图表刷新，避免频繁重排
  - 关闭 ECharts 关系图默认 tooltip，避免和自定义悬浮卡片重叠
  - 悬浮卡片定位优先使用原生鼠标坐标，拿不到时用图表 offset 坐标兜底换算
  - 节点卡片和链路卡片分别使用不同高度估算，避免短节点卡片离鼠标过远
  - 将关系图滚轮缩放改为自定义低灵敏度缩放，保留画布平移和节点拖拽
- 修改 [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
  - 主题切换时同步更新链路延迟折线图配色
  - 悬浮卡片因为使用 `Teleport` 渲染到 `body`，所以单独补充主题样式，避免暗色模式下仍显示纯白卡片
  - 折线图每次渲染使用不合并模式清掉旧配置，避免“链路数据加载中”标题在数据加载完成后残留
- 修改 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - 增加链路 metrics 缓存，同一链路和同一时间范围重复悬停时直接复用已有数据
  - 对正在请求中的同一条链路做请求复用，避免鼠标移入移出导致重复请求

当前效果：

- 拖动顶部主题色滑条时，页面主题色会平滑连续变化
- 点击灯光按钮时，会在亮色和暗色氛围之间切换
- 鼠标移动时会出现轻量跟随光晕，在白色卡片和暗色卡片里都更容易定位

验证结果：

- `npx vue-tsc --noEmit` 已通过
- `npx vite build --configLoader native --outDir .codex-build-check` 已通过
- `.codex-build-check` 为一次性验证目录，验证后已删除

### 2026-05-30 第十三次开发：补齐离线节点展示、链路实时刷新与联调说明

目标：

- 继续按 `docs/NetWeaver-API-v1.8.md` 收口 Dashboard 前端
- 修正“节点掉线后直接从前端消失”的问题，让 `offline` 状态能被明确看到
- 修正“同一条链路长时间停留时，折线图只读缓存不再刷新”的问题
- 把真实联调时该改哪里、怎么改、改完要不要重启，全部写进 README

本次具体改动：

- 修改 [src/components/dashboard/OnlineNodeTable.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/OnlineNodeTable.vue)
  - `allNodes` 明确作为“全部节点”使用，不再只拿在线节点参与主体渲染
  - 节点表格从“在线节点”改为“节点列表”，现在会同时展示在线和离线节点
  - 新增 `offlineNodes`、`hasOnlyOfflineNodes` 等计算状态
  - 当当前没有在线节点、但后端仍返回离线节点时，页面不再空白，而是保留离线节点供排查
  - 已选中链路在轮询刷新时，`metrics` 现在会静默强制刷新，而不是长期复用旧缓存
  - 静默刷新时若本地已有旧曲线，会先继续显示旧曲线，再在后台取新数据，避免每 5 秒闪烁一次“加载中”
- 修改 [src/components/dashboard/NodeRelationGraph.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/NodeRelationGraph.vue)
  - 关系图数据源从“仅在线节点”改为“全部节点”
  - 离线节点现在会保留在图里，并通过灰色样式和状态字段体现 `offline`
  - 图标题右侧新增总数、在线数、离线数标签，方便直接核对后端状态同步结果
- 修改 [src/components/dashboard/LinkLatencyChart.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/dashboard/LinkLatencyChart.vue)
  - 节点悬浮卡片新增“公网地址”“当前状态”“最后心跳”
  - 为了适配新增字段，节点卡片的定位高度估算同步调整
- 修改 [README.md](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/README.md)
  - 顶部新增“当前联调配置（请优先看这一节）”
  - 明确说明现在不再依赖 `test/` Mock 后端目录
  - 明确说明开发联调时优先修改 `.env.development.local`
  - 明确说明 `VITE_API_BASE_URL` 与 `VITE_PROXY_TARGET` 的区别
  - 补充本机联调、校园网 / 局域网联调、直接请求后端三种配置示例
  - 补充 `npm install / npm run dev / npm run build / npm run preview` 的使用场景
  - 补充“改完环境变量后需要重启开发服务器”这一容易漏掉的步骤

当前页面效果变化：

- 节点掉线后，只要后端 `nodes` 接口把该节点状态改成 `offline`，前端就会继续把它显示在图和表格里
- 与离线节点相关的真实边仍然只依赖 `edges` 接口；只要后端移除边，前端关系图会同步断开
- 悬停同一条链路较长时间时，折线图数据会随轮询静默更新，不再长期停在旧缓存

验证结果：

- `npx vue-tsc --noEmit` 已通过
- `npm run build` 的 TypeScript 与 Vite transform 阶段已通过，但当前机器仍然无法写入 `frontend/dist/assets`，报错为 `拒绝访问`
- `npx vite build --configLoader native --outDir .codex-build-check` 已通过，用于确认本轮源码可以完整生产构建
- `.codex-build-check` 为一次性验证目录，验证后已立即删除，没有作为遗留目录保留
