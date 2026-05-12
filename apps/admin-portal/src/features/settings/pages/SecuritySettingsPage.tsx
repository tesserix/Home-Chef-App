import { useState } from 'react';
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
            <button
              onClick={() => setDraft({})}
              disabled={Object.keys(draft).length === 0 || save.isPending}
              className="rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              onClick={() => save.mutate(draft)}
              disabled={Object.keys(draft).length === 0 || save.isPending}
              className="rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {save.isPending ? 'Saving...' : 'Save'}
            </button>
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
            <button
              onClick={() =>
                savePolicy.mutate({
                  ...(accessTtl !== null ? { sessionAccessTtlHours: accessTtl } : {}),
                  ...(refreshTtl !== null ? { sessionRefreshTtlDays: refreshTtl } : {}),
                })
              }
              disabled={!hasPolicyChanges || savePolicy.isPending}
              className="rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {savePolicy.isPending ? 'Saving...' : 'Save policy'}
            </button>
          </div>
        </div>
      )}

      <div className="mt-6 border-t border-border pt-4">
        <div className="mb-2 flex items-center justify-between">
          <h4 className="text-sm font-medium text-foreground">Your active sessions</h4>
          <button
            onClick={() => refetch()}
            className="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-foreground"
            title="Refresh"
          >
            <RefreshCw className="h-3.5 w-3.5" />
          </button>
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
                <button
                  onClick={() => revokeOne.mutate(s.id)}
                  disabled={revokeOne.isPending}
                  className="shrink-0 text-paprika hover:text-paprika"
                  title="Revoke"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>
        )}
        <button
          onClick={() => revokeAll.mutate()}
          disabled={revokeAll.isPending}
          className="mt-3 w-full rounded-lg border border-paprika/30 px-3 py-1.5 text-xs font-medium text-paprika hover:bg-paprika-tint disabled:opacity-50"
        >
          Sign out everywhere
        </button>
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
            <button
              onClick={() => setDisabling(true)}
              className="rounded-lg border border-paprika/30 px-3 py-1.5 text-xs font-medium text-paprika hover:bg-paprika-tint"
            >
              Disable
            </button>
          ) : (
            <button
              onClick={() => startEnroll.mutate()}
              disabled={startEnroll.isPending}
              className="rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {startEnroll.isPending ? 'Starting...' : 'Enable'}
            </button>
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
              className="mx-auto my-3 h-48 w-48 rounded border border-border bg-bone p-2"
            />
            <div className="space-y-2">
              <input
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="6-digit code"
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-center text-lg tracking-widest focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => setEnroll(null)}
                  className="flex-1 rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary"
                >
                  Cancel
                </button>
                <button
                  onClick={() => confirmEnroll.mutate()}
                  disabled={code.length !== 6 || confirmEnroll.isPending}
                  className="flex-1 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                >
                  {confirmEnroll.isPending ? 'Verifying...' : 'Verify & enable'}
                </button>
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
              <input
                type="password"
                value={disablePwd}
                onChange={(e) => setDisablePwd(e.target.value)}
                placeholder="Password"
                className="w-full rounded-lg border border-border bg-bone px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
              />
              <input
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="6-digit code"
                className="w-full rounded-lg border border-border bg-bone px-3 py-2 text-center tracking-widest focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => setDisabling(false)}
                  className="flex-1 rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary"
                >
                  Cancel
                </button>
                <button
                  onClick={() => disable.mutate()}
                  disabled={!disablePwd || disableCode.length !== 6 || disable.isPending}
                  className="flex-1 rounded-lg bg-paprika px-3 py-1.5 text-sm font-medium text-paper hover:bg-paprika disabled:opacity-50"
                >
                  {disable.isPending ? 'Disabling...' : 'Disable 2FA'}
                </button>
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
      <p className="text-sm text-foreground">2FA exempt emails</p>
      <p className="mt-1 text-xs text-muted-foreground">
        One per line. Useful for service accounts (E2E tests, automation)
        that can't scan a QR. Leave empty to enforce 2FA on every admin.
      </p>
      <textarea
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
        <button
          onClick={() => setDraft(null)}
          disabled={draft === null || save.isPending}
          className="rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          onClick={() =>
            save.mutate(
              (draft ?? '')
                .split('\n')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
          disabled={draft === null || save.isPending}
          className="rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {save.isPending ? 'Saving...' : 'Save list'}
        </button>
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
            <button
              onClick={copyFullKey}
              className="rounded bg-primary px-2 py-1 text-xs text-primary-foreground hover:bg-primary/90"
            >
              {copied ? <CheckCircle2 className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            </button>
          </div>
          <button
            onClick={() => setRevealed(null)}
            className="mt-2 text-xs text-amber hover:underline"
          >
            I've saved it
          </button>
        </div>
      )}

      {creating ? (
        <div className="mt-4 space-y-3 rounded-lg border border-border bg-secondary/20 p-3">
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Name</label>
            <input
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
                  onClick={() => toggleScope(s)}
                  className={`rounded-lg border px-3 py-1.5 text-xs ${
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
            <button
              onClick={() => setCreating(false)}
              className="flex-1 rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary"
            >
              Cancel
            </button>
            <button
              onClick={() => create.mutate()}
              disabled={!name.trim() || scopes.length === 0 || create.isPending}
              className="flex-1 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {create.isPending ? 'Creating...' : 'Create key'}
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setCreating(true)}
          className="mt-4 inline-flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:border-primary hover:text-primary"
        >
          <Plus className="h-4 w-4" /> New API key
        </button>
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
                <button
                  onClick={() => revoke.mutate(k.id)}
                  className="shrink-0 text-paprika hover:text-paprika"
                  title="Revoke"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
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
  return (
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
      <input
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
        onClick={() => onChange(!value)}
        className={`relative h-5 w-9 rounded-full transition-colors ${
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
