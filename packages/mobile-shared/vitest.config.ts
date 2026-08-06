import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    globals: true,
  },
  // React Native's dev flag. Source branches on it; without it those modules
  // throw ReferenceError on import.
  define: { __DEV__: 'true' },
  resolve: {
    alias: {
      'expo-secure-store': new URL(
        './src/__mocks__/expo-secure-store.ts',
        import.meta.url,
      ).pathname,
      // Mock @tesserix/native since it's a peerDep not installed in this package
      '@tesserix/native': new URL('./src/__mocks__/@tesserix/native.ts', import.meta.url).pathname,
      // Mock react-native so screen files can be imported in the node test env
      'react-native': new URL('./src/__mocks__/react-native.ts', import.meta.url).pathname,
      'react-native-safe-area-context': new URL(
        './src/__mocks__/react-native-safe-area-context.ts',
        import.meta.url,
      ).pathname,
      'react-native-svg': new URL('./src/__mocks__/react-native-svg.ts', import.meta.url).pathname,
      'expo-constants': new URL('./src/__mocks__/expo-constants.ts', import.meta.url).pathname,
    },
  },
});
