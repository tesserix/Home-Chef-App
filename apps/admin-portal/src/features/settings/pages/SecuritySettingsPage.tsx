import { useId, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { ApiError } from '@/shared/types';
import {
  ArrowLeft,
  ShieldCheck,
  Lock,
  KeyRound,
  Smartphone,
  Copy,
  CheckCircle2,
  X,
  Plus,
  Trash2,
  RefreshCw,
} from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';

interface SecurityPolicy {
  passwordMinLength: number;
  passwordRequireUpper: boolean;
  passwordRequireLower: boolean;
  passwordRequireNumber: boolean;
  passwordRequireSpecial: boolean;
  sessionAccessTtlHours: number;
  sessionRefreshTtlDays: number;
  twoFactorRequiredForAdmins: boolean;
  twoFactorExemptEmails?: string[];
}

export default function SecuritySettingsPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to="/settings"
          className="rounded-lg p-2 text-muted-foreground hover:bg-secondary hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="page-title">Security</h1>
          <p className="page-description">
            Password rules, sessions, two-factor auth, and integration keys
          </p>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <PasswordPolicySection />
        <SessionPolicySection />
        <TwoFactorSection />
        <ApiKeysSection />
      </div>
    </div>
  );
}

// ============================================================
// Shared feedback banner hook
// ============================================================

function useFeedback() {
  const [state, setState] = useState<{ kind: 'success' | 'error'; message: string } | null>(null);
  return { state, setState } as const;
}

function FeedbackBanner({
  value,
  onDismiss,
}: {
  value: { kind: 'success' | 'error'; message: string } | null;
  onDismiss: () => void;
}) {
  if (!value) return null;
  return (
    <div
      className={`mt-4 flex items-start justify-between gap-2 rounded-lg border px-3 py-2 text-xs ${
        value.kind === 'success'
          ? 'border-herb/30 bg-herb-tint text-herb'
          : 'border-paprika/30 bg-paprika-tint text-paprika'
      }`}
    >
      <span>{value.message}</span>
      <button
        onClick={onDismiss}
        aria-label="Dismiss"
        className="shrink-0 opacity-60 hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb rounded p-0.5"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

// ============================================================
// Password policy
// ============================================================

function PasswordPolicySection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data, isLoading } = useQuery({
    queryKey: ['security-policy'],
    queryFn: () => apiClient.get<SecurityPolicy>('/admin/security/policy'),
  });

  const [draft, setDraft] = useState<Partial<SecurityPolicy>>({});
  const current = { ...data, ...draft } as SecurityPolicy;

  const save = useMutation({
    mutationFn: (body: Partial<SecurityPolicy>) =>
      apiClient.put<SecurityPolicy>('/admin/security/policy', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['security-policy'] });
      setDraft({});
      feedback.setState({ kind: 'success', message: 'Password rules updated' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to save' });
    },
  });

  return (
    <Card
      icon={<Lock className="h-5 w-5 text-primary" />}
      title="Password policy"
      description="Rules enforced at register, change, and reset"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />
      {isLoading || !data ? (
        <p className="py-4 text-sm text-muted-foreground">Loading...</p>
      ) : (
        <div className="mt-4 space-y-3">
          <NumberField
            label="Minimum length"
            value={current.passwordMinLength}
            onChange={(v) => setDraft((d) => ({ ...d, passwordMinLength: v }))}
            min={6}
            max={64}
          />
          <Toggle
            label="Require uppercase letter"
            value={current.passwordRequireUpper}
            onChange={(v) => setDraft((d) => ({ ...d, passwordRequireUpper: v }))}
          />
          <Toggle
            label="Require lowercase letter"
            value={current.passwordRequireLower}
            onChange={(v) => setDraft((d) => ({ ...d, passwordRequireLower: v }))}
          />
          <Toggle
            label="Require number"
            value={current.passwordRequireNumber}
            onChange={(v) => setDraft((d) => ({ ...d, passwordRequireNumber: v }))}
          />
          <Toggle
            label="Require special character"
            value={current.passwordRequireSpecial}
            onChange={(v) => setDraft((d) => ({ ...d, passwordRequireSpecial: v }))}
          />
          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="outline"
              size="sm"
              disabled={Object.keys(draft).length === 0 || save.isPending}
              onClick={() => setDraft({})}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              size="sm"
              isLoading={save.isPending}
              disabled={Object.keys(draft).length === 0 || save.isPending}
              onClick={() => save.mutate(draft)}
            >
              {save.isPending ? 'Saving...' : 'Save'}
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}

// ============================================================
// Sessions
// ============================================================

interface Session {
  id: string;
  userAgent: string;
  ipAddress: string;
  createdAt: string;
  expiresAt: string;
  lastUsedAt: string | null;
}

function SessionPolicySection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data: policy, isLoading: loadingPolicy } = useQuery({
    queryKey: ['security-policy'],
    queryFn: () => apiClient.get<SecurityPolicy>('/admin/security/policy'),
  });
  const { data: sessionsData, isLoading: loadingSessions, refetch } = useQuery({
    queryKey: ['my-sessions'],
    queryFn: () => apiClient.get<{ sessions: Session[] }>('/security/sessions'),
  });

  const [accessTtl, setAccessTtl] = useState<number | null>(null);
  const [refreshTtl, setRefreshTtl] = useState<number | null>(null);

  const savePolicy = useMutation({
    mutationFn: (body: Partial<SecurityPolicy>) =>
      apiClient.put<SecurityPolicy>('/admin/security/policy', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['security-policy'] });
      setAccessTtl(null);
      setRefreshTtl(null);
      feedback.setState({ kind: 'success', message: 'Session policy updated' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to save' });
    },
  });

  const revokeOne = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/security/sessions/${id}`),
    onSuccess: () => {
      refetch();
      feedback.setState({ kind: 'success', message: 'Session revoked' });
    },
  });

  const revokeAll = useMutation({
    mutationFn: () => apiClient.post('/security/sessions/revoke-all'),
    onSuccess: () => {
      refetch();
      feedback.setState({
        kind: 'success',
        message: 'All other sessions revoked. You may need to sign in again.',
      });
    },
  });

  const hasPolicyChanges = accessTtl !== null || refreshTtl !== null;

  return (
    <Card
      icon={<ShieldCheck className="h-5 w-5 text-primary" />}
      title="Session management"
      description="Token lifetimes and your active devices"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />

      {loadingPolicy || !policy ? (
        <p className="py-4 text-sm text-muted-foreground">Loading...</p>
      ) : (
        <div className="mt-4 space-y-3">
          <NumberField
            label="Access token TTL (hours)"
            value={accessTtl ?? policy.sessionAccessTtlHours}
            onChange={setAccessTtl}
            min={1}
            max={168}
          />
          <NumberField
            label="Refresh token TTL (days)"
            value={refreshTtl ?? policy.sessionRefreshTtlDays}
            onChange={setRefreshTtl}
            min={1}
            max={365}
          />
          <div className="flex justify-end gap-2 pt-1">
            <Button
              variant="primary"
              size="sm"
              isLoading={savePolicy.isPending}
              disabled={!hasPolicyChanges || savePolicy.isPending}
              onClick={() =>
                savePolicy.mutate({
                  ...(accessTtl !== null ? { sessionAccessTtlHours: accessTtl } : {}),
                  ...(refreshTtl !== null ? { sessionRefreshTtlDays: refreshTtl } : {}),
                })
              }
            >
              {savePolicy.isPending ? 'Saving...' : 'Save policy'}
            </Button>
          </div>
        </div>
      )}

      <div className="mt-6 border-t border-border pt-4">
        <div className="mb-2 flex items-center justify-between">
          <h4 className="text-sm font-medium text-foreground">Your active sessions</h4>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Refresh sessions"
            title="Refresh"
            onClick={() => refetch()}
            className="text-muted-foreground hover:bg-secondary hover:text-foreground"
          >
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
        </div>
        {loadingSessions ? (
          <p className="py-4 text-sm text-muted-foreground">Loading...</p>
        ) : (sessionsData?.sessions ?? []).length === 0 ? (
          <p className="py-2 text-xs text-muted-foreground">No active sessions.</p>
        ) : (
          <div className="space-y-2">
            {(sessionsData?.sessions ?? []).map((s) => (
              <div
                key={s.id}
                className="flex items-start justify-between gap-3 rounded-lg border border-border bg-secondary/30 px-3 py-2 text-xs"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate text-foreground" title={s.userAgent}>
                    {s.userAgent || 'Unknown device'}
                  </p>
                  <p className="text-muted-foreground">
                    {s.ipAddress || '—'} · issued {new Date(s.createdAt).toLocaleDateString()}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Revoke session"
                  title="Revoke"
                  disabled={revokeOne.isPending}
                  onClick={() => revokeOne.mutate(s.id)}
                  className="shrink-0 text-paprika hover:bg-paprika/10 hover:text-paprika"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}
          </div>
        )}
        <Button
          variant="outline"
          size="sm"
          fullWidth
          isLoading={revokeAll.isPending}
          disabled={revokeAll.isPending}
          onClick={() => revokeAll.mutate()}
          className="mt-3 border-paprika/30 text-paprika hover:bg-paprika-tint hover:border-paprika/40 hover:text-paprika"
        >
          Sign out everywhere
        </Button>
      </div>
    </Card>
  );
}

// ============================================================
// 2FA TOTP
// ============================================================

interface EnrollPayload {
  secret: string;
  otpAuthUrl: string;
  qrCodeBase64: string;
}

function TwoFactorSection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data: policy } = useQuery({
    queryKey: ['security-policy'],
    queryFn: () => apiClient.get<SecurityPolicy>('/admin/security/policy'),
  });
  const { data: me } = useQuery({
    queryKey: ['me'],
    queryFn: () => apiClient.get<{ totpEnabled: boolean }>('/profile'),
  });

  const [enroll, setEnroll] = useState<EnrollPayload | null>(null);
  const [code, setCode] = useState('');
  const [disablePwd, setDisablePwd] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [disabling, setDisabling] = useState(false);

  const startEnroll = useMutation({
    mutationFn: () => apiClient.post<EnrollPayload>('/security/totp/enroll'),
    onSuccess: (payload) => setEnroll(payload),
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Enroll failed' });
    },
  });
  const confirmEnroll = useMutation({
    mutationFn: () => apiClient.post('/security/totp/confirm', { code }),
    onSuccess: () => {
      setEnroll(null);
      setCode('');
      qc.invalidateQueries({ queryKey: ['me'] });
      feedback.setState({ kind: 'success', message: '2FA enabled successfully' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Invalid code' });
    },
  });
  const disable = useMutation({
    mutationFn: () =>
      apiClient.post('/security/totp/disable', { password: disablePwd, code: disableCode }),
    onSuccess: () => {
      setDisabling(false);
      setDisablePwd('');
      setDisableCode('');
      qc.invalidateQueries({ queryKey: ['me'] });
      feedback.setState({ kind: 'success', message: '2FA disabled' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to disable' });
    },
  });
  const saveEnforce = useMutation({
    mutationFn: (v: boolean) =>
      apiClient.put('/admin/security/policy', { twoFactorRequiredForAdmins: v }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['security-policy'] });
      feedback.setState({ kind: 'success', message: 'Enforcement setting updated' });
    },
  });

  return (
    <Card
      icon={<Smartphone className="h-5 w-5 text-primary" />}
      title="Two-factor authentication"
      description="TOTP via Google Authenticator, Authy, 1Password…"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />

      <div className="mt-4 space-y-4">
        <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
          <div>
            <p className="text-foreground">2FA for your account</p>
            <p className="text-xs text-muted-foreground">
              {me?.totpEnabled
                ? 'Enabled — login requires a 6-digit code'
                : 'Disabled — enable to protect your admin account'}
            </p>
          </div>
          {me?.totpEnabled ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setDisabling(true)}
              className="border-paprika/30 text-paprika hover:bg-paprika-tint hover:border-paprika/40 hover:text-paprika"
            >
              Disable
            </Button>
          ) : (
            <Button
              variant="primary"
              size="sm"
              isLoading={startEnroll.isPending}
              disabled={startEnroll.isPending}
              onClick={() => startEnroll.mutate()}
            >
              {startEnroll.isPending ? 'Starting...' : 'Enable'}
            </Button>
          )}
        </div>

        {enroll && (
          <div className="rounded-lg border border-border bg-background p-4">
            <p className="text-sm font-medium text-foreground">Scan with your authenticator</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Or type this key manually: <code className="rounded bg-muted px-1.5 py-0.5">{enroll.secret}</code>
            </p>
            <img
              src={`data:image/png;base64,${enroll.qrCodeBase64}`}
              alt="TOTP QR code"
              loading="lazy"
              decoding="async"
              className="mx-auto my-3 h-48 w-48 rounded border border-border bg-bone p-2"
            />
            <div className="space-y-2">
              <label htmlFor="totp-enroll-code" className="sr-only">6-digit verification code</label>
              <input
                id="totp-enroll-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="6-digit code"
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-center text-lg tracking-widest focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
              />
              <div className="flex gap-2">
                <Button variant="outline" size="sm" fullWidth onClick={() => setEnroll(null)}>
                  Cancel
                </Button>
                <Button
                  variant="primary"
                  size="sm"
                  fullWidth
                  isLoading={confirmEnroll.isPending}
                  disabled={code.length !== 6 || confirmEnroll.isPending}
                  onClick={() => confirmEnroll.mutate()}
                >
                  {confirmEnroll.isPending ? 'Verifying...' : 'Verify & enable'}
                </Button>
              </div>
            </div>
          </div>
        )}

        {disabling && (
          <div className="rounded-lg border border-paprika/30 bg-paprika-tint/50 p-4">
            <p className="text-sm font-medium text-foreground">Disable 2FA</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Requires your password and a current 6-digit code.
            </p>
            <div className="mt-2 space-y-2">
              <label htmlFor="totp-disable-password" className="sr-only">Password</label>
              <input
                id="totp-disable-password"
                type="password"
                autoComplete="current-password"
                value={disablePwd}
                onChange={(e) => setDisablePwd(e.target.value)}
                placeholder="Password"
                className="w-full rounded-lg border border-border bg-bone px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
              />
              <label htmlFor="totp-disable-code" className="sr-only">6-digit verification code</label>
              <input
                id="totp-disable-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="6-digit code"
                className="w-full rounded-lg border border-border bg-bone px-3 py-2 text-center tracking-widest focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
              />
              <div className="flex gap-2">
                <Button variant="outline" size="sm" fullWidth onClick={() => setDisabling(false)}>
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  fullWidth
                  isLoading={disable.isPending}
                  disabled={!disablePwd || disableCode.length !== 6 || disable.isPending}
                  onClick={() => disable.mutate()}
                >
                  {disable.isPending ? 'Disabling...' : 'Disable 2FA'}
                </Button>
              </div>
            </div>
          </div>
        )}

        <div className="rounded-lg border border-border bg-secondary/30 px-4 py-3">
          <Toggle
            label="Require 2FA for all admins"
            value={policy?.twoFactorRequiredForAdmins ?? false}
            onChange={(v) => saveEnforce.mutate(v)}
          />
          <p className="mt-1 text-xs text-muted-foreground">
            When enabled, admins without 2FA are prompted to enroll on their next login.
          </p>
        </div>

        <TwoFactorExemptList />
      </div>
    </Card>
  );
}

// Admin-facing editor for the TwoFactorExemptEmails list on the SecurityPolicy.
// Kept as a small separate component so the main 2FA card stays readable.
function TwoFactorExemptList() {
  const qc = useQueryClient();
  const { data } = useQuery({
    queryKey: ['security-policy'],
    queryFn: () => apiClient.get<SecurityPolicy>('/admin/security/policy'),
  });
  const [draft, setDraft] = useState<string | null>(null);
  const [saveFeedback, setSaveFeedback] = useState<string | null>(null);

  const current = draft ?? (data?.twoFactorExemptEmails ?? []).join('\n');

  const save = useMutation({
    mutationFn: (emails: string[]) =>
      apiClient.put('/admin/security/policy', { twoFactorExemptEmails: emails }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['security-policy'] });
      setDraft(null);
      setSaveFeedback('Exempt list saved');
      setTimeout(() => setSaveFeedback(null), 3000);
    },
  });

  return (
    <div className="rounded-lg border border-border bg-secondary/30 px-4 py-3">
      <label htmlFor="tfa-exempt-emails" className="text-sm text-foreground">2FA exempt emails</label>
      <p id="tfa-exempt-emails-hint" className="mt-1 text-xs text-muted-foreground">
        One per line. Useful for service accounts (E2E tests, automation)
        that can't scan a QR. Leave empty to enforce 2FA on every admin.
      </p>
      <textarea
        id="tfa-exempt-emails"
        aria-describedby="tfa-exempt-emails-hint"
        value={current}
        onChange={(e) => setDraft(e.target.value)}
        rows={3}
        placeholder="service@fe3dr.com"
        className="mt-2 w-full rounded-lg border border-border bg-bone px-3 py-2 font-mono text-xs focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
      />
      {saveFeedback && (
        <p className="mt-1 text-xs text-herb">{saveFeedback}</p>
      )}
      <div className="mt-2 flex justify-end gap-2">
        <Button
          variant="outline"
          size="xs"
          disabled={draft === null || save.isPending}
          onClick={() => setDraft(null)}
        >
          Cancel
        </Button>
        <Button
          variant="primary"
          size="xs"
          isLoading={save.isPending}
          disabled={draft === null || save.isPending}
          onClick={() =>
            save.mutate(
              (draft ?? '')
                .split('\n')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
        >
          {save.isPending ? 'Saving...' : 'Save list'}
        </Button>
      </div>
    </div>
  );
}

// ============================================================
// API keys
// ============================================================

interface ApiKeyRecord {
  id: string;
  name: string;
  prefix: string;
  scopes: string;
  createdAt: string;
  expiresAt: string | null;
  lastUsedAt: string | null;
  revokedAt: string | null;
}

interface CreateKeyResponse {
  key: ApiKeyRecord;
  fullKey: string;
}

function ApiKeysSection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data, isLoading } = useQuery({
    queryKey: ['api-keys'],
    queryFn: () => apiClient.get<{ keys: ApiKeyRecord[] }>('/admin/security/api-keys'),
  });

  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [expiresInDays, setExpiresInDays] = useState(0);
  const [scopes, setScopes] = useState<string[]>(['read']);
  const [revealed, setRevealed] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const create = useMutation({
    mutationFn: () =>
      apiClient.post<CreateKeyResponse>('/admin/security/api-keys', {
        name,
        scopes,
        expiresInDays,
      }),
    onSuccess: (payload) => {
      setRevealed(payload.fullKey);
      setName('');
      setExpiresInDays(0);
      setScopes(['read']);
      setCreating(false);
      qc.invalidateQueries({ queryKey: ['api-keys'] });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to create key' });
    },
  });
  const revoke = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/admin/security/api-keys/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['api-keys'] });
      feedback.setState({ kind: 'success', message: 'Key revoked' });
    },
  });

  const copyFullKey = async () => {
    if (!revealed) return;
    await navigator.clipboard.writeText(revealed);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const toggleScope = (s: string) => {
    setScopes((prev) => (prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]));
  };

  return (
    <Card
      icon={<KeyRound className="h-5 w-5 text-primary" />}
      title="API keys"
      description="Integration credentials for external systems"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />

      {revealed && (
        <div className="mt-4 rounded-lg border border-amber/30 bg-amber-tint p-3">
          <p className="text-xs font-medium text-amber">
            Copy this key now — you won't see it again
          </p>
          <div className="mt-2 flex items-center gap-2">
            <code className="flex-1 truncate rounded bg-bone px-2 py-1 text-xs">{revealed}</code>
            <Button
              variant="primary"
              size="xs"
              aria-label="Copy full key"
              onClick={copyFullKey}
            >
              {copied ? <CheckCircle2 className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            </Button>
          </div>
          <Button
            variant="link"
            size="sm"
            className="mt-2 text-amber"
            onClick={() => setRevealed(null)}
          >
            I've saved it
          </Button>
        </div>
      )}

      {creating ? (
        <div className="mt-4 space-y-3 rounded-lg border border-border bg-secondary/20 p-3">
          <div>
            <label htmlFor="api-key-name" className="block text-xs font-medium text-muted-foreground mb-1">Name</label>
            <input
              id="api-key-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Partner integration"
              className="w-full rounded-lg border border-border bg-bone px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Scopes</label>
            <div className="flex gap-2">
              {['read', 'write', 'admin'].map((s) => (
                <button
                  key={s}
                  type="button"
                  role="checkbox"
                  aria-checked={scopes.includes(s)}
                  aria-label={`Toggle ${s} scope`}
                  onClick={() => toggleScope(s)}
                  className={`rounded-lg border px-3 py-1.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
                    scopes.includes(s)
                      ? 'border-primary bg-primary text-primary-foreground'
                      : 'border-border bg-bone text-foreground hover:bg-secondary'
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
          <NumberField
            label="Expires in (days; 0 = never)"
            value={expiresInDays}
            onChange={setExpiresInDays}
            min={0}
            max={3650}
          />
          <div className="flex gap-2">
            <Button variant="outline" size="sm" fullWidth onClick={() => setCreating(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              size="sm"
              fullWidth
              isLoading={create.isPending}
              disabled={!name.trim() || scopes.length === 0 || create.isPending}
              onClick={() => create.mutate()}
            >
              {create.isPending ? 'Creating...' : 'Create key'}
            </Button>
          </div>
        </div>
      ) : (
        <Button
          variant="outline"
          size="sm"
          leftIcon={<Plus className="h-4 w-4" />}
          onClick={() => setCreating(true)}
          className="mt-4 border-dashed text-muted-foreground hover:border-primary hover:text-primary hover:bg-transparent"
        >
          New API key
        </Button>
      )}

      <div className="mt-4 space-y-2">
        {isLoading ? (
          <p className="py-2 text-sm text-muted-foreground">Loading...</p>
        ) : (data?.keys ?? []).length === 0 ? (
          <p className="py-2 text-xs text-muted-foreground">No keys yet.</p>
        ) : (
          (data?.keys ?? []).map((k) => (
            <div
              key={k.id}
              className={`flex items-start justify-between gap-3 rounded-lg border border-border bg-secondary/30 px-3 py-2 text-xs ${
                k.revokedAt ? 'opacity-50' : ''
              }`}
            >
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-foreground">{k.name}</p>
                <p className="text-muted-foreground">
                  <code>{k.prefix}…</code> · {k.scopes || 'no scopes'}
                  {k.expiresAt && ` · expires ${new Date(k.expiresAt).toLocaleDateString()}`}
                  {k.revokedAt && ' · revoked'}
                </p>
              </div>
              {!k.revokedAt && (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Revoke API key"
                  title="Revoke"
                  onClick={() => revoke.mutate(k.id)}
                  className="shrink-0 text-paprika hover:bg-paprika/10 hover:text-paprika"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              )}
            </div>
          ))
        )}
      </div>
    </Card>
  );
}

// ============================================================
// Primitives
// ============================================================

function Card({
  icon,
  title,
  description,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          {icon}
        </div>
        <div>
          <h3 className="font-semibold text-foreground">{title}</h3>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
      </div>
      {children}
    </div>
  );
}

function NumberField({
  label,
  value,
  onChange,
  min,
  max,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  min: number;
  max: number;
}) {
  const id = useId();
  return (
    <div>
      <label htmlFor={id} className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
      <input
        id={id}
        type="number"
        value={value}
        min={min}
        max={max}
        onChange={(e) => onChange(Math.max(min, Math.min(max, Number(e.target.value) || min)))}
        className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
      />
    </div>
  );
}

function Toggle({
  label,
  value,
  onChange,
}: {
  label: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-center justify-between gap-2 py-1 text-sm">
      <span className="text-foreground">{label}</span>
      <button
        type="button"
        role="switch"
        aria-checked={value}
        aria-label={label}
        onClick={() => onChange(!value)}
        className={`relative h-5 w-9 rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
          value ? 'bg-primary' : 'bg-border'
        }`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-bone shadow-sm transition-transform ${
            value ? 'translate-x-4' : 'translate-x-0.5'
          }`}
        />
      </button>
    </label>
  );
}
