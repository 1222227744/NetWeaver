export default {
  // PostCSS 可以理解成“CSS 的加工流水线”。
  // 它会在最终输出 CSS 之前，先把我们写的样式做一遍处理。
  plugins: {
    // tailwindcss：负责把 @tailwind 指令展开成真正的 CSS。
    tailwindcss: {},

    // autoprefixer：自动补浏览器前缀。
    // 例如某些 CSS 特性在不同浏览器里需要 -webkit- 之类的前缀，
    // 它会帮我们自动处理，减少手写兼容代码。
    autoprefixer: {}
  }
}
