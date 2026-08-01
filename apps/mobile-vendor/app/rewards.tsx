// Rewards — chef loyalty points (earned per delivered order, converted to
// cashback on the weekly payout once past the threshold) + the refer-a-chef
// program (cash for both kitchens once the new one completes its first
// milestone of delivered orders). All amounts come from the API config —
// nothing is hardcoded.

import {
  Alert,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  Share,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { AlertCircle, ChevronLeft, Gift, Share2, Users } from 'lucide-react-native';
import * as Haptics from 'expo-haptics';
import { theme } from '@homechef/mobile-shared/theme';
import { useChefRewards, useConvertRewards } from '../hooks/useChefRewards';

function fmtInr(value: number): string {
  return `₹${value.toLocaleString('en-IN', { minimumFractionDigits: 0 })}`;
}

function fmtPts(value: number): string {
  return value.toLocaleString('en-IN', { maximumFractionDigits: 0 });
}

const INK_RIPPLE = `${theme.colors.ink.DEFAULT}14`;

export default function RewardsScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useChefRewards();
  const convert = useConvertRewards();

  async function onShare(): Promise<void> {
    if (!data) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    try {
      await Share.share({
        message:
          `Cook with me on Fe3dr! Join as a home chef with my referral code ${data.referral.code} — ` +
          `you earn ${fmtInr(data.referral.refereeAmount)} after your first ${data.referral.milestoneOrders} orders. ` +
          data.referral.link,
      });
    } catch {
      // Share cancelled — ignore.
    }
  }

  function onConvert(): void {
    if (!data) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    Alert.alert(
      'Convert points to cashback?',
      `${fmtPts(data.loyalty.points)} points become ${fmtInr(data.loyalty.convertValue)}, added to your next weekly payout.`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Convert',
          onPress: () =>
            convert.mutate(undefined, {
              onSuccess: (res) => Alert.alert('Done', res.message),
              onError: (err) =>
                Alert.alert(
                  'Could not convert',
                  err instanceof Error ? err.message : 'Please try again.',
                ),
            }),
        },
      ],
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={10}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: INK_RIPPLE, borderless: true }}
        >
          <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Rewards</Text>
        <View style={{ width: 26 }} />
      </View>

      {isError ? (
        <View style={styles.centered}>
          <AlertCircle size={28} color={theme.colors.ink.soft} />
          <Text style={styles.errorTitle}>Something went wrong</Text>
          <Pressable
            onPress={() => void refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry loading rewards"
          >
            {({ pressed }) => (
              <View style={[styles.primaryBtn, pressed && styles.primaryBtnPressed]}>
                <Text style={styles.primaryBtnText}>Try again</Text>
              </View>
            )}
          </Pressable>
        </View>
      ) : isLoading || !data ? (
        <View style={styles.scroll}>
          <View style={[styles.skeleton, { height: 180 }]} />
          <View style={[styles.skeleton, { height: 200 }]} />
        </View>
      ) : (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={
            <RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} />
          }
        >
          {data.pendingPayoutCredit > 0 ? (
            <View style={styles.pendingBanner}>
              <Text style={styles.pendingBannerText}>
                {fmtInr(data.pendingPayoutCredit)} in rewards will be added to your next payout
              </Text>
            </View>
          ) : null}

          {/* Points */}
          <View style={styles.card}>
            <View style={styles.cardHeader}>
              <Gift size={18} color={theme.colors.ink.soft} />
              <Text style={styles.cardTitle}>Reward points</Text>
            </View>
            <Text style={styles.points}>{fmtPts(data.loyalty.points)}</Text>
            <Text style={styles.pointsSub}>
              worth {fmtInr(data.loyalty.convertValue)} · lifetime {fmtPts(data.loyalty.lifetimePoints)} pts
            </Text>
            <View style={styles.progressTrack}>
              <View
                style={[
                  styles.progressFill,
                  {
                    width: `${Math.min(100, (data.loyalty.points / data.loyalty.minConvertPoints) * 100)}%`,
                  },
                ]}
              />
            </View>
            <Text style={styles.hint}>
              Earn {data.loyalty.earnRate} pt per ₹1 of delivered orders. Convert at{' '}
              {fmtPts(data.loyalty.minConvertPoints)} points into cashback on your weekly payout.
            </Text>
            <Pressable
              onPress={onConvert}
              disabled={!data.loyalty.canConvert || convert.isPending}
              accessibilityRole="button"
              accessibilityLabel="Convert points to cashback"
              android_ripple={{ color: INK_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.primaryBtn,
                    (!data.loyalty.canConvert || convert.isPending) && styles.primaryBtnDisabled,
                    pressed && Platform.OS === 'ios' && styles.primaryBtnPressed,
                  ]}
                >
                  <Text style={styles.primaryBtnText}>
                    {data.loyalty.canConvert
                      ? `Convert to ${fmtInr(data.loyalty.convertValue)} cashback`
                      : `${fmtPts(Math.max(0, data.loyalty.minConvertPoints - data.loyalty.points))} points to go`}
                  </Text>
                </View>
              )}
            </Pressable>
          </View>

          {/* Referral */}
          <View style={styles.card}>
            <View style={styles.cardHeader}>
              <Users size={18} color={theme.colors.ink.soft} />
              <Text style={styles.cardTitle}>Refer a chef</Text>
            </View>
            <Text style={styles.body}>
              You earn {fmtInr(data.referral.referrerAmount)}, they earn{' '}
              {fmtInr(data.referral.refereeAmount)} — paid once their kitchen completes{' '}
              {data.referral.milestoneOrders} orders.
            </Text>
            <View style={styles.codeBox}>
              <Text style={styles.code} accessibilityLabel={`Your referral code is ${data.referral.code}`}>
                {data.referral.code}
              </Text>
            </View>
            <Pressable
              onPress={() => void onShare()}
              accessibilityRole="button"
              accessibilityLabel="Share your referral code"
              android_ripple={{ color: INK_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View style={[styles.primaryBtn, pressed && Platform.OS === 'ios' && styles.primaryBtnPressed]}>
                  <Share2 size={16} color={theme.colors.paper} />
                  <Text style={styles.primaryBtnText}>Share code</Text>
                </View>
              )}
            </Pressable>
            {data.referral.totalEarned > 0 ? (
              <Text style={styles.hint}>
                Earned so far: {fmtInr(data.referral.totalEarned)}
              </Text>
            ) : null}
          </View>

          {/* Referral progress */}
          {data.referral.referrals.length > 0 ? (
            <View style={styles.card}>
              <Text style={styles.cardTitle}>Your referrals</Text>
              {data.referral.referrals.map((r, i) => (
                <View
                  key={`${r.kitchenName}-${i}`}
                  style={[styles.refRow, i === data.referral.referrals.length - 1 && styles.refRowLast]}
                >
                  <View style={{ flex: 1 }}>
                    <Text style={styles.refName} numberOfLines={1}>
                      {r.kitchenName || 'New kitchen'}
                    </Text>
                    {r.status === 'pending' ? (
                      <Text style={styles.refSub}>
                        {r.deliveredOrders}/{r.milestoneOrders} orders delivered
                      </Text>
                    ) : null}
                  </View>
                  <Text
                    style={[
                      styles.refStatus,
                      r.status === 'rewarded' && { color: theme.colors.success.DEFAULT },
                    ]}
                  >
                    {r.status === 'rewarded'
                      ? `+${fmtInr(r.reward)}`
                      : r.status === 'pending'
                        ? 'In progress'
                        : 'Rejected'}
                  </Text>
                </View>
              ))}
            </View>
          ) : null}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  headerTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: theme.colors.ink.DEFAULT,
  },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 12, paddingHorizontal: 32 },
  errorTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT },
  scroll: { padding: theme.spacing[4], paddingBottom: 40, gap: theme.spacing[4] },
  skeleton: { backgroundColor: theme.colors.bone, borderRadius: theme.radius.lg },

  pendingBanner: {
    backgroundColor: theme.colors.herb.tint,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  pendingBannerText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: theme.colors.herb.soft,
    fontVariant: ['tabular-nums'],
  },

  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    gap: theme.spacing[3],
    ...theme.shadow[1],
  },
  cardHeader: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  cardTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT },
  points: {
    fontFamily: 'Geist-Bold',
    fontSize: 36,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  pointsSub: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.soft,
    marginTop: -8,
    fontVariant: ['tabular-nums'],
  },
  progressTrack: {
    height: 6,
    borderRadius: 3,
    backgroundColor: theme.colors.mist.DEFAULT,
    overflow: 'hidden',
  },
  progressFill: { height: '100%', borderRadius: 3, backgroundColor: theme.colors.herb.DEFAULT },
  hint: { fontFamily: 'Inter', fontSize: 12, lineHeight: 17, color: theme.colors.ink.muted },
  body: { fontFamily: 'Inter', fontSize: 14, lineHeight: 20, color: theme.colors.ink.soft },

  codeBox: {
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.bone,
    paddingVertical: theme.spacing[3],
    alignItems: 'center',
  },
  code: {
    fontFamily: 'Geist-Bold',
    fontSize: 26,
    letterSpacing: 4,
    color: theme.colors.ink.DEFAULT,
  },

  primaryBtn: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.ink.DEFAULT,
    paddingHorizontal: theme.spacing[5],
  },
  primaryBtnPressed: { opacity: 0.85 },
  primaryBtnDisabled: { backgroundColor: theme.colors.mist.strong },
  primaryBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.paper },

  refRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  refRowLast: { borderBottomWidth: 0 },
  refName: { fontFamily: 'Inter-Medium', fontSize: 15, color: theme.colors.ink.DEFAULT },
  refSub: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.muted, marginTop: 2 },
  refStatus: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: theme.colors.ink.soft,
    fontVariant: ['tabular-nums'],
  },
});
