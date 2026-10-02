// Jest configuration for the vendor app.
//
// Mirrors apps/mobile-customer/jest.config.js. The `jest-expo` preset this app
// used before needs `@react-native/jest-preset`, an unmet peer, so every suite
// died at startup and none of the seven here had ever run (#957 fixed only the
// customer app).
//
// Adding the peer as a devDependency is NOT the fix: `6435cc39` reverted exactly
// that, because it re-keys react-native's peer resolution, relocates the package
// under `node-linker=hoisted`, and leaves Metro unable to resolve
// expo-modules-core. Spelling the transform out instead touches neither the
// dependency graph nor the lockfile.

module.exports = {
  testEnvironment: 'node',
  transform: {
    '^.+\\.(js|jsx|mjs|ts|tsx)$': [
      'babel-jest',
      { caller: { name: 'metro', bundler: 'metro', platform: 'ios' } },
    ],
  },
  transformIgnorePatterns: [
    'node_modules/(?!(?:jest-)?react-native|@react-native|@react-native-community|@react-native-async-storage|expo|expo-.*|@expo|@expo-google-fonts|react-navigation|@react-navigation|nativewind|react-native-css-interop|react-native-svg|react-native-worklets|react-native-reanimated|@tesserix)',
  ],
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  moduleNameMapper: {
    '^react$': require.resolve('react'),
    // `nativewind/babel` sets jsxImportSource, so every .tsx pulls the interop
    // runtime, which resolves its web build under node and dies reaching
    // StyleSheet. These suites test logic, not rendering, so the plain React
    // runtime is enough — and a render test could not prove anything about the
    // interop anyway (see apps/mobile-customer/lib/pressable-style.test.ts).
    '^react-native-css-interop/jsx-(dev-)?runtime$': 'react/jsx-runtime',
  },
  setupFiles: ['<rootDir>/jest.setup.js'],
  testPathIgnorePatterns: ['/node_modules/', '/ios/', '/android/', '/.expo/'],
};
