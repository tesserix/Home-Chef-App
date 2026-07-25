// Refer & Earn (#38) — invite friends with a unique code/link; the reward lands
// as LOYALTY POINTS on the friend's first paid order. Reward amounts +
// stats come from the API (admin-configurable); nothing is hardcoded.

import {
  ActivityIndicator,
  Linking,
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
import {
  AlertCircle,
  ChevronLeft,
  Gift,
  Mail,
  MessageCircle,
  MessageSquare,
  Share2,
} from 'lucide-react-native';
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

  function inviteMessage(): string {
    if (!data) return '';
    return (
      `Join me on Fe3dr! Use my code ${data.code} and we both get rewarded — you get ${money(data.refereeReward)} ` +
      `in points on your first order. ${data.link}`
    );
  }

  // Direct-to-app share so the friend picks the channel they actually use:
  // WhatsApp, Message (SMS), or Email. Each opens the app pre-filled; if that
  // app isn't installed (or the deep link is refused), fall back to the native
  // share sheet so the invite is never a dead end.
  async function shareVia(medium: 'whatsapp' | 'sms' | 'email'): Promise<void> {
    if (!data) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    const msg = inviteMessage();
    const body = encodeURIComponent(msg);
    const url =
      medium === 'whatsapp'
        ? `whatsapp://send?text=${body}`
        : medium === 'sms'
          ? // iOS wants `sms:&body=`, Android `sms:?body=` for a recipient-less draft.
            `sms:${Platform.OS === 'ios' ? '&' : '?'}body=${body}`
          : `mailto:?subject=${encodeURIComponent('Join me on Fe3dr')}&body=${body}`;
    try {
      if (await Linking.canOpenURL(url)) {
        await Linking.openURL(url);
        return;
      }
    } catch {
      // Deep link refused — fall through to the native sheet.
    }
    try {
      await Share.share({ message: msg });
    } catch {
      // Share cancelled — ignore.
    }
  }

  // "More ways to share" — the OS sheet (copy link, other apps).
  async function onShare(): Promise<void> {
    if (!data) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    try {
      await Share.share({ message: inviteMessage() });
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
              Your friend gets {money(data.refereeReward)} in points to spend. You get{' '}
              {money(data.referrerReward)} in points once they place their first order.
            </Text>
          </View>

          {/* Code + share */}
          <View style={[styles.codeCard, cardShadow]}>
            <Text style={styles.codeLabel}>YOUR CODE</Text>
            <Text style={styles.code} accessibilityLabel={`Your referral code is ${data.code}`}>
              {data.code}
            </Text>
            {/* Pick the channel the friend actually uses. */}
            <View style={styles.shareRow}>
              {[
                { key: 'whatsapp' as const, label: 'WhatsApp', Icon: MessageCircle },
                { key: 'sms' as const, label: 'Message', Icon: MessageSquare },
                { key: 'email' as const, label: 'Email', Icon: Mail },
              ].map(({ key, label, Icon }) => (
                <Pressable
                  key={key}
                  onPress={() => void shareVia(key)}
                  accessibilityRole="button"
                  accessibilityLabel={`Share invite via ${label}`}
                  android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
                  style={{ flex: 1 }}
                >
                  {({ pressed }) => (
                    <View style={[styles.mediumBtn, pressed && Platform.OS === 'ios' && styles.mediumBtnPressed]}>
                      <Icon size={20} color={customerColors.coral.DEFAULT} strokeWidth={2} />
                      <Text style={styles.mediumLabel}>{label}</Text>
                    </View>
                  )}
                </Pressable>
              ))}
            </View>
            <Pressable
              onPress={() => void onShare()}
              accessibilityRole="button"
              accessibilityLabel="More ways to share"
              android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View style={[styles.moreShare, pressed && Platform.OS === 'ios' && { opacity: 0.6 }]}>
                  <Share2 size={15} color={customerColors.charcoal.soft} strokeWidth={2} />
                  <Text style={styles.moreShareText}>More ways to share</Text>
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
              <Text style={styles.statLabel}>Rewards earned</Text>
            </View>
          </View>

          {/* How it works */}
          <Text style={styles.sectionLabel}>HOW IT WORKS</Text>
          <View style={[styles.steps, cardShadow]}>
            {[
              'Share your code with friends.',
              'They sign up and place their first order.',
              `You both get loyalty points to spend on your next order.`,
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
  heroTitle: { fontFamily: 'Geist-Bold', fontSize: 22, color: customerColors.charcoal.DEFAULT, textAlign: 'center', fontVariant: ['tabular-nums'] },
  heroSub: { fontFamily: 'Inter', fontSize: 14, lineHeight: 20, color: customerColors.charcoal.soft, textAlign: 'center', fontVariant: ['tabular-nums'] },

  codeCard: { backgroundColor: customerColors.canvas, borderRadius: 16, padding: 20, alignItems: 'center', gap: 12 },
  codeLabel: { fontFamily: 'Inter-SemiBold', fontSize: 12, letterSpacing: 1.4, color: customerColors.charcoal.soft },
  code: {
    fontFamily: 'Geist-Bold',
    fontSize: 30,
    letterSpacing: 4,
    color: customerColors.charcoal.DEFAULT,
  },
  // Three channel buttons (WhatsApp / Message / Email) — equal-width, tinted
  // tiles so no single medium dominates; the coral icon carries the accent.
  shareRow: { flexDirection: 'row', gap: 10, alignSelf: 'stretch' },
  mediumBtn: {
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    paddingVertical: 12,
    borderRadius: 10,
    backgroundColor: customerColors.surface.soft,
    minHeight: 64,
  },
  mediumBtnPressed: { backgroundColor: customerColors.coral.tint },
  mediumLabel: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: customerColors.charcoal.DEFAULT },
  moreShare: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    paddingVertical: 8,
  },
  moreShareText: { fontFamily: 'Inter-Medium', fontSize: 13, color: customerColors.charcoal.soft },

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
