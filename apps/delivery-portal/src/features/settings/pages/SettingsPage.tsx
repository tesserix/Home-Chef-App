import { useEffect, useState } from 'react';
import { useAuth } from '@/app/providers/AuthProvider';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { LogOut, Bell, Shield, HelpCircle, Globe, ExternalLink, CheckCircle2, XCircle } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';

export default function SettingsPage() {
  const { logout } = useAuth();

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Settings</h1>
      </div>

      <StripeConnectCard />

      <div className="space-y-2">
        <button type="button" className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-secondary">
          <Bell className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
          <div>
            <p className="text-sm font-medium text-foreground">Notifications</p>
            <p className="text-xs text-muted-foreground">Manage push & email notifications</p>
          </div>
        </button>

        <button type="button" className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-secondary">
          <Shield className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
          <div>
            <p className="text-sm font-medium text-foreground">Privacy & Security</p>
            <p className="text-xs text-muted-foreground">Manage your account security</p>
          </div>
        </button>

        <button type="button" className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-secondary">
          <HelpCircle className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
          <div>
            <p className="text-sm font-medium text-foreground">Help & Support</p>
            <p className="text-xs text-muted-foreground">Get help with your deliveries</p>
          </div>
        </button>

        <button type="button"
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-xl border border-destructive/30 bg-card p-4 text-left transition-colors hover:bg-destructive/5"
        >
          <LogOut className="h-5 w-5 text-destructive" aria-hidden="true" />
          <div>
            <p className="text-sm font-medium text-destructive">Sign Out</p>
            <p className="text-xs text-muted-foreground">Sign out of your account</p>
          </div>
        </button>
      </div>
    </div>
  );
}

interface StripeStatus {
  connected: boolean;
  accountId: string;
  chargesEnabled: boolean;
  payoutsEnabled: boolean;
  detailsSubmitted: boolean;
  country: string;
  paymentProvider?: 'razorpay' | 'stripe';
  warning?: string;
}

// Connect-supported countries trimmed to the set this platform actually
// onboards drivers in. Same list as the vendor portal so chefs/drivers
// can pair across jurisdictions.
const SUPPORTED_COUNTRIES = [
  { code: 'US', name: 'United States' },
  { code: 'GB', name: 'United Kingdom' },
  { code: 'CA', name: 'Canada' },
  { code: 'AU', name: 'Australia' },
  { code: 'NZ', name: 'New Zealand' },
  { code: 'SG', name: 'Singapore' },
  { code: 'HK', name: 'Hong Kong' },
  { code: 'AE', name: 'United Arab Emirates' },
  { code: 'DE', name: 'Germany' },
  { code: 'FR', name: 'France' },
  { code: 'IT', name: 'Italy' },
  { code: 'ES', name: 'Spain' },
  { code: 'NL', name: 'Netherlands' },
  { code: 'IE', name: 'Ireland' },
  { code: 'IN', name: 'India' },
];

function StripeConnectCard() {
  const queryClient = useQueryClient();
  const [country, setCountry] = useState('US');

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['driver-stripe-status'],
    queryFn: () => apiClient.get<StripeStatus>('/delivery/stripe/status'),
    refetchInterval: (q) => {
      const d = q.state.data as StripeStatus | undefined;
      if (d?.connected && !d.chargesEnabled) return 60_000;
      return false;
    },
  });

  // Same return-from-Stripe handler as the vendor portal's card — clears
  // the ?done=1 / ?refresh=1 query string and refetches capability state.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get('done') === '1' || params.get('refresh') === '1') {
      refetch();
      params.delete('done');
      params.delete('refresh');
      const next = params.toString();
      const url = window.location.pathname + (next ? '?' + next : '');
      window.history.replaceState(null, '', url);
    }
  }, [refetch]);

  const connect = useMutation({
    mutationFn: (c: string) =>
      apiClient.post<{ accountId: string; onboardingUrl: string }>(
        '/delivery/stripe/connect',
        { country: c }
      ),
    onSuccess: (r) => {
      window.location.href = r.onboardingUrl;
    },
  });

  const resume = useMutation({
    mutationFn: () =>
      apiClient.post<{ onboardingUrl: string }>('/delivery/stripe/onboarding-link', {}),
    onSuccess: (r) => {
      window.location.href = r.onboardingUrl;
    },
  });

  const switchProvider = useMutation({
    mutationFn: (p: 'razorpay' | 'stripe') =>
      apiClient.put<{ paymentProvider: string }>('/delivery/payment-provider', { provider: p }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['driver-stripe-status'] });
    },
  });

  const ready = Boolean(data?.connected && data.chargesEnabled && data.payoutsEnabled);
  const activeProvider = data?.paymentProvider ?? 'razorpay';

  return (
    <div className="rounded-xl border border-border bg-card p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Globe className="h-5 w-5 text-primary" aria-hidden="true" />
          <div>
            <p className="text-sm font-medium text-foreground">Stripe (International Payouts)</p>
            <p className="text-xs text-muted-foreground">
              For drivers outside India, or as an alternative to Razorpay.
            </p>
          </div>
        </div>
        {data?.connected ? (
          ready ? (
            <span className="flex items-center gap-1 rounded-full bg-herb-tint px-2 py-0.5 text-xs text-herb">
              <CheckCircle2 className="h-3 w-3" aria-hidden="true" /> Connected
            </span>
          ) : (
            <span className="flex items-center gap-1 rounded-full bg-amber-tint px-2 py-0.5 text-xs text-amber">
              <XCircle className="h-3 w-3" aria-hidden="true" /> Action Required
            </span>
          )
        ) : null}
      </div>

      {isLoading ? (
        <div className="mt-3 h-10 animate-pulse rounded bg-secondary" />
      ) : !data?.connected ? (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-muted-foreground">
            Accept delivery payouts in your local currency. Stripe handles KYC and bank
            verification on its hosted pages.
          </p>
          <div>
            <label htmlFor="stripe-connect-country" className="block text-xs font-medium text-muted-foreground mb-1">Country</label>
            <select
              id="stripe-connect-country"
              value={country}
              onChange={(e) => setCountry(e.target.value)}
              className="w-full max-w-sm rounded-lg border border-border bg-background px-3 py-2 text-sm"
            >
              {SUPPORTED_COUNTRIES.map((c) => (
                <option key={c.code} value={c.code}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
          <button type="button"
            onClick={() => connect.mutate(country)}
            disabled={connect.isPending}
            className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {connect.isPending ? 'Starting…' : 'Connect with Stripe'}
            <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </div>
      ) : (
        <div className="mt-4 space-y-3">
          <div className="grid grid-cols-2 gap-2 text-sm">
            <Cell label="Country" value={data.country || '—'} good />
            <Cell label="Charges" value={data.chargesEnabled ? 'Enabled' : 'Pending'} good={data.chargesEnabled} />
            <Cell label="Payouts" value={data.payoutsEnabled ? 'Enabled' : 'Pending'} good={data.payoutsEnabled} />
            <Cell label="Details" value={data.detailsSubmitted ? 'Submitted' : 'Incomplete'} good={data.detailsSubmitted} />
          </div>

          {data.warning && (
            <div className="rounded-lg border border-amber/30 bg-amber-tint px-3 py-2 text-xs text-amber">
              {data.warning}
            </div>
          )}

          <div className="flex flex-wrap gap-2">
            {!ready && (
              <button type="button"
                onClick={() => resume.mutate()}
                disabled={resume.isPending}
                className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                Resume Onboarding
                <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            )}
            {ready && activeProvider !== 'stripe' && (
              <button type="button"
                onClick={() => switchProvider.mutate('stripe')}
                disabled={switchProvider.isPending}
                className="inline-flex items-center rounded-lg border border-border px-3 py-2 text-sm hover:bg-secondary disabled:opacity-50"
              >
                Make Stripe My Primary Gateway
              </button>
            )}
            {activeProvider === 'stripe' && (
              <button type="button"
                onClick={() => switchProvider.mutate('razorpay')}
                disabled={switchProvider.isPending}
                className="inline-flex items-center rounded-lg border border-border px-3 py-2 text-sm hover:bg-secondary disabled:opacity-50"
              >
                Switch to Razorpay
              </button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            Active payout gateway:{' '}
            <span className="font-medium text-foreground">
              {activeProvider === 'stripe' ? 'Stripe' : 'Razorpay'}
            </span>
          </p>
        </div>
      )}
    </div>
  );
}

function Cell({ label, value, good }: { label: string; value: string; good: boolean }) {
  return (
    <div className="rounded-lg border border-border bg-background px-3 py-2">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className={`text-sm font-medium ${good ? 'text-herb' : 'text-amber'}`}>{value}</p>
    </div>
  );
}
