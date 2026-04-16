// shadcn CLI shim — not used in electron build. See electron.vite.config.ts.
// Also used as the vitest config for renderer unit/integration tests.
import { resolve } from 'path'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  resolve: {
    alias: {
      '@renderer': resolve('src/renderer/src'),
      '@renderer/app': resolve('src/renderer/src/app'),
      '@renderer/features': resolve('src/renderer/src/features'),
      '@renderer/shared': resolve('src/renderer/src/shared'),
      '@': resolve('src/renderer/src')
    }
  },
  plugins: [tailwindcss(), react()],
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/renderer/src/__tests__/**/*.test.{ts,tsx}'],
    setupFiles: []
  }
})
