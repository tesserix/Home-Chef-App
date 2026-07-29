import { useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Inbox } from 'lucide-react';
import { toast } from 'sonner';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/shared/components/ui/Dialog';
import { formatCurrency } from '@/shared/utils/format';
import { usePlanRequests, useRespondToPlanRequest, type PlanRequestDay } from '../hooks/usePlanRequests';

// Review one pending tiffin plan request — the web twin of the mobile review
// screen. The chef leaves every day on to accept the plan whole, or toggles off
// the days they cannot cook and sends back the trimmed set for the customer to
// approve and pay.

function dayLabel(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

/** Sorted so the chef reads the week in the order they will cook it. */
function sortDays(days: PlanRequestDay[]): PlanRequestDay[] {
  const slotRank = (s: string) => (s === 'lunch' ? 0 : 1);
  return [...days].sort(
    (a, b) =>
      new Date(a.date).getTime() - new Date(b.date).getTime() || slotRank(a.slot) - slotRank(b.slot),
  );
}

export function PlanRequestPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: plans = [], isLoading } = usePlanRequests();
  const respond = useRespondToPlanRequest();

  const plan = plans.find((p) => p.id === id);
  const days = useMemo(() => sortDays(plan?.days ?? []), [plan]);

  // Everything starts on: accepting the whole plan is the common case, and the
  // chef opts days OUT rather than having to opt each one in.
  const [declined, setDeclined] = useState<Record<string, boolean>>({});
  const [confirming, setConfirming] = useState(false);

  const acceptedDays = days.filter((d) => !declined[d.id]);
  const acceptAll = acceptedDays.length === days.length;
  const acceptedTotal = acceptedDays.reduce((sum, d) => sum + d.price, 0);

  if (isLoading) {
    return <p className="mx-auto max-w-2xl text-sm text-ink-soft">Loading request…</p>;
  }

  if (!plan) {
    return (
      <div className="mx-auto max-w-2xl">
        <Card className="flex flex-col items-center gap-3 p-10 text-center">
          <Inbox className="h-8 w-8 text-ink-muted" aria-hidden="true" />
          <p className="font-medium text-foreground">This request is no longer available</p>
          <p className="text-sm text-ink-soft">
            It may have been answered already, or expired while waiting for a reply.
          </p>
          <Button asChild variant="outline">
            <Link to="/tiffin-plans">Back to tiffin plans</Link>
          </Button>
        </Card>
      </div>
    );
  }

  const alreadyAnswered = plan.status !== 'pending_chef';

  const submit = async () => {
    setConfirming(false);
    try {
      await respond.mutateAsync({
        planId: plan.id,
        acceptAll,
        acceptedDayIds: acceptedDays.map((d) => d.id),
      });
      toast.success(
        acceptAll
          ? 'Plan accepted — the customer pays the advance to confirm it.'
          : `${acceptedDays.length} of ${days.length} meals accepted — sent back for approval.`,
      );
      navigate('/tiffin-plans');
    } catch {
      toast.error('Could not send your response. Please try again.');
    }
  };

  const customerName =
    [plan.customer?.firstName, plan.customer?.lastName].filter(Boolean).join(' ') || 'Customer';

  return (
    <div className="mx-auto max-w-2xl pb-28">
      <Link
        to="/tiffin-plans"
        className="inline-flex items-center gap-1.5 text-sm text-ink-soft hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Tiffin plans
      </Link>

      <p className="mt-4 text-xs uppercase tracking-wide text-ink-muted">{plan.mealPlanNumber}</p>
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">
        {days.length} {days.length === 1 ? 'meal' : 'meals'} requested
      </h1>
      <p className="mt-1 text-sm text-ink-soft">
        {customerName} · {dayLabel(plan.startDate)} – {dayLabel(plan.endDate)}
      </p>
      <p className="mt-3 text-sm text-ink-soft">
        Turn off any meal you can&apos;t cook. Leave them all on to accept the whole plan.
      </p>

      {alreadyAnswered && (
        <Card className="mt-4 border-amber-500/40 bg-amber-500/10 p-4">
          <p className="text-sm text-foreground">
            You&apos;ve already responded to this request — it&apos;s with the customer now.
          </p>
        </Card>
      )}

      <Card className="mt-5 divide-y divide-border p-0">
        {days.map((d) => {
          const off = !!declined[d.id];
          return (
            <div key={d.id} className="flex items-center gap-4 p-4">
              <div className="min-w-0 flex-1">
                <p className={`font-medium ${off ? 'text-ink-muted line-through' : 'text-foreground'}`}>
                  {dayLabel(d.date)}
                </p>
                <p className="text-sm text-ink-soft">
                  <span className="capitalize">{d.slot}</span>
                  {d.dishName ? ` · ${d.dishName}` : ''}
                  {' · '}
                  <span className={d.variant === 'veg' ? 'text-emerald-600' : 'text-destructive'}>
                    {d.variant === 'veg' ? 'Veg' : 'Non-veg'}
                  </span>
                </p>
              </div>
              <span className={`tabular-nums ${off ? 'text-ink-muted' : 'text-foreground'}`}>
                {formatCurrency(d.price)}
              </span>
              <label className="flex cursor-pointer items-center">
                <span className="sr-only">
                  {off ? 'Accept' : 'Decline'} {d.slot} on {dayLabel(d.date)}
                </span>
                <input
                  type="checkbox"
                  className="h-5 w-5 accent-primary"
                  checked={!off}
                  disabled={alreadyAnswered}
                  onChange={(e) =>
                    setDeclined((prev) => ({ ...prev, [d.id]: !e.target.checked }))
                  }
                />
              </label>
            </div>
          );
        })}
      </Card>

      <div className="mt-6 flex items-center justify-between">
        <span className="text-sm text-ink-soft">
          {acceptedDays.length} of {days.length} accepted
        </span>
        <span className="text-lg font-semibold tabular-nums text-foreground">
          {formatCurrency(acceptedTotal)}
        </span>
      </div>

      <Button
        className="mt-4 w-full"
        size="lg"
        disabled={alreadyAnswered || acceptedDays.length === 0 || respond.isPending}
        onClick={() => setConfirming(true)}
      >
        {acceptedDays.length === 0
          ? 'Accept at least one meal'
          : acceptAll
            ? 'Accept all meals'
            : `Accept ${acceptedDays.length} of ${days.length}`}
      </Button>

      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Confirm response</AlertDialogTitle>
            <AlertDialogDescription>
              {acceptAll
                ? `Accept all ${days.length} meals? The customer still needs to approve & pay before it's confirmed.`
                : `Accept ${acceptedDays.length} of ${days.length} meals? The rest are declined and refunded, and the customer approves the trimmed plan.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={submit}>Confirm</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

/** Compact pending-request row for the Tiffin plans hub. */
export function PlanRequestRow({ plan }: { plan: { id: string; mealPlanNumber: string; total: number; startDate: string; endDate: string; days?: PlanRequestDay[]; customer?: { firstName?: string; lastName?: string } } }) {
  const count = plan.days?.length ?? 0;
  const name =
    [plan.customer?.firstName, plan.customer?.lastName].filter(Boolean).join(' ') || 'Customer';
  return (
    <Link to={`/tiffin-plans/requests/${plan.id}`} className="block">
      <Card className="p-4 transition-colors hover:bg-paper">
        <div className="flex items-baseline justify-between gap-3">
          <span className="font-medium text-foreground">{name}</span>
          <span className="tabular-nums font-semibold text-foreground">
            {formatCurrency(plan.total)}
          </span>
        </div>
        <p className="mt-1 text-sm text-ink-soft">
          {dayLabel(plan.startDate)} – {dayLabel(plan.endDate)} · {count}{' '}
          {count === 1 ? 'meal' : 'meals'}
        </p>
        <div className="mt-2 flex items-center justify-between">
          <Badge variant="warning">Needs your answer</Badge>
          <span className="text-sm font-medium text-primary">Review</span>
        </div>
      </Card>
    </Link>
  );
}

export default PlanRequestPage;
