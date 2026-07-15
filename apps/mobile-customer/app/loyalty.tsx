import React from 'react';
import { ActivityIndicator, Alert, Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Award, Flame, Sparkles, ArrowDownLeft, ArrowUpRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import {
  useLoyalty,
  useLoyaltyTransactions,
  useRedeemLoyalty,
  loyaltyErrorMessage,
} from '../hooks/useLoyalty';

function formatMoney(amount: number): string {
  try {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(amount);
  } catch {
    return `₹${amount.toFixed(2)}`;
  }
}

function formatPoints(p: number): string {
  return Math.round(p).toLocaleString('en-IN');
}

function sourceLabel(s: string): string {
  return s.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

const TIER_LABEL: Record<string, string> = { bronze: 'Bronze', silver: 'Silver', gold: 'Gold' };

const cardShadow = {
  shadowColor: customerColors.charcoal.DEFAULT,
  shadowOffset: { width: 0, height: 1 },
  shadowOpacity: 0.06,
  shadowRadius: 4,
  elevation: 2,
} as const;

export default function LoyaltyScreen() {
  const router = useRouter();
  const { data: loyalty, isLoading } = useLoyalty();
  const { data: txns = [], isLoading: txnLoading } = useLoyaltyTransactions();
  const redeem = useRedeemLoyalty();

  const balance = loyalty?.balance ?? 0;
  const cfg = loyalty?.config;
  const redeemable = Math.floor(balance);
  const minRedeem = cfg?.minRedeem ?? 100;
  const redeemRate = cfg?.redeemRate ?? 0.1;
  const canRedeem = (cfg?.enabled ?? true) && redeemable >= minRedeem;
  const redeemValue = Math.round(redeemable * redeemRate * 100) / 100;
  const pointsToMin = Math.max(0, Math.ceil(minRedeem - redeemable));

  // Streak progress toward the next bonus.
  const threshold = cfg?.streakThreshold ?? 7;
  const streak = loyalty?.currentStreak ?? 0;
  const daysToBonus = threshold > 0 ? threshold - (streak % threshold) : 0;

  const onRedeem = () => {
    if (!canRedeem || redeem.isPending) return;
    Alert.alert(
      'Redeem points',
      `Redeem ${formatPoints(redeemable)} points for ${formatMoney(redeemValue)} of wallet credit?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Redeem',
          onPress: () =>
            redeem.mutate(redeemable, {
              onSuccess: (res) =>
                Alert.alert(
                  'Points redeemed',
                  `${formatMoney(res.walletCredited)} has been added to your Fe3dr wallet.`,
                  [
                    { text: 'View wallet', onPress: () => router.push('/wallet' as never) },
                    { text: 'Done' },
                  ],
                ),
              onError: (err) => Alert.alert('Could not redeem', loyaltyErrorMessage(err)),
            }),
        },
      ],
    );
  };

  return (
    <SafeAreaView edges={['top', 'left', 'right']} className="flex-1 bg-canvas">
      <ScreenHeader title="Rewards" />
      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator color={customerColors.charcoal.soft} />
        </View>
      ) : (
        <ScrollView>
          <View className="p-4">
            {/* Points balance + tier */}
            <View className="rounded-2xl bg-canvas p-5" style={cardShadow}>
              <View className="flex-row items-center justify-between">
                <Text className="text-xs font-semibold text-charcoal-soft">
                  Points balance
                </Text>
                <View className="flex-row items-center rounded-full bg-coral-tint px-2.5 py-1">
                  <Award size={13} color={customerColors.coral.DEFAULT} />
                  <Text className="ml-1 text-xs font-semibold text-coral">
                    {TIER_LABEL[loyalty?.tier ?? 'bronze']}
                  </Text>
                </View>
              </View>
              <Text className="mt-1 text-4xl font-bold text-charcoal">{formatPoints(balance)}</Text>
              <Text className="mt-1 text-sm text-charcoal-soft">
                Worth {formatMoney(redeemValue)} in wallet credit
              </Text>
            </View>

            {/* Streak */}
            <View className="mt-3 rounded-2xl bg-canvas p-5" style={cardShadow}>
              <View className="flex-row items-center">
                <View className="mr-3 h-10 w-10 items-center justify-center rounded-full bg-surface-soft">
                  <Flame size={20} color={streak > 0 ? customerColors.coral.DEFAULT : customerColors.charcoal.soft} />
                </View>
                <View className="flex-1">
                  <Text className="text-base font-semibold text-charcoal">
                    {streak > 0 ? `${streak}-day meal streak` : 'No active streak'}
                  </Text>
                  <Text className="text-xs text-charcoal-soft">
                    {streak > 0
                      ? `${daysToBonus} more ${daysToBonus === 1 ? 'day' : 'days'} for a ${formatPoints(cfg?.streakBonus ?? 50)}-point bonus`
                      : `Keep a daily meal subscription going to earn streak bonuses`}
                  </Text>
                </View>
              </View>
            </View>

            {/* Redeem */}
            <Pressable
              onPress={onRedeem}
              disabled={!canRedeem || redeem.isPending}
              accessibilityRole="button"
              accessibilityLabel="Redeem points to wallet"
              className={`mt-3 flex-row items-center justify-center rounded-2xl px-5 py-4 ${canRedeem ? 'bg-coral' : 'bg-surface-soft'}`}
            >
              {redeem.isPending ? (
                <ActivityIndicator color={canRedeem ? '#FFFFFF' : customerColors.charcoal.soft} />
              ) : (
                <>
                  <Sparkles size={18} color={canRedeem ? '#FFFFFF' : customerColors.charcoal.soft} />
                  <Text className={`ml-2 text-base font-semibold ${canRedeem ? 'text-white' : 'text-charcoal-soft'}`}>
                    {canRedeem
                      ? `Redeem ${formatPoints(redeemable)} pts → ${formatMoney(redeemValue)}`
                      : `Earn ${formatPoints(pointsToMin)} more to redeem`}
                  </Text>
                </>
              )}
            </Pressable>

            <Text className="px-1 pb-2 pt-6 text-xs font-semibold text-charcoal-soft">
              History
            </Text>

            {txnLoading ? (
              <ActivityIndicator className="mt-4" color={customerColors.charcoal.soft} />
            ) : txns.length === 0 ? (
              <View className="items-center rounded-xl bg-canvas p-8" style={cardShadow}>
                <Text className="text-charcoal-soft">
                  No points yet. Earn points on every delivered order.
                </Text>
              </View>
            ) : (
              <View className="overflow-hidden rounded-xl bg-canvas" style={cardShadow}>
                {txns.map((t, i) => {
                  const credit = t.type === 'credit';
                  return (
                    <View key={t.id}>
                      {i > 0 && <View className="ml-16 h-px bg-hairline" />}
                      <View className="min-h-[56px] flex-row items-center px-4 py-3">
                        <View className="mr-3 h-9 w-9 items-center justify-center rounded-full bg-surface-soft">
                          {credit ? (
                            <ArrowDownLeft size={16} color={customerColors.success.DEFAULT} />
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
                        <Text className={`text-base font-semibold ${credit ? 'text-success' : 'text-charcoal'}`}>
                          {credit ? '+' : '−'}
                          {formatPoints(t.points)}
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
