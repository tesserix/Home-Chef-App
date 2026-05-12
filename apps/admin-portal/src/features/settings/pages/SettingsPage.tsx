import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { ApiError } from '@/shared/types';
import { Settings, Shield, Bell, Globe, Database, CreditCard, RefreshCw, Copy, CheckCircle2, Pencil, Save, X, ChevronRight } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';

interface PaymentGatewayStatus {
  configured: boolean;
  mode: string;
  webhookUrl: string;
  webhookSecretSet: boolean;
  keyPrefix: string;
  error: string;
}

interface StripeGatewayStatus extends PaymentGatewayStatus {
  publishableKeySet: boolean;
}

interface UpdateKeysResponse {
  message: string;
  verified?: boolean;
  testError?: string;
}

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Settings</h1>
        <p className="page-description">Configure platform settings and preferences</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <PaymentGatewayCard />
        <StripeGatewayCard />
        <SettingsLinkCard
          icon={Shield}
          title="Security"
          description="Authentication, passwords, and access control"
          to="/settings/security"
          items={[
            { label: 'Two-factor authentication', cta: 'Manage' },
            { label: 'Password policies', cta: 'Manage' },
            { label: 'Session management', cta: 'Manage' },
            { label: 'API keys', cta: 'Manage' },
          ]}
        />
        <SettingsLinkCard
          icon={Bell}
          title="Notifications"
          description="Email, push, and in-app notification preferences"
          to="/settings/notifications"
          items={[
            { label: 'Order alerts', cta: 'Manage' },
            { label: 'Chef verification alerts', cta: 'Manage' },
            { label: 'Delivery updates', cta: 'Manage' },
            { label: 'Marketing & promos', cta: 'Manage' },
          ]}
        />
        <SettingsLinkCard
          icon={Globe}
          title="Platform"
          description="Commission, delivery fees, service areas, hours"
          to="/settings/platform"
          items={[
            { label: 'Commission rates', cta: 'Manage' },
            { label: 'Delivery fees', cta: 'Manage' },
            { label: 'Operating hours', cta: 'Manage' },
            { label: 'Service areas', cta: 'Manage' },
          ]}
        />
        <SettingsLinkCard
          icon={Database}
          title="Data & Exports"
          description="CSV downloads and audit history"
          to="/settings/data-exports"
          items={[
            { label: 'Users CSV', cta: 'Download' },
            { label: 'Orders CSV', cta: 'Download' },
            { label: 'Revenue CSV', cta: 'Download' },
            { label: 'Audit logs', cta: 'View' },
          ]}
        />
      </div>
    </div>
  );
}

function PaymentGatewayCard() {
  const queryClient = useQueryClient();
  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [keyForm, setKeyForm] = useState({ keyId: '', keySecret: '', webhookSecret: '' });
  const [saveFeedback, setSaveFeedback] = useState<
    { kind: 'success' | 'error'; message: string } | null
  >(null);

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['payment-gateway-status'],
    queryFn: () => apiClient.get<PaymentGatewayStatus>('/admin/payment-gateway/status'),
  });

  const saveMutation = useMutation({
    mutationFn: (keys: typeof keyForm) =>
      apiClient.put<UpdateKeysResponse>('/admin/payment-gateway/keys', keys),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['payment-gateway-status'] });
      setKeyForm({ keyId: '', keySecret: '', webhookSecret: '' });
      if (result.verified === false && result.testError) {
        // Saved to Secret Manager but Razorpay rejected them — keep the form
        // open so the admin can correct the keys without re-typing context.
        setSaveFeedback({
          kind: 'error',
          message: `Saved, but Razorpay rejected the keys: ${result.testError}`,
        });
        return;
      }
      setSaveFeedback({
        kind: 'success',
        message: result.message ?? 'Payment gateway keys saved and verified',
      });
      setEditing(false);
    },
    onError: (err) => {
      const apiErr = err as Partial<ApiError>;
      const message =
        apiErr?.error?.message ??
        (err instanceof Error ? err.message : 'Failed to save keys');
      setSaveFeedback({ kind: 'error', message });
    },
  });

  const copyWebhookUrl = async () => {
    if (!data?.webhookUrl) return;
    await navigator.clipboard.writeText(data.webhookUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <CreditCard className="h-5 w-5 text-primary" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-foreground">Payment Gateway</h3>
            {data && (
              <span
                className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                  data.configured
                    ? 'bg-herb-tint text-herb'
                    : 'bg-paprika-tint text-paprika'
                }`}
              >
                {data.configured ? 'Connected' : 'Disconnected'}
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">Razorpay integration status</p>
        </div>
        {!editing && (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Edit keys"
            title="Edit keys"
            onClick={() => {
              setSaveFeedback(null);
              setEditing(true);
            }}
            className="text-muted-foreground hover:bg-secondary hover:text-foreground"
          >
            <Pencil className="h-4 w-4" />
          </Button>
        )}
      </div>

      {saveFeedback && (
        <div
          className={`mt-4 flex items-start justify-between gap-2 rounded-lg border px-3 py-2 text-xs ${
            saveFeedback.kind === 'success'
              ? 'border-herb/30 bg-herb-tint text-herb'
              : 'border-paprika/30 bg-paprika-tint text-paprika'
          }`}
        >
          <span>{saveFeedback.message}</span>
          <button
            type="button"
            aria-label="Dismiss"
            title="Dismiss"
            onClick={() => setSaveFeedback(null)}
            className="shrink-0 rounded opacity-60 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      <div className="mt-4 space-y-2">
        {editing ? (
          <div className="space-y-3">
            <div>
              <label htmlFor="razorpay-key-id" className="block text-xs font-medium text-muted-foreground mb-1">Razorpay Key ID</label>
              <input
                id="razorpay-key-id"
                type="text"
                value={keyForm.keyId}
                onChange={(e) => setKeyForm({ ...keyForm, keyId: e.target.value })}
                placeholder="rzp_test_..."
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label htmlFor="razorpay-key-secret" className="block text-xs font-medium text-muted-foreground mb-1">Razorpay Key Secret</label>
              <input
                id="razorpay-key-secret"
                type="password"
                value={keyForm.keySecret}
                onChange={(e) => setKeyForm({ ...keyForm, keySecret: e.target.value })}
                placeholder="Enter key secret"
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label htmlFor="razorpay-webhook-secret" className="block text-xs font-medium text-muted-foreground mb-1">Webhook Secret</label>
              <input
                id="razorpay-webhook-secret"
                type="password"
                value={keyForm.webhookSecret}
                onChange={(e) => setKeyForm({ ...keyForm, webhookSecret: e.target.value })}
                placeholder="Enter webhook secret"
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Keys are stored securely in GCP Secret Manager. Leave a field empty to keep its current value.
            </p>
            <div className="flex gap-2">
              <Button
                variant="primary"
                isLoading={saveMutation.isPending}
                disabled={saveMutation.isPending || (!keyForm.keyId && !keyForm.keySecret && !keyForm.webhookSecret)}
                leftIcon={!saveMutation.isPending ? <Save className="h-3.5 w-3.5" /> : undefined}
                onClick={() => {
                  setSaveFeedback(null);
                  saveMutation.mutate(keyForm);
                }}
              >
                {saveMutation.isPending ? 'Saving...' : 'Save Keys'}
              </Button>
              <Button
                variant="outline"
                leftIcon={<X className="h-3.5 w-3.5" />}
                onClick={() => {
                  setEditing(false);
                  setKeyForm({ keyId: '', keySecret: '', webhookSecret: '' });
                  setSaveFeedback(null);
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : isLoading ? (
          <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">
            Loading...
          </div>
        ) : isError ? (
          <div className="rounded-lg border border-paprika/30 bg-paprika-tint px-4 py-3 text-sm text-paprika">
            Failed to fetch payment gateway status
          </div>
        ) : data ? (
          <>
            {/* Mode */}
            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Mode</span>
              <span
                className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                  data.mode === 'live'
                    ? 'bg-herb-tint text-herb'
                    : 'bg-amber-tint text-amber'
                }`}
              >
                {data.mode === 'live' ? 'Live Mode' : 'Test Mode'}
              </span>
            </div>

            {/* Key Prefix */}
            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Key Prefix</span>
              <code className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                {data.keyPrefix || '—'}
              </code>
            </div>

            {/* Webhook URL */}
            <div className="flex items-center justify-between gap-3 rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="shrink-0 text-foreground">Webhook URL</span>
              <div className="flex min-w-0 items-center gap-1">
                <code className="truncate rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                  {data.webhookUrl}
                </code>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Copy webhook URL"
                  title="Copy webhook URL"
                  onClick={copyWebhookUrl}
                  className="shrink-0 text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {copied ? (
                    <CheckCircle2 className="h-3.5 w-3.5 text-herb" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </Button>
              </div>
            </div>

            {/* Webhook Secret */}
            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Webhook Secret</span>
              {data.webhookSecretSet ? (
                <span className="flex items-center gap-1 text-xs text-herb">
                  <CheckCircle2 className="h-3.5 w-3.5" /> Configured
                </span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-paprika">
                  ✕ Not configured
                </span>
              )}
            </div>

            {/* Error */}
            {data.error && (
              <div className="rounded-lg border border-paprika/30 bg-paprika-tint px-4 py-3 text-sm text-paprika">
                {data.error}
              </div>
            )}

            {/* Test Connection */}
            <Button
              variant="outline"
              fullWidth
              isLoading={isFetching}
              disabled={isFetching}
              leftIcon={!isFetching ? <RefreshCw className="h-4 w-4" /> : undefined}
              className="mt-2"
              onClick={() => refetch()}
            >
              {isFetching ? 'Testing...' : 'Test Connection'}
            </Button>

            {/* Test Cards Reference */}
            {data.mode === 'test' && (
              <div className="mt-3 rounded-lg border border-dashed border-border px-4 py-3">
                <p className="mb-2 text-xs font-medium text-muted-foreground">Test Cards</p>
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">Visa</span>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">4111 1111 1111 1111</code>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">Mastercard</span>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">5267 3181 8797 5449</code>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">UPI</span>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">success@razorpay</code>
                  </div>
                </div>
              </div>
            )}
          </>
        ) : null}
      </div>
    </div>
  );
}

function StripeGatewayCard() {
  const queryClient = useQueryClient();
  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [keyForm, setKeyForm] = useState({ secretKey: '', publishableKey: '', webhookSecret: '' });
  const [saveFeedback, setSaveFeedback] = useState<
    { kind: 'success' | 'error'; message: string } | null
  >(null);

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['stripe-gateway-status'],
    queryFn: () => apiClient.get<StripeGatewayStatus>('/admin/payment-gateway/stripe/status'),
  });

  const saveMutation = useMutation({
    mutationFn: (keys: typeof keyForm) =>
      apiClient.put<UpdateKeysResponse>('/admin/payment-gateway/stripe/keys', keys),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['stripe-gateway-status'] });
      setKeyForm({ secretKey: '', publishableKey: '', webhookSecret: '' });
      if (result.verified === false && result.testError) {
        setSaveFeedback({
          kind: 'error',
          message: `Saved, but Stripe rejected the keys: ${result.testError}`,
        });
        return;
      }
      setSaveFeedback({
        kind: 'success',
        message: result.message ?? 'Stripe keys saved and verified',
      });
      setEditing(false);
    },
    onError: (err) => {
      const apiErr = err as Partial<ApiError>;
      const message =
        apiErr?.error?.message ??
        (err instanceof Error ? err.message : 'Failed to save keys');
      setSaveFeedback({ kind: 'error', message });
    },
  });

  const copyWebhookUrl = async () => {
    if (!data?.webhookUrl) return;
    await navigator.clipboard.writeText(data.webhookUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <CreditCard className="h-5 w-5 text-primary" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-foreground">Stripe Gateway</h3>
            {data && (
              <span
                className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                  data.configured ? 'bg-herb-tint text-herb' : 'bg-paprika-tint text-paprika'
                }`}
              >
                {data.configured ? 'Connected' : 'Disconnected'}
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">International payouts (markets where Razorpay is unavailable)</p>
        </div>
        {!editing && (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Edit keys"
            title="Edit keys"
            onClick={() => {
              setSaveFeedback(null);
              setEditing(true);
            }}
            className="text-muted-foreground hover:bg-secondary hover:text-foreground"
          >
            <Pencil className="h-4 w-4" />
          </Button>
        )}
      </div>

      {saveFeedback && (
        <div
          className={`mt-4 flex items-start justify-between gap-2 rounded-lg border px-3 py-2 text-xs ${
            saveFeedback.kind === 'success'
              ? 'border-herb/30 bg-herb-tint text-herb'
              : 'border-paprika/30 bg-paprika-tint text-paprika'
          }`}
        >
          <span>{saveFeedback.message}</span>
          <button
            type="button"
            aria-label="Dismiss"
            title="Dismiss"
            onClick={() => setSaveFeedback(null)}
            className="shrink-0 rounded opacity-60 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      <div className="mt-4 space-y-2">
        {editing ? (
          <div className="space-y-3">
            <div>
              <label htmlFor="stripe-secret-key" className="block text-xs font-medium text-muted-foreground mb-1">Stripe Secret Key</label>
              <input
                id="stripe-secret-key"
                type="password"
                value={keyForm.secretKey}
                onChange={(e) => setKeyForm({ ...keyForm, secretKey: e.target.value })}
                placeholder="sk_test_... or sk_live_..."
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label htmlFor="stripe-publishable-key" className="block text-xs font-medium text-muted-foreground mb-1">Stripe Publishable Key</label>
              <input
                id="stripe-publishable-key"
                type="text"
                value={keyForm.publishableKey}
                onChange={(e) => setKeyForm({ ...keyForm, publishableKey: e.target.value })}
                placeholder="pk_test_... or pk_live_..."
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label htmlFor="stripe-webhook-secret" className="block text-xs font-medium text-muted-foreground mb-1">Webhook Signing Secret</label>
              <input
                id="stripe-webhook-secret"
                type="password"
                value={keyForm.webhookSecret}
                onChange={(e) => setKeyForm({ ...keyForm, webhookSecret: e.target.value })}
                placeholder="whsec_..."
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Keys are stored in GCP Secret Manager. Leave a field empty to keep its current value.
            </p>
            <div className="flex gap-2">
              <Button
                variant="primary"
                isLoading={saveMutation.isPending}
                disabled={saveMutation.isPending || (!keyForm.secretKey && !keyForm.publishableKey && !keyForm.webhookSecret)}
                leftIcon={!saveMutation.isPending ? <Save className="h-3.5 w-3.5" /> : undefined}
                onClick={() => {
                  setSaveFeedback(null);
                  saveMutation.mutate(keyForm);
                }}
              >
                {saveMutation.isPending ? 'Saving...' : 'Save Keys'}
              </Button>
              <Button
                variant="outline"
                leftIcon={<X className="h-3.5 w-3.5" />}
                onClick={() => {
                  setEditing(false);
                  setKeyForm({ secretKey: '', publishableKey: '', webhookSecret: '' });
                  setSaveFeedback(null);
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : isLoading ? (
          <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">Loading...</div>
        ) : isError ? (
          <div className="rounded-lg border border-paprika/30 bg-paprika-tint px-4 py-3 text-sm text-paprika">
            Failed to fetch Stripe gateway status
          </div>
        ) : data ? (
          <>
            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Mode</span>
              <span
                className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                  data.mode === 'live'
                    ? 'bg-herb-tint text-herb'
                    : 'bg-amber-tint text-amber'
                }`}
              >
                {data.mode === 'live' ? 'Live Mode' : 'Test Mode'}
              </span>
            </div>

            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Secret Key Prefix</span>
              <code className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                {data.keyPrefix || '—'}
              </code>
            </div>

            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Publishable Key</span>
              {data.publishableKeySet ? (
                <span className="flex items-center gap-1 text-xs text-herb">
                  <CheckCircle2 className="h-3.5 w-3.5" /> Configured
                </span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-paprika">✕ Not configured</span>
              )}
            </div>

            <div className="flex items-center justify-between gap-3 rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="shrink-0 text-foreground">Webhook URL</span>
              <div className="flex min-w-0 items-center gap-1">
                <code className="truncate rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                  {data.webhookUrl}
                </code>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Copy webhook URL"
                  title="Copy webhook URL"
                  onClick={copyWebhookUrl}
                  className="shrink-0 text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {copied ? (
                    <CheckCircle2 className="h-3.5 w-3.5 text-herb" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </Button>
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Webhook Secret</span>
              {data.webhookSecretSet ? (
                <span className="flex items-center gap-1 text-xs text-herb">
                  <CheckCircle2 className="h-3.5 w-3.5" /> Configured
                </span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-paprika">✕ Not configured</span>
              )}
            </div>

            {data.error && (
              <div className="rounded-lg border border-paprika/30 bg-paprika-tint px-4 py-3 text-sm text-paprika">
                {data.error}
              </div>
            )}

            <Button
              variant="outline"
              fullWidth
              isLoading={isFetching}
              disabled={isFetching}
              leftIcon={!isFetching ? <RefreshCw className="h-4 w-4" /> : undefined}
              className="mt-2"
              onClick={() => refetch()}
            >
              {isFetching ? 'Testing...' : 'Test Connection'}
            </Button>

            {data.mode === 'test' && (
              <div className="mt-3 rounded-lg border border-dashed border-border px-4 py-3">
                <p className="mb-2 text-xs font-medium text-muted-foreground">Test Cards</p>
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">Visa</span>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">4242 4242 4242 4242</code>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">Mastercard</span>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">5555 5555 5555 4444</code>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">3D Secure</span>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-muted-foreground">4000 0027 6000 3184</code>
                  </div>
                </div>
              </div>
            )}
          </>
        ) : null}
      </div>
    </div>
  );
}

function SettingsLinkCard({
  icon: Icon,
  title,
  description,
  items,
  to,
}: {
  icon: typeof Settings;
  title: string;
  description: string;
  items: { label: string; cta: string }[];
  to: string;
}) {
  return (
    <Link
      to={to}
      className="group block rounded-xl border border-border bg-card p-6 shadow-card transition-colors hover:border-primary/40"
    >
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <Icon className="h-5 w-5 text-primary" />
        </div>
        <div className="flex-1">
          <h3 className="font-semibold text-foreground">{title}</h3>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        <ChevronRight className="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
      </div>
      <div className="mt-4 space-y-2">
        {items.map((item) => (
          <div
            key={item.label}
            className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm"
          >
            <span className="text-foreground">{item.label}</span>
            <span className="text-xs text-primary">{item.cta}</span>
          </div>
        ))}
      </div>
    </Link>
  );
}

