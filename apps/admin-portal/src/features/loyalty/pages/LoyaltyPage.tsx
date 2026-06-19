import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Award, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

// Admin loyalty program (#40): tune the earn/redeem economics + streak/tier rules
// (loyalty.* settings) and watch the points-economy analytics. The customer
// endpoints + points engine read the same keys, so changes take effect live.

interface LoyaltyConfig {
  enabled: boolean;
  pointsPerRupee: number;
  redeemRate: number;
  minRedeem: number;
  streakThreshold: number;
  streakBonus: number;
  streakGraceDays: number;
  tierSilverAt: number;
  tierGoldAt: number;
}

interface Analytics {
  members: number;
  outstandingPts: number;
  pointsEarned: number;
  pointsRedeemed: number;
  activeStreaks: number;
  longestStreak: number;
}

export default function LoyaltyPage() {
  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Loyalty &amp; Rewards</h1>
        <p className="page-description">
          Reward repeat customers — points on every delivered order, streak bonuses for daily meal
          subscriptions, redeemable for wallet credit. Tune the economics here; changes are live.
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
    queryKey: ['admin-loyalty-config'],
    queryFn: () => apiClient.get<LoyaltyConfig>('/admin/loyalty/config'),
  });
  const config = data as unknown as LoyaltyConfig | undefined;

  const [form, setForm] = useState<LoyaltyConfig | null>(null);
  useEffect(() => {
    if (config && !form) setForm(config);
  }, [config, form]);

  const save = useMutation({
    mutationFn: (body: Partial<LoyaltyConfig>) => apiClient.put('/admin/loyalty/config', body),
    onSuccess: () => {
      toast.success('Loyalty config updated');
      queryClient.invalidateQueries({ queryKey: ['admin-loyalty-config'] });
    },
    onError: () => toast.error('Could not update the config'),
  });

  if (isLoading || !form) {
    return (
      <div className="flex items-center justify-center rounded-xl border border-border bg-card py-10 text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
      </div>
    );
  }

  const num = (k: keyof LoyaltyConfig) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm({ ...form, [k]: Number(e.target.value) });
  const field =
    'h-9 w-full rounded-lg border border-input bg-card px-3 text-sm tabular-nums text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring';
  const label = 'mb-1 block text-xs font-medium text-muted-foreground';

  // Effective earn vs redeem so the admin can see the give-back at a glance.
  const earnBack =
    form.pointsPerRupee > 0 ? Math.round(form.pointsPerRupee * form.redeemRate * 1000) / 10 : 0;

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Award className="h-5 w-5 text-primary" />
          <h3 className="text-sm font-semibold text-foreground">Program rules</h3>
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
      <p className={`mb-1 text-xs font-medium ${form.enabled ? 'text-success' : 'text-muted-foreground'}`}>
        {form.enabled ? '● Enabled' : '○ Disabled'}
      </p>
      <p className="mb-4 text-xs text-muted-foreground">
        Effective give-back: <span className="font-medium tabular-nums text-foreground">{earnBack}%</span> of order value
        (earn × redeem rate).
      </p>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div>
          <label className={label}>Points per ₹1 spent</label>
          <input className={field} type="number" min={0} step={0.01} value={form.pointsPerRupee} onChange={num('pointsPerRupee')} />
        </div>
        <div>
          <label className={label}>Redeem rate (₹ per point)</label>
          <input className={field} type="number" min={0} step={0.01} value={form.redeemRate} onChange={num('redeemRate')} />
        </div>
        <div>
          <label className={label}>Min points to redeem</label>
          <input className={field} type="number" min={1} value={form.minRedeem} onChange={num('minRedeem')} />
        </div>
        <div>
          <label className={label}>Streak threshold (days)</label>
          <input className={field} type="number" min={1} value={form.streakThreshold} onChange={num('streakThreshold')} />
        </div>
        <div>
          <label className={label}>Streak bonus (points)</label>
          <input className={field} type="number" min={0} value={form.streakBonus} onChange={num('streakBonus')} />
        </div>
        <div>
          <label className={label}>Streak grace (days between deliveries)</label>
          <input className={field} type="number" min={1} value={form.streakGraceDays} onChange={num('streakGraceDays')} />
        </div>
        <div>
          <label className={label}>Silver tier at (lifetime pts)</label>
          <input className={field} type="number" min={0} value={form.tierSilverAt} onChange={num('tierSilverAt')} />
        </div>
        <div>
          <label className={label}>Gold tier at (lifetime pts)</label>
          <input className={field} type="number" min={0} value={form.tierGoldAt} onChange={num('tierGoldAt')} />
        </div>
      </div>
      <div className="mt-5 flex justify-end">
        <Button variant="primary" isLoading={save.isPending} onClick={() => save.mutate(form)}>
          Save config
        </Button>
      </div>
    </div>
  );
}

function AnalyticsCard() {
  const { data, isLoading } = useQuery({
    queryKey: ['admin-loyalty-analytics'],
    queryFn: () => apiClient.get<Analytics>('/admin/loyalty/analytics'),
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

  const fmt = (n: number) => Math.round(n).toLocaleString('en-IN');
  const stats = [
    { label: 'Members', value: fmt(a.members), hint: 'have earned points' },
    { label: 'Points earned', value: fmt(a.pointsEarned), hint: 'all-time' },
    { label: 'Points redeemed', value: fmt(a.pointsRedeemed), hint: 'to wallet credit' },
    { label: 'Outstanding points', value: fmt(a.outstandingPts), hint: 'unredeemed liability' },
    { label: 'Active streaks', value: fmt(a.activeStreaks), hint: 'live runs' },
    { label: 'Longest streak', value: `${fmt(a.longestStreak)} days`, hint: 'best ever' },
  ];

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <h3 className="mb-4 text-sm font-semibold text-foreground">Points economy</h3>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
        {stats.map((s) => (
          <div key={s.label} className="rounded-lg bg-muted/50 p-3">
            <p className="text-xs text-muted-foreground">{s.label}</p>
            <p className="mt-0.5 text-xl font-semibold tabular-nums text-foreground">{s.value}</p>
            {s.hint && <p className="text-[11px] text-muted-foreground">{s.hint}</p>}
          </div>
        ))}
      </div>
    </div>
  );
}
