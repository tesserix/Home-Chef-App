
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
      'Well ahead of your meal, you get everything back for that day — food, taxes and delivery — and choose where it goes right away. Closer in, your chef reviews it (they may have started cooking) and there’s a minimum they must refund based on how much notice you gave. This can’t be undone.',
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
                  const r = res as { status?: string; minRefundPercent?: number } | undefined;
                  if (r?.status === 'pending_customer') {
                    // Auto-agreed (top tier) — the customer now picks the medium.
                    promptMedium(dayId);
                  } else {
                    // A lower tier: the chef sets the amount, no lower than the floor the
                    // server pinned. Show that floor so the outcome isn't a black box.
                    const floor = r?.minRefundPercent;
                    showAlert(
                      'Skip requested',
                      floor
                        ? `Your chef will review this (they may have started cooking). You’ll get back at least ${floor}% of what you paid for the day — we’ll notify you when it’s ready to choose.`
                        : 'Your chef will review this (they may have started cooking). We’ll notify you when your refund is ready to choose.',
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
