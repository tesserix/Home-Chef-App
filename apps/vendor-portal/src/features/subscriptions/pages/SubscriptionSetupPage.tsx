import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { CalendarClock, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

// Chef tiffin-subscription setup (#284) — wired to GET/PUT /chef/subscription-config
// (the customer offer + price engine read the same config). Enabling requires a
// published weekly menu (#1).

interface SubConfig {
  enabled: boolean;
  slots: string[];
  cadences: string[];
  perMealPrice: number;
  deliveryFee: number;
  dailyCapacity: number;
  cutoffTime: string;
  trialEnabled: boolean;
  trialDurationDays: number;
  trialPrice: number;
}

interface ConfigResponse {
  config: SubConfig;
  hasPublishedMenu: boolean;
}

const EMPTY: SubConfig = {
  enabled: false, slots: ['lunch'], cadences: ['weekly', 'monthly'], perMealPrice: 0,
  deliveryFee: 0, dailyCapacity: 0, cutoffTime: '21:00', trialEnabled: false, trialDurationDays: 3, trialPrice: 0,
};

export default function SubscriptionSetupPage() {
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ['chef-subscription-config'],
    queryFn: () => apiClient.get<ConfigResponse>('/chef/subscription-config'),
  });
  const res = data as unknown as ConfigResponse | undefined;
  const hasMenu = res?.hasPublishedMenu ?? false;

  const [form, setForm] = useState<SubConfig>(EMPTY);
  useEffect(() => {
    if (res?.config) setForm({ ...EMPTY, ...res.config, slots: res.config.slots ?? ['lunch'], cadences: res.config.cadences ?? ['weekly', 'monthly'] });
  }, [res]);

  const save = useMutation({
    mutationFn: (body: SubConfig) => apiClient.put('/chef/subscription-config', body),
    onSuccess: () => {
      toast.success('Subscription settings saved');
      queryClient.invalidateQueries({ queryKey: ['chef-subscription-config'] });
    },
    onError: (e: unknown) => {
      const msg = e && typeof e === 'object' && 'error' in e ? (e as { error?: { message?: string } }).error?.message : undefined;
      toast.error(msg || 'Could not save settings');
    },
  });

  if (isLoading) {
    return <div className="flex items-center justify-center py-16 text-muted-foreground"><Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…</div>;
  }

  const toggleIn = (arr: string[], v: string) => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
  const field = 'h-10 w-full rounded-lg border border-input bg-card px-3 text-sm tabular-nums text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring';
  const label = 'mb-1 block text-sm font-medium text-foreground';
  const chip = (active: boolean) =>
    `rounded-full border px-4 py-1.5 text-sm ${active ? 'border-primary bg-primary/10 text-primary' : 'border-input bg-card text-muted-foreground'}`;

  const canEnable = hasMenu && form.perMealPrice > 0;

  return (
    <div className="max-w-2xl space-y-6">
      <div className="page-header">
        <h1 className="page-title">Tiffin Subscriptions</h1>
        <p className="page-description">
          Let customers subscribe to your weekly menu — billed weekly or monthly, delivered daily. We
          place each day’s order automatically at your cutoff.
        </p>
      </div>

      {!hasMenu && (
        <div className="rounded-lg border border-warning/30 bg-warning/5 p-4 text-sm text-foreground">
          Publish your <span className="font-medium">weekly menu</span> first — subscriptions deliver that menu each day.
        </div>
      )}

      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CalendarClock className="h-5 w-5 text-primary" />
            <span className="text-sm font-semibold text-foreground">Offer subscriptions</span>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={form.enabled}
            disabled={!canEnable && !form.enabled}
            onClick={() => setForm({ ...form, enabled: !form.enabled })}
            className={`relative h-6 w-11 rounded-full transition-colors ${form.enabled ? 'bg-primary' : 'bg-muted'} ${!canEnable && !form.enabled ? 'opacity-50' : ''}`}
          >
            <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-card transition-transform ${form.enabled ? 'translate-x-5' : 'translate-x-0.5'}`} />
          </button>
        </div>

        <div className="mt-5 grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label className={label}>Meals offered</label>
            <div className="flex gap-2">
              {['lunch', 'dinner'].map((s) => (
                <button key={s} type="button" className={chip(form.slots.includes(s))} onClick={() => setForm({ ...form, slots: toggleIn(form.slots, s) })}>
                  {s === 'lunch' ? 'Lunch' : 'Dinner'}
                </button>
              ))}
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className={label}>Cadences offered</label>
            <div className="flex gap-2">
              {['weekly', 'monthly'].map((c) => (
                <button key={c} type="button" className={chip(form.cadences.includes(c))} onClick={() => setForm({ ...form, cadences: toggleIn(form.cadences, c) })}>
                  {c === 'weekly' ? 'Weekly' : 'Monthly'}
                </button>
              ))}
            </div>
          </div>
          <div>
            <label className={label}>Price per meal (₹)</label>
            <input className={field} type="number" min={0} value={form.perMealPrice} onChange={(e) => setForm({ ...form, perMealPrice: Number(e.target.value) })} />
          </div>
          <div>
            <label className={label}>Flat delivery fee per cycle (₹)</label>
            <input className={field} type="number" min={0} value={form.deliveryFee} onChange={(e) => setForm({ ...form, deliveryFee: Number(e.target.value) })} />
          </div>
          <div>
            <label className={label}>Daily capacity (0 = unlimited)</label>
            <input className={field} type="number" min={0} value={form.dailyCapacity} onChange={(e) => setForm({ ...form, dailyCapacity: Number(e.target.value) })} />
          </div>
          <div>
            <label className={label}>Order cutoff (IST, HH:MM)</label>
            <input className={field} type="time" value={form.cutoffTime} onChange={(e) => setForm({ ...form, cutoffTime: e.target.value })} />
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between">
          <span className="text-sm font-semibold text-foreground">Paid trial</span>
          <button
            type="button"
            role="switch"
            aria-checked={form.trialEnabled}
            onClick={() => setForm({ ...form, trialEnabled: !form.trialEnabled })}
            className={`relative h-6 w-11 rounded-full transition-colors ${form.trialEnabled ? 'bg-primary' : 'bg-muted'}`}
          >
            <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-card transition-transform ${form.trialEnabled ? 'translate-x-5' : 'translate-x-0.5'}`} />
          </button>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">Let new customers sample your kitchen before subscribing.</p>
        {form.trialEnabled && (
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className={label}>Trial duration (days)</label>
              <input className={field} type="number" min={1} value={form.trialDurationDays} onChange={(e) => setForm({ ...form, trialDurationDays: Number(e.target.value) })} />
            </div>
            <div>
              <label className={label}>Trial price (₹)</label>
              <input className={field} type="number" min={0} value={form.trialPrice} onChange={(e) => setForm({ ...form, trialPrice: Number(e.target.value) })} />
            </div>
          </div>
        )}
      </div>

      <div className="flex justify-end">
        <Button variant="primary" isLoading={save.isPending} onClick={() => save.mutate(form)}>
          Save settings
        </Button>
      </div>
    </div>
  );
}
