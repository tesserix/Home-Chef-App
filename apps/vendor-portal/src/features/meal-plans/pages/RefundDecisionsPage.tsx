import { useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { formatCurrency } from '@/shared/utils/format';
import {
  useRefundDecisions,
  useSubmitRefundDecision,
  type RefundChoice,
  type RefundDecisionDay,
} from '../hooks/useRefundDecisions';

// Late skip/cancel refund decisions — the web twin of
// apps/mobile-vendor/app/meal-plans/refund-decisions.tsx.
//
// A customer skipped or cancelled a tiffin day with ≤12h notice, by which point
// the chef has already bought the ingredients. Rather than auto-refunding, the
// chef decides how much to give back. Over 12h it auto-agrees a full refund and
// never appears here.

export function RefundDecisionsPage() {
  const { data: days = [], isLoading } = useRefundDecisions();

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Refund requests</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Late skips and cancellations — you decide how much to refund. Requests made more than 12
        hours ahead are refunded in full automatically and never reach this list.
      </p>

      {isLoading ? (
        <Skeleton className="mt-8 h-32 w-full" />
      ) : days.length === 0 ? (
        <Card className="mt-8 p-6 text-center">
          <p className="font-semibold text-foreground">Nothing waiting on you</p>
          <p className="mt-1 text-sm text-ink-soft">
            When a customer skips or cancels a day at short notice, it appears here for your
            decision.
          </p>
        </Card>
      ) : (
        <div className="mt-6 flex flex-col gap-4">
          {days.map((d) => (
            <DecisionCard key={d.dayId} day={d} />
          ))}
        </div>
      )}
    </div>
  );
}

function DecisionCard({ day }: { day: RefundDecisionDay }) {
  const submit = useSubmitRefundDecision();
  const [pending, setPending] = useState<RefundChoice | 'decline' | null>(null);

  function decide(choice: RefundChoice, decline = false) {
    setPending(decline ? 'decline' : choice);
    submit.mutate(
      { dayId: day.dayId, choice, decline },
      {
        onSuccess: () =>
          toast.success(
            decline
              ? 'Request declined — the day stands and nothing is refunded.'
              : 'Refund decision recorded.',
          ),
        onError: () => toast.error('Could not record that decision. Please try again.'),
        onSettled: () => setPending(null),
      },
    );
  }

  const busy = submit.isPending;

  return (
    <Card className="p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-semibold text-foreground">{day.dishName}</p>
          <p className="text-sm text-ink-soft">
            {day.customerName} · {day.mealPlanNumber}
          </p>
          <p className="text-xs text-ink-muted tabular-nums">
            {format(new Date(day.date), 'EEE d MMM')} · {day.slot}
          </p>
        </div>
        <p className="shrink-0 text-sm font-semibold text-ink tabular-nums">
          {formatCurrency(day.foodPrice)}
        </p>
      </div>

      {/* Every amount is the SERVER's figure. The fee, GST and delivery slices
          are excluded server-side, so halving foodPrice here would quietly
          disagree with what actually gets paid out. */}
      <div className="mt-4 flex flex-wrap gap-2">
        <Button size="sm" disabled={busy} onClick={() => decide('full')}>
          {pending === 'full' ? 'Saving…' : `Full · ${formatCurrency(day.fullRefund)}`}
        </Button>
        <Button size="sm" variant="secondary" disabled={busy} onClick={() => decide('half')}>
          {pending === 'half' ? 'Saving…' : `Half · ${formatCurrency(day.halfRefund)}`}
        </Button>
        <Button size="sm" variant="outline" disabled={busy} onClick={() => decide('none')}>
          {pending === 'none' ? 'Saving…' : 'No refund'}
        </Button>
        <Button size="sm" variant="ghost" disabled={busy} onClick={() => decide('none', true)}>
          {pending === 'decline' ? 'Saving…' : 'Decline request'}
        </Button>
      </div>
      <p className="mt-2 text-xs text-ink-muted">
        &ldquo;No refund&rdquo; keeps the payment and serves the day as planned. &ldquo;Decline&rdquo;
        rejects the request outright.
      </p>
    </Card>
  );
}

export default RefundDecisionsPage;
