import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import {
  Settings,
  DollarSign,
  Percent,
  Shield,
  Save,
  Loader2,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';

interface PlatformSettings {
  platformFee: number;
  deliveryFeeBase: number;
  deliveryFeePerKm: number;
  minimumOrderDefault: number;
  maxDeliveryRadius: number;
  orderCancellationWindow: number;
  autoAcceptTimeout: number;
  maintenanceMode: boolean;
  newUserSignups: boolean;
  newChefSignups: boolean;
}

export default function AdminSettingsPage() {
  const [settings, setSettings] = useState<PlatformSettings>({
    platformFee: 15,
    deliveryFeeBase: 2.99,
    deliveryFeePerKm: 0.5,
    minimumOrderDefault: 15,
    maxDeliveryRadius: 20,
    orderCancellationWindow: 5,
    autoAcceptTimeout: 10,
    maintenanceMode: false,
    newUserSignups: true,
    newChefSignups: true,
  });

  const saveMutation = useMutation({
    mutationFn: (data: PlatformSettings) => apiClient.put('/admin/settings', data),
    onSuccess: () => {
      toast.success('Settings saved successfully');
    },
    onError: () => {
      toast.error('Failed to save settings');
    },
  });

  const handleSave = () => {
    saveMutation.mutate(settings);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-paper">Platform Settings</h1>
          <p className="mt-1 text-ink-muted">Configure platform-wide settings</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saveMutation.isPending}
          className="btn-primary"
        >
          {saveMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          Save Changes
        </button>
      </div>

      {/* Fees & Pricing */}
      <div className="rounded-xl bg-ink p-6">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-paper">
          <DollarSign className="h-5 w-5 text-herb-soft" />
          Fees & Pricing
        </h2>

        <div className="mt-6 grid gap-6 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-ink-muted">
              Platform Fee (%)
            </label>
            <div className="relative mt-1">
              <input
                type="number"
                value={settings.platformFee}
                onChange={(e) => setSettings({ ...settings, platformFee: Number(e.target.value) })}
                className="w-full rounded-lg bg-ink-soft border-ink-soft text-paper pr-8"
              />
              <Percent className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-ink-muted" />
            </div>
            <p className="mt-1 text-xs text-ink-muted">Fee charged on each order</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-ink-muted">
              Base Delivery Fee ($)
            </label>
            <div className="relative mt-1">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-muted">$</span>
              <input
                type="number"
                step="0.01"
                value={settings.deliveryFeeBase}
                onChange={(e) =>
                  setSettings({ ...settings, deliveryFeeBase: Number(e.target.value) })
                }
                className="w-full rounded-lg bg-ink-soft border-ink-soft text-paper pl-7"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-ink-muted">
              Delivery Fee per km ($)
            </label>
            <div className="relative mt-1">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-muted">$</span>
              <input
                type="number"
                step="0.01"
                value={settings.deliveryFeePerKm}
                onChange={(e) =>
                  setSettings({ ...settings, deliveryFeePerKm: Number(e.target.value) })
                }
                className="w-full rounded-lg bg-ink-soft border-ink-soft text-paper pl-7"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-ink-muted">
              Default Minimum Order ($)
            </label>
            <div className="relative mt-1">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-muted">$</span>
              <input
                type="number"
                value={settings.minimumOrderDefault}
                onChange={(e) =>
                  setSettings({ ...settings, minimumOrderDefault: Number(e.target.value) })
                }
                className="w-full rounded-lg bg-ink-soft border-ink-soft text-paper pl-7"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Order Settings */}
      <div className="rounded-xl bg-ink p-6">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-paper">
          <Settings className="h-5 w-5 text-herb-soft" />
          Order Settings
        </h2>

        <div className="mt-6 grid gap-6 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-ink-muted">
              Max Delivery Radius (km)
            </label>
            <input
              type="number"
              value={settings.maxDeliveryRadius}
              onChange={(e) =>
                setSettings({ ...settings, maxDeliveryRadius: Number(e.target.value) })
              }
              className="mt-1 w-full rounded-lg bg-ink-soft border-ink-soft text-paper"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-ink-muted">
              Cancellation Window (minutes)
            </label>
            <input
              type="number"
              value={settings.orderCancellationWindow}
              onChange={(e) =>
                setSettings({ ...settings, orderCancellationWindow: Number(e.target.value) })
              }
              className="mt-1 w-full rounded-lg bg-ink-soft border-ink-soft text-paper"
            />
            <p className="mt-1 text-xs text-ink-muted">
              Time window for customers to cancel without penalty
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-ink-muted">
              Auto-Accept Timeout (minutes)
            </label>
            <input
              type="number"
              value={settings.autoAcceptTimeout}
              onChange={(e) =>
                setSettings({ ...settings, autoAcceptTimeout: Number(e.target.value) })
              }
              className="mt-1 w-full rounded-lg bg-ink-soft border-ink-soft text-paper"
            />
            <p className="mt-1 text-xs text-ink-muted">
              Time before order auto-cancels if not accepted
            </p>
          </div>
        </div>
      </div>

      {/* Platform Controls */}
      <div className="rounded-xl bg-ink p-6">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-paper">
          <Shield className="h-5 w-5 text-herb-soft" />
          Platform Controls
        </h2>

        <div className="mt-6 space-y-4">
          <div className="flex items-center justify-between rounded-lg bg-ink-soft/50 p-4">
            <div>
              <p className="font-medium text-paper">Maintenance Mode</p>
              <p className="text-sm text-ink-muted">Temporarily disable the platform</p>
            </div>
            <Toggle
              enabled={settings.maintenanceMode}
              onChange={(enabled) => setSettings({ ...settings, maintenanceMode: enabled })}
              danger
            />
          </div>

          <div className="flex items-center justify-between rounded-lg bg-ink-soft/50 p-4">
            <div>
              <p className="font-medium text-paper">New User Signups</p>
              <p className="text-sm text-ink-muted">Allow new customers to register</p>
            </div>
            <Toggle
              enabled={settings.newUserSignups}
              onChange={(enabled) => setSettings({ ...settings, newUserSignups: enabled })}
            />
          </div>

          <div className="flex items-center justify-between rounded-lg bg-ink-soft/50 p-4">
            <div>
              <p className="font-medium text-paper">New Chef Signups</p>
              <p className="text-sm text-ink-muted">Allow new chefs to apply</p>
            </div>
            <Toggle
              enabled={settings.newChefSignups}
              onChange={(enabled) => setSettings({ ...settings, newChefSignups: enabled })}
            />
          </div>
        </div>
      </div>

      {/* Danger Zone */}
      <div className="rounded-xl bg-paprika/10 border border-paprika/20 p-6">
        <h2 className="text-lg font-semibold text-paprika">Danger Zone</h2>
        <p className="mt-2 text-sm text-paprika/70">
          These actions are irreversible. Please proceed with caution.
        </p>

        <div className="mt-4 flex gap-3">
          <button className="btn-base bg-paprika text-paper hover:bg-paprika">
            Clear All Cache
          </button>
          <button className="btn-base bg-paprika text-paper hover:bg-paprika">
            Rebuild Search Index
          </button>
        </div>
      </div>
    </div>
  );
}

function Toggle({
  enabled,
  onChange,
  danger,
}: {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  danger?: boolean;
}) {
  return (
    <button
      onClick={() => onChange(!enabled)}
      className={`relative h-6 w-11 rounded-full transition-colors ${
        enabled
          ? danger
            ? 'bg-paprika'
            : 'bg-herb'
          : 'bg-ink-soft'
      }`}
    >
      <span
        className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-bone shadow transition-transform ${
          enabled ? 'translate-x-5' : ''
        }`}
      />
    </button>
  );
}
