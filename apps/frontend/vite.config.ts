import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  root: '.',
  publicDir: 'public',
  build: {
    outDir: 'dist',
    rollupOptions: {
      input: {
        'authenticated': './public/authenticated.html',
        'unauthenticated': './public/unauthenticated.html'
      }
    }
  }
})