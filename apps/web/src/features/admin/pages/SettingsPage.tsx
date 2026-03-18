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
          <h1 className="text-2xl font-bold text-white">Platform Settings</h1>
          <p className="mt-1 text-gray-400">Configure platform-wide settings</p>
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
      <div className="rounded-xl bg-gray-800 p-6">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-white">
          <DollarSign className="h-5 w-5 text-brand-400" />
          Fees & Pricing
        </h2>

        <div className="mt-6 grid gap-6 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-gray-300">
              Platform Fee (%)
            </label>
            <div className="relative mt-1">
              <input
                type="number"
                value={settings.platformFee}
                onChange={(e) => setSettings({ ...settings, platformFee: Number(e.target.value) })}
                className="w-full rounded-lg bg-gray-700 border-gray-600 text-white pr-8"
              />
              <Percent className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
            </div>
            <p className="mt-1 text-xs text-gray-500">Fee charged on each order</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300">
              Base Delivery Fee ($)
            </label>
            <div className="relative mt-1">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
              <input
                type="number"
                step="0.01"
                value={settings.deliveryFeeBase}
                onChange={(e) =>
                  setSettings({ ...settings, deliveryFeeBase: Number(e.target.value) })
                }
                className="w-full rounded-lg bg-gray-700 border-gray-600 text-white pl-7"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300">
              Delivery Fee per km ($)
            </label>
            <div className="relative mt-1">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
              <input
                type="number"
                step="0.01"
                value={settings.deliveryFeePerKm}
                onChange={(e) =>
                  setSettings({ ...settings, deliveryFeePerKm: Number(e.target.value) })
                }
                className="w-full rounded-lg bg-gray-700 border-gray-600 text-white pl-7"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300">
              Default Minimum Order ($)
            </label>
            <div className="relative mt-1">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
              <input
                type="number"
                value={settings.minimumOrderDefault}
                onChange={(e) =>
                  setSettings({ ...settings, minimumOrderDefault: Number(e.target.value) })
                }
                className="w-full rounded-lg bg-gray-700 border-gray-600 text-white pl-7"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Order Settings */}
      <div className="rounded-xl bg-gray-800 p-6">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-white">
          <Settings className="h-5 w-5 text-brand-400" />
          Order Settings
        </h2>

        <div className="mt-6 grid gap-6 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-gray-300">
              Max Delivery Radius (km)
            </label>
            <input
              type="number"
              value={settings.maxDeliveryRadius}
              onChange={(e) =>
                setSettings({ ...settings, maxDeliveryRadius: Number(e.target.value) })
              }
              className="mt-1 w-full rounded-lg bg-gray-700 border-gray-600 text-white"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300">
              Cancellation Window (minutes)
            </label>
            <input
              type="number"
              value={settings.orderCancellationWindow}
              onChange={(e) =>
                setSettings({ ...settings, orderCancellationWindow: Number(e.target.value) })
              }
              className="mt-1 w-full rounded-lg bg-gray-700 border-gray-600 text-white"
            />
            <p className="mt-1 text-xs text-gray-500">
              Time window for customers to cancel without penalty
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300">
              Auto-Accept Timeout (minutes)
            </label>
            <input
              type="number"
              value={settings.autoAcceptTimeout}
              onChange={(e) =>
                setSettings({ ...settings, autoAcceptTimeout: Number(e.target.value) })
              }
              className="mt-1 w-full rounded-lg bg-gray-700 border-gray-600 text-white"
            />
            <p className="mt-1 text-xs text-gray-500">
              Time before order auto-cancels if not accepted
            </p>
          </div>
        </div>
      </div>

      {/* Platform Controls */}
      <div className="rounded-xl bg-gray-800 p-6">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-white">
          <Shield className="h-5 w-5 text-brand-400" />
          Platform Controls
        </h2>

        <div className="mt-6 space-y-4">
          <div className="flex items-center justify-between rounded-lg bg-gray-700/50 p-4">
            <div>
              <p className="font-medium text-white">Maintenance Mode</p>
              <p className="text-sm text-gray-400">Temporarily disable the platform</p>
            </div>
            <Toggle
              enabled={settings.maintenanceMode}
              onChange={(enabled) => setSettings({ ...settings, maintenanceMode: enabled })}
              danger
            />
          </div>

          <div className="flex items-center justify-between rounded-lg bg-gray-700/50 p-4">
            <div>
              <p className="font-medium text-white">New User Signups</p>
              <p className="text-sm text-gray-400">Allow new customers to register</p>
            </div>
            <Toggle
              enabled={settings.newUserSignups}
              onChange={(enabled) => setSettings({ ...settings, newUserSignups: enabled })}
            />
          </div>

          <div className="flex items-center justify-between rounded-lg bg-gray-700/50 p-4">
            <div>
              <p className="font-medium text-white">New Chef Signups</p>
              <p className="text-sm text-gray-400">Allow new chefs to apply</p>
            </div>
            <Toggle
              enabled={settings.newChefSignups}
              onChange={(enabled) => setSettings({ ...settings, newChefSignups: enabled })}
            />
          </div>
        </div>
      </div>

      {/* Danger Zone */}
      <div className="rounded-xl bg-red-500/10 border border-red-500/20 p-6">
        <h2 className="text-lg font-semibold text-red-400">Danger Zone</h2>
        <p className="mt-2 text-sm text-red-300/70">
          These actions are irreversible. Please proceed with caution.
        </p>

        <div className="mt-4 flex gap-3">
          <button className="btn-base bg-red-600 text-white hover:bg-red-700">
            Clear All Cache
          </button>
          <button className="btn-base bg-red-600 text-white hover:bg-red-700">
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
            ? 'bg-red-500'
            : 'bg-brand-500'
          : 'bg-gray-600'
      }`}
    >
      <span
        className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${
          enabled ? 'translate-x-5' : ''
        }`}
      />
    </button>
  );
}
