import { describe, it, expect } from '@jest/globals';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';

// `DeliveryMap` inflates a `react-native-maps` `MapView` with `PROVIDER_DEFAULT`,
// which on Android is Google Maps and throws `RuntimeException: API key not found`
// unless the manifest carries `com.google.android.geo.API_KEY`. No environment
// supplies one — `GOOGLE_MAPS_API_KEY` is unset everywhere, so `withMapsKey`
// no-ops.
//
// This is a sharper hazard than the chefs map: that screen was always behind a
// flag, whereas `DeliveryMap` rendered automatically as soon as an order went en
// route, so the crash landed on the busiest post-purchase screen without anyone
// navigating to a map on purpose. It shipped ungated in build 19 because the
// original gating work scoped it out as "a separate surface".
//
// Both render paths must stay behind DELIVERY_MAP_ENABLED: the inline card on
// `order/[id]` and the full-bleed `order/[id]/track` screen, the latter also
// reachable by deep link and push notification.

const ROOT = resolve(__dirname, '..');
const SCANNED = ['app', 'components', 'hooks', 'lib', 'store', 'types'];
const SELF = 'delivery-map-gated.test.ts';

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.tsx?$/.test(entry) && !full.endsWith(SELF) ? [full] : [];
  });
}

describe('DeliveryMap is unreachable while DELIVERY_MAP_ENABLED is false', () => {
  const files = SCANNED.flatMap((d) => sourceFiles(join(ROOT, d)));
  const rel = (f: string) => f.slice(ROOT.length + 1).replace(/\\/g, '/');

  it('every module that renders DeliveryMap checks the flag', () => {
    const offenders = files.filter((f) => {
      const src = readFileSync(f, 'utf8');
      if (!/<DeliveryMap[\s/>]/.test(src)) return false;
      return !src.includes('DELIVERY_MAP_ENABLED');
    });
    expect(offenders.map(rel)).toEqual([]);
  });

  it('no module navigates to the track screen without checking the flag', () => {
    const navigates =
      /router\.(?:push|replace|navigate)\(\s*[`'"][^`'"]*\/track|href=\{?\s*[`'"][^`'"]*\/track/;
    const offenders = files.filter((f) => {
      const src = readFileSync(f, 'utf8');
      return navigates.test(src) && !src.includes('DELIVERY_MAP_ENABLED');
    });
    expect(offenders.map(rel)).toEqual([]);
  });

  it('the track screen itself refuses to render when the flag is off', () => {
    // Assert the file was found first, so a path change fails loudly rather
    // than passing vacuously on an empty list.
    const screen = files.filter((f) => rel(f) === 'app/order/[id]/track.tsx');
    expect(screen.length).toBe(1);
    expect(readFileSync(screen[0] as string, 'utf8')).toContain('DELIVERY_MAP_ENABLED');
  });

  it('the store description does not promise map tracking while it is off', () => {
    const config = JSON.parse(readFileSync(join(ROOT, 'store.config.json'), 'utf8'));
    const info = Object.values(config.apple.info)[0] as {
      description: string;
      releaseNotes: string;
    };
    expect(info.description).not.toMatch(/map tracking/i);
    expect(info.releaseNotes).not.toMatch(/on a map/i);
  });
});
