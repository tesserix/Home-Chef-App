import { useEffect } from 'react';
import { create } from 'zustand';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { placeLabel } from '../lib/address-label';

// useDeviceLocation — where the phone actually is.
//
// A guest has no saved address, so before this the home screen had no point to
// measure from: the header sat on a placeholder and the chef list was ranked
// un-located. This asks the OS once, keeps the answer, and hands back both the
// coordinates and a human label ("12 Queen St, Auckland").
//
// SHARED state, not per-component. An earlier cut used useState inside the hook,
// so every caller got its own private copy: the header requested a fix while
// useCustomerCoords sat forever on an untouched second copy that never asked for
// anything and never received it. One position, one request, one subscriber list.
//
// expo-location is imported LAZILY, inside the function that uses it. Importing
// a native module at the top of a module the app loads on startup is what
// produced the "Cannot find native module ExpoCrypto" redbox on launch (see
// lib/auth/apple.ts) — a build without the native side linked would crash on
// boot instead of simply degrading to "no location".

const CACHE_KEY = 'device-location.v1';

/** Ceiling on the WHOLE acquisition. See the deadline in `request` for why this
 *  cannot be attached to any single call. */
const LOCATION_TIMEOUT_MS = 8000;

/** How old the OS's cached fix may be before we insist on a fresh one. Two
 *  minutes is far longer than it takes to walk somewhere that changes which
 *  kitchens can reach you, and short enough that yesterday's city never wins. */
const LAST_KNOWN_MAX_AGE_MS = 2 * 60 * 1000;

export interface DeviceLocation {
  lat: number;
  lng: number;
  /** "12 Queen St, Auckland" — for the header pill. Empty when reverse geocoding
   *  found nothing, which is not a failure: the coordinates still work. */
  label: string;
  city: string;
  region: string;
  postalCode: string;
}

export type LocationStatus =
  | 'idle'
  | 'locating'
  /** The OS said no. The manual area picker stays the way forward. */
  | 'denied'
  | 'unavailable'
  | 'ready';

interface LocationState {
  location: DeviceLocation | null;
  status: LocationStatus;
  hydrated: boolean;
  hydrate: () => Promise<void>;
  request: () => Promise<DeviceLocation | null>;
}

async function readCache(): Promise<DeviceLocation | null> {
  try {
    const raw = await AsyncStorage.getItem(CACHE_KEY);
    return raw ? (JSON.parse(raw) as DeviceLocation) : null;
  } catch {
    return null;
  }
}

async function writeCache(loc: DeviceLocation): Promise<void> {
  try {
    await AsyncStorage.setItem(CACHE_KEY, JSON.stringify(loc));
  } catch {
    // A cache miss next launch is not worth surfacing.
  }
}

/** Reverse geocode through the OS. Never throws — a missing label is cosmetic,
 *  the coordinates are the part that matters. */
async function describe(
  Location: typeof import('expo-location'),
  lat: number,
  lng: number,
): Promise<Pick<DeviceLocation, 'label' | 'city' | 'region' | 'postalCode'>> {
  const empty = { label: '', city: '', region: '', postalCode: '' };
  try {
    const [place] = await Location.reverseGeocodeAsync({ latitude: lat, longitude: lng });
    if (!place) return empty;
    const city = place.city || place.subregion || place.district || '';
    const region = place.region || '';
    return {
      city,
      region,
      postalCode: place.postalCode || '',
      label: placeLabel({ ...place, city }),
    };
  } catch {
    return empty;
  }
}

/** Distinguishes "the OS refused" from "we got nothing", so the UI can say which. */
const DENIED = Symbol('denied');
const TIMED_OUT = Symbol('timed-out');

export const useDeviceLocationStore = create<LocationState>((set, get) => ({
  location: null,
  status: 'idle',
  hydrated: false,

  hydrate: async () => {
    if (get().hydrated) return;
    set({ hydrated: true });
    const cached = await readCache();
    // Only adopt the cache if nothing better has arrived while we were reading.
    if (cached && get().status === 'idle') set({ location: cached, status: 'ready' });
  },

  request: async () => {
    if (get().status === 'locating') return null; // already in flight
    set({ status: 'locating' });

    // The deadline covers EVERYTHING: the dynamic import, the permission
    // prompt, and both position calls.
    //
    // This is the whole point, and the previous cut got it wrong twice. First
    // the timeout wrapped only getCurrentPositionAsync, so a hang in
    // getLastKnownPositionAsync sailed past it. Then it wrapped both position
    // calls — but it was still *inside* the await chain, so it only started
    // AFTER `import('expo-location')` and requestForegroundPermissionsAsync had
    // resolved, and a stall in either left the header on "Finding you…"
    // forever. None of these APIs reliably reject when the platform simply
    // cannot answer. A deadline that starts after the thing that hangs is not a
    // deadline, so this one is created first and races the entire operation.
    const deadline = new Promise<typeof TIMED_OUT>((resolve) =>
      setTimeout(() => resolve(TIMED_OUT), LOCATION_TIMEOUT_MS),
    );

    const work = (async (): Promise<DeviceLocation | typeof DENIED | null> => {
      const Location = await import('expo-location');
      const { status: permission } = await Location.requestForegroundPermissionsAsync();
      if (permission !== 'granted') return DENIED;

      // Last known first — instant when the OS already has a fix, so the common
      // case never waits on the radio.
      //
      // maxAge is not optional. Without it the OS hands back a fix of ANY age,
      // so someone who flew Delhi → Melbourne opens the app and is shown Delhi
      // kitchens, and an emulator hands back the Mountain View default it was
      // born with. A stale position is worse than none: it is confidently
      // wrong, and it feeds the delivery-distance maths.
      const last = await Location.getLastKnownPositionAsync({
        maxAge: LAST_KNOWN_MAX_AGE_MS,
      });
      // Balanced accuracy: enough to price a delivery distance, and it returns
      // in a second or two rather than holding out for a GPS lock.
      const pos =
        last ??
        (await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced }));
      if (!pos) return null;

      const lat = pos.coords.latitude;
      const lng = pos.coords.longitude;
      return { lat, lng, ...(await describe(Location, lat, lng)) };
    })().catch(() => null); // a build without the native module lands here

    const result = await Promise.race([work, deadline]);

    if (result === TIMED_OUT) {
      set({ status: 'unavailable' });
      return null;
    }
    if (result === DENIED) {
      set({ status: 'denied' });
      return null;
    }
    if (!result) {
      set({ status: 'unavailable' });
      return null;
    }
    set({ location: result, status: 'ready' });
    void writeCache(result);
    return result;
  },
}));

/**
 * The device's location, resolved on demand.
 *
 * `request()` is what triggers the OS permission prompt, so the caller decides
 * when the customer sees it — never on a cold start before they have any idea
 * what the app is.
 */
export function useDeviceLocation(): {
  location: DeviceLocation | null;
  status: LocationStatus;
  request: () => Promise<DeviceLocation | null>;
} {
  const location = useDeviceLocationStore((s) => s.location);
  const status = useDeviceLocationStore((s) => s.status);
  const request = useDeviceLocationStore((s) => s.request);
  const hydrate = useDeviceLocationStore((s) => s.hydrate);

  // Warm from cache so a returning guest never sees the placeholder again.
  // Guarded inside the store, so mounting several consumers reads once.
  useEffect(() => {
    void hydrate();
  }, [hydrate]);

  return { location, status, request };
}
