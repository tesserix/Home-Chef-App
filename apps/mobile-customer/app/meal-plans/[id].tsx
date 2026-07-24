import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';

// Android ripple tints — translucent tokens, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;
import {
  canCancelMealPlan,
  mealPlanAdvanceBreakdown,
  useCancelMealPlan,
  useFinalizeMealPlan,
  useMealPlan,
  useSkipMealPlanDay,
  type MealPlanDay,
} from '../../hooks/useMealPlans';
import {
  useConfirmMealPlanDayReceived,
  useConfirmTodaysTiffin,
} from '../../hooks/useConfirmReceived';
import { mealPlanStatusMeta, isDeclinedDayStatus, toLocalDateKey } from '../../lib/meal-plan';
import { canConfirmReceipt } from '../../lib/payout-hold';
import { friendlyErrorMessage } from '../../lib/errors';
import { MealPlanDayList } from '../../components/meal-plan/MealPlanDayList';

// Plan detail (#196): the booked days with per-day status. When the chef has
// cherry-picked (status awaiting_customer), the customer approves the revised
// set (declined days drop) or rejects it (whole plan cancels).
export default function MealPlanDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading } = useMealPlan(id);
  const finalize = useFinalizeMealPlan();
  const skipDay = useSkipMealPlanDay();
  const cancel = useCancelMealPlan();
  const confirmDay = useConfirmMealPlanDayReceived();
  const confirmTiffin = useConfirmTodaysTiffin();
  const plan = data?.mealPlan;

  function handleCancel() {
    if (!id) return;
    Alert.alert(
      'Cancel this plan?',
      "You haven't been served yet, so you'll be fully refunded. This can't be undone.",
      [
        { text: 'Keep plan', style: 'cancel' },
        {
          text: 'Cancel plan',
          style: 'destructive',
          onPress: () =>
            cancel.mutate(id, {
              onSuccess: () =>
                Alert.alert(
                  'Plan cancelled',
                  'Your plan was cancelled and any advance refunded.',
                  [{ text: 'OK', onPress: () => router.back() }],
                ),
              onError: () => Alert.alert('Something went wrong', 'Please try again.'),
            }),
        },
      ],
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator color={customerColors.coral.DEFAULT} />
      </SafeAreaView>
    );
  }

  if (!plan) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <Header />
        <View style={styles.centered}>
          <Text style={styles.muted}>Meal plan not found.</Text>
        </View>
      </SafeAreaView>
    );
  }

  const meta = mealPlanStatusMeta(plan.status);
  const days = plan.days ?? [];
  const acceptedDays = days.filter((d) => !isDeclinedDayStatus(d.status));
  const acceptedTotal = acceptedDays.reduce((s, d) => s + (d.price ?? 0), 0);

  function act(approve: boolean) {
    if (!plan) return;
    Alert.alert(
      approve ? 'Approve & pay?' : 'Reject plan?',
      approve
        ? `Confirm the ${acceptedDays.length} day${acceptedDays.length === 1 ? '' : 's'} your chef can cook, then pay the advance (food + GST + delivery, shown at checkout) to lock them in.`
        : 'This cancels the whole plan. You can book again any time.',
      [
        { text: 'Back', style: 'cancel' },
        {
          text: approve ? 'Approve' : 'Reject',
          style: approve ? 'default' : 'destructive',
          onPress: () =>
            finalize.mutate(
              { id: plan.id, approve },
              {
                onSuccess: (res) => {
                  // Approve (escrow on): the server minted a Razorpay advance order
                  // for the accepted days — launch checkout. Payment now happens here,
                  // after approval, not at create. verify-payment then confirms + holds.
                  if (approve && res?.paymentError) {
                    Alert.alert('Payment unavailable', res.paymentError);
                    return;
                  }
                  if (approve && res?.razorpayOrderId) {
                    const b = mealPlanAdvanceBreakdown(res.mealPlan);
                    router.push({
                      pathname: '/payment/checkout',
                      params: {
                        kind: 'mealplan',
                        mealPlanId: plan.id,
                        razorpayOrderId: res.razorpayOrderId,
                        razorpayKeyId: res.razorpayKeyId ?? '',
                        amount: String(b.amountPaise),
                        currency: res.mealPlan.currency ?? 'INR',
                      },
                    });
                    return;
                  }
                  // Reject, or escrow-off approve (unpaid handshake → confirmed).
                  Alert.alert(
                    approve ? 'Plan confirmed' : 'Plan cancelled',
                    approve
                      ? 'Your chef has been notified.'
                      : 'No charge — the plan was cancelled.',
                    [{ text: 'OK', onPress: () => router.back() }],
                  );
                },
                onError: () => Alert.alert('Something went wrong', 'Please try again.'),
              },
            ),
        },
      ],
    );
  }

  function confirmSkip(dayId: string) {
    if (!plan) return;
    Alert.alert(
      'Skip this day?',
      'We’ll send a refund request for review. Once approved you get that day’s food back (minus the platform fee) to your wallet — GST and delivery aren’t refunded. This can’t be undone.',
      [
        { text: 'Back', style: 'cancel' },
        {
          text: 'Request skip',
          style: 'destructive',
          onPress: () =>
            skipDay.mutate(
              { planId: plan.id, dayId },
              {
                onSuccess: () =>
                  Alert.alert(
                    'Skip requested',
                    'Your request is in for review. If approved, the day’s food (minus the platform fee) is refunded to your wallet and your chef won’t cook it.',
                  ),
                onError: () =>
                  Alert.alert(
                    'Could not request skip',
                    'It may be too close to when your chef starts cooking this day.',
                  ),
              },
            ),
        },
      ],
    );
  }

  // #617 — escrow fulfilment confirmation for delivered days awaiting the
  // customer's confirmation. The bulk banner is scoped to TODAY's awaiting days
  // (the `/tiffin/confirm-today` endpoint only confirms today's), so the count
  // matches what the button does; the per-day links handle any non-today
  // stragglers. Both are inert while the escrow flags are off (no day ever
  // reaches awaiting).
  const todayKey = toLocalDateKey(new Date().toISOString());
  const todaysAwaitingCount = days.filter(
    (d) => canConfirmReceipt(d) && toLocalDateKey(d.date) === todayKey,
  ).length;

  function confirmReceipt(dayId: string) {
    if (!plan) return;
    Alert.alert(
      'Confirm this meal?',
      "Let us know you received this meal. You can still report an issue if something's wrong.",
      [
        { text: 'Not yet', style: 'cancel' },
        {
          text: 'Confirm received',
          onPress: () =>
            confirmDay.mutate(
              { planId: plan.id, dayId },
              {
                onSuccess: (res) => Alert.alert('Thanks!', res.message),
                onError: (err) =>
                  Alert.alert(
                    'Something went wrong',
                    friendlyErrorMessage(err, 'Could not confirm right now. Please try again.'),
                  ),
              },
            ),
        },
      ],
    );
  }

  function confirmToday() {
    Alert.alert(
      "Confirm today's meals?",
      "Let us know you received today's delivered meals. You can still report an issue if something's wrong.",
      [
        { text: 'Not yet', style: 'cancel' },
        {
          text: 'Confirm',
          onPress: () =>
            confirmTiffin.mutate(undefined, {
              onSuccess: (res) =>
                Alert.alert(
                  'Thanks!',
                  res.confirmed > 0
                    ? `Confirmed ${res.confirmed} meal${res.confirmed === 1 ? '' : 's'}.`
                    : 'Your meals are already confirmed.',
                ),
              onError: (err) =>
                Alert.alert(
                  'Something went wrong',
                  friendlyErrorMessage(err, 'Could not confirm right now. Please try again.'),
                ),
            }),
        },
      ],
    );
  }

  // #618 — report a quality issue on a delivered day. The day's per-day fulfilment
  // shell order is a real order, so reuse the existing order report-issue screen.
  function reportIssue(day: MealPlanDay) {
    if (!day.orderId) return;
    router.push(`/order/${day.orderId}/report-issue`);
  }

  // A confirmed day on a live plan can still be skipped (server enforces the cutoff).
  const canSkip =
    plan.status === 'confirmed' || plan.status === 'active';
  const needsApproval = meta.needsAction;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <Header />
      <ScrollView style={styles.scrollFill} contentContainerStyle={styles.scroll}>
        <View style={styles.summaryRow}>
          <View>
            <Text style={styles.planNo}>{plan.mealPlanNumber}</Text>
            <Text style={styles.chefName}>{plan.chef?.businessName ?? 'Your chef'}</Text>
          </View>
          <View style={[styles.chip, { backgroundColor: meta.bg }]}>
            <Text style={[styles.chipText, { color: meta.color }]}>{meta.label}</Text>
          </View>
        </View>

        {needsApproval ? (
          <View style={styles.banner}>
            <Text style={styles.bannerTitle}>Your chef revised this plan</Text>
            <Text style={styles.bannerText}>
              They can cook {acceptedDays.length} of {days.length} days. Approve
              to confirm those, or reject to cancel the whole plan.
            </Text>
          </View>
        ) : null}

        {/* #617 — bulk-confirm today's delivered meals (tiffin). Shown only when
            a day awaits confirmation; the per-day links below handle single days. */}
        {todaysAwaitingCount > 0 ? (
          <View style={styles.confirmBanner}>
            <View style={{ flex: 1 }}>
              <Text style={styles.confirmBannerTitle}>Confirm today's meals</Text>
              <Text style={styles.confirmBannerText}>
                {todaysAwaitingCount} delivered meal{todaysAwaitingCount === 1 ? '' : 's'} awaiting
                your confirmation.
              </Text>
            </View>
            <Pressable
              onPress={confirmToday}
              disabled={confirmTiffin.isPending}
              accessibilityRole="button"
              accessibilityLabel="Confirm today's delivered meals"
              android_ripple={confirmTiffin.isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.confirmTodayBtn,
                    pressed && Platform.OS === 'ios' && !confirmTiffin.isPending && styles.confirmTodayBtnPressed,
                  ]}
                >
                  {confirmTiffin.isPending ? (
                    <ActivityIndicator color={customerColors.canvas} size="small" />
                  ) : (
                    <Text style={styles.confirmTodayText}>Confirm today</Text>
                  )}
                </View>
              )}
            </Pressable>
          </View>
        ) : null}

        <MealPlanDayList
          days={days}
          onSkip={canSkip ? confirmSkip : undefined}
          skipping={skipDay.isPending}
          onConfirmReceived={confirmReceipt}
          confirming={confirmDay.isPending}
          onReportIssue={reportIssue}
        />

        <View style={styles.totalRow}>
          <Text style={styles.totalLabel}>
            {needsApproval ? 'If approved' : 'Total'}
          </Text>
          <Text style={styles.totalValue}>
            ₹{(needsApproval ? acceptedTotal : plan.total).toFixed(0)}
          </Text>
        </View>
      </ScrollView>

      {needsApproval ? (
        <View style={styles.footer}>
          <Pressable
            style={styles.footerColNarrow}
            onPress={() => act(false)}
            disabled={finalize.isPending}
            accessibilityRole="button"
            accessibilityLabel="Reject plan"
            android_ripple={finalize.isPending ? undefined : { color: GHOST_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.rejectBtn,
                  pressed && Platform.OS === 'ios' && !finalize.isPending && styles.rejectBtnPressed,
                ]}
              >
                <Text style={styles.rejectText}>Reject</Text>
              </View>
            )}
          </Pressable>
          <Pressable
            style={styles.footerColWide}
            onPress={() => act(true)}
            disabled={finalize.isPending}
            accessibilityRole="button"
            accessibilityLabel={`Approve ${acceptedDays.length} days`}
            android_ripple={finalize.isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.approveBtn,
                  pressed && Platform.OS === 'ios' && !finalize.isPending && styles.approveBtnPressed,
                ]}
              >
                {finalize.isPending ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text style={styles.approveText}>Approve &amp; pay</Text>
                )}
              </View>
            )}
          </Pressable>
        </View>
      ) : canCancelMealPlan(plan) ? (
        <View style={styles.footer}>
          <Pressable
            style={styles.footerColNarrow}
            onPress={handleCancel}
            disabled={cancel.isPending}
            accessibilityRole="button"
            accessibilityLabel="Cancel plan"
            android_ripple={cancel.isPending ? undefined : { color: GHOST_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.rejectBtn,
                  pressed && Platform.OS === 'ios' && !cancel.isPending && styles.rejectBtnPressed,
                ]}
              >
                {cancel.isPending ? (
                  <ActivityIndicator color={customerColors.destructive.DEFAULT} />
                ) : (
                  <Text style={styles.rejectText}>Cancel plan</Text>
                )}
              </View>
            )}
          </Pressable>
        </View>
      ) : null}
    </SafeAreaView>
  );
}

function Header() {
  return (
    <View style={styles.header}>
      <Pressable
        onPress={() => router.back()}
        hitSlop={8}
        accessibilityRole="button"
        accessibilityLabel="Go back"
        android_ripple={{ color: ICON_RIPPLE, borderless: true }}
      >
        <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
      </Pressable>
      <Text style={styles.title}>Meal plan</Text>
      <View style={{ width: 24 }} />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  muted: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 18,
    color: customerColors.charcoal.DEFAULT,
  },
  // The ScrollView MUST be flex:1 so it's bounded to the space between the header and
  // the sticky footer. Without it the ScrollView grows to its content and the footer
  // sibling gets pushed off the bottom edge (the Approve/Reject bar went invisible).
  scrollFill: { flex: 1 },
  // Extra bottom padding so the last row clears the absolutely-pinned action bar (~92px).
  scroll: { padding: 16, paddingBottom: 108 },
  // Footer button wrappers carry the flex so the Pressables (not just their inner Views)
  // span the full width — Reject 1 : Approve 2. Without this the Pressables shrink to
  // their text and cram into the left half.
  footerColNarrow: { flex: 1 },
  footerColWide: { flex: 2 },
  summaryRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 16,
  },
  planNo: { fontFamily: 'Inter-Medium', fontSize: 13, color: customerColors.charcoal.soft },
  chefName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
    marginTop: 2,
  },
  chip: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 999 },
  chipText: { fontFamily: 'Inter-SemiBold', fontSize: 12 },
  banner: {
    backgroundColor: customerColors.coral.tint,
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
    gap: 4,
  },
  bannerTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.coral.DEFAULT,
  },
  bannerText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 19,
  },
  // #617 — bulk-confirm banner (coral tint) above the day list.
  confirmBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    backgroundColor: customerColors.coral.tint,
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
  },
  confirmBannerTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.coral.DEFAULT,
  },
  confirmBannerText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 19,
    marginTop: 2,
  },
  confirmTodayBtn: {
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    minHeight: 40,
    paddingHorizontal: 14,
    alignItems: 'center',
    justifyContent: 'center',
  },
  confirmTodayBtnPressed: { backgroundColor: customerColors.coral.pressed },
  confirmTodayText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.canvas,
  },
  totalRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 16,
    paddingHorizontal: 4,
  },
  totalLabel: { fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.soft },
  totalValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 22,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  // Sticky action bar — white, top hairline + shadow[2] (spec §1 floating elements).
  // Absolutely pinned to the bottom so it renders at its natural height. In the flex
  // column the footer was being shrunk to ~0px (its 52px buttons collapsed, leaving
  // only a hairline visible); position:absolute takes it out of flex negotiation.
  // styles.scroll pads the content so the last row clears this bar.
  footer: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: 'row',
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 28,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    shadowColor: customerTheme.shadow[2].shadowColor,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: customerTheme.shadow[2].shadowOpacity,
    shadowRadius: customerTheme.shadow[2].shadowRadius,
    elevation: customerTheme.shadow[2].elevation,
  },
  rejectBtn: {
    flex: 1,
    height: 52,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.hairline,
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
    flex: 2,
    height: 52,
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  approveBtnPressed: { backgroundColor: customerColors.coral.pressed },
  approveText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas },
});
