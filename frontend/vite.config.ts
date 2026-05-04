import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  // 这里读取 frontend/.env.* 里的变量。
  // 以后切换本机后端或局域网后端时，只改环境变量，不改源码。
  const env = loadEnv(mode, process.cwd(), '')

  // 代理目标默认指向本机 8080。
  // 如果后面要接队友机器，只需要把这个地址换成对方局域网 IP。
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://127.0.0.1:8080'

  return {
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
      port: 5173,
      proxy: {
        // 当前前端访问 /api/... 时，开发环境会自动转发到代理目标。
        // 这样浏览器仍然请求 /api/v1/dashboard/nodes，但实际由 Vite 帮我们代理到后端服务。
        '/api': {
          target: proxyTarget,
          changeOrigin: true
        }
      }
    },
    preview: {
      host: '0.0.0.0',
      port: 4173,
      proxy: {
        // 预览生产包时保持同样的代理规则，避免 npm run preview 时接口地址和开发环境不一致。
        '/api': {
          target: proxyTarget,
          changeOrigin: true
        }
      }
    }
  }
})
