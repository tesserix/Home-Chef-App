// Jest environment setup for the delivery app.
//
// AsyncStorage is a native module, so under jest it resolves to null and any
// suite that touches it dies with "Native module is null, cannot access legacy
// storage" — taking the whole worker with it. Stores here persist through zustand's
// `persist` middleware, so any suite touching one could never run.
//
// The package ships no jest mock of its own, so this is a minimal in-memory
// stand-in covering the surface `createJSONStorage` uses.

// Metro injects this; react-native's own entry point reads it at import time.
global.__DEV__ = true;

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
