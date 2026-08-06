
import { router } from 'expo-router';

import {
  mealPlanAdvanceBreakdown,
  useFinalizeMealPlan,
  type MealPlan,
} from './useMealPlans';
import { isDeclinedDayStatus } from '../lib/meal-plan';
import { useAlert } from '@homechef/mobile-shared/ui';

// useMealPlanApproval — the ONE place the "approve & pay" / "reject the whole plan"
// flow lives, so the plan-detail screen, the Home card, and the chef-page sheet all
// behave identically (payment-after-approval; reject cancels the whole plan). Approve
// (escrow on) mints a Razorpay advance order for the accepted days and launches
// checkout; reject cancels the plan outright. `onDone` runs after a non-checkout
// outcome (reject, or escrow-off confirm) so each caller can close/pop as it likes.
export interface MealPlanApproval {
  approve: () => void;
  reject: () => void;
  isPending: boolean;
  /** Number of meals the chef can cook (declined ones excluded) — for button copy. */
  acceptedCount: number;
}

export function useMealPlanApproval(
  plan: MealPlan | undefined,
  opts?: { onDone?: () => void },
): MealPlanApproval {
  const { showAlert } = useAlert();
  const finalize = useFinalizeMealPlan();
  const acceptedCount = (plan?.days ?? []).filter(
    (d) => !isDeclinedDayStatus(d.status),
  ).length;

  function run(approve: boolean) {
    if (!plan) return;
    showAlert(
      approve ? 'Approve & pay?' : 'Reject plan?',
      approve
        ? // "meal", not "day" (#1040) — plan.days holds booked meals, so a plan
          // over 3 dates with lunch + dinner is 6 of these, not 3.
          `Confirm the ${acceptedCount} meal${acceptedCount === 1 ? '' : 's'} your chef can cook, then pay the advance (food + GST + delivery, shown at checkout) to lock them in.`
        : 'This cancels the whole plan. You can book again any time.',
      [
        { text: 'Back', style: 'cancel' },
        {
          text: approve ? 'Approve' : 'Reject',
          style: approve ? 'default' : 'destructive',
          onPress: () =>
            finalize.mutate(
              { id: plan.id, approve },
              {
                onSuccess: (res) => {
                  // Approve (escrow on): the server minted a Razorpay advance order
                  // for the accepted days — launch checkout. Payment happens here,
                  // after approval. verify-payment then confirms + holds.
                  if (approve && res?.paymentError) {
                    showAlert('Payment unavailable', res.paymentError);
                    return;
                  }
                  // Cashfree opens its own sheet (a WebView, not the Razorpay
                  // native one) and has no key id or client signature, so it gets
                  // its own screen. The provider comes from the server — never
                  // guessed here, since only the server knows which rail it minted.
                  if (approve && res?.provider === 'cashfree') {
                    router.push({
                      pathname: '/payment/cashfree',
                      params: {
                        kind: 'mealplan',
                        mealPlanId: plan.id,
                        orderId: plan.id,
                        paymentSessionId: res.cashfreePaymentSessionId ?? '',
                        cashfreeOrderId: res.cashfreeOrderId ?? '',
                        env: res.cashfreeEnv ?? '',
                      },
                    });
                    return;
                  }
                  if (approve && res?.razorpayOrderId) {
                    const b = mealPlanAdvanceBreakdown(res.mealPlan);
                    router.push({
                      pathname: '/payment/checkout',
                      params: {
                        kind: 'mealplan',
                        mealPlanId: plan.id,
                        razorpayOrderId: res.razorpayOrderId,
                        razorpayKeyId: res.razorpayKeyId ?? '',
                        amount: String(b.amountPaise),
                        currency: res.mealPlan.currency ?? 'INR',
                      },
                    });
                    return;
                  }
                  // Reject, or escrow-off approve (unpaid handshake → confirmed).
                  showAlert(
                    approve ? 'Plan confirmed' : 'Plan cancelled',
                    approve
                      ? 'Your chef has been notified.'
                      : 'No charge — the plan was cancelled.',
                    [{ text: 'OK', onPress: () => opts?.onDone?.() }],
                  );
                },
                onError: () => showAlert('Something went wrong', 'Please try again.'),
              },
            ),
        },
      ],
    );
  }

  return {
    approve: () => run(true),
    reject: () => run(false),
    isPending: finalize.isPending,
    acceptedCount,
  };
}
