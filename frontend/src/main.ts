import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'
import './assets/main.css'

// createApp(App) 可以理解成：
// 以 App.vue 作为最外层页面，创建整个 Vue 应用。
const app = createApp(App)

// 全局注册 Element Plus。
// 注册后，我们就能在 .vue 文件里直接使用 <el-button>、<el-menu> 这类组件。
app.use(ElementPlus)

// 把整个 Vue 应用渲染到 index.html 里的 <div id="app"></div> 上。
app.mount('#app')
