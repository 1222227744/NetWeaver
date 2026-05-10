/// <reference types="vite/client" />

// 这个文件不是业务页面，它是“给 TypeScript 看”的说明文件。
// 作用只有一个：告诉 TS，import.meta.env 里会有哪些环境变量。
//
// 为什么需要它？
// 因为在 TypeScript 看来，环境变量默认是不确定的。
// 如果我们不提前声明，代码里写 import.meta.env.VITE_PROXY_TARGET 时，
// 编辑器就可能报类型错误，或者提示“不知道这个字段是什么”。

interface ImportMetaEnv {
  // 可选的完整后端地址。
  // 如果这里有值，axios 会直接请求这个地址。
  readonly VITE_API_BASE_URL?: string

  // Vite 开发服务器的代理目标地址。
  // 本项目当前更推荐改这个变量，因为这样浏览器仍然请求 /api/...，
  // 由 Vite 在开发阶段帮我们转发到真实后端。
  readonly VITE_PROXY_TARGET?: string
}

interface ImportMeta {
  // import.meta.env 的类型定义最终会落到这里。
  readonly env: ImportMetaEnv
}
