// "Weekly plan" tab pane on the chef detail screen. Everything tiffin lives
// here — the plan-a-week and subscribe entry points (slim hairline rows, no
// tinted cards), the MyPlanChip for an existing live plan, and the compact
// weekly-menu preview. The screen only mounts this pane when TIFFIN_ENABLED,
// so the deferred-money-flow gating is unchanged.

import { StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import { CalendarDays } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { WeeklyMenuItem } from '../../hooks/useMealPlans';
import { MyPlanChip } from '../MyPlanChip';
import { ChefActionRow } from './ChefActionRow';
import { WeeklyMenuPreview } from './WeeklyMenuPreview';

export interface ChefWeeklyPlanTabProps {
  chefId: string;
  /** Chef has an active daily-tiffin subscription offer (#283). */
  mealOfferAvailable: boolean;
  /** Chef's published fixed weekly menu items (#1); empty when none. */
  weeklyMenuItems: WeeklyMenuItem[];
  weeklyMenuPublished: boolean;
}

export function ChefWeeklyPlanTab({
  chefId,
  mealOfferAvailable,
  weeklyMenuItems,
  weeklyMenuPublished,
}: ChefWeeklyPlanTabProps) {
  const hasWeeklyMenu = weeklyMenuPublished && weeklyMenuItems.length > 0;

  return (
    <View style={styles.root}>
      <View style={styles.rows}>
        {/* Tiffin pre-booking (#196) — slim row, small coral icon. */}
        <ChefActionRow
          icon={
            <CalendarDays
              size={18}
              color={customerColors.coral.DEFAULT}
              strokeWidth={2}
            />
          }
          title="Plan a week of meals"
          caption="Pre-book tiffin from this chef"
          onPress={() => router.push(`/book-meal-plan?chefId=${chefId}` as never)}
          accessibilityLabel="Plan a week of meals"
        />

        {/* Compact entry to an existing reserved/active plan with this chef —
            opens a sheet with each day's fulfilment status (#434). */}
        <MyPlanChip chefId={chefId} />

        {/* Daily tiffin subscription (#283) — only when the chef offers one. */}
        {mealOfferAvailable ? (
          <ChefActionRow
            icon={
              <CalendarDays
                size={18}
                color={customerColors.coral.DEFAULT}
                strokeWidth={2}
              />
            }
            title="Subscribe to daily tiffin"
            caption="Recurring meals, delivered automatically"
            onPress={() => router.push(`/meal-subscription/${chefId}` as never)}
            accessibilityLabel="Subscribe to a daily tiffin"
          />
        ) : null}
      </View>

      {/* Compact weekly-menu preview (#1) — one day at a time. */}
      {hasWeeklyMenu ? (
        <WeeklyMenuPreview items={weeklyMenuItems} />
      ) : (
        <Text style={styles.emptyText}>
          This chef hasn’t published a weekly menu yet.
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    paddingTop: 4,
    paddingBottom: 24,
  },
  rows: {
    paddingHorizontal: 20,
  },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    paddingHorizontal: 20,
    paddingVertical: 24,
  },
});
