import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Bell, Power, Lock, Trash2, Banknote, CheckCircle2, XCircle } from 'lucide-react';
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
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-500 border-t-transparent" />
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
        <p className="page-description">Manage your vendor preferences</p>
      </motion.div>

      {/* Accepting Orders */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
        <div className="flex items-center gap-3">
          <Power className="h-5 w-5 text-brand-500" />
          <h2 className="text-lg font-semibold text-gray-900">Order Acceptance</h2>
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
              <label className="block text-sm font-medium text-gray-700">
                Auto-accept threshold ($)
              </label>
              <input
                type="number"
                value={localSettings.autoAcceptThreshold}
                onChange={(e) =>
                  setLocalSettings({
                    ...localSettings,
                    autoAcceptThreshold: Number(e.target.value),
                  })
                }
                className="mt-1 w-32 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
              />
              <p className="mt-1 text-xs text-gray-500">
                Orders under this amount will be auto-accepted
              </p>
            </div>
          )}
        </div>
      </motion.div>

      {/* Notifications */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
        <div className="flex items-center gap-3">
          <Bell className="h-5 w-5 text-brand-500" />
          <h2 className="text-lg font-semibold text-gray-900">Notifications</h2>
        </div>
        <div className="mt-4 space-y-4">
          <h3 className="text-sm font-medium text-gray-500">Push Notifications</h3>
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
          <div className="border-t border-gray-100 pt-4">
            <h3 className="text-sm font-medium text-gray-500">Email</h3>
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
          <div className="border-t border-gray-100 pt-4">
            <h3 className="text-sm font-medium text-gray-500">SMS</h3>
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
      <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Banknote className="h-5 w-5 text-brand-500" />
            <h2 className="text-lg font-semibold text-gray-900">Payout Details</h2>
          </div>
          {payoutData?.razorpayConnected ? (
            <span className="flex items-center gap-1.5 rounded-full bg-green-100 px-3 py-1 text-xs font-medium text-green-700">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Razorpay Connected
            </span>
          ) : payoutData?.payoutMethod ? (
            <span className="flex items-center gap-1.5 rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-700">
              <XCircle className="h-3.5 w-3.5" />
              Razorpay Pending
            </span>
          ) : null}
        </div>
        {payoutData?.razorpayConnected && payoutData.razorpayAccountId && (
          <p className="mt-2 text-xs text-gray-400">
            Linked Account: {payoutData.razorpayAccountId}
          </p>
        )}

        {payoutLoading ? (
          <div className="mt-4 flex items-center justify-center py-8">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-brand-500 border-t-transparent" />
          </div>
        ) : !payoutEditing && payoutData?.payoutMethod ? (
          <div className="mt-4 space-y-3">
            {payoutData.payoutMethod === 'bank_transfer' ? (
              <>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Method</span>
                  <span className="font-medium text-gray-900">Bank Transfer</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Account Holder</span>
                  <span className="font-medium text-gray-900">{payoutData.bankAccountName}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Account Number</span>
                  <span className="font-medium text-gray-900">{payoutData.bankAccountNumber}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">IFSC Code</span>
                  <span className="font-medium text-gray-900">{payoutData.bankIFSC}</span>
                </div>
              </>
            ) : (
              <>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Method</span>
                  <span className="font-medium text-gray-900">UPI</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">UPI ID</span>
                  <span className="font-medium text-gray-900">{payoutData.upiId}</span>
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
            <p className="text-sm text-gray-500">No payout details configured yet.</p>
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
                    ? 'bg-brand-500 text-white'
                    : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                Bank Transfer
              </button>
              <button
                type="button"
                onClick={() => setPayoutForm({ ...payoutForm, payoutMethod: 'upi' })}
                className={`relative inline-flex items-center rounded-full px-4 py-1.5 text-sm font-medium transition-colors ${
                  payoutForm.payoutMethod === 'upi'
                    ? 'bg-brand-500 text-white'
                    : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                UPI
              </button>
            </div>

            {payoutForm.payoutMethod === 'bank_transfer' ? (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">Account Holder Name</label>
                  <input
                    type="text"
                    value={payoutForm.bankAccountName}
                    onChange={(e) => setPayoutForm({ ...payoutForm, bankAccountName: e.target.value })}
                    placeholder="Name as on bank account"
                    className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Account Number</label>
                  <input
                    type="text"
                    value={payoutForm.bankAccountNumber}
                    onChange={(e) => setPayoutForm({ ...payoutForm, bankAccountNumber: e.target.value })}
                    placeholder="Enter account number"
                    className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">IFSC Code</label>
                  <input
                    type="text"
                    value={payoutForm.bankIFSC}
                    onChange={(e) => setPayoutForm({ ...payoutForm, bankIFSC: e.target.value.toUpperCase() })}
                    placeholder="e.g. SBIN0001234"
                    className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
                  />
                </div>
              </div>
            ) : (
              <div>
                <label className="block text-sm font-medium text-gray-700">UPI ID</label>
                <input
                  type="text"
                  value={payoutForm.upiId}
                  onChange={(e) => setPayoutForm({ ...payoutForm, upiId: e.target.value })}
                  placeholder="yourname@upi"
                  className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
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

      {/* Change Password — only for email/password accounts, not social logins */}
      {(!localSettings.authProvider || localSettings.authProvider === 'email') ? (
        <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
          <div className="flex items-center gap-3">
            <Lock className="h-5 w-5 text-brand-500" />
            <h2 className="text-lg font-semibold text-gray-900">Change Password</h2>
          </div>
          <div className="mt-4 max-w-md space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Current Password</label>
              <input
                type="password"
                value={passwordForm.currentPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, currentPassword: e.target.value })}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">New Password</label>
              <input
                type="password"
                value={passwordForm.newPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, newPassword: e.target.value })}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Confirm New Password</label>
              <input
                type="password"
                value={passwordForm.confirmPassword}
                onChange={(e) => setPasswordForm({ ...passwordForm, confirmPassword: e.target.value })}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
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
        <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
          <div className="flex items-center gap-3">
            <Lock className="h-5 w-5 text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Change Password</h2>
          </div>
          <p className="mt-3 text-sm text-gray-500">
            Your account is linked to{' '}
            <span className="font-medium capitalize">{localSettings.authProvider}</span> login.
            Password management is handled by your social login provider.
          </p>
        </motion.div>
      )}

      {/* Danger Zone */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-red-200 bg-white p-6">
        <div className="flex items-center gap-3">
          <Trash2 className="h-5 w-5 text-red-500" />
          <h2 className="text-lg font-semibold text-red-900">Danger Zone</h2>
        </div>
        <p className="mt-2 text-sm text-gray-600">
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
        <p className="text-sm font-medium text-gray-900">{label}</p>
        <p className="text-xs text-gray-500">{description}</p>
      </div>
      <button
        onClick={onChange}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
          checked ? 'bg-brand-500' : 'bg-gray-200'
        }`}
      >
        <span
          className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
            checked ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
    </div>
  );
}
