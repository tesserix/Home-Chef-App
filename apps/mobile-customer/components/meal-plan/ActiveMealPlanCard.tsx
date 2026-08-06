// ActiveMealPlanCard — the plan-holder's meal on the Home screen (#1037).
//
// Home was pure chef browsing: someone mid-plan had no idea dinner was already
// booked, let alone that the chef had started cooking it. Today's (or the next)
// meal now sits above the chef grid, live status included, and taps through to
// the plan.
//
// It reads as a sibling of ActiveOrderCard — same 16px radius, same hairline
// card, same chevron affordance — because they answer the same question ("what
// food is coming?"). It is INLINE rather than floating: the order card owns the
// floating anchor above the dock, and two cards fighting for that spot would
// overlap for anyone who has both.
//
// Every piece of day rendering (veg mark, status pill, cooking animation) is the
// same primitive MealPlanDayList uses, so a status can never read one way here
// and another on the plan screen.

import { useMemo } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import { CalendarCheck, ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

import { useMyMealPlans } from '../../hooks/useMealPlans';
import { useActiveOrder } from '../../hooks/useActiveOrder';
import { mealPlanDayStatusMeta, toLocalDateKey } from '../../lib/meal-plan';
import { selectActiveMealPlanMeal } from '../../lib/active-meal-plan';
import { DietIcon } from '@homechef/mobile-shared/ui';
import { CookingIndicator } from '../status/CookingIndicator';
import { HAIRLINE } from '../../lib/hairline';

/** "Today" / "Tomorrow" / "Fri 8 Aug" — relative reads faster on a glance card. */
function whenLabel(iso: string, todayKey: string): string {
  const key = toLocalDateKey(iso);
  if (key === todayKey) return 'Today';
  const tomorrow = new Date(`${todayKey}T00:00:00`);
  tomorrow.setDate(tomorrow.getDate() + 1);
  if (key === toLocalDateKey(tomorrow.toISOString())) return 'Tomorrow';
  return new Date(iso).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

export function ActiveMealPlanCard() {
  const { data } = useMyMealPlans();

  // Recomputed per render against the day the DEVICE is on, so a phone left open
  // past midnight rolls to the new day on the next data tick rather than
  // insisting yesterday's meal is still "Today".
  const todayKey = toLocalDateKey(new Date().toISOString());

  // A meal that is actually cooking already has its own floating active-order
  // card (the plan mints a real order per meal), so hand it over and show what
  // is booked NEXT instead of describing the same food twice.
  const { orders: activeOrders } = useActiveOrder();
  const activeOrderIds = useMemo(
    () => new Set(activeOrders.map((o) => o.id)),
    [activeOrders],
  );

  const selected = useMemo(
    () => selectActiveMealPlanMeal(data?.data, todayKey, activeOrderIds),
    [data?.data, todayKey, activeOrderIds],
  );

  if (!selected) return null;

  const { plan, day, mealNumber, totalMeals, isToday } = selected;
  const meta = mealPlanDayStatusMeta(day.status);
  const chefName = plan.chef?.businessName ?? 'Your chef';
  const isVeg = day.variant === 'veg';
  const slot = day.slot === 'lunch' ? 'Lunch' : 'Dinner';
  const when = whenLabel(day.date, todayKey);

  return (
    <Pressable
      onPress={() => router.push(`/meal-plans/${plan.id}` as never)}
      accessibilityRole="button"
      // One sentence, in reading order — a screen-reader user gets the same
      // glance the sighted card gives, not five disconnected fragments.
      accessibilityLabel={`Your meal plan from ${chefName}. ${when} ${slot}: ${
        day.dishName ?? 'dish to be confirmed'
      }, ${isVeg ? 'vegetarian' : 'non-vegetarian'}. ${meta.label}. Meal ${mealNumber} of ${totalMeals}. Tap to open the plan.`}
    >
      {({ pressed }) => (
        <View style={[styles.card, isToday && styles.cardToday, pressed && styles.cardPressed]}>
          <View style={styles.topRow}>
            {/* Leading coral calendar mark — the same glyph MyPlanChip and the
                Plans tab use for a tiffin plan. Without it this card was a white
                rectangle on a white page (canvas and surface.DEFAULT are BOTH
                #FFFFFF) and read as a stray chef row. */}
            <View style={[styles.iconWrap, isToday && styles.iconWrapToday]}>
              <CalendarCheck size={15} color={customerColors.coral.DEFAULT} />
            </View>
            <Text style={styles.when} numberOfLines={1}>
              {when} · {slot}
            </Text>
            <View style={styles.statusWrap}>
              {/* Same animated indicator the plan screen uses while a dish is
                  actually on the stove. */}
              {meta.cooking ? (
                <CookingIndicator size={13} color={customerColors.coral.DEFAULT} />
              ) : null}
              <View style={[styles.statusPill, { backgroundColor: meta.bg }]}>
                <Text style={[styles.statusText, { color: meta.color }]}>{meta.label}</Text>
              </View>
            </View>
          </View>

          <View style={styles.dishRow}>
            {/* DietIcon, not a coloured dot: geometry carries the veg/non-veg
                distinction so it survives colour-blindness (WCAG 1.4.1). */}
            <DietIcon kind={isVeg ? 'veg' : 'non-veg'} size={12} />
            <Text style={styles.dish} numberOfLines={1}>
              {day.dishName ?? 'Dish to be confirmed'}
            </Text>
          </View>

          <View style={styles.bottomRow}>
            <Text style={styles.meta} numberOfLines={1}>
              {/* "Meal", not "Day" — plan.days holds booked MEALS and a plan can
                  book both lunch and dinner on one date (#1040). */}
              Meal {mealNumber} of {totalMeals} · {chefName}
            </Text>
            <ChevronRight
              size={18}
              color={customerColors.charcoal.soft}
              accessibilityElementsHidden
            />
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  // Separation comes from ELEVATION, not a heavier border: `canvas` and
  // `surface.DEFAULT` are both #FFFFFF, so a hairline alone left this card
  // invisible against the page — it read as loose text between the filter bar
  // and the first chef photo. This is the same shadow[2] ActiveOrderCard uses,
  // which is the point: the two cards answer the same question and should sit
  // at the same depth.
  card: {
    marginBottom: 12,
    padding: 14,
    borderRadius: 16,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: HAIRLINE,
    borderColor: customerColors.hairline,
    gap: 6,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 12,
    elevation: 4,
  },
  // Today's meal earns the accent surface; a meal three days out does not.
  // Emphasis tracks urgency rather than being spent permanently, which keeps
  // "one accent, used sparingly" honest on a Home screen that already spends
  // coral on the selected cuisine, the dock and the active-order progress bar.
  cardToday: {
    backgroundColor: customerColors.coral.tint,
    borderColor: customerColors.coral.DEFAULT,
  },
  cardPressed: { opacity: 0.94 },

  // Tinted disc behind the calendar glyph — the same treatment as the
  // refund banner's icon in MealPlanList. Flips to solid canvas on the
  // coral-tint surface so the glyph keeps its contrast.
  iconWrap: {
    width: 26,
    height: 26,
    borderRadius: 13,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.coral.tint,
  },
  iconWrapToday: { backgroundColor: customerColors.canvas },

  topRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  when: {
    flex: 1,
    fontFamily: 'Geist',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.1,
  },
  statusWrap: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  statusPill: { borderRadius: 9999, paddingHorizontal: 10, paddingVertical: 3 },
  statusText: { fontFamily: 'Inter-SemiBold', fontSize: 11 },

  dishRow: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  dish: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
  },

  bottomRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 8,
  },
  meta: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
});
