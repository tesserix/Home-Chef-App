import type { ConfigContext, ExpoConfig } from 'expo/config';

import { withMapsKey } from './lib/maps-config';

// Extends app.json so the Google Maps API key can be injected from the
// environment instead of being committed (#759).
//
// react-native-maps crashes outright without it — "API key not found" — which
// took out live order tracking (components/tracking/DeliveryMap.tsx) and the
// chefs map (app/chefs-map.tsx). Expo prefers this file over app.json when both
// exist, and `config` is app.json already parsed.
//
// The key must be restricted in GCP to this app's package/bundle id plus its
// signing certificate — it ships inside the binary, as all mobile Maps keys do,
// so restriction is what secures it, not secrecy.
export default ({ config }: ConfigContext): ExpoConfig =>
  withMapsKey(config, process.env.GOOGLE_MAPS_API_KEY) as ExpoConfig;
