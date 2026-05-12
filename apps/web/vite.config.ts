import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react-swc';
import path from 'path';

export default defineConfig({
  plugins: [tailwindcss(), react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  // @tesserix/web's deep `export *` chains aren't all preserved by Vite's
  // optimizeDeps bundler — only the directly-named export `cn` survives.
  // We work around this in src by not importing DS components from the
  // package (we only use `cn`), so optimizeDeps can pre-bundle normally.
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
      '/bff': {
        target: 'https://identity.fe3dr.com',
        changeOrigin: true,
        rewrite: (path: string) => path.replace(/^\/bff/, ''),
      },
    },
  },
  build: {
    outDir: 'dist',
    // Sourcemaps off in production — they leak unminified source + auth/payment
    // logic to the browser. Keep on for dev so devtools mapping works locally.
    sourcemap: process.env.NODE_ENV !== 'production',
  },
});
