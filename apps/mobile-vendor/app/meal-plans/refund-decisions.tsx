// Chef refund decisions (refund policy v3 — docs/refund-policy-v3-spec.md, #834). A customer's
// late skip/cancel routes here: the chef may have started prep, so they choose HOW MUCH to refund
// — anything from the day's lead-time floor up to the whole amount — or Decline (they'll serve it).
// The base is everything the customer paid for the day: food, GST and delivery. The floor is
// pinned when the request is raised and enforced server-side; this screen only makes it visible.
// Once the chef decides, the CUSTOMER picks where the money goes (wallet or original method).

import { useState } from 'react';
import {
  ActivityIndicator,
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
  refundAtPercent,
  useChefPendingRefundDecisions,
  useChefRefundDecision,
  type RefundDecisionDay,
} from '../../hooks/useMealPlans';
import { useAlert } from '@homechef/mobile-shared/ui';

function dayLabel(iso: string): string {
  const d = new Date(iso + 'T00:00:00');
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' });
}

function money(n: number): string {
  return `₹${Math.round(n) === n ? n : n.toFixed(2)}`;
}

export default function RefundDecisionsScreen() {
  const { showAlert } = useAlert();
  const { data, isLoading, isError, refetch, isRefetching } = useChefPendingRefundDecisions();
  const decide = useChefRefundDecision();
  const days = data?.data ?? [];

  function act(d: RefundDecisionDay, percent: number, decline?: boolean) {
    const amount = refundAtPercent(d, percent);
    const title = decline
      ? 'Keep this day?'
      : percent === 0
        ? 'Refund nothing?'
        : `Refund ${money(amount)}?`;
    const body = decline
      ? "You'll cook and deliver this day as planned — the customer is charged in full."
      : percent === 0
        ? "No refund — you keep the full payout (you'd started prep). The day won't be delivered."
        : `${money(amount)} (${percent}% of this day's refundable value) goes back to the customer; your payout for this day is reduced by the same share. They choose where it lands.`;
    showAlert(title, body, [
      { text: 'Back', style: 'cancel' },
      {
        text: decline ? 'Keep it' : 'Confirm',
        style: percent > 0 ? 'destructive' : 'default',
        onPress: () =>
          decide.mutate(
            { dayId: d.dayId, percent, decline },
            { onError: () => showAlert('Something went wrong', 'Please try again.') },
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
          style={styles.backBtn}
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
                When a customer skips or cancels a day at short notice, it shows here for you to
                decide how much to refund.
              </Text>
            </View>
          ) : (
            days.map((d) => (
              <DecisionCard key={d.dayId} day={d} busy={decide.isPending} onAct={act} />
            ))
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

/** One request, with the chef's amount picker. Opens at the FLOOR — the option that
 *  protects a chef who has already started cooking, and the one they most often want. */
function DecisionCard({
  day,
  busy,
  onAct,
}: {
  day: RefundDecisionDay;
  busy: boolean;
  onAct: (d: RefundDecisionDay, percent: number, decline?: boolean) => void;
}) {
  const [percent, setPercent] = useState(day.minPercent);
  // Presets inside the permitted range, deduplicated — the common answers stay one tap away.
  const presets = [day.minPercent, 75, 100].filter(
    (p, i, all) => p >= day.minPercent && all.indexOf(p) === i,
  );
  const step = (delta: number) =>
    setPercent((p) => Math.min(100, Math.max(day.minPercent, p + delta)));

  return (
    <View style={styles.card}>
      <View style={styles.cardHead}>
        <Text style={styles.cardDate}>{dayLabel(day.date)}</Text>
        <Text style={styles.cardSlot}>{day.slot === 'lunch' ? 'Lunch' : 'Dinner'}</Text>
      </View>
      <Text style={styles.cardDish}>{day.dishName || '—'}</Text>
      <Text style={styles.cardSub}>
        {day.customerName || 'Customer'} · {day.mealPlanNumber} · food {money(day.foodPrice)}
      </Text>

      <View style={styles.picker}>
        <View style={styles.pickerRow}>
          <Pressable
            disabled={busy || percent <= day.minPercent}
            onPress={() => step(-5)}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Refund less"
            style={[
              styles.stepBtn,
              (busy || percent <= day.minPercent) && styles.btnDisabled,
            ]}
          >
            <Text style={styles.stepBtnText}>−</Text>
          </Pressable>
          <View style={styles.pickerValue}>
            <Text style={styles.pickerAmount}>{money(refundAtPercent(day, percent))}</Text>
            <Text style={styles.pickerPercent}>{percent}% of the day's refund</Text>
          </View>
          <Pressable
            disabled={busy || percent >= 100}
            onPress={() => step(5)}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Refund more"
            style={[styles.stepBtn, (busy || percent >= 100) && styles.btnDisabled]}
          >
            <Text style={styles.stepBtnText}>+</Text>
          </Pressable>
        </View>
        <View style={styles.presetRow}>
          {presets.map((p) => (
            <Pressable
              key={p}
              disabled={busy}
              onPress={() => setPercent(p)}
              style={[styles.preset, percent === p && styles.presetOn]}
            >
              <Text style={[styles.presetText, percent === p && styles.presetTextOn]}>{p}%</Text>
            </Pressable>
          ))}
        </View>
        <Text style={styles.pickerHint}>
          {day.minPercent > 0
            ? `At this much notice the least you can refund is ${day.minPercent}% (${money(day.minRefund)}).`
            : 'The meal is imminent, so no refund is owed — but you can still give one.'}
        </Text>
      </View>

      <View style={styles.actions}>
        <Pressable
          disabled={busy}
          onPress={() => onAct(day, percent)}
          style={[styles.btnPrimary, busy && styles.btnDisabled]}
        >
          <Text style={styles.btnPrimaryText}>Refund {money(refundAtPercent(day, percent))}</Text>
        </Pressable>
        <Pressable
          disabled={busy}
          onPress={() => onAct(day, 0, true)}
          style={[styles.btnGhost, busy && styles.btnDisabled]}
        >
          <Text style={styles.btnGhostText}>Keep this day — cook it as planned</Text>
        </Pressable>
      </View>
    </View>
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
  picker: {
    marginTop: 12,
    gap: 10,
    padding: 12,
    borderRadius: 12,
    backgroundColor: theme.colors.paper,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  pickerRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  pickerValue: { alignItems: 'center', flex: 1 },
  pickerAmount: { fontFamily: 'Geist-SemiBold', fontSize: 22, color: theme.colors.ink.DEFAULT },
  pickerPercent: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft, marginTop: 2 },
  pickerHint: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.muted, lineHeight: 17 },
  stepBtn: {
    width: 48,
    height: 48,
    borderRadius: 10,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  stepBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 22, color: theme.colors.ink.DEFAULT },
  presetRow: { flexDirection: 'row', gap: 8 },
  preset: {
    flex: 1,
    minHeight: 44,
    borderRadius: 10,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  presetOn: { backgroundColor: theme.colors.ink.DEFAULT, borderColor: theme.colors.ink.DEFAULT },
  presetText: { fontFamily: 'Inter-Medium', fontSize: 14, color: theme.colors.ink.soft },
  presetTextOn: { color: theme.colors.paper },
  btnPrimary: { backgroundColor: theme.colors.ink.DEFAULT, borderRadius: 10, minHeight: 48, alignItems: 'center', justifyContent: 'center' },
  btnPrimaryText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.paper },
  btnOutline: { borderRadius: 10, minHeight: 48, alignItems: 'center', justifyContent: 'center', borderWidth: StyleSheet.hairlineWidth, borderColor: theme.colors.ink.DEFAULT },
  btnOutlineText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  btnGhost: { borderRadius: 10, minHeight: 44, alignItems: 'center', justifyContent: 'center', borderWidth: StyleSheet.hairlineWidth, borderColor: theme.colors.mist.DEFAULT },
  btnGhostText: { fontFamily: 'Inter-Medium', fontSize: 14, color: theme.colors.ink.soft },
  btnDisabled: { opacity: 0.5 },
});
