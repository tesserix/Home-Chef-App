import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Gift, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

// Admin win-back program (#42): tune the auto-offer policy (winback.* settings)
// and watch the reactivation analytics.

interface WinbackConfig {
  enabled: boolean;
  discountPercent: number;
  maxDiscount: number;
  validityDays: number;
  lapseThresholdDays: number;
  cooldownDays: number;
}

interface Analytics {
  total: number;
  offered: number;
  reactivated: number;
  expired: number;
  reactivationRate: number;
  byTrigger: { trigger: string; total: number; reactivated: number }[];
}

const TRIGGER_LABEL: Record<string, string> = {
  lapsed: 'Customer lapsed',
  subscription_cancelled: 'Subscription cancelled',
  subscription_suspended: 'Subscription suspended',
};

export default function WinbackPage() {
  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Win-back Offers</h1>
        <p className="page-description">
          Auto-offer a discounted promo when a customer lapses or a subscriber cancels/suspends —
          delivered by push + email — to protect subscription &amp; order LTV.
        </p>
      </div>
      <ConfigCard />
      <AnalyticsCard />
    </div>
  );
}

function ConfigCard() {
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ['admin-winback-config'],
    queryFn: () => apiClient.get<WinbackConfig>('/admin/winback/config'),
  });
  const config = data as unknown as WinbackConfig | undefined;

  const [form, setForm] = useState<WinbackConfig | null>(null);
  useEffect(() => {
    if (config && !form) setForm(config);
  }, [config, form]);

  const save = useMutation({
    mutationFn: (body: Partial<WinbackConfig>) => apiClient.put('/admin/winback/config', body),
    onSuccess: () => {
      toast.success('Win-back policy updated');
      queryClient.invalidateQueries({ queryKey: ['admin-winback-config'] });
    },
    onError: () => toast.error('Could not update the policy'),
  });

  if (isLoading || !form) {
    return (
      <div className="flex items-center justify-center rounded-xl border border-border bg-card py-10 text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
      </div>
    );
  }

  const num = (k: keyof WinbackConfig) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm({ ...form, [k]: Number(e.target.value) });
  const field =
    'h-9 w-full rounded-lg border border-input bg-card px-3 text-sm tabular-nums text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring';
  const label = 'mb-1 block text-xs font-medium text-muted-foreground';

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Gift className="h-5 w-5 text-primary" />
          <h3 className="text-sm font-semibold text-foreground">Offer policy</h3>
        </div>
        <Button
          variant="ghost"
          size="sm"
          isLoading={save.isPending}
          onClick={() => save.mutate({ enabled: !form.enabled })}
        >
          {form.enabled ? 'Disable program' : 'Enable program'}
        </Button>
      </div>
      <p className={`mb-4 text-xs font-medium ${form.enabled ? 'text-success' : 'text-muted-foreground'}`}>
        {form.enabled ? '● Enabled' : '○ Disabled'}
      </p>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div>
          <label className={label}>Discount %</label>
          <input className={field} type="number" min={1} max={100} value={form.discountPercent} onChange={num('discountPercent')} />
        </div>
        <div>
          <label className={label}>Max discount (₹, 0 = none)</label>
          <input className={field} type="number" min={0} value={form.maxDiscount} onChange={num('maxDiscount')} />
        </div>
        <div>
          <label className={label}>Code validity (days)</label>
          <input className={field} type="number" min={1} value={form.validityDays} onChange={num('validityDays')} />
        </div>
        <div>
          <label className={label}>Lapse threshold (days)</label>
          <input className={field} type="number" min={1} value={form.lapseThresholdDays} onChange={num('lapseThresholdDays')} />
        </div>
        <div>
          <label className={label}>Re-offer cooldown (days)</label>
          <input className={field} type="number" min={1} value={form.cooldownDays} onChange={num('cooldownDays')} />
        </div>
      </div>
      <div className="mt-5 flex justify-end">
        <Button variant="primary" isLoading={save.isPending} onClick={() => save.mutate(form)}>
          Save policy
        </Button>
      </div>
    </div>
  );
}

function AnalyticsCard() {
  const { data, isLoading } = useQuery({
    queryKey: ['admin-winback-analytics'],
    queryFn: () => apiClient.get<Analytics>('/admin/winback/analytics'),
    refetchInterval: 60_000,
  });
  const a = data as unknown as Analytics | undefined;

  if (isLoading || !a) {
    return (
      <div className="flex items-center justify-center rounded-xl border border-border bg-card py-10 text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading analytics…
      </div>
    );
  }

  const stats = [
    { label: 'Offers sent', value: String(a.total) },
    { label: 'Reactivated', value: String(a.reactivated) },
    { label: 'Expired', value: String(a.expired) },
    { label: 'Reactivation rate', value: `${a.reactivationRate}%`, hint: 'of resolved offers' },
  ];

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <h3 className="mb-4 text-sm font-semibold text-foreground">Reactivation</h3>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {stats.map((s) => (
          <div key={s.label} className="rounded-lg bg-muted/50 p-3">
            <p className="text-xs text-muted-foreground">{s.label}</p>
            <p className="mt-0.5 text-xl font-semibold tabular-nums text-foreground">{s.value}</p>
            {s.hint && <p className="text-[11px] text-muted-foreground">{s.hint}</p>}
          </div>
        ))}
      </div>

      {a.byTrigger.length > 0 && (
        <div className="mt-5">
          <p className="mb-2 text-xs font-medium text-muted-foreground">By trigger</p>
          <div className="overflow-hidden rounded-lg border border-border">
            {a.byTrigger.map((t, i) => {
              const rate = t.total > 0 ? Math.round((t.reactivated / t.total) * 100) : 0;
              return (
                <div
                  key={t.trigger}
                  className={`flex items-center justify-between px-3 py-2 text-sm ${i > 0 ? 'border-t border-border' : ''}`}
                >
                  <span className="text-foreground">{TRIGGER_LABEL[t.trigger] ?? t.trigger}</span>
                  <span className="tabular-nums text-muted-foreground">
                    {t.reactivated}/{t.total} reactivated · {rate}%
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
