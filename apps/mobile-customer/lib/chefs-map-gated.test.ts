import { describe, it, expect } from '@jest/globals';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';

// The chefs map inflates a `react-native-maps` `MapView` with `PROVIDER_DEFAULT`,
// which on Android is Google Maps and throws `RuntimeException: API key not found`
// unless the manifest carries `com.google.android.geo.API_KEY`. No environment
// supplies one — `GOOGLE_MAPS_API_KEY` is unset everywhere, so `withMapsKey`
// no-ops. The route must therefore be unreachable while `CHEFS_MAP_ENABLED` is
// false, including by deep link: hiding the button alone does not stop
// `homechef-customer://chefs-map`. Guards both halves of that gate, because the
// next screen that wants a map will be copied from this one.

const ROOT = resolve(__dirname, '..');
const SCANNED = ['app', 'components', 'hooks', 'lib', 'store', 'types'];

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.tsx?$/.test(entry) && !full.endsWith('chefs-map-gated.test.ts') ? [full] : [];
  });
}

describe('the chefs map is unreachable while CHEFS_MAP_ENABLED is false', () => {
  const files = SCANNED.flatMap((d) => sourceFiles(join(ROOT, d)));

  it('no module navigates to /chefs-map without checking the flag', () => {
    const navigates =
      /router\.(?:push|replace|navigate)\(\s*['"`]\/chefs-map|href=\{?\s*['"`]\/chefs-map/;
    const offenders = files.filter((f) => {
      const src = readFileSync(f, 'utf8');
      return navigates.test(src) && !src.includes('CHEFS_MAP_ENABLED');
    });
    expect(offenders.map((f) => f.slice(ROOT.length + 1))).toEqual([]);
  });

  it('the screen itself refuses to render when the flag is off', () => {
    const screen = files.filter((f) => f.replace(/\\/g, '/').endsWith('app/chefs-map.tsx'));
    // Assert the file was found first, so a path typo fails loudly rather than
    // passing on an empty list.
    expect(screen.length).toBe(1);
    expect(readFileSync(screen[0] as string, 'utf8')).toContain('CHEFS_MAP_ENABLED');
  });
});
