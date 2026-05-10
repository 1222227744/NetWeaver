import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig, loadEnv } from 'vite'

// 这个文件是 Vite 的总配置。
// 你可以把 Vite 理解成“前端开发服务器 + 打包工具”。
//
// 新手最常遇到的两个问题，几乎都和这个文件有关：
// 1. 为什么我改了后端地址，前端还是连错地方？
// 2. 为什么我写了 @/components/xxx 这样的路径却还能生效？
//
// 上面两个问题的答案都在这个文件里：
// - 代理地址由 proxy 决定
// - @ 别名由 resolve.alias 决定

export default defineConfig(({ mode }) => {
  // 这里读取 frontend/.env.* 里的变量。
  // 以后切换本机后端或局域网后端时，只改环境变量，不改源码。
  // mode 代表当前运行模式，例如 development 或 production。
  const env = loadEnv(mode, process.cwd(), '')

  // 代理目标默认指向本机 8080。
  // 如果后面要接队友机器，只需要把这个地址换成对方局域网 IP。
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://127.0.0.1:8080'

  return {
    // 告诉 Vite：这个项目要支持 .vue 单文件组件。
    // 没有这个插件，Vite 不知道怎么处理 .vue 文件。
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
      // 如果这里不是 0.0.0.0，队友在同一局域网里通常就访问不到你的前端开发页。
      host: '0.0.0.0',
      // 本地开发默认端口。
      port: 5173,
      proxy: {
        // 当前前端访问 /api/... 时，开发环境会自动转发到代理目标。
        // 这样浏览器仍然请求 /api/v1/dashboard/nodes，但实际由 Vite 帮我们代理到后端服务。
        // 这就是“为什么代码里没写完整后端地址也能联调成功”的原因。
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
          // changeOrigin 为 true 的意思可以粗略理解成：
          // 让转发出去的请求更像是“真正发给目标服务器”的请求。
          changeOrigin: true
        }
      }
    }
  }
})
