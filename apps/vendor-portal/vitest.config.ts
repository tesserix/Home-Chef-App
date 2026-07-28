import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react-swc';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    // Node's ESM resolver rejects @tesserix/web's internal directory imports
    // ("…/components/accordion"). Inlining routes the package through Vite's
    // resolver instead, which handles them — without this, any test that
    // renders a component importing the design system fails to even load.
    server: { deps: { inline: ['@tesserix/web'] } },
    // Per-file `// @vitest-environment jsdom` opts DOM tests in; pure logic
    // tests stay on the faster node environment.
    environment: 'node',
  },
});
