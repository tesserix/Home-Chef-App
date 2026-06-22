import Constants from 'expo-constants';
import { Platform } from 'react-native';

// "1.0.0+12" — semver version from app.json + native build number so backend
// logs can correlate an exact binary. Sent on every API call as
// X-App-Version (see packages/mobile-shared/src/api/client.ts).
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
