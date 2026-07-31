import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Button } from '@/shared/components/ui';
import { openRazorpayCheckout } from '@/shared/utils/razorpay';
import { openCashfreeCheckout } from '@/shared/utils/cashfree';
import {
  apiErrorMessage,
  useApproveMealPlan,
  useMealPlan,
  useMealPlanAction,
  useSkipMealPlanDay,
  useVerifyMealPlanPayment,
  type MealPlanDay,
  type MealPlanDayStatus,
} from '@/features/customer/hooks/useMealPlans';

// One meal plan: the accepted days, and whatever decision is currently the
// customer's to make — approve & pay the advance, reject the chef's trim, skip a
// day, or cancel the plan.

const DAY_LABEL: Record<MealPlanDayStatus, string> = {
  requested: 'Requested',
  accepted: 'Chef accepted',
  declined: 'Declined',
  confirmed: 'Scheduled',
  prepared: 'Cooking',
  delivered: 'Delivered',
  skip_req: 'Skip requested',
  skipped: 'Skipped',
  cancelled: 'Cancelled',
  refunded: 'Refunded',
  failed: 'Delivery failed',
};

/** A day is still ahead of the kitchen only in these states. */
const SKIPPABLE: MealPlanDayStatus[] = ['confirmed', 'accepted'];

function dayLabel(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

function sortDays(days: MealPlanDay[]): MealPlanDay[] {
  const slotRank = (s: string) => (s === 'lunch' ? 0 : 1);
  return [...days].sort(
    (a, b) =>
      new Date(a.date).getTime() - new Date(b.date).getTime() || slotRank(a.slot) - slotRank(b.slot),
  );
}

export default function MealPlanDetailPage() {
  const { id } = useParams<{ id: string }>();
  const fp = useFormatPrice();
  const { data: plan, isLoading } = useMealPlan(id);
  const approve = useApproveMealPlan();
  const verify = useVerifyMealPlanPayment();
  const action = useMealPlanAction();
  const skip = useSkipMealPlanDay();
  const [paying, setPaying] = useState(false);

  if (isLoading) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  if (!plan || !id) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-16 text-center">
        <p className="font-medium text-ink">We couldn&apos;t find that meal plan.</p>
        <Button asChild variant="outline" className="mt-4">
          <Link to="/meal-plans">Back to my plans</Link>
        </Button>
      </div>
    );
  }

  const days = sortDays(plan.days ?? []);
  const liveDays = days.filter((d) => d.status !== 'declined');
  const needsApproval = plan.status === 'awaiting_customer';
  const canCancel = ['confirmed', 'active'].includes(plan.status);

  /** Approve → mint the advance → open the gateway sheet → verify. One click. */
  const approveAndPay = async () => {
    setPaying(true);
    try {
      const res = await approve.mutateAsync(id);
      if (res.paymentError) {
        toast.error(res.paymentError);
        return;
      }
      // Cashfree has no key id and hands back no signature, so it settles by a
      // server-side re-fetch: any non-error close calls verify with an empty body.
      // The provider comes from the server — only it knows which rail it minted.
      if (res.provider === 'cashfree' && res.cashfreePaymentSessionId) {
        await openCashfreeCheckout({
          data: {
            cashfreePaymentSessionId: res.cashfreePaymentSessionId,
            cashfreeOrderId: res.cashfreeOrderId ?? '',
            cashfreeEnv: res.cashfreeEnv,
            amount: Math.round((res.mealPlan?.total ?? plan.total) * 100),
            currency: plan.currency ?? 'INR',
          },
          onSettled: async () => {
            await verify.mutateAsync({ id });
            toast.success('Paid — your plan is confirmed.');
          },
          onDismiss: () => setPaying(false),
        });
        return;
      }
      if (!res.razorpayOrderId || !res.razorpayKeyId) {
        // Escrow off — approval alone confirms the plan, nothing to charge.
        toast.success('Plan confirmed.');
        return;
      }
      openRazorpayCheckout({
        data: {
          razorpayOrderId: res.razorpayOrderId,
          razorpayKeyId: res.razorpayKeyId,
          amount: Math.round((res.mealPlan?.total ?? plan.total) * 100),
          currency: plan.currency ?? 'INR',
        },
        description: `Meal plan ${plan.mealPlanNumber}`,
        onVerified: async (resp) => {
          await verify.mutateAsync({
            id,
            razorpayOrderId: resp.razorpay_order_id,
            razorpayPaymentId: resp.razorpay_payment_id,
            razorpaySignature: resp.razorpay_signature,
          });
          toast.success('Paid — your plan is confirmed.');
        },
        onDismiss: () => setPaying(false),
      });
    } catch (err) {
      toast.error(apiErrorMessage(err) || 'Could not start the payment. Please try again.');
    } finally {
      setPaying(false);
    }
  };

  const reject = async () => {
    if (!window.confirm('Reject this plan? The whole booking is cancelled.')) return;
    try {
      await action.mutateAsync({ id, action: 'reject' });
      toast.success('Plan rejected.');
    } catch (err) {
      toast.error(apiErrorMessage(err) || 'Could not reject the plan.');
    }
  };

  const cancel = async () => {
    if (!window.confirm('Cancel this plan? Undelivered days are refunded.')) return;
    try {
      await action.mutateAsync({ id, action: 'cancel' });
      toast.success('Plan cancelled.');
    } catch (err) {
      toast.error(apiErrorMessage(err) || 'Could not cancel the plan.');
    }
  };

  const requestSkip = async (dayId: string) => {
    if (!window.confirm('Ask to skip this day? Your chef decides the refund.')) return;
    try {
      await skip.mutateAsync({ id, dayId });
      toast.success('Skip requested.');
    } catch (err) {
      toast.error(apiErrorMessage(err) || 'Could not request the skip.');
    }
  };

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <Link
        to="/meal-plans"
        className="inline-flex items-center gap-1.5 text-sm text-ink-soft hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        My meal plans
      </Link>

      <p className="mt-4 text-xs uppercase tracking-wide text-ink-muted">{plan.mealPlanNumber}</p>
      <h1 className="text-2xl font-semibold tracking-tight text-ink">
        {plan.chef?.businessName ?? 'Your chef'}
      </h1>

      {needsApproval && (
        <div className="mt-4 rounded-lg bg-herb/10 p-4">
          <p className="font-medium text-herb">Your chef has answered</p>
          <p className="mt-1 text-sm text-ink-soft">
            They can cook {liveDays.length} of {days.length}{' '}
            {days.length === 1 ? 'meal' : 'meals'}. Approve and pay the advance to lock them in, or
            reject to cancel the whole plan.
          </p>
        </div>
      )}

      <ul className="mt-5 divide-y divide-mist rounded-lg border border-mist bg-bone">
        {days.map((d) => {
          const dropped = d.status === 'declined';
          return (
            <li key={d.id} className="flex items-center gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className={`font-medium ${dropped ? 'text-ink-muted line-through' : 'text-ink'}`}>
                  {dayLabel(d.date)}
                </p>
                <p className="text-sm text-ink-soft">
                  <span className="capitalize">{d.slot}</span>
                  {d.dishName ? ` · ${d.dishName}` : ''}
                </p>
                <span className="mt-1 inline-block rounded-full bg-mist px-2 py-0.5 text-xs text-ink-soft">
                  {DAY_LABEL[d.status] ?? d.status}
                </span>
              </div>
              <div className="flex flex-col items-end gap-1">
                <span className={`tabular-nums ${dropped ? 'text-ink-muted' : 'text-ink'}`}>
                  {fp(d.price)}
                </span>
                {plan.status === 'active' && SKIPPABLE.includes(d.status) && (
                  <button
                    type="button"
                    onClick={() => void requestSkip(d.id)}
                    disabled={skip.isPending}
                    className="text-sm font-medium text-herb hover:underline disabled:opacity-50"
                  >
                    Skip
                  </button>
                )}
              </div>
            </li>
          );
        })}
      </ul>

      <div className="mt-5 flex items-center justify-between border-t border-mist pt-4">
        <span className="text-sm text-ink-soft">{needsApproval ? 'If approved' : 'Total'}</span>
        <span className="text-lg font-semibold tabular-nums text-ink">{fp(plan.total)}</span>
      </div>
      {plan.tax > 0 && (
        <p className="mt-1 text-xs text-ink-muted">
          Includes GST and per-day delivery on top of {fp(plan.subtotal)} of food.
        </p>
      )}

      {needsApproval && (
        <div className="mt-6 flex gap-3">
          <Button variant="outline" className="flex-1" onClick={() => void reject()} disabled={action.isPending}>
            Reject
          </Button>
          <Button
            className="flex-1"
            onClick={() => void approveAndPay()}
            disabled={paying || approve.isPending}
          >
            {paying || approve.isPending ? 'Starting payment…' : 'Approve & pay'}
          </Button>
        </div>
      )}

      {canCancel && (
        <Button variant="outline" className="mt-6 w-full" onClick={() => void cancel()} disabled={action.isPending}>
          Cancel plan
        </Button>
      )}
    </div>
  );
}
