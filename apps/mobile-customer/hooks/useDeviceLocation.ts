import { useCallback, useEffect, useState } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

// useDeviceLocation — where the phone actually is.
//
// A guest has no saved address, so before this the home screen had no point to
// measure from: the header sat on a placeholder and the chef list was ranked
// un-located. This asks the OS once, keeps the answer, and hands back both the
// coordinates and a human label ("Bhubaneswar, Odisha").
//
// expo-location is imported LAZILY, inside the functions that use it. Importing
// a native module at the top of a module that the app loads on startup is what
// produced the "Cannot find native module ExpoCrypto" redbox on launch (see
// lib/auth/apple.ts) — a build without the native side linked would crash on
// boot instead of simply degrading to "no location".

const CACHE_KEY = 'device-location.v1';

/** How long to wait for a fix before giving up and offering the manual picker.
 *  Long enough for a cold GPS lock outdoors, short enough that a customer
 *  indoors isn't left staring at a spinner. */
const LOCATION_TIMEOUT_MS = 8000;

/** How old the OS's cached fix may be before we insist on a fresh one. Two
 *  minutes is far longer than it takes to walk somewhere that changes which
 *  kitchens can reach you, and short enough that yesterday's city never wins. */
const LAST_KNOWN_MAX_AGE_MS = 2 * 60 * 1000;

export interface DeviceLocation {
  lat: number;
  lng: number;
  /** "Bhubaneswar, Odisha" — for the header pill. Empty when reverse geocoding
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

/** Cached so a returning guest sees their area immediately, without a second
 *  permission prompt or a GPS fix they have to wait for. */
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
  lat: number,
  lng: number,
): Promise<Pick<DeviceLocation, 'label' | 'city' | 'region' | 'postalCode'>> {
  const empty = { label: '', city: '', region: '', postalCode: '' };
  try {
    const Location = await import('expo-location');
    const [place] = await Location.reverseGeocodeAsync({ latitude: lat, longitude: lng });
    if (!place) return empty;
    const city = place.city || place.subregion || place.district || '';
    const region = place.region || '';
    return {
      city,
      region,
      postalCode: place.postalCode || '',
      label: [city, region].filter(Boolean).join(', '),
    };
  } catch {
    return empty;
  }
}

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
  const [location, setLocation] = useState<DeviceLocation | null>(null);
  const [status, setStatus] = useState<LocationStatus>('idle');

  // Warm from cache so a returning guest never sees the placeholder again.
  useEffect(() => {
    let live = true;
    void readCache().then((cached) => {
      if (live && cached) {
        setLocation(cached);
        setStatus('ready');
      }
    });
    return () => {
      live = false;
    };
  }, []);

  const request = useCallback(async (): Promise<DeviceLocation | null> => {
    setStatus('locating');
    try {
      const Location = await import('expo-location');
      const { status: permission } = await Location.requestForegroundPermissionsAsync();
      if (permission !== 'granted') {
        setStatus('denied');
        return null;
      }
      // ONE deadline around the whole acquisition, not around a single call.
      //
      // Neither of these APIs reliably rejects when the platform simply cannot
      // produce a fix: indoors, in a lift, or on an emulator whose fused
      // provider has nothing to give, they just never settle. An earlier cut
      // put the timeout only on getCurrentPositionAsync — and then
      // getLastKnownPositionAsync was the one that hung, so the race was never
      // reached and the header sat on "Finding you…" indefinitely. Wrapping the
      // sequence means the UI resolves in bounded time whichever call stalls.
      const acquire = (async () => {
        // Last known first — instant when the OS already has a fix, so the
        // common case never waits on the radio.
        //
        // maxAge is not optional. Without it the OS hands back a fix of ANY
        // age, so someone who flew Delhi → Melbourne opens the app and is shown
        // Delhi kitchens, and an emulator hands back the Mountain View default
        // it was born with. A stale position is worse than none: it is
        // confidently wrong, and it feeds the delivery-distance maths.
        const last = await Location.getLastKnownPositionAsync({
          maxAge: LAST_KNOWN_MAX_AGE_MS,
        });
        if (last) return last;
        // Balanced accuracy: enough to price a delivery distance, and it
        // returns in a second or two rather than holding out for a GPS lock.
        return Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      })();

      const pos = await Promise.race([
        acquire,
        new Promise<null>((resolve) => setTimeout(() => resolve(null), LOCATION_TIMEOUT_MS)),
      ]);
      if (!pos) {
        setStatus('unavailable');
        return null;
      }
      const lat = pos.coords.latitude;
      const lng = pos.coords.longitude;
      const next: DeviceLocation = { lat, lng, ...(await describe(lat, lng)) };
      setLocation(next);
      setStatus('ready');
      void writeCache(next);
      return next;
    } catch {
      // A build without the native module linked, a simulator with no fix, or a
      // hardware failure all land here. Discovery stays un-located and the
      // manual picker still works — the screen must not break over this.
      setStatus('unavailable');
      return null;
    }
  }, []);

  return { location, status, request };
}
