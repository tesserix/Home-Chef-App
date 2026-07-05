// MealPlanSheet — a closeable bottom sheet showing a tiffin plan's per-day
// fulfilment (#434). Built on the canonical shared Sheet and the shared
// MealPlanDayList, so the "Show my plan" chips and the plan-detail screen all
// present the same rows.
//
// It re-fetches the plan by id via useMealPlan (which polls every 20s while the
// plan is confirmed/active), so the sheet shows near-real-time cooking/delivery
// status — reusing the exact hook the full detail screen uses. The `summary`
// plan (already in hand from the list) seeds the rows until the fresh fetch
// lands, so the sheet never opens empty.

import { forwardRef } from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { Sheet, type SheetHandle } from '@homechef/mobile-shared/ui';

import { useMealPlan, type MealPlan } from '../../hooks/useMealPlans';
import { formatDateRange } from '../../lib/meal-plan';
import { MealPlanDayList } from './MealPlanDayList';

export interface MealPlanSheetProps {
  /** The live plan to show (from the list) — seeds the title + rows. */
  summary: MealPlan;
}

export const MealPlanSheet = forwardRef<SheetHandle, MealPlanSheetProps>(function MealPlanSheet(
  { summary },
  ref,
) {
  // Poll the single plan for near-real-time per-day status while it's live.
  const { data } = useMealPlan(summary.id);
  const plan = data?.mealPlan ?? summary;

  return (
    <Sheet
      ref={ref}
      title={`My plan · ${formatDateRange(plan.startDate, plan.endDate)}`}
      cancelLabel="Close"
      snapPoints={['70%']}
    >
      <ScrollView style={styles.scroll} contentContainerStyle={styles.content}>
        {/* Read-only in the sheet: no Skip, no price column — a glanceable
            fulfilment view. The full detail screen keeps those actions. */}
        <MealPlanDayList days={plan.days ?? []} showPrice={false} />
      </ScrollView>
    </Sheet>
  );
});

const styles = StyleSheet.create({
  scroll: { marginTop: 4 },
  content: { paddingBottom: 24 },
});
