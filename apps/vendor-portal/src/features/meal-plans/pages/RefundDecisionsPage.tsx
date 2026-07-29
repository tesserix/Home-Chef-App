import { useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { formatCurrency } from '@/shared/utils/format';
import {
  refundAtPercent,
  useRefundDecisions,
  useSubmitRefundDecision,
  type RefundDecisionDay,
} from '../hooks/useRefundDecisions';

// Late skip/cancel refund decisions — the web twin of
// apps/mobile-vendor/app/meal-plans/refund-decisions.tsx.
//
// Refund policy v3 (#834): the chef sets the amount rather than picking Full/Half/None,
// bounded below by the day's lead-time floor. The nearer the meal, the lower the floor —
// the chef has more prep to be compensated for.

export function RefundDecisionsPage() {
  const { data: days = [], isLoading } = useRefundDecisions();

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Refund requests</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Late skips and cancellations — you choose how much to refund, down to the minimum for how
        much notice you were given. Requests made well ahead of cooking are refunded in full
        automatically and never reach this list.
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
  // Open at the floor: the chef's most protective option, and the one they are most
  // likely to want when they have already started cooking.
  const [percent, setPercent] = useState(day.minPercent);
  const [pending, setPending] = useState<'refund' | 'decline' | null>(null);

  function decide(decline: boolean) {
    setPending(decline ? 'decline' : 'refund');
    submit.mutate(
      { dayId: day.dayId, percent, decline },
      {
        onSuccess: () =>
          toast.success(
            decline
              ? 'Request declined — the day stands and nothing is refunded.'
              : `Refunding ${percent}% — ${formatCurrency(refundAtPercent(day, percent))}.`,
          ),
        onError: () => toast.error('Could not record that decision. Please try again.'),
        onSettled: () => setPending(null),
      },
    );
  }

  const busy = submit.isPending;
  // Presets inside the permitted range, so the common answers stay one click away.
  const presets = [day.minPercent, 75, 100].filter(
    (p, i, all) => p >= day.minPercent && all.indexOf(p) === i,
  );

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

      <div className="mt-4 rounded-lg bg-mist/40 p-4">
        <div className="flex items-baseline justify-between">
          <label htmlFor={`pct-${day.dayId}`} className="text-sm text-ink-soft">
            Refund to the customer
          </label>
          <p className="text-lg font-semibold text-ink tabular-nums">
            {formatCurrency(refundAtPercent(day, percent))}
            <span className="ml-1 text-sm font-normal text-ink-muted">({percent}%)</span>
          </p>
        </div>
        <input
          id={`pct-${day.dayId}`}
          type="range"
          className="mt-3 w-full accent-primary"
          min={day.minPercent}
          max={100}
          step={5}
          value={percent}
          disabled={busy}
          onChange={(e) => setPercent(Number(e.target.value))}
          aria-valuetext={`${percent} percent, ${formatCurrency(refundAtPercent(day, percent))}`}
        />
        <div className="mt-2 flex flex-wrap items-center gap-2">
          {presets.map((p) => (
            <Button
              key={p}
              size="sm"
              variant={percent === p ? 'secondary' : 'outline'}
              disabled={busy}
              onClick={() => setPercent(p)}
            >
              {p}%
            </Button>
          ))}
        </div>
        <p className="mt-2 text-xs text-ink-muted">
          {day.minPercent > 0
            ? `At this much notice the minimum is ${day.minPercent}% (${formatCurrency(day.minRefund)}). Amounts are the customer's food, tax and delivery, net of the platform's service charge.`
            : 'The meal is imminent, so no refund is owed — but you can still give one.'}
        </p>
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        <Button size="sm" disabled={busy} onClick={() => decide(false)}>
          {pending === 'refund' ? 'Saving…' : `Refund ${formatCurrency(refundAtPercent(day, percent))}`}
        </Button>
        <Button size="sm" variant="ghost" disabled={busy} onClick={() => decide(true)}>
          {pending === 'decline' ? 'Saving…' : 'Decline request'}
        </Button>
      </div>
      <p className="mt-2 text-xs text-ink-muted">
        &ldquo;Decline&rdquo; rejects the request outright — you cook the day as planned and the
        customer is charged in full.
      </p>
    </Card>
  );
}

export default RefundDecisionsPage;
