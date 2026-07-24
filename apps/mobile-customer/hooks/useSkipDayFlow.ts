import { Alert } from 'react-native';

import { useSkipMealPlanDay } from './useMealPlans';

// Shared "request to skip a day" flow — the confirm dialog, the skip request, and the
// success/error copy — so the plan-detail screen and the "My plan" sheet behave identically
// (one source of truth, no drift). The server enforces the exact guardrail: a day can only be
// skipped while it is still `confirmed` (no order generated) and at least ~12h before the chef
// starts cooking it; if it is too late the request is rejected and we explain why.
export function useSkipDayFlow(planId: string | undefined) {
  const skipDay = useSkipMealPlanDay();

  function confirmSkip(dayId: string) {
    if (!planId) return;
    Alert.alert(
      'Skip this day?',
      'More than 12 hours before your meal? You’re refunded to your wallet right away. Closer than that, your chef reviews it (they may have started cooking). The refund is the food only — the platform fee, GST, and delivery aren’t refunded. This can’t be undone.',
      [
        { text: 'Back', style: 'cancel' },
        {
          text: 'Request skip',
          style: 'destructive',
          onPress: () =>
            skipDay.mutate(
              { planId, dayId },
              {
                // The server tells us the outcome: an auto-refund (>12h) or a pending chef review
                // (≤12h). Show its message verbatim so the copy always matches what happened.
                onSuccess: (res) => {
                  const r = res as { status?: string; message?: string } | undefined;
                  Alert.alert(
                    r?.status === 'refunded' ? 'Refunded to your wallet' : 'Skip requested',
                    r?.message ??
                      'Your request is in. If approved, the day’s food (minus the platform fee) goes to your wallet.',
                  );
                },
                onError: () =>
                  Alert.alert(
                    'Could not request skip',
                    'It may be too close to when your chef starts cooking this day.',
                  ),
              },
            ),
        },
      ],
    );
  }

  return { confirmSkip, skipping: skipDay.isPending };
}
