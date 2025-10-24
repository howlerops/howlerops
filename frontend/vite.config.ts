import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from "@tailwindcss/vite"
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // Vendor chunks
          if (id.includes('node_modules')) {
            if (id.includes('react') || id.includes('react-dom')) {
              return 'vendor-react';
            }
            if (id.includes('@radix-ui')) {
              return 'vendor-ui';
            }
            if (id.includes('recharts')) {
              return 'vendor-charts';
            }
            if (id.includes('codemirror') || id.includes('@uiw/react-codemirror')) {
              return 'vendor-editor';
            }
            if (id.includes('date-fns') || id.includes('lodash')) {
              return 'vendor-utils';
            }
            if (id.includes('sql-formatter')) {
              return 'vendor-sql';
            }
            return 'vendor-misc';
          }
          // Feature chunks
          if (id.includes('src/components/query')) {
            return 'feature-query';
          }
          if (id.includes('src/components/sync')) {
            return 'feature-sync';
          }
          if (id.includes('src/lib/ai')) {
            return 'feature-ai';
          }
        },
      },
    },
    chunkSizeWarningLimit: 500,
    reportCompressedSize: true,
  },
  server: {
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
})
