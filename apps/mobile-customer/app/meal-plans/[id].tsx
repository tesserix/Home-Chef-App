import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  canCancelMealPlan,
  useCancelMealPlan,
  useFinalizeMealPlan,
  useMealPlan,
  useSkipMealPlanDay,
  type MealPlanDay,
} from '../../hooks/useMealPlans';
import { mealPlanStatusMeta, isDeclinedDayStatus } from '../../lib/meal-plan';
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
      approve ? 'Approve plan?' : 'Reject plan?',
      approve
        ? `Confirm the ${acceptedDays.length} day${acceptedDays.length === 1 ? '' : 's'} your chef can cook (₹${acceptedTotal.toFixed(0)}).`
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
                onSuccess: () =>
                  Alert.alert(
                    approve ? 'Plan confirmed' : 'Plan cancelled',
                    approve
                      ? 'Your chef has been notified.'
                      : 'No charge — the plan was cancelled.',
                    [{ text: 'OK', onPress: () => router.back() }],
                  ),
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
      'You won’t be charged for it. This can’t be undone, but you can book again any time.',
      [
        { text: 'Back', style: 'cancel' },
        {
          text: 'Skip day',
          style: 'destructive',
          onPress: () =>
            skipDay.mutate(
              { planId: plan.id, dayId },
              {
                onError: () =>
                  Alert.alert(
                    'Could not skip',
                    'It may be too close to the delivery day.',
                  ),
              },
            ),
        },
      ],
    );
  }

  // A confirmed day on a live plan can still be skipped (server enforces the cutoff).
  const canSkip =
    plan.status === 'confirmed' || plan.status === 'active';
  const needsApproval = meta.needsAction;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <Header />
      <ScrollView contentContainerStyle={styles.scroll}>
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

        <MealPlanDayList
          days={days}
          onSkip={canSkip ? confirmSkip : undefined}
          skipping={skipDay.isPending}
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
            onPress={() => act(false)}
            disabled={finalize.isPending}
            style={styles.rejectBtn}
            accessibilityRole="button"
          >
            <Text style={styles.rejectText}>Reject</Text>
          </Pressable>
          <Pressable
            onPress={() => act(true)}
            disabled={finalize.isPending}
            style={styles.approveBtn}
            accessibilityRole="button"
          >
            {finalize.isPending ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.approveText}>Approve {acceptedDays.length} days</Text>
            )}
          </Pressable>
        </View>
      ) : canCancelMealPlan(plan) ? (
        <View style={styles.footer}>
          <Pressable
            onPress={handleCancel}
            disabled={cancel.isPending}
            style={styles.rejectBtn}
            accessibilityRole="button"
          >
            {cancel.isPending ? (
              <ActivityIndicator color={customerColors.destructive.DEFAULT} />
            ) : (
              <Text style={styles.rejectText}>Cancel plan</Text>
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
      <Pressable onPress={() => router.back()} hitSlop={8} accessibilityLabel="Go back">
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
  scroll: { padding: 16, paddingBottom: 24 },
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
  },
  footer: {
    flexDirection: 'row',
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
  },
  rejectBtn: {
    flex: 1,
    height: 52,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    alignItems: 'center',
    justifyContent: 'center',
  },
  rejectText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
  },
  approveBtn: {
    flex: 2,
    height: 52,
    borderRadius: 12,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  approveText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: '#fff' },
});
