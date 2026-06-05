import { defineConfig } from 'vite'

export default defineConfig({
  publicDir: 'public',
  build: { outDir: 'dist' },
  server: { port: 5173 }
})
