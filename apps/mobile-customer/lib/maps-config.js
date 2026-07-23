/**
 * Injects the Google Maps API key into the Expo config (#759).
 *
 * react-native-maps needs `com.google.android.geo.API_KEY` in AndroidManifest
 * and `GMSApiKey` on iOS. Without them the app does not degrade — it crashes
 * with "API key not found" the moment a map renders, which took out both live
 * order tracking and the chefs map.
 *
 * Plain JS on purpose: app.config.ts is transpiled by Expo without resolving
 * local `.ts` imports, so a TypeScript helper here cannot be required at config
 * load time.
 *
 * app.json is static, so the key is merged in at config-resolution time from
 * the environment rather than committed. The key still ships inside the binary
 * (unavoidable, and normal — mobile Maps keys are secured by package/bundle
 * restrictions, not secrecy), so restricting it in GCP is the step that matters.
 *
 * @param {Record<string, any>} config  the parsed app.json
 * @param {string | undefined} apiKey
 * @returns {Record<string, any>} a copy with the key applied
 */
function withMapsKey(config, apiKey) {
  const key = (apiKey || '').trim();
  const next = {
    ...config,
    android: { ...(config.android || {}) },
    ios: { ...(config.ios || {}) },
  };
  if (!key) return next;

  next.android.config = {
    ...((config.android && config.android.config) || {}),
    googleMaps: { apiKey: key },
  };
  next.ios.config = {
    ...((config.ios && config.ios.config) || {}),
    googleMapsApiKey: key,
  };
  return next;
}

module.exports = { withMapsKey };
