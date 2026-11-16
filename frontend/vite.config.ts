import { defineConfig, loadEnv, Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import checker from 'vite-plugin-checker'
import { resolve } from 'path'

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (character) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  })[character] || character)
}

function isSafeImageUrl(value: string): boolean {
  const trimmed = value.trim()
  if ((trimmed.startsWith('/') && !trimmed.startsWith('//')) || /^data:image\//i.test(trimmed)) {
    return true
  }
  try {
    const parsed = new URL(trimmed)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

function injectBranding(html: string, config: { site_name?: string; site_logo?: string }): string {
  let brandedHtml = html
  const siteName = config.site_name?.trim()
  if (siteName) {
    brandedHtml = brandedHtml.replace(
      /<title>[^<]*<\/title>/i,
      `<title>${escapeHtml(siteName)} - AI API Gateway</title>`,
    )
  }

  const siteLogo = config.site_logo?.trim()
  if (siteLogo && isSafeImageUrl(siteLogo)) {
    brandedHtml = brandedHtml.replace(
      /<link\s+rel=["']icon["'][^>]*>/i,
      `<link rel="icon" href="${escapeHtml(siteLogo)}" />`,
    )
  }
  return brandedHtml
}

/**
 * Vite 插件：开发模式下注入公开配置到 index.html
 * 与生产模式的后端注入行为保持一致，消除闪烁
 */
function injectPublicSettings(backendUrl: string): Plugin {
  return {
    name: 'inject-public-settings',
    apply: 'serve',
    transformIndexHtml: {
      order: 'pre',
      async handler(html) {
        try {
          const response = await fetch(`${backendUrl}/api/v1/settings/public`, {
            signal: AbortSignal.timeout(2000)
          })
          if (response.ok) {
            const data = await response.json()
            if (data.code === 0 && data.data) {
              const script = `<script>window.__APP_CONFIG__=${JSON.stringify(data.data)};</script>`
              return injectBranding(html, data.data).replace('</head>', `${script}\n</head>`)
            }
          }
        } catch (e) {
          console.warn('[vite] 无法获取公开配置，将回退到 API 调用:', (e as Error).message)
        }
        return html
      }
    }
  }
}

/**
 * Vite 插件（仅构建）：在 index.html 里提前预加载语言包和主字体。
 * 语言包是动态加载的，默认要等入口脚本执行后才开始下载；这里用一段内联脚本按
 * 与 i18n 相同的规则判断语言，随 HTML 一起立刻开始下载，已登录用户同时预加载后台文案。
 * 内联脚本带 CSP nonce 占位符，由后端在返回 HTML 时替换。
 */
function preloadCriticalAssets(): Plugin {
  return {
    name: 'preload-critical-assets',
    apply: 'build',
    transformIndexHtml: {
      order: 'post',
      handler(html, ctx) {
        const bundle = ctx.bundle
        if (!bundle) return html
        const base = '/'
        const chunkFor = (suffix: string) => {
          for (const item of Object.values(bundle)) {
            if (item.type === 'chunk' && item.facadeModuleId?.replace(/\\/g, '/').endsWith(suffix)) {
              return base + item.fileName
            }
          }
          return ''
        }
        const files = {
          zh: chunkFor('/src/i18n/locales/zh/base.ts'),
          en: chunkFor('/src/i18n/locales/en/base.ts'),
          zhAdmin: chunkFor('/src/i18n/locales/zh/adminPack.ts'),
          enAdmin: chunkFor('/src/i18n/locales/en/adminPack.ts')
        }
        const tags: string[] = []
        const font = Object.values(bundle).find(
          (item) => item.type === 'asset' && /geist-latin-wght-normal.*\.woff2$/.test(item.fileName)
        )
        if (font) {
          tags.push(`<link rel="preload" as="font" type="font/woff2" href="${base}${font.fileName}" crossorigin />`)
        }
        if (files.zh && files.en) {
          const script = `(function(){try{var f=${JSON.stringify(files)};var s=localStorage.getItem('sub2api_locale');` +
            `var l=s==='zh'||s==='en'?s:(navigator.language||'').toLowerCase().indexOf('zh')===0?'zh':'en';` +
            `var a=[f[l]];if(localStorage.getItem('auth_token'))a.push(f[l+'Admin']);` +
            `a.forEach(function(h){if(!h)return;var k=document.createElement('link');k.rel='modulepreload';k.href=h;document.head.appendChild(k)})}catch(e){}})()`
          tags.push(`<script nonce="__CSP_NONCE_VALUE__">${script}</script>`)
        }
        return tags.length ? html.replace('</head>', `    ${tags.join('\n    ')}\n  </head>`) : html
      }
    }
  }
}

export default defineConfig(({ mode }) => {
  // 加载环境变量
  const env = loadEnv(mode, process.cwd(), '')
  const backendUrl = env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080'
  const devPort = Number(env.VITE_DEV_PORT || 3000)

  return {
    plugins: [
      vue(),
      checker({
        vueTsc: true
      }),
      injectPublicSettings(backendUrl),
      preloadCriticalAssets()
    ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      // 使用 vue-i18n 运行时版本，避免 CSP unsafe-eval 问题；
      // v10 起 JIT 编译恒开启，消息编译为 AST 对象而非 JS 代码，无需 unsafe-eval
      'vue-i18n': 'vue-i18n/dist/vue-i18n.runtime.esm-bundler.js'
    }
  },
  build: {
    outDir: '../backend/internal/web/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        /**
         * 手动分包配置
         * 分离第三方库并按功能合并应用代码，避免循环依赖
         */
        manualChunks(id: string) {
          if (id.includes('node_modules')) {
            // Vue 核心库
            if (
              id.includes('/vue/') ||
              id.includes('/vue-router/') ||
              id.includes('/pinia/') ||
              id.includes('/@vue/')
            ) {
              return 'vendor-vue'
            }

            // 表格导出库体积很大（约 140KB gzip），只在导出时动态加载，必须独占一个包，
            // 否则会被同包的其他库连带进普通页面。
            if (id.includes('/xlsx/')) {
              return 'vendor-xlsx'
            }

            if (id.includes('/@vueuse/')) {
              return 'vendor-vueuse'
            }

            // 图表库
            if (id.includes('/chart.js/') || id.includes('/vue-chartjs/') || id.includes('/@kurkle/color/')) {
              return 'vendor-chart'
            }

            // 国际化
            if (id.includes('/vue-i18n/') || id.includes('/@intlify/')) {
              return 'vendor-i18n'
            }

            // Stripe 仅在支付流程中按需加载，避免进入首页公共依赖。
            if (id.includes('/@stripe/stripe-js/')) {
              return 'vendor-stripe'
            }

            // 请求库每个页面都用，单独成包以便长期缓存。
            if (id.includes('/axios/')) {
              return 'vendor-http'
            }

            // 其余第三方库（支付 SDK、二维码、拖拽、Markdown、新手引导等）不再合并成一个公共包：
            // 合并后入口脚本会静态依赖它，首页就要下载并执行全部库（Airwallex SDK 还会去拉第三方脚本）。
            // 交给 Rollup 按实际引用自动分包，只有用到的页面才加载。
            return undefined
          }

          // 应用代码：按入口点自动分包，不手动干预
          // 这样可以避免循环依赖，同时保持合理的 chunk 数量
        }
      }
    }
  },
    server: {
      host: '0.0.0.0',
      port: devPort,
      proxy: {
        '/api': {
          target: backendUrl,
          changeOrigin: true
        },
        '/v1': {
          target: backendUrl,
          changeOrigin: true
        },
        '/setup': {
          target: backendUrl,
          changeOrigin: true
        },
        // 新手引导里自托管的 CC Switch 安装包（后端 DOWNLOADS_DIR）
        '/downloads': {
          target: backendUrl,
          changeOrigin: true
        }
      }
    }
  }
})
