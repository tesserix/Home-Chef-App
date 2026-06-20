import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Bell, Power, Lock, Trash2, Banknote, CheckCircle2, XCircle, Globe, ExternalLink } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

interface SettingsData {
  notifications: {
    pushNewOrder: boolean;
    pushOrderUpdate: boolean;
    emailDailySummary: boolean;
    emailWeeklyReport: boolean;
    smsNewOrder: boolean;
  };
  autoAcceptOrders: boolean;
  autoAcceptThreshold: number;
  acceptingOrders: boolean;
  authProvider?: string;
}

interface PayoutData {
  payoutMethod: string;
  bankAccountName: string;
  bankAccountNumber: string;
  bankIFSC: string;
  upiId: string;
  razorpayConnected: boolean;
  razorpayAccountId: string;
  stripeConnected?: boolean;
  stripeAccountId?: string;
  paymentProvider?: 'razorpay' | 'stripe';
  payoutCountry?: string;
}

export default function SettingsPage() {
  const queryClient = useQueryClient();

  const { data: settings, isLoading } = useQuery({
    queryKey: ['chef-settings'],
    queryFn: () => apiClient.get<SettingsData>('/chef/settings'),
  });

  const { data: payoutData, isLoading: payoutLoading } = useQuery({
    queryKey: ['chef-payout'],
    queryFn: () => apiClient.get<PayoutData>('/chef/payout'),
  });

  const [localSettings, setLocalSettings] = useState<SettingsData | null>(null);

  const [payoutForm, setPayoutForm] = useState({
    payoutMethod: 'bank_transfer' as 'bank_transfer' | 'upi',
    bankAccountName: '',
    bankAccountNumber: '',
    bankIFSC: '',
    upiId: '',
  });
  const [payoutEditing, setPayoutEditing] = useState(false);

  useEffect(() => {
    if (settings) setLocalSettings(settings);
  }, [settings]);

  useEffect(() => {
    if (payoutData) {
      setPayoutForm({
        payoutMethod: (payoutData.payoutMethod || 'bank_transfer') as 'bank_transfer' | 'upi',
        bankAccountName: payoutData.bankAccountName || '',
        bankAccountNumber: '',  // Don't populate masked value
        bankIFSC: payoutData.bankIFSC || '',
        upiId: '',  // Don't populate masked value
      });
    }
  }, [payoutData]);

  const saveMutation = useMutation({
    mutationFn: (data: SettingsData) => apiClient.put('/chef/settings', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-settings'] });
      toast.success('Settings saved');
    },
    onError: () => toast.error('Failed to save settings'),
  });

  const payoutMutation = useMutation({
    mutationFn: (data: typeof payoutForm) => apiClient.post<PayoutData>('/chef/payout', data),
    onSuccess: (data) => {
      // Use the response directly — don't refetch, since secrets are stored async
      queryClient.setQueryData(['chef-payout'], data);
      toast.success('Payout details saved');
      setPayoutEditing(false);
    },
    onError: () => toast.error('Failed to save payout details'),
  });

  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });

  if (isLoading || !localSettings) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-herb border-t-transparent" />
      </div>
    );
  }

  const toggleNotification = (key: keyof SettingsData['notifications']) => {
    setLocalSettings({
      ...localSettings,
      notifications: {
        ...localSettings.notifications,
        [key]: !localSettings.notifications[key],
      },
    });
  };

  return (
    <motion.div variants={staggerContainer} initial="hidden" animate="visible" className="space-y-6">
      <motion.div variants={fadeInUp} className="page-header">
        <h1 className="page-title">Settings</h1>
        <p className="page-description">Manage your chef preferences</p>
      </motion.div>

      {/* Accepting Orders */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
        <div className="flex items-center gap-3">
          <Power className="h-5 w-5 text-herb" />
          <h2 className="text-lg font-semibold text-ink">Order Acceptance</h2>
        </div>
        <div className="mt-4 space-y-4">
          <ToggleRow
            label="Accepting Orders"
            description="Master toggle - turn off to stop receiving new orders"
            checked={localSettings.acceptingOrders}
            onChange={() =>
              setLocalSettings({ ...localSettings, acceptingOrders: !localSettings.acceptingOrders })
            }
          />
          <ToggleRow
            label="Auto-accept Orders"
            description="Automatically accept orders below a threshold"
            checked={localSettings.autoAcceptOrders}
            onChange={() =>
              setLocalSettings({ ...localSettings, autoAcceptOrders: !localSettings.autoAcceptOrders })
            }
          />
          {localSettings.autoAcceptOrders && (
            <div className="ml-8">
              <label htmlFor="settings-auto-accept-threshold" className="block text-sm font-medium text-ink-soft">
                Auto-accept threshold ($)
              </label>
              <input
                id="settings-auto-accept-threshold"
                type="number"
                inputMode="numeric"
                min={0}
                value={localSettings.autoAcceptThreshold}
                onChange={(e) =>
                  setLocalSettings({
                    ...localSettings,
                    autoAcceptThreshold: Number(e.target.value),
                  })
                }
                className="mt-1 w-32 rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
              />
              <p className="mt-1 text-xs text-ink-muted">
                Orders under this amount will be auto-accepted
              </p>
            </div>
          )}
        </div>
      </motion.div>

      {/* Notifications */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
        <div className="flex items-center gap-3">
          <Bell className="h-5 w-5 text-herb" />
          <h2 className="text-lg font-semibold text-ink">Notifications</h2>
        </div>
        <div className="mt-4 space-y-4">
          <h3 className="text-sm font-medium text-ink-muted">Push Notifications</h3>
          <ToggleRow
            label="New order alerts"
            description="Get notified when a new order comes in"
            checked={localSettings.notifications.pushNewOrder}
            onChange={() => toggleNotification('pushNewOrder')}
          />
          <ToggleRow
            label="Order updates"
            description="Notifications for order status changes"
            checked={localSettings.notifications.pushOrderUpdate}
            onChange={() => toggleNotification('pushOrderUpdate')}
          />
          <div className="border-t border-mist pt-4">
            <h3 className="text-sm font-medium text-ink-muted">Email</h3>
          </div>
          <ToggleRow
            label="Daily summary"
            description="Receive a daily email with order and earnings summary"
            checked={localSettings.notifications.emailDailySummary}
            onChange={() => toggleNotification('emailDailySummary')}
          />
          <ToggleRow
            label="Weekly report"
            description="Weekly performance report with analytics"
            checked={localSettings.notifications.emailWeeklyReport}
            onChange={() => toggleNotification('emailWeeklyReport')}
          />
          <div className="border-t border-mist pt-4">
            <h3 className="text-sm font-medium text-ink-muted">SMS</h3>
          </div>
          <ToggleRow
            label="New order SMS"
            description="Get an SMS for each new order"
            checked={localSettings.notifications.smsNewOrder}
            onChange={() => toggleNotification('smsNewOrder')}
          />
        </div>
      </motion.div>

      {/* Payout Details */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Banknote className="h-5 w-5 text-herb" />
            <h2 className="text-lg font-semibold text-ink">Payout Details</h2>
          </div>
          {payoutData?.razorpayConnected ? (
            <span className="flex items-center gap-1.5 rounded-full bg-herb-tint px-3 py-1 text-xs font-medium text-herb">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Razorpay Connected
            </span>
          ) : payoutData?.payoutMethod ? (
            <span className="flex items-center gap-1.5 rounded-full bg-amber-tint px-3 py-1 text-xs font-medium text-amber">
              <XCircle className="h-3.5 w-3.5" />
              Razorpay Pending
            </span>
          ) : null}
        </div>
        {payoutData?.razorpayConnected && payoutData.razorpayAccountId && (
          <p className="mt-2 text-xs text-ink-muted">
            Linked Account: {payoutData.razorpayAccountId}
          </p>
        )}

        {payoutLoading ? (
          <div className="mt-4 flex items-center justify-center py-8">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-herb border-t-transparent" />
          </div>
        ) : !payoutEditing && payoutData?.payoutMethod ? (
          <div className="mt-4 space-y-3">
            {payoutData.payoutMethod === 'bank_transfer' ? (
              <>
                <div className="flex justify-between text-sm">
                  <span className="text-ink-muted">Method</span>
                  <span className="font-medium text-ink">Bank Transfer</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-ink-muted">Account Holder</span>
                  <span className="font-medium text-ink">{payoutData.bankAccountName}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-ink-muted">Account Number</span>
                  <span className="font-medium text-ink">{payoutData.bankAccountNumber}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-ink-muted">IFSC Code</span>
                  <span className="font-medium text-ink">{payoutData.bankIFSC}</span>
                </div>
              </>
            ) : (
              <>
                <div className="flex justify-between text-sm">
                  <span className="text-ink-muted">Method</span>
                  <span className="font-medium text-ink">UPI</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-ink-muted">UPI ID</span>
                  <span className="font-medium text-ink">{payoutData.upiId}</span>
                </div>
              </>
            )}
            <Button
              size="sm"
              variant="outline"
              className="mt-2"
              onClick={() => setPayoutEditing(true)}
            >
              Edit
            </Button>
          </div>
        ) : !payoutEditing ? (
          <div className="mt-4">
            <p className="text-sm text-ink-muted">No payout details configured yet.</p>
            <Button
              size="sm"
              className="mt-3"
              onClick={() => setPayoutEditing(true)}
            >
              Set Up Payouts
            </Button>
          </div>
        ) : (
          <div className="mt-4 max-w-md space-y-4">
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setPayoutForm({ ...payoutForm, payoutMethod: 'bank_transfer' })}
                className={`relative inline-flex items-center rounded-full px-4 py-1.5 text-sm font-medium transition-colors ${
                  payoutForm.payoutMethod === 'bank_transfer'
                    ? 'bg-herb text-paper'
                    : 'bg-mist text-ink-soft hover:bg-mist-strong'
                }`}
              >
                Bank Transfer
              </button>
              <button
                type="button"
                onClick={() => setPayoutForm({ ...payoutForm, payoutMethod: 'upi' })}
                className={`relative inline-flex items-center rounded-full px-4 py-1.5 text-sm font-medium transition-colors ${
                  payoutForm.payoutMethod === 'upi'
                    ? 'bg-herb text-paper'
                    : 'bg-mist text-ink-soft hover:bg-mist-strong'
                }`}
              >
                UPI
              </button>
            </div>

            {payoutForm.payoutMethod === 'bank_transfer' ? (
              <div className="space-y-4">
                <div>
                  <label htmlFor="payout-bank-name" className="block text-sm font-medium text-ink-soft">Account Holder Name</label>
                  <input
                    id="payout-bank-name"
                    type="text"
                    autoComplete="name"
                    value={payoutForm.bankAccountName}
                    onChange={(e) => setPayoutForm({ ...payoutForm, bankAccountName: e.target.value })}
                    placeholder="Name as on bank account"
                    className="mt-1 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
                  />
                </div>
                <div>
                  <label htmlFor="payout-bank-number" className="block text-sm font-medium text-ink-soft">Account Number</label>
                  <input
                    id="payout-bank-number"
                    type="text"
                    inputMode="numeric"
                    value={payoutForm.bankAccountNumber}
                    onChange={(e) => setPayoutForm({ ...payoutForm, bankAccountNumber: e.target.value })}
                    placeholder="Enter account number"
                    className="mt-1 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
                  />
                </div>
                <div>
                  <label htmlFor="payout-bank-ifsc" className="block text-sm font-medium text-ink-soft">IFSC Code</label>
                  <input
                    id="payout-bank-ifsc"
                    type="text"
                    value={payoutForm.bankIFSC}
                    onChange={(e) => setPayoutForm({ ...payoutForm, bankIFSC: e.target.value.toUpperCase() })}
                    placeholder="e.g. SBIN0001234"
                    className="mt-1 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
                  />
                </div>
              </div>
            ) : (
              <div>
                <label htmlFor="payout-upi-id" className="block text-sm font-medium text-ink-soft">UPI ID</label>
                <input
                  id="payout-upi-id"
                  type="text"
                  value={payoutForm.upiId}
                  onChange={(e) => setPayoutForm({ ...payoutForm, upiId: e.target.value })}
                  placeholder="yourname@upi"
                  className="mt-1 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
                />
              </div>
            )}

            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={() => payoutMutation.mutate(payoutForm)}
                isLoading={payoutMutation.isPending}
              >
                Save
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setPayoutEditing(false);
                  if (payoutData) {
                    setPayoutForm({
                      payoutMethod: (payoutData.payoutMethod || 'bank_transfer') as 'bank_transfer' | 'upi',
                      bankAccountName: payoutData.bankAccountName || '',
                      bankAccountNumber: '',
                      bankIFSC: payoutData.bankIFSC || '',
                      upiId: '',
                    });
                  }
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}
      </motion.div>

      {/* Stripe Connect — international payouts. Lives alongside Razorpay
          so chefs in any country can get paid. Payments to a Stripe-onboarded
          chef use Stripe at checkout; Razorpay-onboarded chefs still use Razorpay. */}
      <StripeConnectCard />

      {/* Change Password — only for email/password accounts, not social logins */}
      {(!localSettings.authProvider || localSettings.authProvider === 'email') ? (
        <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
          <div className="flex items-center gap-3">
            <Lock className="h-5 w-5 text-herb" />
            <h2 className="text-lg font-semibold text-ink">Change Password</h2>
          </div>
          <div className="mt-4 max-w-md space-y-4">
            <div>
              <label htmlFor="settings-current-password" className="block text-sm font-medium text-ink-soft">Current Password</label>
              <input
                id="settings-current-password"
                type="password"
                autoComplete="current-password"
                value={passwordForm.currentPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, currentPassword: e.target.value })}
                className="mt-1 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
              />
            </div>
            <div>
              <label htmlFor="settings-new-password" className="block text-sm font-medium text-ink-soft">New Password</label>
              <input
                id="settings-new-password"
                type="password"
                autoComplete="new-password"
                value={passwordForm.newPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, newPassword: e.target.value })}
                className="mt-1 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
              />
            </div>
            <div>
              <label htmlFor="settings-confirm-password" className="block text-sm font-medium text-ink-soft">Confirm New Password</label>
              <input
                id="settings-confirm-password"
                type="password"
                autoComplete="new-password"
                value={passwordForm.confirmPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, confirmPassword: e.target.value })}
                className="mt-1 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
              />
            </div>
            <Button
              size="sm"
              disabled={!passwordForm.currentPassword || !passwordForm.newPassword || passwordForm.newPassword !== passwordForm.confirmPassword}
              onClick={async () => {
                try {
                  await apiClient.put('/profile/password', {
                    currentPassword: passwordForm.currentPassword,
                    newPassword: passwordForm.newPassword,
                  });
                  toast.success('Password updated successfully');
                  setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
                } catch {
                  toast.error('Failed to update password. Check your current password.');
                }
              }}
            >
              Update Password
            </Button>
          </div>
        </motion.div>
      ) : (
        <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
          <div className="flex items-center gap-3">
            <Lock className="h-5 w-5 text-ink-muted" />
            <h2 className="text-lg font-semibold text-ink">Change Password</h2>
          </div>
          <p className="mt-3 text-sm text-ink-muted">
            Your account is linked to{' '}
            <span className="font-medium capitalize">{localSettings.authProvider}</span> login.
            Password management is handled by your social login provider.
          </p>
        </motion.div>
      )}

      {/* Danger Zone */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-paprika/30 bg-bone p-6">
        <div className="flex items-center gap-3">
          <Trash2 className="h-5 w-5 text-paprika" />
          <h2 className="text-lg font-semibold text-paprika">Danger Zone</h2>
        </div>
        <p className="mt-2 text-sm text-ink-soft">
          Deactivating your account will hide your kitchen from customers and pause all orders.
        </p>
        <Button
          variant="danger"
          size="sm"
          className="mt-4"
          onClick={() => toast.error('Account deactivation is not available in demo mode')}
        >
          Deactivate Account
        </Button>
      </motion.div>

      {/* Save Button */}
      <motion.div variants={fadeInUp} className="flex justify-end">
        <Button
          size="lg"
          onClick={() => saveMutation.mutate(localSettings)}
          isLoading={saveMutation.isPending}
        >
          Save Settings
        </Button>
      </motion.div>
    </motion.div>
  );
}

function ToggleRow({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-medium text-ink">{label}</p>
        <p className="text-xs text-ink-muted">{description}</p>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        onClick={onChange}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
          checked ? 'bg-herb' : 'bg-mist'
        }`}
      >
        <span
          className={`inline-block h-4 w-4 rounded-full bg-bone transition-transform ${
            checked ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
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

// ISO-3166 countries Stripe Connect supports for Express accounts — shortened
// to the set our vendors actually operate in. Extend as needed.
const SUPPORTED_STRIPE_COUNTRIES: { code: string; name: string }[] = [
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
    queryKey: ['chef-stripe-status'],
    queryFn: () => apiClient.get<StripeStatus>('/chef/stripe/status'),
    // Poll once a minute while onboarding is in progress so the UI reflects
    // completion without requiring a manual refresh.
    refetchInterval: (q) => {
      const d = q.state.data as StripeStatus | undefined;
      if (d?.connected && !d.chargesEnabled) return 60_000;
      return false;
    },
  });

  // After Stripe redirects the chef back with ?done=1 or ?refresh=1, clear
  // the query string and refetch status so the UI updates immediately.
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

  const connectMutation = useMutation({
    mutationFn: (c: string) =>
      apiClient.post<{ accountId: string; onboardingUrl: string; country: string }>(
        '/chef/stripe/connect',
        { country: c }
      ),
    onSuccess: (res) => {
      window.location.href = res.onboardingUrl;
    },
    onError: () => toast.error('Failed to start Stripe onboarding'),
  });

  const resumeMutation = useMutation({
    mutationFn: () =>
      apiClient.post<{ onboardingUrl: string }>('/chef/stripe/onboarding-link', {}),
    onSuccess: (res) => {
      window.location.href = res.onboardingUrl;
    },
    onError: () => toast.error('Failed to resume onboarding'),
  });

  const switchProviderMutation = useMutation({
    mutationFn: (provider: 'razorpay' | 'stripe') =>
      apiClient.put<{ paymentProvider: string }>('/chef/payment-provider', { provider }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-stripe-status'] });
      queryClient.invalidateQueries({ queryKey: ['chef-payout'] });
      toast.success('Payment provider updated');
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : 'Failed to update provider';
      toast.error(msg);
    },
  });

  const ready = Boolean(data?.connected && data.chargesEnabled && data.payoutsEnabled);
  const activeProvider = data?.paymentProvider ?? 'razorpay';

  return (
    <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Globe className="h-5 w-5 text-herb" />
          <div>
            <h2 className="text-lg font-semibold text-ink">Stripe (International Payouts)</h2>
            <p className="text-xs text-ink-muted">
              For chefs outside India, or as an alternative to Razorpay.
            </p>
          </div>
        </div>
        {data?.connected ? (
          ready ? (
            <span className="flex items-center gap-1.5 rounded-full bg-herb-tint px-3 py-1 text-xs font-medium text-herb">
              <CheckCircle2 className="h-3.5 w-3.5" /> Connected
            </span>
          ) : (
            <span className="flex items-center gap-1.5 rounded-full bg-amber-tint px-3 py-1 text-xs font-medium text-amber">
              <XCircle className="h-3.5 w-3.5" /> Action Required
            </span>
          )
        ) : null}
      </div>

      {isLoading ? (
        <div className="mt-4 flex items-center justify-center py-8">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-herb border-t-transparent" />
        </div>
      ) : !data?.connected ? (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-ink-soft">
            Accept payments and receive payouts in your local currency. Stripe handles KYC and bank
            verification on their hosted pages — just pick your country and follow the flow.
          </p>
          <div>
            <label htmlFor="stripe-connect-country" className="block text-sm font-medium text-ink-soft">Country</label>
            <select
              id="stripe-connect-country"
              value={country}
              onChange={(e) => setCountry(e.target.value)}
              className="mt-1 w-full max-w-sm rounded-lg border border-mist-strong px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/20"
            >
              {SUPPORTED_STRIPE_COUNTRIES.map((c) => (
                <option key={c.code} value={c.code}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
          <Button
            size="sm"
            onClick={() => connectMutation.mutate(country)}
            isLoading={connectMutation.isPending}
          >
            Connect with Stripe
            <ExternalLink className="ml-1 h-3.5 w-3.5" />
          </Button>
        </div>
      ) : (
        <div className="mt-4 space-y-3">
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="rounded-lg border border-mist bg-paper px-3 py-2">
              <p className="text-xs text-ink-muted">Country</p>
              <p className="font-medium text-ink">{data.country || '—'}</p>
            </div>
            <div className="rounded-lg border border-mist bg-paper px-3 py-2">
              <p className="text-xs text-ink-muted">Charges</p>
              <p className={`font-medium ${data.chargesEnabled ? 'text-herb' : 'text-amber'}`}>
                {data.chargesEnabled ? 'Enabled' : 'Pending'}
              </p>
            </div>
            <div className="rounded-lg border border-mist bg-paper px-3 py-2">
              <p className="text-xs text-ink-muted">Payouts</p>
              <p className={`font-medium ${data.payoutsEnabled ? 'text-herb' : 'text-amber'}`}>
                {data.payoutsEnabled ? 'Enabled' : 'Pending'}
              </p>
            </div>
            <div className="rounded-lg border border-mist bg-paper px-3 py-2">
              <p className="text-xs text-ink-muted">Details Submitted</p>
              <p className={`font-medium ${data.detailsSubmitted ? 'text-herb' : 'text-amber'}`}>
                {data.detailsSubmitted ? 'Yes' : 'No'}
              </p>
            </div>
          </div>

          {data.warning && (
            <div className="rounded-lg border border-amber/30 bg-amber-tint px-3 py-2 text-xs text-amber">
              {data.warning}
            </div>
          )}

          {!ready && (
            <div className="flex items-start gap-2 rounded-lg border border-amber/30 bg-amber-tint px-3 py-2 text-xs text-amber">
              <span>
                Stripe needs more information before you can accept payments. Resume onboarding to
                finish.
              </span>
            </div>
          )}

          <div className="flex flex-wrap gap-2">
            {!ready && (
              <Button
                size="sm"
                onClick={() => resumeMutation.mutate()}
                isLoading={resumeMutation.isPending}
              >
                Resume Onboarding
                <ExternalLink className="ml-1 h-3.5 w-3.5" />
              </Button>
            )}
            {ready && activeProvider !== 'stripe' && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => switchProviderMutation.mutate('stripe')}
                isLoading={switchProviderMutation.isPending}
              >
                Make Stripe My Primary Gateway
              </Button>
            )}
            {activeProvider === 'stripe' && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => switchProviderMutation.mutate('razorpay')}
                isLoading={switchProviderMutation.isPending}
              >
                Switch Back to Razorpay
              </Button>
            )}
            <Button size="sm" variant="outline" onClick={() => refetch()}>
              Refresh Status
            </Button>
          </div>

          <p className="text-xs text-ink-muted">
            Active gateway for your orders:{' '}
            <span className="font-medium text-ink-soft">
              {activeProvider === 'stripe' ? 'Stripe' : 'Razorpay'}
            </span>
          </p>
        </div>
      )}
    </motion.div>
  );
}
