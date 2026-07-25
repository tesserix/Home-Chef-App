import { useEffect, useState } from 'react';
import { Loader2, ShieldCheck } from 'lucide-react';

import { apiClient } from '@/shared/services/api-client';
import {
  setDeviceToken,
  setMFAChallengeHandler,
  type MFAChallengePrompt,
} from '@/shared/services/mfa';

// Covers the portal while a two-factor challenge is pending.
//
// Rendered above the router rather than as a route, because a challenge can be
// raised by ANY request. A route would mean every page has to know to redirect,
// and any page that forgot would show an empty state with no explanation.
// Covering the app is also honest: until the code is accepted every other
// request 403s, so there is nothing behind this.

export function MFAChallengeOverlay() {
  const [challenge, setChallenge] = useState<MFAChallengePrompt | null>(null);
  const [code, setCode] = useState('');
  const [backupCode, setBackupCode] = useState('');
  const [useBackup, setUseBackup] = useState(false);
  const [remember, setRemember] = useState(true);
  const [masked, setMasked] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setMFAChallengeHandler((next) => {
      // Ignore repeats while one is open: a dashboard firing several parallel
      // queries produces several 403s, and re-setting state on each would wipe
      // a half-typed code.
      setChallenge((current) => current ?? next);
    });
    return () => setMFAChallengeHandler(null);
  }, []);

  // Send the email code as soon as a challenge opens. Email is the only channel
  // deliverable without further interaction.
  useEffect(() => {
    if (!challenge?.channels.includes('email')) return;
    void (async () => {
      try {
        const res = await apiClient.post<{ masked: string }>('/auth/mfa/challenge', {
          channel: 'email',
        });
        setMasked(res.masked);
      } catch {
        setError('Could not send the code. Try resending.');
      }
    })();
  }, [challenge]);

  if (!challenge) return null;

  const canSubmit = useBackup ? backupCode.trim().length > 0 : code.length === 6;

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      const res = await apiClient.post<{ deviceToken: string }>('/auth/mfa/verify', {
        ...(useBackup ? { backupCode: backupCode.trim() } : { channel: 'email', code }),
        rememberDevice: remember,
        deviceLabel: navigator.userAgent.slice(0, 80),
        platform: 'web',
      });
      // Stored BEFORE the overlay closes, so the retried request already carries
      // it rather than bouncing straight into another 403.
      setDeviceToken(res.deviceToken);
      setChallenge(null);
      window.location.reload();
    } catch (err: unknown) {
      setError(
        (err as { error?: { message?: string } })?.error?.message ?? 'That code did not work.'
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="mfa-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-neutral-900/60 p-4"
    >
      <div className="w-full max-w-md space-y-4 rounded-xl bg-white p-6 shadow-xl">
        <h2 id="mfa-title" className="flex items-center gap-2 text-lg font-semibold text-neutral-900">
          <ShieldCheck className="h-5 w-5" /> Confirm it's you
        </h2>
        <p className="text-sm text-neutral-600">
          {useBackup
            ? 'Enter one of the backup codes you saved when you turned on two-factor.'
            : masked
              ? `We sent a 6-digit code to ${masked}.`
              : 'Sending your verification code…'}
        </p>

        {useBackup ? (
          <input
            value={backupCode}
            onChange={(e) => setBackupCode(e.target.value.toUpperCase())}
            placeholder="XXXX-XXXX-XXXX-XXXX"
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 font-mono text-sm"
          />
        ) : (
          <input
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            value={code}
            onChange={(e) => setCode(e.target.value.replace(/[^0-9]/g, '').slice(0, 6))}
            placeholder="123456"
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-center text-lg tracking-widest tabular-nums"
          />
        )}

        {error && <p className="text-sm text-red-700">{error}</p>}

        <label className="flex items-center gap-2 text-sm text-neutral-700">
          <input
            type="checkbox"
            checked={remember}
            onChange={(e) => setRemember(e.target.checked)}
          />
          Remember this browser
        </label>
        <p className="text-xs text-neutral-500">
          {remember
            ? "You won't be asked again on this browser until you remove it from Security settings."
            : "You'll be asked again next time you sign in."}
        </p>

        <button
          type="button"
          disabled={!canSubmit || busy}
          onClick={() => void submit()}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-neutral-900 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-40"
        >
          {busy && <Loader2 className="h-4 w-4 animate-spin" />} Verify
        </button>

        <button
          type="button"
          className="w-full text-sm text-neutral-600 underline"
          onClick={() => {
            setUseBackup((b) => !b);
            setCode('');
            setBackupCode('');
            setError(null);
          }}
        >
          {useBackup ? 'Use a verification code instead' : "I can't access my email"}
        </button>
      </div>
    </div>
  );
}
