// ActivePlanChip — a glanceable "your tiffin plan" pill for the customer home
// (#434). When the customer has any live plan (across chefs) it shows a compact
// "N days left · today: <status>" summary; tapping opens the shared MealPlanSheet
// with the full per-day fulfilment. This is the discoverability entry the issue
// adds — previously a plan was only reachable via Profile → list → detail.
//
// Reuses pickLiveMealPlan + summarizeLivePlan (pure, tested) and the shared
// MealPlanSheet — no day-row logic here.

import { useMemo, useRef } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { CalendarCheck } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { type SheetHandle } from '@homechef/mobile-shared/ui';

import { useMyMealPlans } from '../../hooks/useMealPlans';
import {
  pickLiveMealPlan,
  summarizeLivePlan,
  mealPlanDayStatusMeta,
  toLocalDateKey,
} from '../../lib/meal-plan';
import { MealPlanSheet } from './MealPlanSheet';

export function ActivePlanChip() {
  const { data } = useMyMealPlans();
  const sheetRef = useRef<SheetHandle>(null);

  const plan = pickLiveMealPlan(data?.data);

  const summary = useMemo(
    () => (plan ? summarizeLivePlan(plan.days ?? [], toLocalDateKey(new Date().toISOString())) : null),
    [plan],
  );

  if (!plan || !summary) return null;

  const daysLeftText = `${summary.daysLeft} day${summary.daysLeft === 1 ? '' : 's'} left`;
  const todayText = summary.todayStatus
    ? ` · today: ${mealPlanDayStatusMeta(summary.todayStatus).label}`
    : '';

  return (
    <>
      <Pressable
        onPress={() => sheetRef.current?.present()}
        style={styles.chip}
        accessibilityRole="button"
        accessibilityLabel="Show my tiffin plan"
      >
        <CalendarCheck size={16} color={customerColors.coral.DEFAULT} />
        <Text style={styles.chipText} numberOfLines={1}>
          {daysLeftText}
          {todayText}
        </Text>
      </Pressable>

      <MealPlanSheet ref={sheetRef} summary={plan} />
    </>
  );
}

// Neutral chrome (accent discipline): this is a status, not a promotion — the
// small coral calendar icon is the only accent, so the chip doesn't compete
// with the dock pill / selected filters for the eye.
const styles = StyleSheet.create({
  chip: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    alignSelf: 'flex-start',
    marginHorizontal: 16,
    paddingVertical: 8,
    paddingHorizontal: 12,
    borderRadius: 999,
    backgroundColor: customerColors.canvas,
    borderWidth: 1,
    borderColor: customerColors.hairline,
  },
  chipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
  },
});
