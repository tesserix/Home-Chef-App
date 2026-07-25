import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Loader2, ShieldCheck, Smartphone, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

import { apiClient } from '@/shared/services/api-client';

// Security settings for the vendor portal — the web counterpart of the shared
// mobile screen.
//
// Backup codes are shown once, on their own panel, and require an explicit
// acknowledgement. They cannot be retrieved afterwards, so surfacing them in a
// toast would guarantee a support queue of locked-out chefs.

interface MFAStatus {
  featureEnabled: boolean;
  enabled: boolean;
  emailEnrolled: boolean;
  phoneEnrolled: boolean;
  maskedEmail: string;
  maskedPhone: string;
  backupCodesRemaining: number;
  channels: Array<'email' | 'phone'>;
}

interface TrustedDevice {
  id: string;
  app: string;
  label: string;
  platform: string;
  lastSeenAt: string;
}

export function SecurityPage() {
  const queryClient = useQueryClient();
  const [enrollingEmail, setEnrollingEmail] = useState(false);
  const [emailCode, setEmailCode] = useState('');
  const [freshCodes, setFreshCodes] = useState<string[] | null>(null);

  const { data: status, isLoading } = useQuery<MFAStatus>({
    queryKey: ['mfa', 'status'],
    queryFn: () => apiClient.get<MFAStatus>('/auth/mfa/status'),
  });

  const { data: devices } = useQuery<{ devices: TrustedDevice[] }>({
    queryKey: ['mfa', 'devices'],
    queryFn: () => apiClient.get<{ devices: TrustedDevice[] }>('/auth/mfa/devices'),
    enabled: !!status?.enabled,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['mfa'] });
  };

  const fail = (fallback: string) => (err: unknown) =>
    toast.error((err as { error?: { message?: string } })?.error?.message ?? fallback);

  const requestEmail = useMutation({
    mutationFn: () => apiClient.post('/auth/mfa/enroll/email/request', {}),
    onSuccess: () => setEnrollingEmail(true),
    onError: fail('Could not send the code.'),
  });

  const verifyEmail = useMutation({
    mutationFn: () => apiClient.post('/auth/mfa/enroll/email/verify', { code: emailCode }),
    onSuccess: () => {
      setEnrollingEmail(false);
      setEmailCode('');
      invalidate();
    },
    onError: fail('That code did not work.'),
  });

  const enable = useMutation({
    mutationFn: () => apiClient.post<{ backupCodes: string[] }>('/auth/mfa/enable', {}),
    onSuccess: (res) => setFreshCodes(res.backupCodes),
    onError: fail('Could not turn two-factor on.'),
  });

  const disable = useMutation({
    mutationFn: () => apiClient.post('/auth/mfa/disable', {}),
    onSuccess: invalidate,
    onError: fail('Could not turn two-factor off.'),
  });

  const regenerate = useMutation({
    mutationFn: () => apiClient.post<{ backupCodes: string[] }>('/auth/mfa/backup-codes/regenerate', {}),
    onSuccess: (res) => setFreshCodes(res.backupCodes),
    onError: fail('Could not generate new codes.'),
  });

  const revoke = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/auth/mfa/devices/${id}`),
    onSuccess: invalidate,
    onError: fail('Could not remove that device.'),
  });

  if (isLoading) {
    return (
      <div className="flex min-h-64 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
      </div>
    );
  }

  if (!status?.featureEnabled) {
    return (
      <div className="mx-auto max-w-2xl p-6">
        <p className="text-sm text-neutral-600">Two-factor authentication isn't available yet.</p>
      </div>
    );
  }

  if (freshCodes) {
    return (
      <div className="mx-auto max-w-2xl space-y-4 p-6">
        <h1 className="text-xl font-semibold text-neutral-900">Save your backup codes</h1>
        <p className="text-sm text-neutral-600">
          Each code works once. They are the only way back in if you lose access to your email and
          phone. This is the only time they are shown.
        </p>
        <div className="grid grid-cols-2 gap-2 rounded-lg bg-neutral-50 p-4 font-mono text-sm tabular-nums">
          {freshCodes.map((c) => (
            <span key={c} className="text-center text-neutral-900">
              {c}
            </span>
          ))}
        </div>
        <button
          type="button"
          className="w-full rounded-lg bg-neutral-900 px-4 py-2.5 text-sm font-semibold text-white"
          onClick={() => {
            setFreshCodes(null);
            invalidate();
          }}
        >
          I've saved them
        </button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl space-y-8 p-6">
      <header className="space-y-1">
        <h1 className="flex items-center gap-2 text-xl font-semibold text-neutral-900">
          <ShieldCheck className="h-5 w-5" /> Two-factor authentication
        </h1>
        <p className="text-sm text-neutral-600">
          {status.enabled
            ? "You'll be asked for a code when you sign in on a browser you haven't remembered."
            : 'Add a second step at sign-in so a stolen password on its own is not enough.'}
        </p>
      </header>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold text-neutral-900">Where we send your code</h2>
        <div className="flex items-center justify-between rounded-lg border border-neutral-200 p-4">
          <div>
            <p className="text-sm font-medium text-neutral-900">Email</p>
            <p className="text-sm text-neutral-500">
              {status.emailEnrolled ? status.maskedEmail : 'Not set up'}
            </p>
          </div>
          {!status.emailEnrolled && !enrollingEmail && (
            <button
              type="button"
              className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-medium"
              disabled={requestEmail.isPending}
              onClick={() => requestEmail.mutate()}
            >
              {requestEmail.isPending ? 'Sending…' : 'Set up'}
            </button>
          )}
        </div>

        {enrollingEmail && (
          <div className="flex gap-2 rounded-lg border border-neutral-200 p-4">
            <input
              inputMode="numeric"
              maxLength={6}
              value={emailCode}
              onChange={(e) => setEmailCode(e.target.value.replace(/[^0-9]/g, '').slice(0, 6))}
              placeholder="6-digit code"
              className="flex-1 rounded-lg border border-neutral-300 px-3 py-2 text-sm tabular-nums"
            />
            <button
              type="button"
              className="rounded-lg bg-neutral-900 px-4 py-2 text-sm font-semibold text-white disabled:opacity-40"
              disabled={emailCode.length !== 6 || verifyEmail.isPending}
              onClick={() => verifyEmail.mutate()}
            >
              Confirm
            </button>
          </div>
        )}
      </section>

      <section className="space-y-3">
        {status.enabled ? (
          <>
            <p className="text-sm text-neutral-600">
              {status.backupCodesRemaining} backup code
              {status.backupCodesRemaining === 1 ? '' : 's'} left
            </p>
            <button
              type="button"
              className="rounded-lg border border-neutral-300 px-4 py-2 text-sm font-medium"
              onClick={() => regenerate.mutate()}
            >
              Generate new backup codes
            </button>
            <button
              type="button"
              className="block text-sm font-medium text-red-700"
              onClick={() => {
                if (
                  window.confirm(
                    'Turn off two-factor? Your password alone will be enough to sign in, and every remembered browser will be forgotten.'
                  )
                ) {
                  disable.mutate();
                }
              }}
            >
              Turn off two-factor
            </button>
          </>
        ) : (
          <button
            type="button"
            className="w-full rounded-lg bg-neutral-900 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-40"
            disabled={!status.emailEnrolled && !status.phoneEnrolled}
            onClick={() => enable.mutate()}
          >
            Turn on two-factor
          </button>
        )}
        {!status.enabled && !status.emailEnrolled && !status.phoneEnrolled && (
          <p className="text-sm text-neutral-500">Set up email first.</p>
        )}
      </section>

      {status.enabled && (
        <section className="space-y-3">
          <h2 className="text-sm font-semibold text-neutral-900">Remembered devices</h2>
          <p className="text-sm text-neutral-500">
            These skip the code at sign-in. They stay trusted until you remove them.
          </p>
          {(devices?.devices ?? []).length === 0 ? (
            <p className="text-sm text-neutral-500">No remembered devices.</p>
          ) : (
            (devices?.devices ?? []).map((d) => (
              <div
                key={d.id}
                className="flex items-center justify-between rounded-lg border border-neutral-200 p-4"
              >
                <div className="flex items-center gap-3">
                  <Smartphone className="h-4 w-4 text-neutral-400" />
                  <div>
                    <p className="text-sm font-medium text-neutral-900">
                      {d.label || d.platform || 'Unknown device'}
                    </p>
                    <p className="text-sm text-neutral-500">
                      {d.app} · last used {new Date(d.lastSeenAt).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <button
                  type="button"
                  aria-label={`Remove ${d.label || 'device'}`}
                  className="text-neutral-500 hover:text-red-700"
                  onClick={() => revoke.mutate(d.id)}
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))
          )}
        </section>
      )}
    </div>
  );
}

export default SecurityPage;
