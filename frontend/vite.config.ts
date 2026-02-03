import tailwindcss from "@tailwindcss/vite"
import react from '@vitejs/plugin-react'
import path from 'path'
import { defineConfig, loadEnv } from 'vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  // Check if we're in Wails development mode (has WAILS_DEV or running in Wails context)
  // In production web builds (Vercel), use the mock runtime
  const isWailsMode = env.WAILS_DEV === 'true' || process.env.WAILS === 'true'
  const useRealRuntime = isWailsMode && mode !== 'production'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
        // Use mock runtime for web builds (Vercel), real runtime for Wails desktop
        ...(!useRealRuntime && {
          "@wailsio/runtime": path.resolve(__dirname, "./src/lib/wails-runtime-mock.ts"),
        }),
      },
    },
    build: {
      // Let Vite handle chunking automatically to avoid React bundling issues
      // Manual chunking was causing "Cannot read properties of undefined" errors
      // when React-dependent packages were split across different chunks
      chunkSizeWarningLimit: 1500,
      reportCompressedSize: true,
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
