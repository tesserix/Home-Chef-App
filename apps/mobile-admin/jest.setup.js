// Jest environment setup for the admin app.
//
// Native modules resolve to null under jest, so any suite reaching one dies and
// takes the worker with it. Two stand-ins and one global are what the vendor
// suites need; each is the minimum surface the code under test actually calls.

// react-native/index.js reads __DEV__ at module scope, so importing anything
// from 'react-native' (AppState in useLiveUpdates) throws without it.
global.__DEV__ = true;

// expo-modules-core reads its primitives off `globalThis.expo`, which the native
// runtime installs and jest has no equivalent of — so every expo-* package (
// device, notifications, secure-store) throws at import. One shim covers them
// all; the per-package mocks below only stub behaviour the tests rely on.
class NoopEventEmitter {
  addListener() {
    return { remove() {} };
  }
  removeAllListeners() {}
  removeSubscription() {}
  emit() {}
}
globalThis.expo = globalThis.expo ?? {
  EventEmitter: NoopEventEmitter,
  NativeModule: class extends NoopEventEmitter {},
  SharedObject: class extends NoopEventEmitter {},
  SharedRef: class extends NoopEventEmitter {},
  // requireNativeModule throws on a missing entry, and every expo-* package asks
  // for its own, so the registry answers for any name.
  modules: new Proxy(
    {},
    {
      get: () => new NoopEventEmitter(),
      has: () => true,
    },
  ),
  uuidv4: () => '00000000-0000-4000-8000-000000000000',
  uuidv5: () => '00000000-0000-5000-8000-000000000000',
  getViewConfig: () => null,
  reloadAppAsync: () => Promise.resolve(),
};

// expo-secure-store backs the shared auth storage that the api client imports,
// so it is pulled in by every hook that talks to the API.
jest.mock('expo-secure-store', () => {
  const store = {};
  return {
    __esModule: true,
    getItemAsync: jest.fn((key) => Promise.resolve(key in store ? store[key] : null)),
    setItemAsync: jest.fn((key, value) => {
      store[key] = String(value);
      return Promise.resolve();
    }),
    deleteItemAsync: jest.fn((key) => {
      delete store[key];
      return Promise.resolve();
    }),
    isAvailableAsync: jest.fn(() => Promise.resolve(true)),
    WHEN_UNLOCKED: 'whenUnlocked',
    AFTER_FIRST_UNLOCK: 'afterFirstUnlock',
  };
});

// expo-notifications reaches expo-modules-core's EventEmitter at import time; it
// arrives through the shared push helper that the live-update hooks import.
jest.mock('expo-notifications', () => ({
  __esModule: true,
  getExpoPushTokenAsync: jest.fn(),
  getDevicePushTokenAsync: jest.fn(),
  getPermissionsAsync: jest.fn(() => Promise.resolve({ status: 'granted' })),
  requestPermissionsAsync: jest.fn(() => Promise.resolve({ status: 'granted' })),
  setNotificationHandler: jest.fn(),
  addNotificationReceivedListener: jest.fn(() => ({ remove: jest.fn() })),
  addNotificationResponseReceivedListener: jest.fn(() => ({ remove: jest.fn() })),
  setNotificationChannelAsync: jest.fn(),
  AndroidImportance: { MAX: 5, HIGH: 4, DEFAULT: 3 },
}));

// Platform resolves through the PlatformConstants TurboModule, which does not
// exist off-device. Suites here are platform-agnostic; iOS is the arbitrary pick.
jest.mock('react-native/Libraries/Utilities/Platform', () => ({
  __esModule: true,
  default: {
    OS: 'ios',
    Version: 26,
    isPad: false,
    isTV: false,
    select: (spec) => (spec.ios !== undefined ? spec.ios : spec.default),
  },
}));

// expo-constants reads RN's NativeModules at import; lib/app-version.ts needs
// only the version fields off expoConfig.
jest.mock('expo-constants', () => ({
  __esModule: true,
  default: {
    expoConfig: {
      version: '1.0.0',
      ios: { buildNumber: '1', bundleIdentifier: 'com.tesserix.homechef.admin' },
      android: { versionCode: 1, package: 'com.tesserix.homechef.admin' },
    },
  },
}));

// The mfa barrel re-exports MFAChallengeScreen, so importing the two helpers
// lib/api.ts actually uses would pull a component tree whose StyleSheet.create
// needs the native runtime. The tests here exercise request logic, not the
// challenge UI.
jest.mock('@homechef/mobile-shared/mfa', () => ({
  __esModule: true,
  getDeviceToken: jest.fn(() => Promise.resolve(null)),
  emitMFARequired: jest.fn(),
}));

// expo-router drags in react-native-safe-area-context, whose native component
// spec throws under jest. lib/api.ts imports it only to route on a hard 401.
jest.mock('expo-router', () => ({
  __esModule: true,
  router: { push: jest.fn(), replace: jest.fn(), back: jest.fn() },
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), back: jest.fn() }),
  useLocalSearchParams: () => ({}),
  usePathname: () => '/',
}));

jest.mock('@react-native-async-storage/async-storage', () => {
  let store = {};
  return {
    __esModule: true,
    default: {
      getItem: jest.fn((key) => Promise.resolve(key in store ? store[key] : null)),
      setItem: jest.fn((key, value) => {
        store[key] = String(value);
        return Promise.resolve();
      }),
      removeItem: jest.fn((key) => {
        delete store[key];
        return Promise.resolve();
      }),
      clear: jest.fn(() => {
        store = {};
        return Promise.resolve();
      }),
      getAllKeys: jest.fn(() => Promise.resolve(Object.keys(store))),
      multiGet: jest.fn((keys) =>
        Promise.resolve(keys.map((k) => [k, k in store ? store[k] : null])),
      ),
      multiSet: jest.fn((pairs) => {
        pairs.forEach(([k, v]) => {
          store[k] = String(v);
        });
        return Promise.resolve();
      }),
      multiRemove: jest.fn((keys) => {
        keys.forEach((k) => delete store[k]);
        return Promise.resolve();
      }),
    },
  };
});
