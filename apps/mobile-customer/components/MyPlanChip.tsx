// MyPlanChip — a compact "Show my plan" pill shown on a chef page when the
// customer has a live tiffin plan with THAT chef (#434). Tapping opens the
// shared MealPlanSheet listing each day's scheduled / being-prepared / delivered
// status. All the day-row + status logic lives in the shared MealPlanDayList /
// MealPlanSheet, so this file is just the chef-scoped chip.

import { useRef } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { CalendarCheck } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { type SheetHandle } from '@homechef/mobile-shared/ui';

import { useMyMealPlans } from '../hooks/useMealPlans';
import { pickLiveMealPlan, mealPlanStatusMeta } from '../lib/meal-plan';
import { MealPlanSheet } from './meal-plan/MealPlanSheet';

export function MyPlanChip({ chefId }: { chefId: string }) {
  const { data } = useMyMealPlans();
  const sheetRef = useRef<SheetHandle>(null);

  const plan = pickLiveMealPlan(data?.data, chefId);
  if (!plan) return null;
  const meta = mealPlanStatusMeta(plan.status);

  return (
    <>
      <Pressable
        onPress={() => sheetRef.current?.present()}
        style={styles.chip}
        accessibilityRole="button"
        accessibilityLabel="Show my plan"
      >
        <CalendarCheck size={16} color={customerColors.coral.DEFAULT} />
        <Text style={styles.chipText}>Show my plan</Text>
        <View style={[styles.statusPill, { backgroundColor: meta.bg }]}>
          <Text style={[styles.statusPillText, { color: meta.color }]}>{meta.label}</Text>
        </View>
      </Pressable>

      <MealPlanSheet ref={sheetRef} summary={plan} />
    </>
  );
}

const styles = StyleSheet.create({
  chip: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    alignSelf: 'flex-start',
    paddingVertical: 8,
    paddingHorizontal: 12,
    borderRadius: 999,
    backgroundColor: customerColors.coral.tint,
  },
  chipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.coral.DEFAULT,
  },
  statusPill: { borderRadius: 999, paddingVertical: 2, paddingHorizontal: 8 },
  statusPillText: { fontFamily: 'Inter-SemiBold', fontSize: 11 },
});
