import legacyPlugin from '@vitejs/plugin-legacy'
import { viteLogo } from './src/core/config'
import Banner from 'vite-plugin-banner'
import * as path from 'path'
import { loadEnv } from 'vite'
import vuePlugin from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import VueFilePathPlugin from './vitePlugin/componentName/index.js'
import { svgBuilder } from 'vite-auto-import-svg'
import vueRootValidator from 'vite-check-multiple-dom'
import { AddSecret } from './vitePlugin/secret'
import UnoCSS from '@unocss/vite'

const createManualChunk = (id) => {
  if (!id.includes('node_modules')) {
    return undefined
  }
  if (id.includes('ace-builds') || id.includes('vue3-ace-editor')) {
    return 'vendor-ace'
  }
  if (id.includes('@wangeditor/')) {
    return 'vendor-wangeditor'
  }
  if (id.includes('@vue-office/')) {
    return 'vendor-vue-office'
  }
  if (id.includes('vue-cropper')) {
    return 'vendor-image-tools'
  }
  if (id.includes('echarts') || id.includes('vue-echarts')) {
    return 'vendor-echarts'
  }
  if (id.includes('@form-create/') || id.includes('vform3-builds')) {
    return 'vendor-form-create'
  }
  return undefined
}

// @see https://cn.vitejs.dev/config/
export default ({ command, mode }) => {
  AddSecret('')
  const env = loadEnv(mode, process.cwd())
  viteLogo(env)

  const timestamp = Date.parse(new Date())
  const isServe = command === 'serve'
  const isFastBuild = command === 'build' && env.VITE_ENABLE_LEGACY === 'false'
  const isReleaseBuild = command === 'build' && !isFastBuild
  const reportCompressedSize = env.VITE_REPORT_COMPRESSED_SIZE
    ? env.VITE_REPORT_COMPRESSED_SIZE === 'true'
    : isReleaseBuild

  const optimizeDeps = {}

  const alias = {
    '@': path.resolve(__dirname, './src'),
    vue$: 'vue/dist/vue.runtime.esm-bundler.js'
  }

  const esbuild = {}

  const rollupOptions = {
    output: {
      entryFileNames: 'assets/087AC4D233B64EB0[name].[hash].js',
      chunkFileNames: 'assets/087AC4D233B64EB0[name].[hash].js',
      assetFileNames: 'assets/087AC4D233B64EB0[name].[hash].[ext]',
      manualChunks: createManualChunk
    }
  }

  const base = '/'
  const root = './'
  const outDir = 'dist'
  const allowedHosts = [
    'localhost',
    '127.0.0.1',
    '.ngrok-free.app',
    '.ngrok.app'
  ]

  if (env.VITE_ALLOWED_HOSTS) {
    env.VITE_ALLOWED_HOSTS
      .split(',')
      .map((host) => host.trim())
      .filter(Boolean)
      .forEach((host) => {
        if (!allowedHosts.includes(host)) {
          allowedHosts.push(host)
        }
      })
  }

  const config = {
    base: base, // 编译后js导入的资源路径
    root: root, // index.html文件所在位置
    publicDir: 'public', // 静态资源文件夹
    resolve: {
      alias
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: 'modern-compiler' // or "modern"
        }
      }
    },
    server: {
      // 如果使用docker-compose开发模式，设置为false
      host: '0.0.0.0',
      open: isServe && env.VITE_AUTO_OPEN !== 'false',
      port: Number(env.VITE_CLI_PORT),
      allowedHosts,
      proxy: {
        // 把key的路径代理到target位置
        // detail: https://cli.vuejs.org/config/#devserver-proxy
        [env.VITE_BASE_API]: {
          // 需要代理的路径   例如 '/api'
          target: `${env.VITE_BASE_PATH}:${env.VITE_SERVER_PORT}/`, // 代理到 目标路径
          changeOrigin: true,
          rewrite: (path) =>
            path.replace(new RegExp('^' + env.VITE_BASE_API), '')
        },
        '/plugin': {
          // 需要代理的路径   例如 '/api'
          target: `https://plugin.gin-vue-admin.com/api/`, // 代理到 目标路径
          changeOrigin: true,
          rewrite: (path) =>
            path.replace(new RegExp('^/plugin'), '')
        }
      }
    },
    build: {
      minify: isFastBuild ? 'esbuild' : 'terser', // 本地快速构建走 esbuild，完整构建保留 terser
      cssMinify: isFastBuild ? 'esbuild' : undefined,
      manifest: false, // 是否产出manifest.json
      sourcemap: false, // 是否产出sourcemap.json
      outDir: outDir, // 产出目录
      reportCompressedSize,
      chunkSizeWarningLimit: isFastBuild ? 2200 : 1400,
      terserOptions: isFastBuild
        ? undefined
        : {
          compress: {
            //生产环境时移除console
            drop_console: true,
            drop_debugger: true
          }
        },
      rollupOptions
    },
    esbuild,
    optimizeDeps,
    plugins: [
      env.VITE_POSITION === 'open' &&
      vueDevTools({ launchEditor: env.VITE_EDITOR }),
      isReleaseBuild && legacyPlugin({
        targets: [
          'Android > 39',
          'Chrome >= 60',
          'Safari >= 10.1',
          'iOS >= 10.3',
          'Firefox >= 54',
          'Edge >= 15'
        ]
      }),
      vuePlugin(),
      svgBuilder(['./src/plugin/', './src/assets/icons/'], base, outDir, 'assets', mode),
      [Banner(`\n Build based on gin-vue-admin \n Time : ${timestamp}`)],
      VueFilePathPlugin('./src/pathInfo.json'),
      UnoCSS(),
      vueRootValidator()
    ].filter(Boolean)
  }
  return config
}
