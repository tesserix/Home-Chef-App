// MealPlanDayList — the shared per-day fulfilment list for a tiffin plan (#434).
// One rendering used by BOTH the full plan-detail screen and the "Show my plan"
// sheet, so the two never drift and the meal-row logic lives in one place.
//
// Rows are grouped by calendar date (#photo-forward-pass): a plan can book both
// lunch and dinner on one date, and rendering "Fri, Aug 7" twice as two
// unrelated rows read as a data bug, not a design. The date is now a group
// heading with that day's meals underneath, not a repeated label. Today's
// group keeps the coral-tint treatment the first pass introduced, adapted
// from a per-row tint to a whole-group one.
//
// Each meal row also now carries its dish photo (#photo-forward-pass): this is
// a food app whose design principle is "photo-forward, chrome-light" and the
// list was previously 100% text even though the dish image is already resolved
// on this screen (WeeklyMenuItem.imageUrl via the chef's weekly menu). A row
// with no matched image degrades to a neutral placeholder — never a blank or
// broken box — so the grid never shifts between rows that have a photo and
// rows that don't.
//
// Per meal row: an accept/declined badge on the photo corner, an FSSAI veg/non-veg
// mark with the slot + dish, the live status pill for EVERY status (not just the
// cooking ones — Scheduled / Being prepared / Delivered / Skipped / Refunded …)
// with the animated CookingIndicator while the dish is being prepared, the
// price, and an optional Skip action (detail screen only — the sheet is
// read-only).
//
// Row anatomy (visual pass, #row-hierarchy): each row is two lines, not one flat
// stack. Line 1 (photo + dish + price) is the "what and how much". Line 2
// (status pill + actions) is the "what can I do about it" — putting the status
// pill between the price above and the Skip/Confirm/Report link below stops the
// destructive-ish Skip link from reading as an appendage of the price figure.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { Image } from 'expo-image';
import { AlertTriangle, Check, UtensilsCrossed, X } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { DietIcon } from '@homechef/mobile-shared/ui';
import { findItemConflicts } from '@homechef/mobile-shared/dietary';

import { type MealPlanDay, type WeeklyMenuItem, useChefWeeklyMenu } from '../../hooks/useMealPlans';
import { useProfile } from '../../hooks/useProfile';
import { mealPlanDayStatusMeta, isDeclinedDayStatus, toLocalDateKey } from '../../lib/meal-plan';
import { canConfirmReceipt } from '../../lib/payout-hold';
import { CookingIndicator } from '../status/CookingIndicator';
import { formatMoney } from '../../lib/format';
import { HAIRLINE } from '../../lib/hairline';

// Same blurhash placeholder ChefCard/MenuItemCard/ChefReviewList use — one photo
// placeholder token across the app, not a per-screen invention.
const PHOTO_BLURHASH = 'L6PZfSi_.AyE_3t7t7R**0o#DgR4';

function dayLabel(d: MealPlanDay): string {
  return new Date(d.date).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

// A calendar-date group of booked meals — `plan.days` holds booked MEALS
// (a date can carry both lunch and dinner), so the date is a grouping key,
// not a per-row label.
interface DateGroup {
  dateKey: string;
  label: string;
  isToday: boolean;
  meals: MealPlanDay[];
}

function groupByDate(days: MealPlanDay[], todayKey: string): DateGroup[] {
  const groups: DateGroup[] = [];
  const indexByKey = new Map<string, number>();
  for (const d of days) {
    const dateKey = toLocalDateKey(d.date);
    const existingIndex = indexByKey.get(dateKey);
    if (existingIndex === undefined) {
      indexByKey.set(dateKey, groups.length);
      groups.push({ dateKey, label: dayLabel(d), isToday: dateKey === todayKey, meals: [d] });
    } else {
      groups[existingIndex]?.meals.push(d);
    }
  }
  return groups;
}

export interface MealPlanDayListProps {
  days: MealPlanDay[];
  /** The plan's chef — resolves each day's dish against the chef's weekly
   *  menu for the dietary-conflict warning (#901) and its dish photo. */
  chefId: string;
  /** When provided, a Skip link renders on still-skippable (confirmed) days. */
  onSkip?: (dayId: string) => void;
  /** Disables the Skip links while a skip request is in flight. */
  skipping?: boolean;
  /** When provided, a "Confirm received" link renders on a delivered day whose
   *  escrow hold awaits confirmation (#617). Inert while the flags are off. */
  onConfirmReceived?: (dayId: string) => void;
  /** Disables the confirm links while a confirm request is in flight. */
  confirming?: boolean;
  /** When provided, a "Report an issue" link renders on a delivered day (#618).
   *  Routes to the day's shell-order report screen; needs the day's orderId. */
  onReportIssue?: (day: MealPlanDay) => void;
  /** Show the per-day price column (detail screen). Defaults to true. */
  showPrice?: boolean;
}

export function MealPlanDayList({
  days,
  chefId,
  onSkip,
  skipping,
  onConfirmReceived,
  confirming,
  onReportIssue,
  showPrice = true,
}: MealPlanDayListProps) {
  // Dietary-conflict + photo data (#901, #photo-forward-pass) — fetched once
  // per render, not per row (hooks rule: no per-row hook calls in the .map()
  // below).
  const { data: profile } = useProfile();
  const { data: weeklyMenu } = useChefWeeklyMenu(chefId);
  const menuById = new Map<string, WeeklyMenuItem>();
  for (const it of weeklyMenu?.items ?? []) {
    if (it.id) menuById.set(it.id, it);
  }
  const todayKey = toLocalDateKey(new Date().toISOString());
  const groups = groupByDate(days, todayKey);

  return (
    <View style={styles.card}>
      {groups.map((group, gi) => (
        <View
          key={group.dateKey}
          style={[
            styles.dateGroup,
            gi < groups.length - 1 && styles.groupDivider,
            group.isToday && styles.dateGroupToday,
          ]}
        >
          <Text style={[styles.dateHeading, group.isToday && styles.dateHeadingToday]}>
            {group.isToday ? `Today · ${group.label}` : group.label}
          </Text>
          {group.meals.map((d, mi) => {
            const declined = isDeclinedDayStatus(d.status);
            const meta = mealPlanDayStatusMeta(d.status);
            // Resolve the day's dish against the chef's weekly menu when
            // possible; even unmatched, the day's own variant still catches a
            // veg/non-veg clash, so this never silently shows nothing (#901
            // quality bar) — and the photo just falls back to the placeholder.
            const matched = d.weeklyMenuItemId ? menuById.get(d.weeklyMenuItemId) : undefined;
            const conflicts = profile
              ? findItemConflicts(
                  { dietaryPreferences: profile.dietaryPreferences, foodAllergies: profile.foodAllergies },
                  { dietaryTags: matched?.dietaryTags, allergens: matched?.allergens, isVeg: d.variant === 'veg' },
                )
              : [];
            const showSkip = !!onSkip && d.status === 'confirmed';
            const showConfirm = !!onConfirmReceived && canConfirmReceipt(d);
            const showReport = !!onReportIssue && d.status === 'delivered' && !!d.orderId;
            const isLastMealInGroup = mi === group.meals.length - 1;
            return (
              <View
                key={d.id}
                style={[styles.mealRow, !isLastMealInGroup && styles.mealDivider]}
              >
                <View style={styles.thumbWrap}>
                  {matched?.imageUrl ? (
                    <Image
                      source={{ uri: matched.imageUrl }}
                      style={styles.thumb}
                      contentFit="cover"
                      placeholder={{ blurhash: PHOTO_BLURHASH }}
                      transition={150}
                      accessibilityElementsHidden
                    />
                  ) : (
                    // Missing image — surface-soft placeholder + utensil
                    // glyph (R2), same size as the photo so no row ever
                    // shifts between a matched and an unmatched dish.
                    <View style={[styles.thumb, styles.thumbPlaceholder]}>
                      <UtensilsCrossed
                        size={20}
                        color={customerColors.charcoal.soft}
                        strokeWidth={1.5}
                        accessibilityElementsHidden
                      />
                    </View>
                  )}
                  <View
                    style={[styles.statusBadge, declined ? styles.statusBad : styles.statusOk]}
                    accessibilityElementsHidden
                  >
                    {declined ? (
                      <X size={11} color={customerColors.canvas} strokeWidth={3} />
                    ) : (
                      <Check size={11} color={customerColors.canvas} strokeWidth={3} />
                    )}
                  </View>
                </View>

                <View style={styles.mealContent}>
                  <View style={styles.rowTop}>
                    <View style={{ flex: 1 }}>
                      {/* Veg/non-veg was an 8px green-or-red dot: hue ONLY — no label, no
                          a11y text, no shape difference — on the screen where you review a
                          whole week of food. WCAG 1.4.1, and red/green is the worst pair
                          for deuteranopia. DietIcon is the FSSAI square-outline mark the
                          dish cards already use: it differs by GEOMETRY, not just colour.
                          The text label makes it survive both colour-blindness and a
                          screen reader. */}
                      <View style={styles.dayMeta}>
                        <DietIcon kind={d.variant === 'veg' ? 'veg' : 'non-veg'} size={12} />
                        <Text style={[styles.daySub, declined && styles.dim]} numberOfLines={1}>
                          {d.variant === 'veg' ? 'Veg' : 'Non-veg'} ·{' '}
                          {d.slot === 'lunch' ? 'Lunch' : 'Dinner'} · {d.dishName ?? '—'}
                        </Text>
                      </View>
                      {/* Dietary-conflict warning (#901) — calm, factual, informational only. */}
                      {conflicts.length > 0 ? (
                        <View style={styles.warnRow}>
                          <AlertTriangle size={12} color={customerColors.destructive.DEFAULT} strokeWidth={2} />
                          <Text style={styles.warnText} numberOfLines={2}>
                            {conflicts.map((cf) => cf.detail).join(' · ')}
                          </Text>
                        </View>
                      ) : null}
                    </View>
                    {showPrice ? (
                      <Text style={[styles.price, declined && styles.dim]}>
                        {formatMoney((d.price ?? 0))}
                      </Text>
                    ) : null}
                  </View>

                  {/* Line 2 — status on the left, actions on the right. Deliberately a
                      separate row from the price above it (#skip-affordance): Skip is
                      a semi-destructive action and reading it as "attached to" the
                      price it sits under made it feel like a price modifier, not an
                      independent choice. */}
                  <View style={styles.rowBottom}>
                    <View style={styles.dayStatusRow}>
                      {meta.cooking ? (
                        <CookingIndicator size={14} color={customerColors.coral.DEFAULT} />
                      ) : null}
                      <View style={[styles.dayStatusPill, { backgroundColor: meta.bg }]}>
                        <Text style={[styles.dayStatusText, { color: meta.color }]}>{meta.label}</Text>
                      </View>
                    </View>
                    {showSkip || showConfirm || showReport ? (
                      <View style={styles.rowActions}>
                        {showSkip ? (
                          <Pressable
                            onPress={() => onSkip?.(d.id)}
                            hitSlop={8}
                            disabled={skipping}
                            accessibilityRole="button"
                            accessibilityLabel={`Skip ${dayLabel(d)}`}
                            accessibilityState={{ disabled: !!skipping, busy: !!skipping }}
                            style={styles.tapTarget}
                          >
                            <Text style={styles.skipLink}>Skip</Text>
                          </Pressable>
                        ) : null}
                        {/* #617 — per-day confirm receipt (delivered + awaiting confirmation).
                            Mutually exclusive with Skip (different day statuses). */}
                        {showConfirm ? (
                          <Pressable
                            onPress={() => onConfirmReceived?.(d.id)}
                            hitSlop={8}
                            disabled={confirming}
                            accessibilityRole="button"
                            accessibilityLabel={`Confirm you received ${dayLabel(d)}`}
                            accessibilityState={{ disabled: !!confirming, busy: !!confirming }}
                            style={styles.tapTarget}
                          >
                            <Text style={styles.confirmLink}>Confirm received</Text>
                          </Pressable>
                        ) : null}
                        {/* #618 — report a quality issue on a delivered day. Routes to the day's
                            shell-order report screen (needs orderId). Shown alongside Confirm. */}
                        {showReport ? (
                          <Pressable
                            onPress={() => onReportIssue?.(d)}
                            hitSlop={8}
                            accessibilityRole="button"
                            accessibilityLabel={`Report an issue with ${dayLabel(d)}`}
                            style={styles.tapTarget}
                          >
                            <Text style={styles.reportLink}>Report an issue</Text>
                          </Pressable>
                        ) : null}
                      </View>
                    ) : null}
                  </View>
                </View>
              </View>
            );
          })}
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 12,
    borderWidth: HAIRLINE,
    borderColor: customerColors.hairline,
    // Groups now paint their own full-bleed background (today's tint), so the
    // card must clip them to its own corner radius.
    overflow: 'hidden',
  },
  // One date heading + its meals. Rows within share a divider (mealDivider);
  // groups are separated by the heading's own top padding + a hairline
  // (groupDivider, omitted on the last group so the card's own border closes
  // the list instead of doubling it).
  dateGroup: {},
  groupDivider: {
    borderBottomWidth: HAIRLINE,
    borderBottomColor: customerColors.hairline,
  },
  // Today's group — the one thing on this screen worth finding at a glance among
  // otherwise-identical date groups. Background tint, not a border, so it reads
  // as a state (the meals that matter right now) rather than a selection outline.
  dateGroupToday: { backgroundColor: customerColors.coral.tint },
  dateHeading: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    letterSpacing: 0.1,
    fontVariant: ['tabular-nums'],
    paddingHorizontal: 16,
    paddingTop: 14,
    paddingBottom: 6,
  },
  dateHeadingToday: { color: customerColors.coral.DEFAULT },
  mealRow: { flexDirection: 'row', gap: 12, paddingHorizontal: 16, paddingVertical: 12 },
  mealDivider: { borderBottomWidth: HAIRLINE, borderBottomColor: customerColors.hairline },
  // Dish thumbnail — surface-soft background (on the Image itself) + blurhash
  // placeholder, so there is no blank flash before the 150ms fade-in, and a
  // fixed 56×56 footprint so an unmatched dish's placeholder never shifts the
  // row layout relative to a matched one.
  thumbWrap: { width: 56, height: 56 },
  thumb: { width: 56, height: 56, borderRadius: 10, backgroundColor: customerColors.surface.soft },
  thumbPlaceholder: { alignItems: 'center', justifyContent: 'center' },
  // Accept/declined badge — a small solid-fill corner badge on the photo
  // (avatar-status-dot convention) rather than a standalone icon column, now
  // that the photo itself anchors the row. White ring for contrast against
  // whatever the dish photo is doing underneath it.
  statusBadge: {
    position: 'absolute',
    bottom: -4,
    right: -4,
    width: 20,
    height: 20,
    borderRadius: 10,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 2,
    borderColor: customerColors.surface.DEFAULT,
  },
  statusOk: { backgroundColor: customerColors.success.DEFAULT },
  statusBad: { backgroundColor: customerColors.destructive.DEFAULT },
  mealContent: { flex: 1 },
  rowTop: { flexDirection: 'row', alignItems: 'flex-start', gap: 12 },
  dayMeta: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  daySub: { flex: 1, fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  // Dietary-conflict warning (#901) — matches WeeklyMenuDishCard/MenuItemCard tone.
  warnRow: { flexDirection: 'row', alignItems: 'flex-start', gap: 4, marginTop: 5 },
  warnText: { flex: 1, fontFamily: 'Inter-SemiBold', fontSize: 11, lineHeight: 15, color: customerColors.destructive.DEFAULT },
  price: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  dim: { color: customerColors.charcoal.soft, textDecorationLine: 'line-through' },
  rowBottom: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    // center, not flex-start: the status pill has its own vertical padding and
    // the action is bare text, so top-aligning them left the action sitting
    // visibly below the pill it shares a line with.
    alignItems: 'center',
    marginTop: 8,
    gap: 12,
  },
  dayStatusRow: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  dayStatusPill: { borderRadius: 9999, paddingHorizontal: 8, paddingVertical: 2, alignSelf: 'flex-start' },
  dayStatusText: { fontFamily: 'Inter-SemiBold', fontSize: 11, letterSpacing: 0.2 },
  rowActions: { alignItems: 'flex-end', gap: 2 },
  // WCAG 2.2 target floor. These were 12px text + hitSlop 8 -> ~32px: the
  // most-used controls in the product (Skip / Confirm / Report) were the
  // smallest things on the screen. hitSlop helps a mouse-free tap but does not
  // show up as a real target, and it does not scale with system text size.
  tapTarget: { minHeight: 44, justifyContent: 'center' },
  skipLink: { fontFamily: 'Inter-Medium', fontSize: 12, color: customerColors.coral.DEFAULT },
  // #617 — per-day "Confirm received" link (coral, slightly heavier than Skip).
  confirmLink: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: customerColors.coral.DEFAULT, textAlign: 'right' },
  // #618 — per-day "Report an issue" link (muted secondary, below Confirm).
  reportLink: { fontFamily: 'Inter-Medium', fontSize: 12, color: customerColors.charcoal.soft, textAlign: 'right' },
});
