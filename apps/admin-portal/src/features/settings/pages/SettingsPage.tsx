import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { Settings, Shield, Bell, Globe, Database, CreditCard, RefreshCw, Copy, CheckCircle2, Pencil, Save, X } from 'lucide-react';

interface PaymentGatewayStatus {
  configured: boolean;
  mode: string;
  webhookUrl: string;
  webhookSecretSet: boolean;
  keyPrefix: string;
  error: string;
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
        <SettingsCard
          icon={Shield}
          title="Security"
          description="Authentication, passwords, and access control"
          items={['Two-factor authentication', 'Password policies', 'Session management', 'API keys']}
        />
        <SettingsCard
          icon={Bell}
          title="Notifications"
          description="Email, push, and in-app notification preferences"
          items={['Order alerts', 'Chef verification alerts', 'Revenue reports', 'System notifications']}
        />
        <SettingsCard
          icon={Globe}
          title="Platform"
          description="General platform configuration"
          items={['Service areas', 'Commission rates', 'Delivery fees', 'Operating hours']}
        />
        <SettingsCard
          icon={Database}
          title="Data & Exports"
          description="Data management and report exports"
          items={['Export user data', 'Order reports', 'Revenue reports', 'Audit logs']}
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

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['payment-gateway-status'],
    queryFn: () => apiClient.get<PaymentGatewayStatus>('/admin/payment-gateway/status'),
  });

  const saveMutation = useMutation({
    mutationFn: (keys: typeof keyForm) => apiClient.put('/admin/payment-gateway/keys', keys),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['payment-gateway-status'] });
      setEditing(false);
      setKeyForm({ keyId: '', keySecret: '', webhookSecret: '' });
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
                    ? 'bg-green-100 text-green-700'
                    : 'bg-red-100 text-red-700'
                }`}
              >
                {data.configured ? 'Connected' : 'Disconnected'}
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">Razorpay integration status</p>
        </div>
        {!editing && (
          <button
            onClick={() => setEditing(true)}
            className="rounded-lg p-2 text-muted-foreground hover:bg-secondary hover:text-foreground"
            title="Edit keys"
          >
            <Pencil className="h-4 w-4" />
          </button>
        )}
      </div>

      <div className="mt-4 space-y-2">
        {editing ? (
          <div className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-1">Razorpay Key ID</label>
              <input
                type="text"
                value={keyForm.keyId}
                onChange={(e) => setKeyForm({ ...keyForm, keyId: e.target.value })}
                placeholder="rzp_test_..."
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-1">Razorpay Key Secret</label>
              <input
                type="password"
                value={keyForm.keySecret}
                onChange={(e) => setKeyForm({ ...keyForm, keySecret: e.target.value })}
                placeholder="Enter key secret"
                className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-1">Webhook Secret</label>
              <input
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
              <button
                onClick={() => saveMutation.mutate(keyForm)}
                disabled={saveMutation.isPending || (!keyForm.keyId && !keyForm.keySecret && !keyForm.webhookSecret)}
                className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                <Save className="h-3.5 w-3.5" />
                {saveMutation.isPending ? 'Saving...' : 'Save Keys'}
              </button>
              <button
                onClick={() => { setEditing(false); setKeyForm({ keyId: '', keySecret: '', webhookSecret: '' }); }}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-secondary"
              >
                <X className="h-3.5 w-3.5" />
                Cancel
              </button>
            </div>
          </div>
        ) : isLoading ? (
          <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">
            Loading...
          </div>
        ) : isError ? (
          <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
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
                    ? 'bg-green-100 text-green-700'
                    : 'bg-yellow-100 text-yellow-700'
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
            <div className="flex items-center justify-between gap-2 rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="shrink-0 text-foreground">Webhook URL</span>
              <div className="flex items-center gap-1">
                <code className="truncate rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground max-w-[180px]">
                  {data.webhookUrl}
                </code>
                <button
                  onClick={copyWebhookUrl}
                  className="shrink-0 rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
                  title="Copy webhook URL"
                >
                  {copied ? (
                    <CheckCircle2 className="h-3.5 w-3.5 text-green-600" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </button>
              </div>
            </div>

            {/* Webhook Secret */}
            <div className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm">
              <span className="text-foreground">Webhook Secret</span>
              {data.webhookSecretSet ? (
                <span className="flex items-center gap-1 text-xs text-green-600">
                  <CheckCircle2 className="h-3.5 w-3.5" /> Configured
                </span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-red-600">
                  ✕ Not configured
                </span>
              )}
            </div>

            {/* Error */}
            {data.error && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                {data.error}
              </div>
            )}

            {/* Test Connection */}
            <button
              onClick={() => refetch()}
              disabled={isFetching}
              className="mt-2 flex w-full items-center justify-center gap-2 rounded-lg border border-border bg-secondary/50 px-4 py-2.5 text-sm font-medium text-foreground hover:bg-secondary disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
              {isFetching ? 'Testing...' : 'Test Connection'}
            </button>

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

function SettingsCard({
  icon: Icon,
  title,
  description,
  items,
}: {
  icon: typeof Settings;
  title: string;
  description: string;
  items: string[];
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <Icon className="h-5 w-5 text-primary" />
        </div>
        <div>
          <h3 className="font-semibold text-foreground">{title}</h3>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
      </div>

      <div className="mt-4 space-y-2">
        {items.map((item) => (
          <div
            key={item}
            className="flex items-center justify-between rounded-lg border border-border bg-secondary/30 px-4 py-3 text-sm"
          >
            <span className="text-foreground">{item}</span>
            <span className="text-xs text-muted-foreground">Configure</span>
          </div>
        ))}
      </div>
    </div>
  );
}
