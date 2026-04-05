import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    globals: true,
  },
  resolve: {
    alias: {
      // Mock @tesserix/native since it's a peerDep not installed in this package
      '@tesserix/native': new URL('./src/__mocks__/@tesserix/native.ts', import.meta.url).pathname,
    },
  },
});
