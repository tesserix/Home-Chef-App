import React from 'react';
import { ActivityIndicator, Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import Animated, { Easing, FadeInDown, useReducedMotion } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { AlertCircle, Award, Flame, Sparkles, ArrowDownLeft, ArrowUpRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import {
  useLoyalty,
  useLoyaltyTransactions,
  useRedeemLoyalty,
  loyaltyErrorMessage,
} from '../hooks/useLoyalty';
import { useAlert } from '@homechef/mobile-shared/ui';

// Entrance easing — ease-out-quart, matches the app-wide motion spec.
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

// Android ripple tint — translucent token, never a new literal colour.
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

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
const NEXT_TIER_LABEL: Record<string, string> = { bronze: 'Silver', silver: 'Gold' };

// Shadow[2] — card lift. `canvas` and `surface.DEFAULT` are both pure white,
// so every white card on this white page needs elevation (not a hairline
// alone) to read as a separate surface. Matches ActiveOrderCard / MealPlanList.
const CARD_SHADOW = {
  shadowColor: '#000000',
  shadowOffset: { width: 0, height: 4 },
  shadowOpacity: 0.1,
  shadowRadius: 12,
  elevation: 4,
} as const;

/** Slim continuous progress bar toward the next loyalty tier. Real config +
 *  lifetime-points data only — no invented figures. */
function TierProgressBar({ progress }: { progress: number }) {
  return (
    <View style={progressStyles.tierTrack}>
      <View style={[progressStyles.tierFill, { width: `${Math.round(progress * 100)}%` }]} />
    </View>
  );
}

/** Segmented streak bar — same visual language as ActiveOrderCard's step
 *  progress: coral-filled segments on a hairline track, no bounce or glow. */
function StreakSegments({ filled, total }: { filled: number; total: number }) {
  const segments = Array.from({ length: Math.max(total, 1) });
  return (
    <View style={progressStyles.streakTrack}>
      {segments.map((_, i) => (
        <View
          key={i}
          style={[
            progressStyles.streakSegment,
            i < filled ? progressStyles.streakSegmentFilled : progressStyles.streakSegmentEmpty,
          ]}
        />
      ))}
    </View>
  );
}

const progressStyles = StyleSheet.create({
  tierTrack: {
    marginTop: 12,
    height: 4,
    borderRadius: 2,
    backgroundColor: customerColors.hairline,
    overflow: 'hidden',
  },
  tierFill: {
    height: 4,
    borderRadius: 2,
    backgroundColor: customerColors.coral.DEFAULT,
  },
  streakTrack: {
    flexDirection: 'row',
    gap: 3,
    height: 3,
    marginTop: 12,
  },
  streakSegment: {
    flex: 1,
    height: 3,
    borderRadius: 2,
  },
  streakSegmentFilled: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  streakSegmentEmpty: {
    backgroundColor: customerColors.hairline,
  },
});

export default function LoyaltyScreen() {
  const { showAlert } = useAlert();
  const router = useRouter();
  const reduceMotion = useReducedMotion();
  const { data: loyalty, isLoading, isError, refetch } = useLoyalty();
  const { data: txns = [], isLoading: txnLoading, isError: txnError, refetch: refetchTxns } = useLoyaltyTransactions();
  const redeem = useRedeemLoyalty();

  const balance = loyalty?.balance ?? 0;
  const cfg = loyalty?.config;
  const redeemable = Math.floor(balance);
  const minRedeem = cfg?.minRedeem ?? 100;
  const redeemRate = cfg?.redeemRate ?? 0.1;
  const canRedeem = (cfg?.enabled ?? true) && redeemable >= minRedeem;
  const redeemValue = Math.round(redeemable * redeemRate * 100) / 100;
  const pointsToMin = Math.max(0, Math.ceil(minRedeem - redeemable));

  // Tier progress — derived entirely from API-served config + lifetime points,
  // never a guessed figure. Gold has no next tier, so the bar is omitted.
  const tier = loyalty?.tier ?? 'bronze';
  const lifetimePoints = loyalty?.lifetimePoints ?? 0;
  const nextTierAt = tier === 'bronze' ? cfg?.tierSilverAt : tier === 'silver' ? cfg?.tierGoldAt : undefined;
  const nextTierLabel = NEXT_TIER_LABEL[tier];
  const tierProgress = nextTierAt ? Math.min(1, lifetimePoints / nextTierAt) : undefined;
  const pointsToNextTier = nextTierAt ? Math.max(0, Math.ceil(nextTierAt - lifetimePoints)) : undefined;

  // Streak progress toward the next bonus.
  const threshold = cfg?.streakThreshold ?? 7;
  const streak = loyalty?.currentStreak ?? 0;
  const daysToBonus = threshold > 0 ? threshold - (streak % threshold) : 0;
  const streakFilled = streak > 0 ? (streak % threshold === 0 ? threshold : streak % threshold) : 0;

  const onRedeem = () => {
    if (!canRedeem || redeem.isPending) return;
    showAlert(
      'Redeem points',
      `Redeem ${formatPoints(redeemable)} points for ${formatMoney(redeemValue)} of wallet credit?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Redeem',
          onPress: () =>
            redeem.mutate(redeemable, {
              onSuccess: (res) =>
                showAlert(
                  'Points redeemed',
                  `${formatMoney(res.walletCredited)} has been added to your Fe3dr wallet.`,
                  [
                    { text: 'View wallet', onPress: () => router.push('/wallet' as never) },
                    { text: 'Done' },
                  ],
                ),
              onError: (err) => showAlert('Could not redeem', loyaltyErrorMessage(err)),
            }),
        },
      ],
    );
  };

  return (
    <SafeAreaView edges={['top', 'left', 'right']} className="flex-1 bg-canvas">
      <ScreenHeader title="Rewards" />
      {isError ? (
        <View className="flex-1 items-center justify-center px-8 gap-4 pt-16">
          <View className="w-16 h-16 rounded-full bg-surface-soft items-center justify-center">
            <AlertCircle size={28} color={customerColors.charcoal.soft} />
          </View>
          <Text className="text-lg font-semibold text-charcoal text-center font-display">
            Something went wrong
          </Text>
          <Text className="text-sm text-charcoal-soft text-center">
            We could not load your rewards. Please try again.
          </Text>
          <Pressable
            onPress={() => void refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry loading rewards"
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
      ) : isLoading ? (
        <View className="p-4 gap-3">
          <View className="rounded-2xl bg-surface-soft" style={{ height: 132 }} />
          <View className="rounded-2xl bg-surface-soft" style={{ height: 88 }} />
          <View className="rounded-lg bg-surface-soft" style={{ height: 52 }} />
          <View className="rounded-2xl bg-surface-soft mt-3" style={{ height: 112 }} />
        </View>
      ) : (
        <ScrollView>
          <View className="p-4">
            {/* Points balance + tier — the one true display moment, so it
                carries the Geist display face at weight 700. */}
            <Animated.View
              entering={reduceMotion ? undefined : FadeInDown.delay(0).duration(250).easing(ENTRANCE_EASING)}
              className="rounded-2xl border border-hairline bg-canvas p-5"
              style={CARD_SHADOW}
            >
              <View className="flex-row items-center justify-between">
                <Text className="text-xs font-semibold text-charcoal-soft">
                  Points balance
                </Text>
                <View className="flex-row items-center rounded-full bg-coral-tint px-2.5 py-1">
                  <Award size={13} color={customerColors.coral.DEFAULT} />
                  <Text className="ml-1 text-xs font-semibold text-coral">
                    {TIER_LABEL[tier]}
                  </Text>
                </View>
              </View>
              <Text
                className="mt-1.5 text-4xl text-charcoal"
                style={{ fontFamily: 'Geist-Bold', letterSpacing: -0.5, fontVariant: ['tabular-nums'] }}
              >
                {formatPoints(balance)}
              </Text>
              <Text className="mt-1 text-sm text-charcoal-soft">
                Worth {formatMoney(redeemValue)} in wallet credit
              </Text>

              {nextTierAt && tierProgress !== undefined ? (
                <>
                  <TierProgressBar progress={tierProgress} />
                  <Text
                    className="mt-2 text-xs text-charcoal-soft"
                    style={{ fontVariant: ['tabular-nums'] }}
                  >
                    {formatPoints(pointsToNextTier ?? 0)} points to {nextTierLabel}
                  </Text>
                </>
              ) : null}
            </Animated.View>

            {/* Streak */}
            <Animated.View
              entering={reduceMotion ? undefined : FadeInDown.delay(60).duration(250).easing(ENTRANCE_EASING)}
              className="mt-3 rounded-2xl border border-hairline bg-canvas p-5"
              style={CARD_SHADOW}
            >
              <View className="flex-row items-center">
                <View
                  className={`mr-3 h-10 w-10 items-center justify-center rounded-full ${
                    streak > 0 ? 'bg-coral-tint' : 'bg-surface-soft'
                  }`}
                >
                  <Flame size={20} color={streak > 0 ? customerColors.coral.DEFAULT : customerColors.charcoal.soft} />
                </View>
                <View className="flex-1">
                  <Text
                    className="text-base font-semibold text-charcoal"
                    style={{ fontVariant: ['tabular-nums'] }}
                  >
                    {streak > 0 ? `${streak}-day meal streak` : 'No active streak'}
                  </Text>
                  <Text className="text-xs text-charcoal-soft">
                    {streak > 0
                      ? `${daysToBonus} more ${daysToBonus === 1 ? 'day' : 'days'} for a ${formatPoints(cfg?.streakBonus ?? 50)}-point bonus`
                      : `Keep a daily meal subscription going to earn streak bonuses`}
                  </Text>
                </View>
              </View>
              {streak > 0 ? <StreakSegments filled={streakFilled} total={threshold} /> : null}
            </Animated.View>

            {/* Redeem — coral filled, radius 8, 52pt (spec §3) */}
            <Animated.View
              entering={reduceMotion ? undefined : FadeInDown.delay(120).duration(250).easing(ENTRANCE_EASING)}
            >
              <Pressable
                onPress={onRedeem}
                disabled={!canRedeem || redeem.isPending}
                accessibilityRole="button"
                accessibilityLabel={
                  canRedeem
                    ? `Redeem ${formatPoints(redeemable)} points for ${formatMoney(redeemValue)}`
                    : 'Redeem points to wallet'
                }
                android_ripple={
                  !canRedeem || redeem.isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }
                }
              >
                {({ pressed }) => (
                  <View
                    className={`mt-3 min-h-[52px] flex-row items-center justify-center rounded-lg px-5 ${
                      canRedeem ? 'bg-coral' : 'bg-surface-soft'
                    } ${canRedeem && pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : ''}`}
                  >
                    {redeem.isPending ? (
                      <ActivityIndicator color={canRedeem ? customerColors.canvas : customerColors.charcoal.soft} />
                    ) : (
                      <>
                        <Sparkles size={18} color={canRedeem ? customerColors.canvas : customerColors.charcoal.soft} />
                        <Text
                          className={`ml-2 text-base font-semibold ${canRedeem ? 'text-white' : 'text-charcoal-soft'}`}
                          style={{ fontVariant: ['tabular-nums'] }}
                        >
                          {canRedeem
                            ? `Redeem ${formatPoints(redeemable)} pts → ${formatMoney(redeemValue)}`
                            : `Earn ${formatPoints(pointsToMin)} more to redeem`}
                        </Text>
                      </>
                    )}
                  </View>
                )}
              </Pressable>
            </Animated.View>

            <Text className="px-1 pb-2 pt-6 text-xs font-semibold text-charcoal-soft">
              History
            </Text>

            <Animated.View
              entering={reduceMotion ? undefined : FadeInDown.delay(160).duration(250).easing(ENTRANCE_EASING)}
            >
              {txnLoading ? (
                <View
                  className="overflow-hidden rounded-2xl border border-hairline bg-canvas"
                  style={CARD_SHADOW}
                >
                  {[0, 1].map((i) => (
                    <View key={i}>
                      {i > 0 && <View className="ml-16 h-px bg-hairline" />}
                      <View className="min-h-[56px] flex-row items-center px-4 py-3">
                        <View className="mr-3 h-9 w-9 rounded-full bg-surface-soft" />
                        <View className="flex-1 gap-2">
                          <View className="h-3.5 rounded bg-hairline" style={{ width: '55%' }} />
                          <View className="h-3 rounded bg-hairline" style={{ width: '35%' }} />
                        </View>
                      </View>
                    </View>
                  ))}
                </View>
              ) : txnError ? (
                <View
                  className="items-center rounded-2xl border border-hairline bg-canvas p-6 gap-3"
                  style={CARD_SHADOW}
                >
                  <Text className="text-charcoal-soft text-center">Could not load your history.</Text>
                  <Pressable
                    onPress={() => void refetchTxns()}
                    accessibilityRole="button"
                    accessibilityLabel="Retry loading rewards history"
                    android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
                  >
                    <View className="min-h-[40px] px-4 items-center justify-center rounded-lg border border-hairline">
                      <Text className="text-sm font-semibold text-charcoal">Try again</Text>
                    </View>
                  </Pressable>
                </View>
              ) : txns.length === 0 ? (
                <View
                  className="items-center rounded-2xl border border-hairline bg-canvas p-8"
                  style={CARD_SHADOW}
                >
                  <View className="mb-3 h-12 w-12 items-center justify-center rounded-full bg-surface-soft">
                    <Award size={22} color={customerColors.charcoal.soft} />
                  </View>
                  <Text className="text-charcoal-soft text-center">
                    No points yet. Earn points on every delivered order.
                  </Text>
                </View>
              ) : (
                <View
                  className="overflow-hidden rounded-2xl border border-hairline bg-canvas"
                  style={CARD_SHADOW}
                >
                  {txns.map((t, i) => {
                    const credit = t.type === 'credit';
                    return (
                      <View key={t.id}>
                        {i > 0 && <View className="ml-16 h-px bg-hairline" />}
                        <View className="min-h-[56px] flex-row items-center px-4 py-3">
                          <View
                            className={`mr-3 h-9 w-9 items-center justify-center rounded-full ${
                              credit ? 'bg-success-tint' : 'bg-surface-soft'
                            }`}
                          >
                            {credit ? (
                              <ArrowDownLeft size={16} color={customerColors.success.DEFAULT} />
                            ) : (
                              <ArrowUpRight size={16} color={customerColors.charcoal.soft} />
                            )}
                          </View>
                          <View className="flex-1">
                            <Text className="text-base text-charcoal">{sourceLabel(t.source)}</Text>
                            <Text
                              className="text-xs text-charcoal-soft"
                              style={{ fontVariant: ['tabular-nums'] }}
                            >
                              {new Date(t.createdAt).toLocaleDateString()}
                              {t.reason ? ` · ${t.reason}` : ''}
                            </Text>
                          </View>
                          <Text
                            className={`text-base font-semibold ${credit ? 'text-success' : 'text-charcoal'}`}
                            style={{ fontVariant: ['tabular-nums'] }}
                          >
                            {credit ? '+' : '−'}
                            {formatPoints(t.points)}
                          </Text>
                        </View>
                      </View>
                    );
                  })}
                </View>
              )}
            </Animated.View>
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
