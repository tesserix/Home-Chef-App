// Hooks over the two-factor endpoints.
//
// Plain useState rather than react-query: mobile-shared has no query-client
// peer dependency, and every one of these is a one-shot imperative action
// driven by a button press rather than cached server state worth invalidating.

import { useCallback, useEffect, useState } from 'react';

import { getServerErrorMessage } from '../api/error';
import { createMFAApi, type MFAApi, type MFAChannel, type MFAStatus, type TrustedDevice } from './api';
import { setDeviceToken } from './device-token';

import type { AxiosInstance } from 'axios';

/** Loads and exposes the caller's two-factor state for a settings screen. */
export function useMFAStatus(api: AxiosInstance) {
  const [status, setStatus] = useState<MFAStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setStatus(await createMFAApi(api).status());
    } catch (err: unknown) {
      setError(getServerErrorMessage(err, 'Could not load your security settings.'));
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { status, loading, error, reload };
}

/** The remembered-device list, with revocation. */
export function useTrustedDevices(api: AxiosInstance) {
  const [devices, setDevices] = useState<TrustedDevice[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      setDevices(await createMFAApi(api).devices());
      setError(null);
    } catch (err: unknown) {
      setError(getServerErrorMessage(err, 'Could not load your devices.'));
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const revoke = useCallback(
    async (id: string) => {
      // Optimistic: the row disappears immediately, and a failure reloads the
      // true list rather than leaving a phantom gap.
      setDevices((prev) => prev.filter((d) => d.id !== id));
      try {
        await createMFAApi(api).revokeDevice(id);
      } catch {
        await reload();
      }
    },
    [api, reload]
  );

  const revokeAll = useCallback(async () => {
    setDevices([]);
    try {
      await createMFAApi(api).revokeAllDevices();
    } catch {
      await reload();
    }
  }, [api, reload]);

  return { devices, loading, error, reload, revoke, revokeAll };
}

export interface ChallengeState {
  sending: boolean;
  verifying: boolean;
  /** Which channel a code was last sent on. */
  sentOn: MFAChannel | null;
  masked: string | null;
  error: string | null;
}

/**
 * Drives the login challenge: send a code, then verify it.
 *
 * On success the returned token is persisted before onVerified fires, so the
 * very next request already carries it. Doing it the other way round sends the
 * caller back into the app and straight into another 403.
 */
export function useMFAChallenge(api: AxiosInstance, onVerified: () => void) {
  const [state, setState] = useState<ChallengeState>({
    sending: false,
    verifying: false,
    sentOn: null,
    masked: null,
    error: null,
  });

  const send = useCallback(
    async (channel: MFAChannel) => {
      setState((s) => ({ ...s, sending: true, error: null }));
      try {
        const res = await createMFAApi(api).challenge(channel);
        setState((s) => ({ ...s, sending: false, sentOn: channel, masked: res.masked }));
        return res;
      } catch (err: unknown) {
        setState((s) => ({
          ...s,
          sending: false,
          error: getServerErrorMessage(err, 'Could not send the code.'),
        }));
        return null;
      }
    },
    [api]
  );

  const verify = useCallback(
    async (input: Parameters<MFAApi['verify']>[0]) => {
      setState((s) => ({ ...s, verifying: true, error: null }));
      try {
        const res = await createMFAApi(api).verify(input);
        await setDeviceToken(res.deviceToken);
        setState((s) => ({ ...s, verifying: false }));
        onVerified();
        return true;
      } catch (err: unknown) {
        setState((s) => ({
          ...s,
          verifying: false,
          error: getServerErrorMessage(err, 'That code did not work.'),
        }));
        return false;
      }
    },
    [api, onVerified]
  );

  return { ...state, send, verify };
}
