// Jest configuration for the delivery app. Mirrors the customer/vendor/admin
// apps: no `jest-expo` preset, because that preset requires
// `@react-native/jest-preset`, an unmet peer, and jest died at startup without
// it (#957).
//
// Adding the peer as a devDependency is known to break the build — `6435cc39`
// reverted exactly that on the vendor app: it changes react-native's own
// peer-resolution key, re-keys every package downstream of it, and under
// `node-linker=hoisted` relocates react-native on disk until Metro can no
// longer resolve expo-modules-core.
//
// So the transform is spelled out instead. Nothing is added to the dependency
// graph, the lockfile is untouched, and Metro is unaffected.

module.exports = {
  testEnvironment: 'node',
  // babel-preset-expo (via the app's babel.config.js) handles TS, JSX and the
  // Flow syntax react-native ships. The metro caller is what tells the preset it
  // is compiling for a native target rather than web.
  transform: {
    '^.+\\.(js|jsx|mjs|ts|tsx)$': [
      'babel-jest',
      { caller: { name: 'metro', bundler: 'metro', platform: 'ios' } },
    ],
  },
  // react-native and the expo packages ship untranspiled ESM/Flow, so they must
  // go through babel rather than being skipped like the rest of node_modules.
  transformIgnorePatterns: [
    'node_modules/(?!(?:jest-)?react-native|@react-native|@react-native-community|@react-native-async-storage|expo|expo-.*|@expo|@expo-google-fonts|react-navigation|@react-navigation|nativewind|react-native-css-interop|react-native-svg|react-native-worklets|react-native-reanimated|@tesserix)',
  ],
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  setupFiles: ['<rootDir>/jest.setup.js'],
  // The iOS/Android build trees contain copies of the JS under
  // ios/build/generated — without this, jest discovers and runs tests twice.
  testPathIgnorePatterns: ['/node_modules/', '/ios/', '/android/', '/.expo/'],
};
