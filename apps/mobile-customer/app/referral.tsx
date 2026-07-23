// Refer & Earn (#38) — invite friends with a unique code/link; the reward lands
// in the store-credit wallet on the friend's first paid order. Reward amounts +
// stats come from the API (admin-configurable); nothing is hardcoded.

import {
  ActivityIndicator,
  Platform,
  Pressable,
  ScrollView,
  Share,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { AlertCircle, ChevronLeft, Gift, Share2 } from 'lucide-react-native';
import * as Haptics from 'expo-haptics';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';
import { useReferral, useReferralHistory } from '../hooks/useReferral';

// Cards float on the white canvas via shadow[2] (spec §1) rather than a grey
// page background — same recipe wallet.tsx uses.
const cardShadow = {
  shadowColor: customerTheme.shadow[2].shadowColor,
  shadowOffset: customerTheme.shadow[2].shadowOffset,
  shadowOpacity: customerTheme.shadow[2].shadowOpacity,
  shadowRadius: customerTheme.shadow[2].shadowRadius,
  elevation: customerTheme.shadow[2].elevation,
} as const;

// Android ripple tints — translucent tokens, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

export default function ReferralScreen() {
  const { data, isLoading, isError, refetch } = useReferral();
  const { data: history } = useReferralHistory();

  async function onShare() {
    if (!data) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    const msg =
      `Join me on Fe3dr! Use my code ${data.code} and we both get ${money(data.refereeReward)} ` +
      `in credit on your first order. ${data.link}`;
    try {
      await Share.share({ message: msg });
    } catch {
      // Share cancelled — ignore.
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={10}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: ICON_RIPPLE, borderless: true }}
        >
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Refer &amp; Earn</Text>
        <View style={{ width: 26 }} />
      </View>

      {isError ? (
        <View style={styles.centered}>
          <View style={styles.errorIconWrap}>
            <AlertCircle size={28} color={customerColors.charcoal.soft} />
          </View>
          <Text style={styles.heroTitle}>Something went wrong</Text>
          <Text style={styles.heroSub}>We could not load your referral program. Please try again.</Text>
          <Pressable
            onPress={() => void refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry loading referral program"
            android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.retryBtn,
                  pressed && Platform.OS === 'ios' && styles.retryBtnPressed,
                ]}
              >
                <Text style={styles.retryBtnText}>Try again</Text>
              </View>
            )}
          </Pressable>
        </View>
      ) : isLoading || !data ? (
        <View style={styles.scroll}>
          <View style={[styles.skeletonBlock, { height: 140, borderRadius: 16 }]} />
          <View style={[styles.skeletonBlock, { height: 150, borderRadius: 16 }]} />
          <View style={styles.statsRow}>
            <View style={[styles.skeletonBlock, { flex: 1, height: 76 }]} />
            <View style={[styles.skeletonBlock, { flex: 1, height: 76 }]} />
          </View>
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {/* Hero */}
          <View style={styles.hero}>
            <View style={styles.heroIcon}>
              <Gift size={26} color={customerColors.coral.DEFAULT} />
            </View>
            <Text style={styles.heroTitle}>
              Give {money(data.refereeReward)}, get {money(data.referrerReward)}
            </Text>
            <Text style={styles.heroSub}>
              Your friend gets {money(data.refereeReward)} off their first order. You get{' '}
              {money(data.referrerReward)} once they order.
            </Text>
          </View>

          {/* Code + share */}
          <View style={[styles.codeCard, cardShadow]}>
            <Text style={styles.codeLabel}>YOUR CODE</Text>
            <Text style={styles.code} accessibilityLabel={`Your referral code is ${data.code}`}>
              {data.code}
            </Text>
            <Pressable
              onPress={onShare}
              accessibilityRole="button"
              accessibilityLabel="Share your invite"
              android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View style={[styles.shareBtn, pressed && Platform.OS === 'ios' && styles.shareBtnPressed]}>
                  <Share2 size={18} color={customerColors.canvas} strokeWidth={2} />
                  <Text style={styles.shareBtnText}>Share invite</Text>
                </View>
              )}
            </Pressable>
          </View>

          {/* Stats */}
          <View style={styles.statsRow}>
            <View style={[styles.statCard, cardShadow]}>
              <Text style={styles.statValue}>{data.stats.rewardedCount}</Text>
              <Text style={styles.statLabel}>Friends joined</Text>
            </View>
            <View style={[styles.statCard, cardShadow]}>
              <Text style={styles.statValue}>{money(data.stats.totalEarned)}</Text>
              <Text style={styles.statLabel}>Credit earned</Text>
            </View>
          </View>

          {/* How it works */}
          <Text style={styles.sectionLabel}>HOW IT WORKS</Text>
          <View style={[styles.steps, cardShadow]}>
            {[
              'Share your code with friends.',
              'They sign up and place their first order.',
              `You both get credit in your Fe3dr wallet.`,
            ].map((step, i) => (
              <View key={i} style={styles.stepRow}>
                <View style={styles.stepNum}>
                  <Text style={styles.stepNumText}>{i + 1}</Text>
                </View>
                <Text style={styles.stepText}>{step}</Text>
              </View>
            ))}
          </View>

          {/* History */}
          {(history?.length ?? 0) > 0 ? (
            <>
              <Text style={styles.sectionLabel}>YOUR REFERRALS</Text>
              <View style={[styles.historyCard, cardShadow]}>
                {history!.map((h, i) => (
                  <View key={i} style={[styles.historyRow, i === history!.length - 1 && styles.historyRowLast]}>
                    <Text style={styles.historyName} numberOfLines={1}>
                      {h.refereeName}
                    </Text>
                    <Text style={[styles.historyStatus, h.status === 'rewarded' && styles.historyStatusDone]}>
                      {h.status === 'rewarded' ? `+${money(h.reward)}` : h.status === 'pending' ? 'Pending' : '—'}
                    </Text>
                  </View>
                ))}
              </View>
            </>
          ) : null}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  // White-first canvas per surface model §1 — the hero/code/stat cards
  // separate via their own shadow, not a grey page background.
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 32, gap: 4 },
  errorIconWrap: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  retryBtn: {
    marginTop: 16,
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: 24,
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
  },
  retryBtnPressed: { backgroundColor: customerColors.coral.pressed },
  retryBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.canvas },
  scroll: { padding: 16, paddingBottom: 40, gap: 16 },
  skeletonBlock: { backgroundColor: customerColors.surface.soft, borderRadius: 12 },

  hero: {
    backgroundColor: customerColors.surface.soft,
    borderRadius: 16,
    padding: 20,
    alignItems: 'center',
    gap: 8,
  },
  heroIcon: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: customerColors.coral.tint,
    alignItems: 'center',
    justifyContent: 'center',
  },
  heroTitle: { fontFamily: 'Geist-Bold', fontSize: 22, color: customerColors.charcoal.DEFAULT, textAlign: 'center' },
  heroSub: { fontFamily: 'Inter', fontSize: 14, lineHeight: 20, color: customerColors.charcoal.soft, textAlign: 'center' },

  codeCard: { backgroundColor: customerColors.canvas, borderRadius: 16, padding: 20, alignItems: 'center', gap: 12 },
  codeLabel: { fontFamily: 'Inter-SemiBold', fontSize: 12, letterSpacing: 1.4, color: customerColors.charcoal.soft },
  code: {
    fontFamily: 'Geist-Bold',
    fontSize: 30,
    letterSpacing: 4,
    color: customerColors.charcoal.DEFAULT,
  },
  shareBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 10,
    paddingHorizontal: 24,
    minHeight: 48,
    justifyContent: 'center',
    alignSelf: 'stretch',
  },
  shareBtnPressed: { backgroundColor: customerColors.coral.pressed },
  shareBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas },

  statsRow: { flexDirection: 'row', gap: 12 },
  statCard: { flex: 1, backgroundColor: customerColors.canvas, borderRadius: 12, padding: 16, alignItems: 'center', gap: 2 },
  statValue: { fontFamily: 'Geist-Bold', fontSize: 22, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  statLabel: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft },

  sectionLabel: { fontFamily: 'Inter-SemiBold', fontSize: 12, letterSpacing: 1.4, color: customerColors.charcoal.soft },
  steps: { backgroundColor: customerColors.canvas, borderRadius: 12, padding: 16, gap: 14, marginTop: -4 },
  stepRow: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  stepNum: {
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: customerColors.coral.tint,
    alignItems: 'center',
    justifyContent: 'center',
  },
  stepNumText: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: customerColors.coral.pressed },
  stepText: { flex: 1, fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT },

  historyCard: { backgroundColor: customerColors.canvas, borderRadius: 12, paddingHorizontal: 16, marginTop: -4 },
  historyRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  historyRowLast: { borderBottomWidth: 0 },
  historyName: { flex: 1, fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  historyStatus: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  historyStatusDone: { color: customerColors.success.DEFAULT },
});
