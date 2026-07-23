import { describe, it, expect } from '@jest/globals';

import { withMapsKey } from './maps-config';

// #759: react-native-maps is used by components/tracking/DeliveryMap.tsx and
// app/chefs-map.tsx, but no Maps key was configured for any mobile app. Opening
// order tracking crashed outright with "API key not found. Check that
// <meta-data android:name="com.google.android.geo.API_KEY" ...>".
//
// The key ships inside the binary either way — mobile Maps keys are secured by
// package/bundle restrictions, not secrecy — but it must not be hard-coded into
// app.json, so it is injected at config-resolution time from the environment.

const base = {
  name: 'Fe3dr',
  android: { package: 'com.tesserix.homechef.customer', permissions: ['x'] },
  ios: { bundleIdentifier: 'com.tesserix.homechef.customer' },
};

describe('withMapsKey', () => {
  it('injects the key into both platforms', () => {
    const cfg = withMapsKey(base, 'KEY123');
    expect(cfg.android.config.googleMaps.apiKey).toBe('KEY123');
    expect(cfg.ios.config.googleMapsApiKey).toBe('KEY123');
  });

  it('leaves existing android config untouched', () => {
    // Dropping `package` or `permissions` would break the build far more
    // visibly than the missing key ever did.
    const cfg = withMapsKey(base, 'KEY123');
    expect(cfg.android.package).toBe('com.tesserix.homechef.customer');
    expect(cfg.android.permissions).toEqual(['x']);
    expect(cfg.ios.bundleIdentifier).toBe('com.tesserix.homechef.customer');
  });

  it('adds no config keys at all when the env var is absent', () => {
    // A literal "undefined" reaching AndroidManifest is worse than no key: it
    // looks configured and fails at runtime with the same opaque crash.
    const cfg = withMapsKey(base, undefined);
    expect(cfg.android.config).toBeUndefined();
    expect(cfg.ios.config).toBeUndefined();
  });

  it('treats an empty or whitespace value as absent', () => {
    expect(withMapsKey(base, '').android.config).toBeUndefined();
    expect(withMapsKey(base, '   ').ios.config).toBeUndefined();
  });

  it('does not mutate the config it was given', () => {
    const input = JSON.parse(JSON.stringify(base));
    withMapsKey(input, 'KEY123');
    expect(input.android.config).toBeUndefined();
  });

  it('merges alongside pre-existing platform config', () => {
    const withExisting = {
      ...base,
      android: { ...base.android, config: { someOther: true } },
    };
    const cfg = withMapsKey(withExisting, 'KEY123');
    expect(cfg.android.config.someOther).toBe(true);
    expect(cfg.android.config.googleMaps.apiKey).toBe('KEY123');
  });
});
