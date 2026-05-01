# Frontend 开发记录

本文档用于记录 `frontend/` 目录内已经完成的开发内容、涉及文件和验证结果。目标是让前端侧改动可追踪、可回看、可对照。

## 当前范围

- 技术栈：`Vue 3 + TypeScript + Vite + Tailwind CSS + Element Plus`
- 当前主页面入口：[src/views/AdminDashboardPage.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/views/AdminDashboardPage.vue)
- 当前布局骨架：[src/components/layout/AdminShell.vue](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/frontend/src/components/layout/AdminShell.vue)

## 目录说明

```text
frontend/
├── src/
│   ├── api/                 # 前端请求层 / mock 数据入口
│   ├── assets/              # 全局样式与静态资源
│   ├── components/          # 公共组件
│   │   ├── dashboard/       # 业务组件（在线节点表格）
│   │   └── layout/          # 布局组件（侧边栏、顶部栏、页面骨架）
│   └── views/               # 页面级组件
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

- [NetWeaver v1.0 API接口文档.md](/d:/Documents/WorkSpace/30-Playground/frontend/planA/NetWeaver/NetWeaver%20v1.0%20API接口文档.md)

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
