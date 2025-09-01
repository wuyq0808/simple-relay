import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  root: '.',
  build: {
    outDir: 'dist',
    rollupOptions: {
      input: {
        'main-authenticated': './src/main-authenticated.tsx',
        'main-unauthenticated': './src/main-unauthenticated.tsx'
      }
    }
  }
})