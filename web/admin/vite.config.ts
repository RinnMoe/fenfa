import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: '/static/admin/',
  resolve: {
    alias: {
      // Use the full build of Vue that includes the runtime compiler
      'vue': 'vue/dist/vue.esm-bundler.js'
    }
  },
  server: {
    host: '127.0.0.1',
    port: 5174,
    strictPort: true,
    hmr: { host: '127.0.0.1', port: 5174 },
    proxy: {
      '/api': 'http://127.0.0.1:8000',
      '/admin/api': 'http://127.0.0.1:8000',
      '/d': 'http://127.0.0.1:8000',
      '/ios': 'http://127.0.0.1:8000',
      '/udid': 'http://127.0.0.1:8000',
      '/upload': 'http://127.0.0.1:8000',
      '/admin/exports': 'http://127.0.0.1:8000',
    },
  },
  build: {
    outDir: '../../internal/web/dist/admin',
    emptyOutDir: true,
    rollupOptions: {
      input: 'index.html',
      output: {
        entryFileNames: 'app.js',
        chunkFileNames: (chunkInfo) => {
          const name = String(chunkInfo.name || 'chunk')
            .replace(/^_+/, 'chunk-')
            .replace(/[^a-zA-Z0-9_-]/g, '-')
          return `assets/${name}-[hash].js`
        },
        assetFileNames: (chunkInfo) => {
          if (chunkInfo.name && chunkInfo.name.endsWith('.css')) return 'style.css'
          return '[name][extname]'
        },
      },
    },
  },
})

