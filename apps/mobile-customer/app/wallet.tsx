import React from 'react';
import { Platform, Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { AlertCircle, ArrowDownLeft, ArrowUpRight } from 'lucide-react-native';
import { router } from 'expo-router';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import { BalanceCard } from '../components/wallet/BalanceCard';
import { useWallet, useWalletTransactions } from '../hooks/useWallet';
import { useLoyalty } from '../hooks/useLoyalty';

// Android ripple tint — translucent token, never a new literal colour.
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const CHARCOAL_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

function formatMoney(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: currency || 'INR',
    }).format(amount);
  } catch {
    return `${currency || 'INR'} ${amount.toFixed(2)}`;
  }
}

function sourceLabel(s: string): string {
  return s.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

// Shadow[2] — canvas and the card surface are both pure white, so a hairline
// alone doesn't separate a card from the page; this is the actual lift.
// Lives on the OUTER, unclipped view of any card — pairing this with
// `overflow: hidden` on the same view clips the shadow to nothing on iOS,
// which is why every rounded-corner + hairline-divided list below is built
// as an outer/inner pair (cardShadow outside, overflow-hidden inside).
const cardShadow = {
  shadowColor: '#000000',
  shadowOffset: { width: 0, height: 4 },
  shadowOpacity: 0.1,
  shadowRadius: 12,
  elevation: 4,
} as const;

// A fixed-width, right-aligned column so every amount's decimal point lines
// up under the one above it — a wallet is a ledger, and ledgers align.
// Paired with tabular-nums (equal-width digits) on the Text itself.
const AMOUNT_COLUMN_WIDTH = 100;

// ─── Loading skeleton — matches balance card + transaction row proportions (R8) ─

function WalletSkeleton() {
  return (
    <View className="p-4">
      <View className="rounded-2xl bg-canvas p-5" style={cardShadow}>
        <View className="flex-row items-center">
          <View className="w-12 h-12 rounded-full bg-surface-soft mr-3" />
          <View className="gap-2">
            <View className="h-3 rounded bg-hairline" style={{ width: 110 }} />
            <View className="h-7 rounded bg-hairline" style={{ width: 140 }} />
          </View>
        </View>
      </View>
      <View className="h-3 rounded bg-hairline mt-8 mb-3 ml-1" style={{ width: 96 }} />
      <View className="rounded-2xl bg-canvas" style={cardShadow}>
        <View className="overflow-hidden rounded-2xl">
          {[0, 1, 2].map((i) => (
            <View key={i}>
              {i > 0 && <View className="h-px bg-hairline ml-16" />}
              <View className="flex-row items-center px-4 py-3 min-h-[56px]">
                <View className="w-9 h-9 rounded-full bg-surface-soft mr-3" />
                <View className="flex-1 gap-2">
                  <View className="h-3.5 rounded bg-hairline" style={{ width: '55%' }} />
                  <View className="h-3 rounded bg-hairline" style={{ width: '35%' }} />
                </View>
                <View className="h-4 rounded bg-hairline" style={{ width: 56 }} />
              </View>
            </View>
          ))}
        </View>
      </View>
    </View>
  );
}

// ─── Error state (R8) ─────────────────────────────────────────────────────────

function WalletErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <View className="flex-1 items-center justify-center px-8 gap-4 pt-16">
      <View className="w-16 h-16 rounded-full bg-surface-soft items-center justify-center">
        <AlertCircle size={28} color={customerColors.charcoal.soft} />
      </View>
      <Text
        className="text-lg text-charcoal text-center"
        style={{ fontFamily: 'Geist-Bold' }}
      >
        Something went wrong
      </Text>
      <Text
        className="text-sm text-charcoal-soft text-center"
        style={{ fontFamily: 'Inter' }}
      >
        We could not load your wallet. Please try again.
      </Text>
      <Pressable
        onPress={onRetry}
        accessibilityRole="button"
        accessibilityLabel="Retry loading wallet"
        android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
      >
        {({ pressed }) => (
          <View
            className={`bg-coral rounded-lg px-6 py-3 min-h-[44px] items-center justify-center ${
              pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : ''
            }`}
          >
            <Text
              className="text-canvas text-sm"
              style={{ fontFamily: 'Inter-SemiBold' }}
            >
              Try again
            </Text>
          </View>
        )}
      </Pressable>
    </View>
  );
}

// ─── Transaction row ───────────────────────────────────────────────────────────

function TransactionRow({
  transaction,
  isFirst,
}: {
  transaction: {
    id: string;
    type: 'credit' | 'debit';
    source: string;
    amount: number;
    currency: string;
    reason?: string;
    createdAt: string;
  };
  isFirst: boolean;
}) {
  const credit = transaction.type === 'credit';
  const dateLabel = new Date(transaction.createdAt).toLocaleDateString();
  const amountLabel = formatMoney(transaction.amount, transaction.currency);
  const label = sourceLabel(transaction.source);

  return (
    <View>
      {!isFirst && <View className="h-px bg-hairline ml-16" />}
      <View
        className="flex-row items-center px-4 py-3 min-h-[56px]"
        // One coherent sentence for a screen reader — "Credit of ₹120.00 for
        // Refund, 6 Aug 2026" — rather than three unrelated fragments.
        accessible
        accessibilityLabel={`${credit ? 'Credit' : 'Debit'} of ${amountLabel} for ${label}${
          transaction.reason ? `, ${transaction.reason}` : ''
        }, on ${dateLabel}`}
      >
        {/* Direction is legible before the label is read: tinted green for
            money in, neutral for money out — matches the checkout / receipt
            convention rather than treating coral (an accent, not a status
            colour) as a loss indicator. */}
        <View
          className="w-9 h-9 rounded-full items-center justify-center mr-3"
          style={{ backgroundColor: credit ? customerColors.success.tint : customerColors.surface.soft }}
        >
          {credit ? (
            <ArrowDownLeft size={16} color={customerColors.success.DEFAULT} />
          ) : (
            <ArrowUpRight size={16} color={customerColors.charcoal.soft} />
          )}
        </View>
        <View className="flex-1">
          <Text
            className="text-base text-charcoal"
            style={{ fontFamily: 'Inter-Medium' }}
            numberOfLines={1}
          >
            {label}
          </Text>
          <Text
            className="text-xs text-charcoal-soft"
            style={{ fontFamily: 'Inter', fontVariant: ['tabular-nums'] }}
            numberOfLines={1}
          >
            {dateLabel}
            {transaction.reason ? ` · ${transaction.reason}` : ''}
          </Text>
        </View>
        <Text
          className="text-base"
          style={{
            fontFamily: 'Inter-SemiBold',
            fontVariant: ['tabular-nums'],
            color: credit ? customerColors.success.DEFAULT : customerColors.charcoal.DEFAULT,
            width: AMOUNT_COLUMN_WIDTH,
            textAlign: 'right',
          }}
        >
          {credit ? '+' : '−'}
          {amountLabel}
        </Text>
      </View>
    </View>
  );
}

export default function WalletScreen() {
  const { data: wallet, isLoading, isError, refetch } = useWallet();
  const {
    data: txns = [],
    isLoading: txnLoading,
    isError: txnError,
    refetch: refetchTxns,
  } = useWalletTransactions();
  const { data: loyalty } = useLoyalty();
  const currency = wallet?.currency ?? 'INR';
  // Points are worth redeemRate rupees each; the rate is admin-tunable at runtime
  // so it must come from the live config, never a client constant.
  const pointsValue = (loyalty?.balance ?? 0) * (loyalty?.config?.redeemRate ?? 0);

  return (
    <SafeAreaView edges={['top', 'left', 'right']} className="flex-1 bg-canvas">
      <ScreenHeader title="Wallet" />
      {isLoading ? (
        <WalletSkeleton />
      ) : isError ? (
        <WalletErrorState onRetry={() => void refetch()} />
      ) : (
        <ScrollView>
          <View className="p-4">
            {/* One headline figure, two sources beneath. Wallet credit and points
                are both spendable on an order, so showing them apart on separate
                screens left the customer without a single answer to "how much can
                I spend?". */}
            <BalanceCard
              walletBalance={wallet?.balance ?? 0}
              pointsBalance={loyalty?.balance ?? 0}
              pointsValue={pointsValue}
              maxRedeemPct={loyalty?.config?.maxRedeemPct}
              monthlyCap={loyalty?.config?.monthlyRedeemCap}
              expiryDays={loyalty?.config?.expiryDays}
              currency={currency}
              onPressPoints={() => router.push('/loyalty')}
            />

            <Text
              className="text-xs text-charcoal-soft px-1 pt-6 pb-2"
              style={{ fontFamily: 'Inter-SemiBold', letterSpacing: 0.3 }}
            >
              Transactions
            </Text>

            {txnLoading ? (
              <View className="rounded-2xl bg-canvas" style={cardShadow}>
                <View className="overflow-hidden rounded-2xl">
                  {[0, 1].map((i) => (
                    <View key={i}>
                      {i > 0 && <View className="h-px bg-hairline ml-16" />}
                      <View className="flex-row items-center px-4 py-3 min-h-[56px]">
                        <View className="w-9 h-9 rounded-full bg-surface-soft mr-3" />
                        <View className="flex-1 gap-2">
                          <View className="h-3.5 rounded bg-hairline" style={{ width: '55%' }} />
                          <View className="h-3 rounded bg-hairline" style={{ width: '35%' }} />
                        </View>
                      </View>
                    </View>
                  ))}
                </View>
              </View>
            ) : txnError ? (
              <View className="rounded-2xl bg-canvas p-6 items-center gap-3" style={cardShadow}>
                <Text
                  className="text-charcoal-soft text-center"
                  style={{ fontFamily: 'Inter' }}
                >
                  Could not load your transactions.
                </Text>
                <Pressable
                  onPress={() => void refetchTxns()}
                  accessibilityRole="button"
                  accessibilityLabel="Retry loading transactions"
                  android_ripple={{ color: CHARCOAL_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      className={`min-h-[44px] px-4 items-center justify-center rounded-lg border border-hairline ${
                        pressed && Platform.OS === 'ios' ? 'bg-surface-soft' : ''
                      }`}
                    >
                      <Text
                        className="text-sm text-charcoal"
                        style={{ fontFamily: 'Inter-SemiBold' }}
                      >
                        Try again
                      </Text>
                    </View>
                  )}
                </Pressable>
              </View>
            ) : txns.length === 0 ? (
              <View className="rounded-2xl bg-canvas p-8 items-center" style={cardShadow}>
                <Text
                  className="text-charcoal-soft text-center"
                  style={{ fontFamily: 'Inter' }}
                >
                  No transactions yet. Refunds and credits will appear here.
                </Text>
              </View>
            ) : (
              <View className="rounded-2xl bg-canvas" style={cardShadow}>
                <View className="overflow-hidden rounded-2xl">
                  {txns.map((t, i) => (
                    <TransactionRow key={t.id} transaction={t} isFirst={i === 0} />
                  ))}
                </View>
              </View>
            )}
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
