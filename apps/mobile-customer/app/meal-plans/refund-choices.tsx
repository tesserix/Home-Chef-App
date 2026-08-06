import { useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { CheckCircle2, ChevronLeft, Clock, Wallet } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

import {
  useCustomerRefundChoices,
  useChooseRefundMedium,
  type RefundChoiceDay,
} from '../../hooks/useMealPlans';
import { formatMoney } from '../../lib/format';
import { HAIRLINE } from '../../lib/hairline';

// Slot → display label ('lunch' → 'Lunch'). Trivial + local; not worth a lib export.
const slotLabel = (slot: string) => (slot ? slot.charAt(0).toUpperCase() + slot.slice(1) : '');

// Refund choices inbox (v2 refund flow, RBI): the chef has agreed a refund (Full/Half) on a
// skipped/cancelled day and — per RBI — the CUSTOMER decides where it goes. Wallet is instant store
// credit (reusable); the original method is a real reversal in ~5–7 business days. Each amount here
// covers the food + that day's delivery fee, excluding GST + the platform fee. The ≤12h counterpart of the immediate
// prompt in useSkipDayFlow (there the customer is present; here they choose later, from a notification).

const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export default function RefundChoicesScreen() {
  const { data, isLoading, refetch, isRefetching } = useCustomerRefundChoices();
  const choices = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
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
        <Text style={styles.title}>Choose your refund</Text>
        <View style={{ width: 24 }} />
      </View>

      {isLoading ? (
        <View style={styles.center}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : (
        <FlatList
          data={choices}
          keyExtractor={(c) => c.dayId}
          renderItem={({ item }) => <RefundCard choice={item} />}
          contentContainerStyle={styles.listContent}
          onRefresh={refetch}
          refreshing={isRefetching}
          ListHeaderComponent={
            choices.length > 0 ? (
              <Text style={styles.intro}>
                Your chef agreed a refund. Pick where it goes — your HomeChef wallet is instant, or
                we can reverse it to your original payment method (per RBI, that takes 5–7 business
                days). It covers the food and that day’s delivery fee; the GST and platform fee aren’t
                refundable.
              </Text>
            ) : null
          }
          ListEmptyComponent={
            <View style={styles.empty}>
              <View style={styles.emptyIconWrap}>
                <CheckCircle2 size={30} color={customerColors.charcoal.soft} strokeWidth={1.5} />
              </View>
              <Text style={styles.emptyTitle}>No refunds to choose</Text>
              <Text style={styles.emptyText}>
                When a chef agrees a refund on a skipped or cancelled day, it’ll show up here for you
                to send to your wallet or original payment method.
              </Text>
            </View>
          }
        />
      )}
    </SafeAreaView>
  );
}

function RefundCard({ choice }: { choice: RefundChoiceDay }) {
  const choose = useChooseRefundMedium();
  // Track which medium is being submitted so only the tapped button spins.
  const [pending, setPending] = useState<'wallet' | 'source' | null>(null);

  function pick(medium: 'wallet' | 'source') {
    if (choose.isPending) return;
    setPending(medium);
    choose.mutate(
      { planId: choice.mealPlanId, dayId: choice.dayId, medium },
      { onSettled: () => setPending(null) },
    );
  }

  const busy = choose.isPending;

  return (
    <View style={styles.card}>
      <View style={styles.cardTop}>
        <Text style={styles.planNo}>{choice.mealPlanNumber}</Text>
        <Text style={styles.amount}>{formatMoney(choice.amount)}</Text>
      </View>
      <Text style={styles.dish} numberOfLines={1}>
        {choice.dishName || 'Meal'}
      </Text>
      <Text style={styles.meta}>
        {choice.date} · {slotLabel(choice.slot)}
      </Text>

      <View style={styles.actions}>
        <Pressable
          onPress={() => pick('wallet')}
          disabled={busy}
          accessibilityRole="button"
          accessibilityLabel="Refund to HomeChef wallet, instant"
          style={[styles.btn, styles.btnWallet, busy && styles.btnDisabled]}
        >
          {pending === 'wallet' ? (
            <ActivityIndicator size="small" color={customerColors.canvas} />
          ) : (
            <>
              <Wallet size={16} color={customerColors.canvas} />
              <Text style={styles.btnWalletText}>Wallet · instant</Text>
            </>
          )}
        </Pressable>

        <Pressable
          onPress={() => pick('source')}
          disabled={busy}
          accessibilityRole="button"
          accessibilityLabel="Refund to original payment method, 5 to 7 business days"
          style={[styles.btn, styles.btnSource, busy && styles.btnDisabled]}
        >
          {pending === 'source' ? (
            <ActivityIndicator size="small" color={customerColors.charcoal.DEFAULT} />
          ) : (
            <>
              <Clock size={16} color={customerColors.charcoal.DEFAULT} />
              <Text style={styles.btnSourceText}>Original · 5–7 days</Text>
            </>
          )}
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 18, color: customerColors.charcoal.DEFAULT },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  listContent: { padding: 16, paddingTop: 8, gap: 12, flexGrow: 1 },
  intro: {
    fontFamily: 'Inter',
    fontSize: 13,
    lineHeight: 19,
    color: customerColors.charcoal.soft,
    marginBottom: 4,
  },
  card: {
    backgroundColor: customerColors.canvas,
    borderRadius: 12,
    borderWidth: HAIRLINE,
    borderColor: customerColors.hairline,
    padding: 16,
    gap: 4,
  },
  cardTop: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  planNo: {
    fontFamily: 'Inter-Medium',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  amount: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  dish: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.charcoal.DEFAULT },
  meta: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  actions: { flexDirection: 'row', gap: 10, marginTop: 12 },
  btn: {
    flex: 1,
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    borderRadius: 8,
    paddingHorizontal: 8,
  },
  btnDisabled: { opacity: 0.6 },
  btnWallet: { backgroundColor: customerColors.coral.DEFAULT },
  btnWalletText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.canvas },
  btnSource: {
    backgroundColor: customerColors.surface.soft,
    borderWidth: HAIRLINE,
    borderColor: customerColors.hairline,
  },
  btnSourceText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
  },
  empty: { alignItems: 'center', paddingHorizontal: 32, paddingTop: 80, gap: 4, flex: 1 },
  emptyIconWrap: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  emptyTitle: { fontFamily: 'Inter-SemiBold', fontSize: 17, color: customerColors.charcoal.DEFAULT },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 20,
  },
});
