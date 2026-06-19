import { toast } from 'sonner';
import { Loader2 } from 'lucide-react';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Button } from '@/shared/components/ui';
import {
  useMealSubscriptions,
  useMealSubAction,
  type MealSubscription,
} from '@/features/customer/hooks/useMealSubscription';

// My tiffin subscriptions (#283, web): list + manage (pause / resume / cancel).

const STATUS_LABEL: Record<MealSubscription['status'], string> = {
  trialing: 'Trial',
  active: 'Active',
  paused: 'Paused',
  past_due: 'Payment due',
  cancelled: 'Cancelled',
};

export default function SubscriptionsPage() {
  const fp = useFormatPrice();
  const { data, isLoading } = useMealSubscriptions();
  const action = useMealSubAction();
  const subs = (data as unknown as { data: MealSubscription[] } | undefined)?.data ?? [];

  const run = (sub: MealSubscription, a: 'pause' | 'resume' | 'cancel') => {
    if (a === 'cancel' && !window.confirm('Cancel this subscription? You can resubscribe anytime.')) return;
    action.mutate(
      { id: sub.id, action: a },
      {
        onSuccess: () => toast.success(a === 'cancel' ? 'Subscription cancelled' : a === 'pause' ? 'Paused' : 'Resumed'),
        onError: () => toast.error('Could not update the subscription'),
      },
    );
  };

  if (isLoading) {
    return <div className="flex min-h-[50vh] items-center justify-center"><Loader2 className="h-8 w-8 animate-spin text-herb" /></div>;
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-10">
      <h1 className="font-display text-2xl font-semibold text-ink">My subscriptions</h1>
      {subs.length === 0 ? (
        <p className="mt-6 text-ink-soft">No tiffin subscriptions yet.</p>
      ) : (
        <div className="mt-6 space-y-3">
          {subs.map((s) => {
            const active = s.status === 'active';
            const paused = s.status === 'paused';
            const terminal = s.status === 'cancelled';
            return (
              <div key={s.id} className="rounded-xl bg-bone p-5">
                <div className="flex items-center justify-between">
                  <p className="font-medium text-ink">
                    {s.slots.map((x) => (x === 'lunch' ? 'Lunch' : 'Dinner')).join(' + ')} · {s.variant === 'veg' ? 'Veg' : 'Non-veg'}
                  </p>
                  <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${active ? 'bg-herb-tint text-herb' : 'bg-mist text-ink-soft'}`}>
                    {STATUS_LABEL[s.status]}
                  </span>
                </div>
                <p className="mt-1 text-sm text-ink-soft">
                  {s.days.length} days/week · {s.cadence === 'monthly' ? 'Monthly' : 'Weekly'} · {fp(s.cycleAmount)}
                </p>
                {s.creditBalance > 0 && (
                  <p className="mt-1 text-xs font-medium text-herb">{fp(s.creditBalance)} credit applies to your next cycle</p>
                )}
                {!terminal && (
                  <div className="mt-3 flex gap-2">
                    {active && <Button variant="outline" size="sm" onClick={() => run(s, 'pause')}>Pause</Button>}
                    {paused && <Button variant="outline" size="sm" onClick={() => run(s, 'resume')}>Resume</Button>}
                    <Button variant="ghost" size="sm" onClick={() => run(s, 'cancel')}>Cancel</Button>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
