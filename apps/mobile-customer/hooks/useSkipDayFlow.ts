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
      'We’ll send a refund request for review. Once approved you get that day’s food back (minus the platform fee) to your wallet — GST and delivery aren’t refunded. This can’t be undone.',
      [
        { text: 'Back', style: 'cancel' },
        {
          text: 'Request skip',
          style: 'destructive',
          onPress: () =>
            skipDay.mutate(
              { planId, dayId },
              {
                onSuccess: () =>
                  Alert.alert(
                    'Skip requested',
                    'Your request is in for review. If approved, the day’s food (minus the platform fee) is refunded to your wallet and your chef won’t cook it.',
                  ),
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
