# Frontend 开发记录

本文档用于记录 `frontend/` 目录内已经完成的开发内容、涉及文件和验证结果。目标是让前端侧改动可追踪、可回看、可对照。

## 当前范围

- 技术栈：`Vue 3 + TypeScript + Vite + Tailwind CSS + Element Plus`
- 当前主页面入口：[src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
- 当前布局骨架：[src/components/layout/AdminShell.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminShell.vue)

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

- 由于接口文档中的 `metrics` 需要 `node_id + target_id`，而当前页面还没有“目标对端节点选择器”，本次先用“当前悬停节点自己的 node_id”去接通真实请求链路
- 这意味着：如果后端还没有按这个最小调用方式返回数据，前端会正确显示空态，而不是自己伪造一条曲线
