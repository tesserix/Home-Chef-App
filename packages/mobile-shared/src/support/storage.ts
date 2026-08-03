// secureStoreKV — a KVStorage adapter over expo-secure-store, used to persist
// the support send-outbox so queued messages survive a cold start. SecureStore
// keys allow only [A-Za-z0-9._-], so keys are sanitised at the boundary (the
// outbox uses a ":" separator that's valid for MMKV/AsyncStorage but not
// SecureStore).
//
// Goes through the hardened secureGet/secureSet/secureDelete rather than
// expo-secure-store directly: a Keychain that throws (unsigned simulator, or the
// transient cold-start race behind #428) would otherwise drop the very queue
// this exists to protect.
//
// Note: SecureStore caps values at ~2KB on iOS. A typical outbox (a few short
// queued messages) is well under that.
import { secureDelete, secureGet, secureSet } from "../utils/storage";

import type { KVStorage } from "./outbox";

const safeKey = (k: string): string => k.replace(/[^A-Za-z0-9._-]/g, "_");

export const secureStoreKV: KVStorage = {
  getItem: (key) => secureGet(safeKey(key)),
  setItem: (key, value) => secureSet(safeKey(key), value),
  removeItem: (key) => secureDelete(safeKey(key)),
};
