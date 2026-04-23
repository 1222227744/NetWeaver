import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  // 告诉 Vite：这个项目要支持 .vue 单文件组件。
  plugins: [vue()],
  resolve: {
    alias: {
      // 以后写 '@/components/xxx' 时，@ 就表示 src 目录。
      // 这样路径更短，也更容易看。
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    // 允许局域网或本机通过开发服务器访问当前项目。
    host: '0.0.0.0',
    // 本地开发默认端口。
    port: 5173
  }
})
