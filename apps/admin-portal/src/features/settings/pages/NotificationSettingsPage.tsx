import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { ApiError } from '@/shared/types';
import { ArrowLeft, Bell, X } from 'lucide-react';

interface NotificationPreference {
  id?: string;
  userId?: string;
  category: string;
  emailEnabled: boolean;
  pushEnabled: boolean;
  smsEnabled: boolean;
}

interface PreferencesResponse {
  categories: string[];
  preferences: NotificationPreference[];
}

const CATEGORY_LABELS: Record<string, { title: string; description: string }> = {
  order: {
    title: 'Order updates',
    description: 'Receipts, status changes, cancellations, delivery confirmation.',
  },
  chef: {
    title: 'Chef activity',
    description: 'New orders and verification outcomes (chef accounts only).',
  },
  delivery: {
    title: 'Delivery updates',
    description: 'Driver assigned, pickup and drop-off notifications.',
  },
  account: {
    title: 'Account',
    description: 'Welcome messages, password changes, approval outcomes.',
  },
  marketing: {
    title: 'Promotions',
    description: 'Discounts, newsletters, and marketing campaigns. Opt-in.',
  },
};

export default function NotificationSettingsPage() {
  const qc = useQueryClient();
  const [feedback, setFeedback] = useState<{
    kind: 'success' | 'error';
    message: string;
  } | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['notification-preferences'],
    queryFn: () => apiClient.get<PreferencesResponse>('/notifications/preferences'),
  });

  const save = useMutation({
    mutationFn: (body: NotificationPreference) =>
      apiClient.put('/notifications/preferences', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notification-preferences'] });
      setFeedback({ kind: 'success', message: 'Preference saved' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      setFeedback({ kind: 'error', message: e?.error?.message ?? 'Failed to save' });
    },
  });

  const togglePref = (pref: NotificationPreference, channel: 'email' | 'push' | 'sms', value: boolean) => {
    save.mutate({
      category: pref.category,
      emailEnabled: channel === 'email' ? value : pref.emailEnabled,
      pushEnabled: channel === 'push' ? value : pref.pushEnabled,
      smsEnabled: channel === 'sms' ? value : pref.smsEnabled,
    });
  };

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
          <h1 className="page-title">Notifications</h1>
          <p className="page-description">
            Control which channels deliver each notification category for your
            account. Transactional order updates stay on by default.
          </p>
        </div>
      </div>

      {feedback && (
        <div
          className={`flex items-start justify-between gap-2 rounded-lg border px-3 py-2 text-xs ${
            feedback.kind === 'success'
              ? 'border-green-200 bg-green-50 text-green-700'
              : 'border-red-200 bg-red-50 text-red-700'
          }`}
        >
          <span>{feedback.message}</span>
          <button onClick={() => setFeedback(null)} className="shrink-0 opacity-60 hover:opacity-100">
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      <div className="rounded-xl border border-border bg-card p-6 shadow-card">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <Bell className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold text-foreground">Notification channels</h3>
            <p className="text-sm text-muted-foreground">
              Toggle email, push, or SMS per category.
            </p>
          </div>
        </div>

        {isLoading ? (
          <p className="mt-6 text-sm text-muted-foreground">Loading...</p>
        ) : (
          <div className="mt-6 overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-xs uppercase tracking-wide text-muted-foreground">
                  <th className="py-2 pr-2 font-medium">Category</th>
                  <th className="py-2 px-2 text-center font-medium">Email</th>
                  <th className="py-2 px-2 text-center font-medium">Push</th>
                  <th className="py-2 px-2 text-center font-medium">SMS</th>
                </tr>
              </thead>
              <tbody>
                {(data?.preferences ?? []).map((pref) => {
                  const label = CATEGORY_LABELS[pref.category] ?? {
                    title: pref.category,
                    description: '',
                  };
                  return (
                    <tr key={pref.category} className="border-b border-border last:border-b-0">
                      <td className="py-3 pr-2">
                        <p className="font-medium text-foreground">{label.title}</p>
                        <p className="text-xs text-muted-foreground">{label.description}</p>
                      </td>
                      <td className="py-3 px-2 text-center">
                        <Toggle
                          value={pref.emailEnabled}
                          onChange={(v) => togglePref(pref, 'email', v)}
                        />
                      </td>
                      <td className="py-3 px-2 text-center">
                        <Toggle
                          value={pref.pushEnabled}
                          onChange={(v) => togglePref(pref, 'push', v)}
                        />
                      </td>
                      <td className="py-3 px-2 text-center">
                        <Toggle
                          value={pref.smsEnabled}
                          onChange={(v) => togglePref(pref, 'sms', v)}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

function Toggle({
  value,
  onChange,
}: {
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      onClick={() => onChange(!value)}
      className={`relative inline-block h-5 w-9 rounded-full transition-colors ${
        value ? 'bg-primary' : 'bg-border'
      }`}
    >
      <span
        className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
          value ? 'translate-x-4' : 'translate-x-0.5'
        }`}
      />
    </button>
  );
}
