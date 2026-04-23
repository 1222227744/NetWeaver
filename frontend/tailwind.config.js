/** @type {import('tailwindcss').Config} */
export default {
  // Tailwind 会扫描这些文件，只有这里真正用到的类名才会被打包。
  // 如果以后你新增目录，但 Tailwind 样式不生效，先检查这里有没有把新目录写进去。
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // 这里扩展了一组项目主色。
        // 以后在模板里可以直接写 bg-brand-500、text-brand-700 这类类名。
        brand: {
          50: '#eefbf7',
          100: '#d5f5ec',
          200: '#aee8d8',
          300: '#7fd4be',
          400: '#4cb59f',
          500: '#258d7d',
          600: '#1b7165',
          700: '#175b53',
          800: '#164944',
          900: '#153d39'
        }
      },
      boxShadow: {
        // 这是自定义阴影名字。
        // 页面里写 shadow-panel 时，实际用的就是这段阴影配置。
        panel: '0 22px 50px -28px rgba(15, 23, 42, 0.35)'
      }
    }
  },
  plugins: []
}
