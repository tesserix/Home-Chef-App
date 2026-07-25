// Chef refund decisions (v2 refund flow — docs/meal-plan-refund-flow-design.md). A customer's
// skip/cancel that lands within 12h of cook-start routes here: the chef may have started prep, so
// they choose how much of the food to refund — Full / Half / None — or Decline (they'll serve it).
// The amounts cover the food + that day's delivery fee, excluding GST + the platform fee. After the chef decides Full/Half, an
// admin pays it to the customer's wallet or original method.

import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft, Inbox } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';

import {
  useChefPendingRefundDecisions,
  useChefRefundDecision,
  type RefundDecisionDay,
} from '../../hooks/useMealPlans';

function dayLabel(iso: string): string {
  const d = new Date(iso + 'T00:00:00');
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' });
}

function money(n: number): string {
  return `₹${Math.round(n) === n ? n : n.toFixed(2)}`;
}

export default function RefundDecisionsScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useChefPendingRefundDecisions();
  const decide = useChefRefundDecision();
  const days = data?.data ?? [];

  function act(d: RefundDecisionDay, choice?: 'full' | 'half' | 'none', decline?: boolean) {
    const title = decline
      ? 'Keep this day?'
      : choice === 'none'
        ? 'Refund nothing?'
        : `Refund ${money(choice === 'half' ? d.halfRefund : d.fullRefund)}?`;
    const body = decline
      ? "You'll cook and deliver this day as planned — the customer is charged in full."
      : choice === 'none'
        ? "No refund — you keep the full payout (you'd started prep). The day won't be delivered."
        : choice === 'half'
          ? `Half the food refunded to the customer; you keep the other half. An admin pays it out.`
          : `Full food refunded to the customer; your payout for this day is reversed. An admin pays it out.`;
    Alert.alert(title, body, [
      { text: 'Back', style: 'cancel' },
      {
        text: decline ? 'Keep it' : 'Confirm',
        style: choice === 'full' || choice === 'half' ? 'destructive' : 'default',
        onPress: () =>
          decide.mutate(
            { dayId: d.dayId, choice, decline },
            { onError: () => Alert.alert('Something went wrong', 'Please try again.') },
          ),
      },
    ]);
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Back"
          style={({ pressed }) => [styles.backBtn, pressed && Platform.OS === 'ios' && { opacity: 0.6 }]}
        >
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Refund requests</Text>
      </View>

      {isLoading ? (
        <View style={styles.centre}>
          <ActivityIndicator color={theme.colors.ink.DEFAULT} />
        </View>
      ) : isError ? (
        <View style={styles.centre}>
          <Text style={styles.muted}>Couldn't load requests.</Text>
          <Pressable onPress={() => void refetch()} style={styles.retry}>
            <Text style={styles.retryText}>Try again</Text>
          </Pressable>
        </View>
      ) : (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={<RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} tintColor={theme.colors.ink.soft} />}
        >
          {days.length === 0 ? (
            <View style={styles.empty}>
              <Inbox size={40} color={theme.colors.ink.muted} />
              <Text style={styles.emptyTitle}>No refund requests</Text>
              <Text style={styles.muted}>
                When a customer skips or cancels a day within 12 hours of cooking, it shows here for
                you to decide.
              </Text>
            </View>
          ) : (
            days.map((d) => {
              const busy = decide.isPending;
              return (
                <View key={d.dayId} style={styles.card}>
                  <View style={styles.cardHead}>
                    <Text style={styles.cardDate}>{dayLabel(d.date)}</Text>
                    <Text style={styles.cardSlot}>{d.slot === 'lunch' ? 'Lunch' : 'Dinner'}</Text>
                  </View>
                  <Text style={styles.cardDish}>{d.dishName || '—'}</Text>
                  <Text style={styles.cardSub}>
                    {d.customerName || 'Customer'} · {d.mealPlanNumber} · food {money(d.foodPrice)}
                  </Text>

                  <View style={styles.actions}>
                    <Pressable
                      disabled={busy}
                      onPress={() => act(d, 'full')}
                      style={({ pressed }) => [styles.btnPrimary, pressed && { opacity: 0.9 }, busy && styles.btnDisabled]}
                    >
                      <Text style={styles.btnPrimaryText}>Refund {money(d.fullRefund)} · full</Text>
                    </Pressable>
                    <Pressable
                      disabled={busy}
                      onPress={() => act(d, 'half')}
                      style={({ pressed }) => [styles.btnOutline, pressed && { opacity: 0.7 }, busy && styles.btnDisabled]}
                    >
                      <Text style={styles.btnOutlineText}>Refund {money(d.halfRefund)} · half</Text>
                    </Pressable>
                    <View style={styles.actionRow}>
                      <Pressable
                        disabled={busy}
                        onPress={() => act(d, 'none')}
                        style={({ pressed }) => [styles.btnGhost, pressed && { opacity: 0.7 }, busy && styles.btnDisabled]}
                      >
                        <Text style={styles.btnGhostText}>No refund</Text>
                      </Pressable>
                      <Pressable
                        disabled={busy}
                        onPress={() => act(d, undefined, true)}
                        style={({ pressed }) => [styles.btnGhost, pressed && { opacity: 0.7 }, busy && styles.btnDisabled]}
                      >
                        <Text style={styles.btnGhostText}>Keep this day</Text>
                      </Pressable>
                    </View>
                  </View>
                </View>
              );
            })
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  header: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: 12, paddingVertical: 8, gap: 4 },
  backBtn: { width: 40, height: 40, alignItems: 'center', justifyContent: 'center' },
  title: { fontFamily: 'Geist-SemiBold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  centre: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 12, padding: 24 },
  scroll: { padding: 16, gap: 12, paddingBottom: 40 },
  muted: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.soft, textAlign: 'center', lineHeight: 20 },
  retry: { paddingHorizontal: 16, paddingVertical: 8, borderRadius: 8, borderWidth: StyleSheet.hairlineWidth, borderColor: theme.colors.mist.DEFAULT },
  retryText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT },
  empty: { alignItems: 'center', gap: 10, paddingVertical: 64, paddingHorizontal: 24 },
  emptyTitle: { fontFamily: 'Geist-SemiBold', fontSize: 17, color: theme.colors.ink.DEFAULT },
  card: {
    backgroundColor: theme.colors.bone,
    borderRadius: 16,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    padding: 16,
    gap: 4,
  },
  cardHead: { flexDirection: 'row', alignItems: 'baseline', justifyContent: 'space-between' },
  cardDate: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT },
  cardSlot: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft },
  cardDish: { fontFamily: 'Inter-Medium', fontSize: 15, color: theme.colors.ink.DEFAULT, marginTop: 2 },
  cardSub: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft, marginTop: 2 },
  actions: { marginTop: 12, gap: 8 },
  actionRow: { flexDirection: 'row', gap: 8 },
  btnPrimary: { backgroundColor: theme.colors.ink.DEFAULT, borderRadius: 10, minHeight: 48, alignItems: 'center', justifyContent: 'center' },
  btnPrimaryText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.paper },
  btnOutline: { borderRadius: 10, minHeight: 48, alignItems: 'center', justifyContent: 'center', borderWidth: StyleSheet.hairlineWidth, borderColor: theme.colors.ink.DEFAULT },
  btnOutlineText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  btnGhost: { flex: 1, borderRadius: 10, minHeight: 44, alignItems: 'center', justifyContent: 'center', borderWidth: StyleSheet.hairlineWidth, borderColor: theme.colors.mist.DEFAULT },
  btnGhostText: { fontFamily: 'Inter-Medium', fontSize: 14, color: theme.colors.ink.soft },
  btnDisabled: { opacity: 0.5 },
});
