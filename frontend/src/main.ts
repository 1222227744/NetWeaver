import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'
import './assets/main.css'

// 推荐新手按这个顺序理解前端启动流程：
// 1. 浏览器先打开 frontend/index.html
// 2. index.html 里加载 src/main.ts
// 3. main.ts 创建 Vue 应用，并把它挂到 #app 上
// 4. App.vue 决定整个前端当前显示哪个页面
// 5. 这个项目当前让 App.vue 直接显示 AdminDashboardPage.vue
//
// 所以如果你完全不知道“页面是从哪里冒出来的”，
// 前端排查的第一站永远应该是：index.html -> main.ts -> App.vue。

// createApp(App) 可以理解成：
// 以 App.vue 作为最外层页面，创建整个 Vue 应用。
const app = createApp(App)

// 全局注册 Element Plus。
// 注册后，我们就能在 .vue 文件里直接使用 <el-button>、<el-menu> 这类组件。
// 如果以后你发现 Element Plus 组件突然不能用了，先检查这一行还在不在。
app.use(ElementPlus)

// 把整个 Vue 应用渲染到 index.html 里的 <div id="app"></div> 上。
// mount('#app') 里的 '#app'，对应的就是 index.html 里那个 id="app" 的根节点。
app.mount('#app')
