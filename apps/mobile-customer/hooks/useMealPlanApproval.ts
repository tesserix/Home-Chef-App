import { Alert } from 'react-native';
import { router } from 'expo-router';

import {
  mealPlanAdvanceBreakdown,
  useFinalizeMealPlan,
  type MealPlan,
} from './useMealPlans';
import { isDeclinedDayStatus } from '../lib/meal-plan';

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
  /** Number of days the chef can cook (declined days excluded) — for button copy. */
  acceptedCount: number;
}

export function useMealPlanApproval(
  plan: MealPlan | undefined,
  opts?: { onDone?: () => void },
): MealPlanApproval {
  const finalize = useFinalizeMealPlan();
  const acceptedCount = (plan?.days ?? []).filter(
    (d) => !isDeclinedDayStatus(d.status),
  ).length;

  function run(approve: boolean) {
    if (!plan) return;
    Alert.alert(
      approve ? 'Approve & pay?' : 'Reject plan?',
      approve
        ? `Confirm the ${acceptedCount} day${acceptedCount === 1 ? '' : 's'} your chef can cook, then pay the advance (food + GST + delivery, shown at checkout) to lock them in.`
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
                    Alert.alert('Payment unavailable', res.paymentError);
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
                  Alert.alert(
                    approve ? 'Plan confirmed' : 'Plan cancelled',
                    approve
                      ? 'Your chef has been notified.'
                      : 'No charge — the plan was cancelled.',
                    [{ text: 'OK', onPress: () => opts?.onDone?.() }],
                  );
                },
                onError: () => Alert.alert('Something went wrong', 'Please try again.'),
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
