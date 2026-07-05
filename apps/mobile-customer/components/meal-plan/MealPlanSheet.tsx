// MealPlanSheet — a closeable bottom sheet showing a tiffin plan's per-day
// fulfilment (#434). Built on the canonical shared Sheet and the shared
// MealPlanDayList, so the "Show my plan" chips and the plan-detail screen all
// present the same rows.
//
// It renders the plan passed by the chip, which comes from useMyMealPlans — that
// hook polls every 20s while a plan is confirmed/active, so the rows show
// near-real-time cooking/delivery status from the SAME source that feeds the
// chip's "today: …" copy (no separate query, no chip/sheet drift).

import { forwardRef } from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { Sheet, type SheetHandle } from '@homechef/mobile-shared/ui';

import { type MealPlan } from '../../hooks/useMealPlans';
import { formatDateRange } from '../../lib/meal-plan';
import { MealPlanDayList } from './MealPlanDayList';

export interface MealPlanSheetProps {
  /** The live plan to show — from useMyMealPlans (kept fresh by its poll). */
  summary: MealPlan;
}

export const MealPlanSheet = forwardRef<SheetHandle, MealPlanSheetProps>(function MealPlanSheet(
  { summary },
  ref,
) {
  return (
    <Sheet
      ref={ref}
      title={`My plan · ${formatDateRange(summary.startDate, summary.endDate)}`}
      cancelLabel="Close"
      snapPoints={['70%']}
    >
      <ScrollView style={styles.scroll} contentContainerStyle={styles.content}>
        {/* Read-only in the sheet: no Skip, no price column — a glanceable
            fulfilment view. The full detail screen keeps those actions. */}
        <MealPlanDayList days={summary.days ?? []} showPrice={false} />
      </ScrollView>
    </Sheet>
  );
});

const styles = StyleSheet.create({
  scroll: { marginTop: 4 },
  content: { paddingBottom: 24 },
});
