import Constants from 'expo-constants';
import { Platform } from 'react-native';

// "1.0.0+12" — semver version from app.json + native build number / iOS
// buildNumber so backend logs can correlate an exact binary. Mobile sends
// this on every API call as X-App-Version (see packages/mobile-shared/src/api/client.ts).
function buildString(): string | undefined {
  if (Platform.OS === 'ios') {
    return Constants.expoConfig?.ios?.buildNumber;
  }
  if (Platform.OS === 'android') {
    const code = Constants.expoConfig?.android?.versionCode;
    return code != null ? String(code) : undefined;
  }
  return undefined;
}

export const appVersion: string = (() => {
  const version = Constants.expoConfig?.version ?? '0.0.0';
  const build = buildString();
  return build ? `${version}+${build}` : version;
})();

export const appPlatform: 'ios' | 'android' | undefined =
  Platform.OS === 'ios' ? 'ios' : Platform.OS === 'android' ? 'android' : undefined;

// Compare two dotted-number semvers. Returns true when `a < b`.
// Mirrors the Go-side parseSemver in apps/api/middleware/version_check.go
// so the client renders the same wall the server enforces.
export function semverLess(a: string, b: string): boolean {
  const pa = parseSemver(a);
  const pb = parseSemver(b);
  for (let i = 0; i < 3; i++) {
    if (pa[i] !== pb[i]) return pa[i] < pb[i];
  }
  return false;
}

function parseSemver(v: string): [number, number, number] {
  const stripped = v.replace(/^v/, '').split(/[+-]/, 1)[0] ?? '0';
  const parts = stripped.split('.');
  const out: [number, number, number] = [0, 0, 0];
  for (let i = 0; i < 3 && i < parts.length; i++) {
    const n = parseInt(parts[i] ?? '0', 10);
    out[i] = Number.isFinite(n) ? n : 0;
  }
  return out;
}
