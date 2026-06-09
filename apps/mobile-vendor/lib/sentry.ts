// Sentry wrapper — keeps Sentry concerns out of _layout.tsx and gates
// the init on EXPO_PUBLIC_SENTRY_DSN so dev / preview builds without a
// DSN configured no-op cleanly instead of throwing on startup (which
// would brick every TestFlight install — see Wave 1 mobile design
// §4.1 risk #2).
//
// SENTRY_AUTH_TOKEN is consumed only at build time by the Expo config
// plugin for source-map upload; we don't read it at runtime.

import Constants from 'expo-constants';
import * as Sentry from '@sentry/react-native';
import { appVersion } from './app-version';

let initialized = false;

export function initSentry(): void {
  if (initialized) return;
  const dsn = process.env.EXPO_PUBLIC_SENTRY_DSN;
  if (!dsn) return;

  try {
    Sentry.init({
      dsn,
      environment:
        process.env.EXPO_PUBLIC_SENTRY_ENVIRONMENT ??
        (__DEV__ ? 'development' : 'production'),
      // Release id matches the EAS source-map upload target so traces
      // resolve to readable JS. Format: <slug>@<semver>+<buildNumber>.
      release: `${Constants.expoConfig?.slug ?? 'homechef-vendor'}@${appVersion}`,
      // 10% trace sampling to stay under the free-tier 5k/mo error cap
      // while keeping enough perf signal to spot regressions. All
      // unhandled errors + native crashes are captured regardless.
      tracesSampleRate: __DEV__ ? 0 : 0.1,
      enableAutoSessionTracking: true,
      // Keep PII out of breadcrumbs — chef email + phone live in our
      // user context if/when we attach it explicitly.
      sendDefaultPii: false,
    });
    initialized = true;
  } catch (err) {
    // Sentry init should never crash the app. If it does, log and move
    // on; the rest of the runtime carries on without telemetry.
    console.warn('[sentry] init failed', err);
  }
}

/** Identify the authenticated chef on Sentry — call after sign-in. */
export function setSentryUser(user: { id: string; email?: string }): void {
  if (!initialized) return;
  Sentry.setUser({ id: user.id, email: user.email });
}

/** Clear the Sentry identity on sign-out. */
export function clearSentryUser(): void {
  if (!initialized) return;
  Sentry.setUser(null);
}

/** Wrap the root component for navigation tracing. No-op if not initialized. */
export const wrapWithSentry: <T>(component: T) => T = (component) => {
  if (!initialized) return component;
  return Sentry.wrap(component as never) as unknown as typeof component;
};
