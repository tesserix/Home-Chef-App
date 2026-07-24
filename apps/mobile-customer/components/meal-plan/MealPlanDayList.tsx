// MealPlanDayList — the shared per-day fulfilment list for a tiffin plan (#434).
// One rendering used by BOTH the full plan-detail screen and the "Show my plan"
// sheet, so the two never drift and the day-row logic lives in one place.
//
// Each row shows: an accept/declined icon, the day label, an FSSAI veg/non-veg mark with
// the slot + dish, the live status pill for EVERY status (not just the cooking
// ones — Scheduled / Being prepared / Delivered / Skipped / Refunded …) with the
// animated CookingIndicator while the dish is being prepared, the price, and an
// optional Skip action (detail screen only — the sheet is read-only).

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { Check, X } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { DietIcon } from '@homechef/mobile-shared/ui';

import { type MealPlanDay } from '../../hooks/useMealPlans';
import { mealPlanDayStatusMeta, isDeclinedDayStatus } from '../../lib/meal-plan';
import { canConfirmReceipt } from '../../lib/payout-hold';
import { CookingIndicator } from '../status/CookingIndicator';

function dayLabel(d: MealPlanDay): string {
  return new Date(d.date).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

export interface MealPlanDayListProps {
  days: MealPlanDay[];
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
  onSkip,
  skipping,
  onConfirmReceived,
  confirming,
  onReportIssue,
  showPrice = true,
}: MealPlanDayListProps) {
  return (
    <View style={styles.card}>
      {days.map((d, i) => {
        const declined = isDeclinedDayStatus(d.status);
        const meta = mealPlanDayStatusMeta(d.status);
        return (
          <View key={d.id} style={[styles.dayRow, i < days.length - 1 && styles.divider]}>
            <View style={[styles.statusIcon, declined ? styles.statusBad : styles.statusOk]}>
              {declined ? (
                <X size={14} color={customerColors.destructive.DEFAULT} strokeWidth={2.5} />
              ) : (
                <Check size={14} color={customerColors.success.DEFAULT} strokeWidth={2.5} />
              )}
            </View>
            <View style={{ flex: 1 }}>
              <Text style={[styles.dayDate, declined && styles.dim]}>{dayLabel(d)}</Text>
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
              {/* Live per-day status (#50) — a pill for every status, animated
                  while the dish is being prepared. */}
              <View style={styles.dayStatusRow}>
                {meta.cooking ? (
                  <CookingIndicator size={14} color={customerColors.coral.DEFAULT} />
                ) : null}
                <View style={[styles.dayStatusPill, { backgroundColor: meta.bg }]}>
                  <Text style={[styles.dayStatusText, { color: meta.color }]}>{meta.label}</Text>
                </View>
              </View>
            </View>
            <View style={{ alignItems: 'flex-end' }}>
              {showPrice ? (
                <Text style={[styles.price, declined && styles.dim]}>
                  ₹{(d.price ?? 0).toFixed(0)}
                </Text>
              ) : null}
              {onSkip && d.status === 'confirmed' ? (
                <Pressable
                  onPress={() => onSkip(d.id)}
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
              {onConfirmReceived && canConfirmReceipt(d) ? (
                <Pressable
                  onPress={() => onConfirmReceived(d.id)}
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
              {onReportIssue && d.status === 'delivered' && d.orderId ? (
                <Pressable
                  onPress={() => onReportIssue(d)}
                  hitSlop={8}
                  accessibilityRole="button"
                  accessibilityLabel={`Report an issue with ${dayLabel(d)}`}
                  style={styles.tapTarget}
                >
                  <Text style={styles.reportLink}>Report an issue</Text>
                </Pressable>
              ) : null}
            </View>
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    paddingHorizontal: 16,
  },
  dayRow: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingVertical: 12 },
  divider: { borderBottomWidth: StyleSheet.hairlineWidth, borderBottomColor: customerColors.hairline },
  statusIcon: { width: 24, height: 24, borderRadius: 12, alignItems: 'center', justifyContent: 'center' },
  statusOk: { backgroundColor: customerColors.success.tint },
  statusBad: { backgroundColor: customerColors.destructive.tint },
  dayDate: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  dayMeta: { flexDirection: 'row', alignItems: 'center', gap: 6, marginTop: 3 },
  dayStatusRow: { flexDirection: 'row', alignItems: 'center', gap: 6, marginTop: 5 },
  dayStatusPill: { borderRadius: 9999, paddingHorizontal: 8, paddingVertical: 2, alignSelf: 'flex-start' },
  dayStatusText: { fontFamily: 'Inter-SemiBold', fontSize: 11, letterSpacing: 0.2 },
  daySub: { flex: 1, fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  price: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  dim: { color: customerColors.charcoal.soft, textDecorationLine: 'line-through' },
  // WCAG 2.2 target floor. These were 12px text + hitSlop 8 -> ~32px: the
  // most-used controls in the product (Skip / Confirm / Report) were the
  // smallest things on the screen. hitSlop helps a mouse-free tap but does not
  // show up as a real target, and it does not scale with system text size.
  tapTarget: { minHeight: 44, justifyContent: 'center' },
  skipLink: { fontFamily: 'Inter-Medium', fontSize: 12, color: customerColors.coral.DEFAULT, marginTop: 4 },
  // #617 — per-day "Confirm received" link (coral, slightly heavier than Skip).
  confirmLink: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: customerColors.coral.DEFAULT, marginTop: 4, textAlign: 'right' },
  // #618 — per-day "Report an issue" link (muted secondary, below Confirm).
  reportLink: { fontFamily: 'Inter-Medium', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 6, textAlign: 'right' },
});
