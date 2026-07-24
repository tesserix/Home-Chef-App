import React from 'react';
import { Platform, Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { AlertCircle, Wallet as WalletIcon, ArrowDownLeft, ArrowUpRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import { useWallet, useWalletTransactions } from '../hooks/useWallet';

// Android ripple tint — translucent token, never a new literal colour.
const CANVAS_RIPPLE = `${customerColors.canvas}33`;

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

const cardShadow = {
  shadowColor: customerColors.charcoal.DEFAULT,
  shadowOffset: { width: 0, height: 1 },
  shadowOpacity: 0.06,
  shadowRadius: 4,
  elevation: 2,
} as const;

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
      <View className="rounded-xl overflow-hidden bg-canvas" style={cardShadow}>
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
  );
}

// ─── Error state (R8) ─────────────────────────────────────────────────────────

function WalletErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <View className="flex-1 items-center justify-center px-8 gap-4 pt-16">
      <View className="w-16 h-16 rounded-full bg-surface-soft items-center justify-center">
        <AlertCircle size={28} color={customerColors.charcoal.soft} />
      </View>
      <Text className="text-lg font-semibold text-charcoal text-center font-display">
        Something went wrong
      </Text>
      <Text className="text-sm text-charcoal-soft text-center">
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
            <Text className="text-canvas font-semibold text-sm">Try again</Text>
          </View>
        )}
      </Pressable>
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
  const currency = wallet?.currency ?? 'INR';

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
            {/* Balance card */}
            <View
              className="rounded-2xl bg-canvas p-5"
              style={cardShadow}
            >
              <View className="flex-row items-center">
                <View className="w-12 h-12 rounded-full bg-surface-soft items-center justify-center mr-3">
                  <WalletIcon size={24} color={customerColors.charcoal.soft} />
                </View>
                <View>
                  <Text className="text-xs font-semibold text-charcoal-soft">
                    Available balance
                  </Text>
                  <Text
                    className="text-3xl font-bold text-charcoal"
                    style={{ fontVariant: ['tabular-nums'] }}
                  >
                    {formatMoney(wallet?.balance ?? 0, currency)}
                  </Text>
                </View>
              </View>
            </View>

            <Text className="text-xs font-semibold text-charcoal-soft px-1 pt-6 pb-2">
              Transactions
            </Text>

            {txnLoading ? (
              <View className="rounded-xl overflow-hidden bg-canvas" style={cardShadow}>
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
            ) : txnError ? (
              <View className="rounded-xl bg-canvas p-6 items-center gap-3" style={cardShadow}>
                <Text className="text-charcoal-soft text-center">
                  Could not load your transactions.
                </Text>
                <Pressable
                  onPress={() => void refetchTxns()}
                  accessibilityRole="button"
                  accessibilityLabel="Retry loading transactions"
                  android_ripple={{ color: `${customerColors.charcoal.DEFAULT}14`, borderless: false }}
                >
                  <View className="min-h-[40px] px-4 items-center justify-center rounded-lg border border-hairline">
                    <Text className="text-sm font-semibold text-charcoal">Try again</Text>
                  </View>
                </Pressable>
              </View>
            ) : txns.length === 0 ? (
              <View className="rounded-xl bg-canvas p-8 items-center" style={cardShadow}>
                <Text className="text-charcoal-soft">
                  No transactions yet. Refunds and credits will appear here.
                </Text>
              </View>
            ) : (
              <View className="rounded-xl overflow-hidden bg-canvas" style={cardShadow}>
                {txns.map((t, i) => {
                  const credit = t.type === 'credit';
                  return (
                    <View key={t.id}>
                      {i > 0 && <View className="h-px bg-hairline ml-16" />}
                      <View className="flex-row items-center px-4 py-3 min-h-[56px]">
                        <View className="w-9 h-9 rounded-full bg-surface-soft items-center justify-center mr-3">
                          {credit ? (
                            <ArrowDownLeft size={16} color={customerColors.charcoal.soft} />
                          ) : (
                            <ArrowUpRight size={16} color={customerColors.charcoal.soft} />
                          )}
                        </View>
                        <View className="flex-1">
                          <Text className="text-base text-charcoal">{sourceLabel(t.source)}</Text>
                          <Text className="text-xs text-charcoal-soft">
                            {new Date(t.createdAt).toLocaleDateString()}
                            {t.reason ? ` · ${t.reason}` : ''}
                          </Text>
                        </View>
                        <Text
                          className="text-base font-semibold text-charcoal"
                          style={{ fontVariant: ['tabular-nums'] }}
                        >
                          {credit ? '+' : '−'}
                          {formatMoney(t.amount, t.currency)}
                        </Text>
                      </View>
                    </View>
                  );
                })}
              </View>
            )}
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
