import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';

import type { AxiosInstance } from 'axios';

import { MFAChallengeScreen } from './MFAChallengeScreen';
import { loadDeviceToken } from './device-token';
import type { MFAChannel } from './api';

// Holds the pending two-factor challenge for a whole app.
//
// The alternative — a challenge route each app pushes onto its own navigator —
// means every app reimplements the same interception, and any screen that
// forgets shows a broken empty state instead. A challenge can be raised by ANY
// request, so it is handled above the router: while one is pending this renders
// over the app entirely, which is honest, because until it is answered every
// other request 403s and there is nothing behind it.

interface PendingChallenge {
  channels: MFAChannel[];
  masked: Partial<Record<MFAChannel, string>>;
}

interface MFAGateValue {
  /** Raise a challenge. Wire this to createApiClient's onMFARequired. */
  requireMFA: (challenge: PendingChallenge) => void;
  /** True while the challenge screen is covering the app. */
  challenging: boolean;
}

const MFAGateContext = createContext<MFAGateValue | null>(null);

// Module-level bridge to the provider.
//
// Each app builds its axios client at module scope, long before any component
// mounts, so onMFARequired cannot close over React state directly. The provider
// publishes its handler here on mount and the client calls through it — the same
// shape DialogProvider uses for showAlertOutsideReact.
let mountedRequireMFA: ((challenge: PendingChallenge) => void) | null = null;

/**
 * Pass this as createApiClient's onMFARequired. Safe to reference at module
 * scope: it is a no-op until a provider mounts.
 */
export function emitMFARequired(challenge: PendingChallenge): void {
  mountedRequireMFA?.(challenge);
}

export interface MFAGateProviderProps {
  children: ReactNode;
  api: AxiosInstance;
  /** Signs the user out — the only way past the challenge without a code. */
  onSignOut: () => void;
  /** Called after a successful challenge, e.g. to refetch what 403'd. */
  onVerified?: () => void;
  accentColor?: string;
  deviceLabel?: string;
}

export function MFAGateProvider({
  children,
  api,
  onSignOut,
  onVerified,
  accentColor,
  deviceLabel,
}: MFAGateProviderProps) {
  const [pending, setPending] = useState<PendingChallenge | null>(null);

  // Warm the in-memory device token before anything can fire a request, so a
  // remembered device is not challenged again on every cold start.
  useEffect(() => {
    void loadDeviceToken();
  }, []);

  const requireMFA = useCallback((challenge: PendingChallenge) => {
    // Ignore repeats: a screen firing five parallel queries produces five 403s,
    // and re-setting state on each would remount the challenge and wipe a
    // half-typed code.
    setPending((current) => current ?? challenge);
  }, []);

  const handleVerified = useCallback(() => {
    setPending(null);
    onVerified?.();
  }, [onVerified]);

  const handleSignOut = useCallback(() => {
    setPending(null);
    onSignOut();
  }, [onSignOut]);

  // Publish to module-level callers (the axios interceptor) while mounted.
  useEffect(() => {
    mountedRequireMFA = requireMFA;
    return () => {
      if (mountedRequireMFA === requireMFA) mountedRequireMFA = null;
    };
  }, [requireMFA]);

  const value = useMemo(
    () => ({ requireMFA, challenging: pending !== null }),
    [requireMFA, pending]
  );

  return (
    <MFAGateContext.Provider value={value}>
      {pending ? (
        <MFAChallengeScreen
          api={api}
          channels={pending.channels}
          masked={pending.masked}
          onVerified={handleVerified}
          onSignOut={handleSignOut}
          accentColor={accentColor}
          deviceLabel={deviceLabel}
        />
      ) : (
        children
      )}
    </MFAGateContext.Provider>
  );
}

/**
 * Access the gate. Returns a no-op outside a provider rather than throwing —
 * the api client wires requireMFA at construction time, which can happen before
 * any provider mounts, and a throw there would take down app start-up.
 */
export function useMFAGate(): MFAGateValue {
  return useContext(MFAGateContext) ?? { requireMFA: () => {}, challenging: false };
}
