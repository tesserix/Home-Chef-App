// MyPlanChip — a compact "Show my plan" pill shown on a chef page when the
// customer has a reserved/active tiffin plan with THAT chef (#434). Tapping opens
// a closeable sheet listing each day with its scheduled / being-prepared /
// delivered status. Reuses the shared Sheet + the day-status presentation from
// lib/meal-plan, so it stays consistent with the full plan-detail screen and adds
// no duplicate status logic.

import { useMemo, useRef } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { CalendarCheck } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { Sheet, type SheetHandle } from '@homechef/mobile-shared/ui';
import { useMyMealPlans, type MealPlan } from '../hooks/useMealPlans';
import {
  formatDateRange,
  mealPlanDayStatusMeta,
  mealPlanStatusMeta,
} from '../lib/meal-plan';

// Statuses worth surfacing on the chef page — a plan still in flight or running.
const LIVE_STATUSES = new Set([
  'pending_chef',
  'awaiting_customer',
  'chef_modified',
  'chef_accepted_full',
  'confirmed',
  'active',
]);

export function MyPlanChip({ chefId }: { chefId: string }) {
  const { data } = useMyMealPlans();
  const sheetRef = useRef<SheetHandle>(null);

  const plan = useMemo<MealPlan | undefined>(() => {
    const mine = (data?.data ?? []).filter(
      (p) => p.chefId === chefId && LIVE_STATUSES.has(p.status),
    );
    // Most recent live plan first.
    return mine.sort((a, b) => (b.startDate ?? '').localeCompare(a.startDate ?? ''))[0];
  }, [data, chefId]);

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

      <Sheet
        ref={sheetRef}
        title={`My plan · ${formatDateRange(plan.startDate, plan.endDate)}`}
        cancelLabel="Close"
        snapPoints={['70%']}
      >
        <ScrollView style={styles.sheetScroll} contentContainerStyle={styles.sheetContent}>
          {(plan.days ?? []).map((d) => {
            const dm = mealPlanDayStatusMeta(d.status);
            return (
              <View key={d.id} style={styles.dayRow}>
                <View style={{ flex: 1 }}>
                  <Text style={styles.dayDate}>
                    {new Date(d.date).toLocaleDateString(undefined, {
                      weekday: 'short',
                      day: 'numeric',
                      month: 'short',
                    })}
                  </Text>
                  <Text style={styles.dayDish} numberOfLines={1}>
                    {d.dishName ??
                      `${d.slot === 'lunch' ? 'Lunch' : 'Dinner'} · ${d.variant === 'veg' ? 'Veg' : 'Non-veg'}`}
                  </Text>
                </View>
                <View style={[styles.dayPill, { backgroundColor: dm.bg }]}>
                  <Text style={[styles.dayPillText, { color: dm.color }]}>{dm.label}</Text>
                </View>
              </View>
            );
          })}
        </ScrollView>
      </Sheet>
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
  sheetScroll: { marginTop: 4 },
  sheetContent: { paddingBottom: 24, gap: 4 },
  dayRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.surface.soft,
  },
  dayDate: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
  },
  dayDish: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginTop: 2 },
  dayPill: { borderRadius: 999, paddingVertical: 4, paddingHorizontal: 10 },
  dayPillText: { fontFamily: 'Inter-SemiBold', fontSize: 12 },
});
