// Is react-native-webview actually in THIS binary?
//
// The package is a native module, so a JS bundle that imports it will hard-crash
// with "RNCWebViewModule could not be found" on any build that predates it —
// which is every reload during development and every OTA update shipped ahead of
// a store build. The in-app Cashfree sheet therefore asks before using it, and
// the hosted checkout stays as the fallback rather than the app dying.
import type { ComponentType } from 'react';

let cached: ComponentType<Record<string, unknown>> | null | undefined;

/** The WebView component, or null when this binary has no native module. */
export function getWebView(): ComponentType<Record<string, unknown>> | null {
  if (cached !== undefined) return cached;
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const mod = require('react-native-webview') as {
      WebView?: ComponentType<Record<string, unknown>>;
    };
    cached = mod?.WebView ?? null;
  } catch {
    cached = null;
  }
  return cached;
}

/** Whether the in-app payment sheet can be opened on this build. */
export function hasInAppWebView(): boolean {
  return getWebView() !== null;
}
