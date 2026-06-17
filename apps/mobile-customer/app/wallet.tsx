import React from 'react';
import { ActivityIndicator, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Stack } from 'expo-router';
import { Wallet as WalletIcon, ArrowDownLeft, ArrowUpRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useWallet, useWalletTransactions } from '../hooks/useWallet';

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

export default function WalletScreen() {
  const { data: wallet, isLoading } = useWallet();
  const { data: txns = [], isLoading: txnLoading } = useWalletTransactions();
  const currency = wallet?.currency ?? 'INR';

  return (
    <SafeAreaView edges={['bottom']} className="flex-1 bg-canvas">
      <Stack.Screen options={{ title: 'Wallet' }} />
      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator color={customerColors.charcoal.soft} />
        </View>
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
                  <Text className="text-xs font-semibold text-charcoal-soft uppercase tracking-wide">
                    Available balance
                  </Text>
                  <Text className="text-3xl font-bold text-charcoal">
                    {formatMoney(wallet?.balance ?? 0, currency)}
                  </Text>
                </View>
              </View>
            </View>

            <Text className="text-xs font-semibold text-charcoal-soft uppercase tracking-wide px-1 pt-6 pb-2">
              Transactions
            </Text>

            {txnLoading ? (
              <ActivityIndicator className="mt-4" color={customerColors.charcoal.soft} />
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
                        <Text className="text-base font-semibold text-charcoal">
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
