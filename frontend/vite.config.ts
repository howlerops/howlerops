import tailwindcss from "@tailwindcss/vite"
import react from '@vitejs/plugin-react'
import path from 'path'
import { defineConfig, loadEnv } from 'vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  // Mock @wailsio/runtime for all production builds and web-only builds.
  // The real Wails runtime is only injected during `wails3 dev` which sets
  // up the module dynamically. For standalone builds (npm run build, Vercel),
  // we need the mock.
  const needsMockRuntime = mode === 'production' || process.env.VERCEL === '1'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
        // Use mock runtime for production builds; real runtime only during wails3 dev
        ...(needsMockRuntime && {
          "@wailsio/runtime": path.resolve(__dirname, "./src/lib/wails-runtime-mock.ts"),
        }),
      },
    },
    build: {
      chunkSizeWarningLimit: 1100, // AG Grid community module is ~1MB and can't be split
      reportCompressedSize: true,
      rollupOptions: {
        // Suppress warnings for modules that are both statically and dynamically
        // imported. The dynamic imports are intentional (SSR/web compatibility,
        // cached loading patterns) and can't be converted to static.
        onwarn(warning, defaultHandler) {
          if (warning.message?.includes('dynamic import will not move module into another chunk')) return
          defaultHandler(warning)
        },
        output: {
          manualChunks(id) {
            // React core must stay together to prevent "undefined" errors
            if (id.includes('node_modules/react/') ||
                id.includes('node_modules/react-dom/') ||
                id.includes('node_modules/react-router-dom/') ||
                id.includes('node_modules/scheduler/')) {
              return 'vendor-react'
            }
            // CodeMirror editor ecosystem (~300KB)
            if (id.includes('node_modules/@codemirror/') ||
                id.includes('node_modules/codemirror') ||
                id.includes('node_modules/@lezer/')) {
              return 'vendor-editor'
            }
            // AG Grid (~400KB)
            if (id.includes('node_modules/ag-grid')) {
              return 'vendor-grid'
            }
            // Recharts + D3 deps (~200KB)
            if (id.includes('node_modules/recharts') ||
                id.includes('node_modules/d3-')) {
              return 'vendor-charts'
            }
            // React Flow (~200KB)
            if (id.includes('node_modules/reactflow') ||
                id.includes('node_modules/@reactflow/')) {
              return 'vendor-flow'
            }
            // Radix UI primitives
            if (id.includes('node_modules/@radix-ui/')) {
              return 'vendor-radix'
            }
            // TanStack (react-query, react-table, react-virtual)
            if (id.includes('node_modules/@tanstack/')) {
              return 'vendor-data'
            }
            // Framer Motion
            if (id.includes('node_modules/framer-motion/')) {
              return 'vendor-animation'
            }
            // Utility libraries
            if (id.includes('node_modules/lodash') ||
                id.includes('node_modules/date-fns') ||
                id.includes('node_modules/clsx') ||
                id.includes('node_modules/tailwind-merge') ||
                id.includes('node_modules/class-variance-authority')) {
              return 'vendor-utils'
            }
            // Network/gRPC/Protobuf
            if (id.includes('node_modules/@grpc/') ||
                id.includes('node_modules/@improbable-eng/') ||
                id.includes('node_modules/grpc-web') ||
                id.includes('node_modules/google-protobuf') ||
                id.includes('node_modules/axios') ||
                id.includes('node_modules/socket.io')) {
              return 'vendor-network'
            }
          },
        },
      },
    },
    server: {
      port: 5173,
      proxy: {
        // REST API proxy (for backward compatibility)
        '/api': {
          target: 'http://localhost:8500',
          changeOrigin: true,
          secure: false,
        },
        // gRPC-Web HTTP Gateway proxy
        '/grpc': {
          target: 'http://localhost:8500',
          changeOrigin: true,
          secure: false,
          rewrite: (path) => path.replace(/^\/grpc/, ''),
        },
        // WebSocket proxy (for legacy WebSocket connections)
        '/ws': {
          target: 'ws://localhost:8500',
          ws: true,
          changeOrigin: true,
        },
        // gRPC service proxies
        '/sqlstudio.database.DatabaseService': {
          target: 'http://localhost:8500',
          changeOrigin: true,
          secure: false,
        },
        '/sqlstudio.query.QueryService': {
          target: 'http://localhost:8500',
          changeOrigin: true,
          secure: false,
        },
        '/sqlstudio.table.TableService': {
          target: 'http://localhost:8500',
          changeOrigin: true,
          secure: false,
        },
        '/sqlstudio.auth.AuthService': {
          target: 'http://localhost:8500',
          changeOrigin: true,
          secure: false,
        },
      },
    },
  }
})
