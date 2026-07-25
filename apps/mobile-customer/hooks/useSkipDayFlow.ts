
import { useSkipMealPlanDay, useChooseRefundMedium } from './useMealPlans';
import { useAlert } from '@homechef/mobile-shared/ui';

// Shared skip + refund-medium flow (v2, docs/meal-plan-refund-flow-design.md), so the plan-detail
// screen and the "My plan" sheet behave identically. A skip >12h before cooking agrees a full
// refund immediately, then — per RBI — the CUSTOMER picks the medium (wallet instant vs original
// 5–7 days); a skip ≤12h goes to the chef to decide the amount first (customer picks the medium
// later, from the notification). The refund covers the food + that day's delivery, excluding GST + the platform fee.
export function useSkipDayFlow(planId: string | undefined) {
  const { showAlert } = useAlert();
  const skipDay = useSkipMealPlanDay();
  const chooseMedium = useChooseRefundMedium();

  // promptMedium: the RBI medium choice for a refund that's agreed and awaiting the customer.
  // Reusable from a "choose refund" action on any pending_customer day.
  function promptMedium(dayId: string) {
    if (!planId) return;
    showAlert(
      'Where would you like your refund?',
      'HomeChef Wallet is instant — use it on your next order. Your original payment method takes ~5–7 business days (per RBI).',
      [
        {
          text: 'HomeChef Wallet (instant)',
          onPress: () =>
            chooseMedium.mutate(
              { planId, dayId, medium: 'wallet' },
              {
                onSuccess: (r) => showAlert('Done', r?.message ?? 'Refunded to your wallet — ready to use.'),
                onError: () => showAlert('Something went wrong', 'Please try again.'),
              },
            ),
        },
        {
          text: 'Original method (5–7 days)',
          onPress: () =>
            chooseMedium.mutate(
              { planId, dayId, medium: 'source' },
              {
                onSuccess: (r) =>
                  showAlert('On its way', r?.message ?? 'We’ll refund your original payment method in 5–7 business days.'),
                onError: () => showAlert('Something went wrong', 'Please try again.'),
              },
            ),
        },
      ],
    );
  }

  function confirmSkip(dayId: string) {
    if (!planId) return;
    showAlert(
      'Skip this day?',
      'More than 12 hours before your meal? You choose your refund right away. Closer than that, your chef reviews it (they may have started cooking). The refund covers the food and that day’s delivery fee — the GST and platform fee aren’t refunded. This can’t be undone.',
      [
        { text: 'Back', style: 'cancel' },
        {
          text: 'Request skip',
          style: 'destructive',
          onPress: () =>
            skipDay.mutate(
              { planId, dayId },
              {
                onSuccess: (res) => {
                  const r = res as { status?: string } | undefined;
                  if (r?.status === 'pending_customer') {
                    // Agreed (>12h) — the customer now picks the medium.
                    promptMedium(dayId);
                  } else {
                    // Within 12h — the chef decides the amount first.
                    showAlert(
                      'Skip requested',
                      'Your chef will review this (they may have started cooking). We’ll notify you when your refund is ready to choose.',
                    );
                  }
                },
                onError: () =>
                  showAlert(
                    'Could not request skip',
                    'It may be too close to when your chef starts cooking this day.',
                  ),
              },
            ),
        },
      ],
    );
  }

  return { confirmSkip, promptMedium, skipping: skipDay.isPending || chooseMedium.isPending };
}
