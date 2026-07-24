// HomeMealPlanCard — surfaces the customer's single live tiffin meal plan as a
// card near the top of the Home screen (#434 follow-up), so an active plan is
// reachable without digging through Profile → My meal plans.
//
// Two states, driven entirely by the plan's status meta:
//   • needsAction (awaiting_customer / chef_modified): a prominent card with the
//     accepted-days total and inline Reject + Approve & pay buttons — the same
//     approve/reject flow (and button styling) as the plan-detail footer.
//   • otherwise (pending_chef / confirmed / active / …): a compact status row
//     that taps through to the plan detail.
//
// Self-contained (no props). Reuses the shared approval hook + pure meal-plan
// helpers — no duplicated flow logic here. Renders nothing when the customer has
// no live plan.

import {
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { router } from 'expo-router';
import { CalendarCheck, ChevronRight } from 'lucide-react-native';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';

import { useMyMealPlans } from '../../hooks/useMealPlans';
import {
  pickLiveMealPlan,
  mealPlanStatusMeta,
  summarizeLivePlan,
  isDeclinedDayStatus,
  toLocalDateKey,
} from '../../lib/meal-plan';
import { useMealPlanApproval } from '../../hooks/useMealPlanApproval';

// Android ripple tints — translucent tokens (mirrors the plan-detail footer),
// never a fresh literal colour. White ripple on the coral CTA; a faint ink
// ghost on the outline button and the whole-card tap.
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export function HomeMealPlanCard() {
  const { data } = useMyMealPlans();
  const plan = pickLiveMealPlan(data?.data);

  // Called unconditionally (hook rules); the hook no-ops when plan is undefined.
  // The Home card stays put after approve/reject — React Query refetch swaps the
  // card between its two states — so onDone does nothing here.
  const { approve, reject, isPending } = useMealPlanApproval(plan, {
    onDone: () => {},
  });

  if (!plan) return null;

  const meta = mealPlanStatusMeta(plan.status);
  const chefName = plan.chef?.businessName ?? 'Your chef';

  // ── Approval state — the chef trimmed the plan; the customer must approve
  //    (pay the advance for the days the chef can cook) or reject (cancel). ──
  if (meta.needsAction) {
    const acceptedTotal = (plan.days ?? [])
      .filter((d) => !isDeclinedDayStatus(d.status))
      .reduce((s, d) => s + (d.price ?? 0), 0);

    return (
      <View style={styles.actionCard}>
        {/* Header taps through to the detail screen so the customer can REVIEW
            which days the chef can cook before approving (the buttons below are
            the quick path once they trust the change). */}
        <Pressable
          onPress={() => router.push(('/meal-plans/' + plan.id) as never)}
          accessibilityRole="button"
          accessibilityLabel="Review the plan changes"
          android_ripple={{ color: ROW_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.actionHeader,
                pressed && Platform.OS === 'ios' && styles.actionHeaderPressed,
              ]}
            >
              <View style={styles.actionHeaderText}>
                <Text style={styles.chefName} numberOfLines={1}>
                  {chefName}
                </Text>
                <View style={styles.eyebrowRow}>
                  <Text style={styles.actionEyebrow} numberOfLines={1}>
                    Your approval needed · Review changes
                  </Text>
                  <ChevronRight size={13} color={customerColors.coral.DEFAULT} />
                </View>
              </View>
              <Text style={styles.actionTotal}>₹{acceptedTotal}</Text>
            </View>
          )}
        </Pressable>

        <View style={styles.buttonRow}>
          <Pressable
            style={styles.colNarrow}
            onPress={reject}
            disabled={isPending}
            accessibilityRole="button"
            accessibilityLabel="Reject plan"
            android_ripple={isPending ? undefined : { color: GHOST_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.rejectBtn,
                  pressed && Platform.OS === 'ios' && !isPending && styles.rejectBtnPressed,
                ]}
              >
                <Text style={styles.rejectText}>Reject</Text>
              </View>
            )}
          </Pressable>
          <Pressable
            style={styles.colWide}
            onPress={approve}
            disabled={isPending}
            accessibilityRole="button"
            accessibilityLabel="Approve plan and pay"
            android_ripple={isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.approveBtn,
                  pressed && Platform.OS === 'ios' && !isPending && styles.approveBtnPressed,
                ]}
              >
                {isPending ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text style={styles.approveText}>Approve &amp; pay</Text>
                )}
              </View>
            )}
          </Pressable>
        </View>
      </View>
    );
  }

  // ── Compact state — a live plan with nothing for the customer to do. Whole
  //    row taps through to the plan detail. ──
  const todayISO = toLocalDateKey(new Date().toISOString());
  const { daysLeft } = summarizeLivePlan(plan.days ?? [], todayISO);
  const daysLeftText = `${daysLeft} day${daysLeft === 1 ? '' : 's'} left`;

  return (
    <Pressable
      onPress={() => router.push(('/meal-plans/' + plan.id) as never)}
      accessibilityRole="button"
      accessibilityLabel={`Your tiffin plan with ${chefName}, view details`}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.compactCard,
            pressed && Platform.OS === 'ios' && styles.compactCardPressed,
          ]}
        >
          <View style={styles.compactIcon}>
            <CalendarCheck size={18} color={customerColors.coral.DEFAULT} />
          </View>
          <View style={styles.compactText}>
            <Text style={styles.chefName} numberOfLines={1}>
              {chefName}
            </Text>
            <Text style={styles.compactStatus} numberOfLines={1}>
              {daysLeftText}
            </Text>
          </View>
          <View style={[styles.pill, { backgroundColor: meta.bg }]}>
            <Text style={[styles.pillText, { color: meta.color }]} numberOfLines={1}>
              {meta.label}
            </Text>
          </View>
          <ChevronRight size={18} color={customerColors.charcoal.soft} />
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  // ── Shared ──
  chefName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },

  // ── Approval (prominent) card ──
  // White card, hairline border + card-lift shadow so it reads as the one thing
  // asking for the customer's attention without shouting (coral stays on the CTA).
  actionCard: {
    marginHorizontal: 16,
    marginTop: 4,
    padding: 16,
    borderRadius: 16,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    gap: 14,
    shadowColor: customerTheme.shadow[2].shadowColor,
    shadowOffset: customerTheme.shadow[2].shadowOffset,
    shadowOpacity: customerTheme.shadow[2].shadowOpacity,
    shadowRadius: customerTheme.shadow[2].shadowRadius,
    elevation: customerTheme.shadow[2].elevation,
  },
  actionHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  actionHeaderPressed: { opacity: 0.6 },
  actionHeaderText: {
    flex: 1,
    gap: 2,
  },
  eyebrowRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 2,
  },
  actionEyebrow: {
    fontFamily: 'Inter-Medium',
    fontSize: 13,
    color: customerColors.coral.DEFAULT,
    flexShrink: 1,
  },
  actionTotal: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },

  // Button row — Reject 1 : Approve 2, matching the plan-detail footer. The
  // flex lives on the Pressable wrappers so they (not just their inner Views)
  // span the row.
  buttonRow: {
    flexDirection: 'row',
    gap: 12,
  },
  colNarrow: { flex: 1 },
  colWide: { flex: 2 },
  rejectBtn: {
    height: 48,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
    alignItems: 'center',
    justifyContent: 'center',
  },
  rejectBtnPressed: { backgroundColor: customerColors.surface.soft },
  rejectText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
  },
  approveBtn: {
    height: 48,
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  approveBtnPressed: { backgroundColor: customerColors.coral.pressed },
  approveText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.canvas,
  },

  // ── Compact (no-action) card ──
  compactCard: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginHorizontal: 16,
    marginTop: 4,
    minHeight: 44,
    paddingVertical: 12,
    paddingHorizontal: 14,
    borderRadius: 12,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
  },
  compactCardPressed: { backgroundColor: customerColors.surface.soft },
  compactIcon: {
    width: 32,
    height: 32,
    borderRadius: 16,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.coral.tint,
  },
  compactText: {
    flex: 1,
    gap: 2,
  },
  compactStatus: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },
  pill: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 999,
  },
  pillText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
  },
});
