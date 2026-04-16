import { resolve } from 'path'
import { defineConfig } from 'electron-vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  main: {},
  preload: {},
  renderer: {
    resolve: {
      alias: {
        '@renderer': resolve('src/renderer/src'),
        '@renderer/app': resolve('src/renderer/src/app'),
        '@renderer/features': resolve('src/renderer/src/features'),
        '@renderer/shared': resolve('src/renderer/src/shared'),
        '@': resolve('src/renderer/src')
      }
    },
    plugins: [tailwindcss(), react()]
  }
})
