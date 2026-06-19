// Premium chef tier — upgrade flow + perks (#44). All prices and perks come from
// the API (admin-configurable); nothing is hardcoded. The chef toggles between
// the standard and premium tiers; perks (Verified-Pro badge, priority ranking,
// lower commission, advanced analytics) activate immediately on upgrade.

import { useState } from 'react';
import { ActivityIndicator, Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft, Check, Sparkles } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import {
  useChefSubscription,
  useSubscriptionPlans,
  useChangeTier,
  type PlanOption,
} from '../hooks/useSubscription';

const CURRENCY_SYMBOL: Record<string, string> = { INR: '₹', USD: '$' };

function money(amount: number, currency: string): string {
  const sym = CURRENCY_SYMBOL[currency] ?? '';
  return `${sym}${Math.round(amount).toLocaleString('en-IN')}`;
}

const INTERVAL_LABEL: Record<PlanOption['interval'], string> = {
  monthly: 'per month',
  quarterly: 'per quarter',
  yearly: 'per year',
};

export default function PremiumScreen() {
  const sub = useChefSubscription();
  const plans = useSubscriptionPlans();
  const changeTier = useChangeTier();
  const [pendingTier, setPendingTier] = useState<'premium' | 'standard' | null>(null);

  const isPremium = sub.data?.subscription?.tier === 'premium';
  const loading = sub.isLoading || plans.isLoading;

  function onChange(tier: 'premium' | 'standard') {
    setPendingTier(tier);
    changeTier.mutate(tier, {
      onSuccess: () => {
        setPendingTier(null);
        Alert.alert(
          tier === 'premium' ? 'Welcome to Premium' : 'Switched to Standard',
          tier === 'premium'
            ? 'Your premium perks are now active.'
            : 'Your subscription is back on the standard tier.',
        );
      },
      onError: (err: unknown) => {
        setPendingTier(null);
        const status = (err as { response?: { status?: number } })?.response?.status;
        Alert.alert(
          'Could not change tier',
          status === 404
            ? 'Start your subscription first, then upgrade to Premium.'
            : 'Please try again in a moment.',
        );
      },
    });
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={10} accessibilityLabel="Go back">
          <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Premium</Text>
        <View style={{ width: 26 }} />
      </View>

      {loading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={theme.colors.herb.DEFAULT} />
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {/* Status hero */}
          <View style={[styles.hero, isPremium && styles.heroActive]}>
            <View style={styles.heroIcon}>
              <Sparkles size={22} color={isPremium ? theme.colors.paper : theme.colors.herb.DEFAULT} />
            </View>
            <Text style={[styles.heroTitle, isPremium && styles.heroTitleActive]}>
              {isPremium ? "You're on Premium" : 'Upgrade to Premium'}
            </Text>
            <Text style={[styles.heroSub, isPremium && styles.heroSubActive]}>
              {isPremium
                ? 'Your Verified-Pro perks are active.'
                : 'Stand out, rank higher, and keep more of every order.'}
            </Text>
          </View>

          {/* Perks */}
          <Text style={styles.sectionLabel}>WHAT YOU GET</Text>
          <View style={styles.card}>
            {(plans.data?.premiumPerks ?? []).map((perk) => (
              <View key={perk} style={styles.perkRow}>
                <Check size={18} color={theme.colors.herb.DEFAULT} strokeWidth={2.5} />
                <Text style={styles.perkText}>{perk}</Text>
              </View>
            ))}
          </View>

          {/* Pricing — admin-configurable, fetched from the API */}
          <Text style={styles.sectionLabel}>PREMIUM PRICING</Text>
          <View style={styles.card}>
            {(plans.data?.premiumPlans ?? []).map((p) => (
              <View key={p.interval} style={styles.priceRow}>
                <View style={{ flex: 1 }}>
                  <Text style={styles.priceInterval}>{INTERVAL_LABEL[p.interval]}</Text>
                  {p.savingsPercent && p.savingsPercent > 0 ? (
                    <Text style={styles.priceSaving}>Save {Math.round(p.savingsPercent)}%</Text>
                  ) : null}
                </View>
                <Text style={styles.priceAmount}>{money(p.amount, plans.data?.currency ?? 'INR')}</Text>
              </View>
            ))}
            <Text style={styles.priceNote}>
              Billed on your current cycle. Premium replaces the standard plan fee.
            </Text>
          </View>

          {/* CTA */}
          <View style={styles.ctaWrap}>
            {isPremium ? (
              <Pressable
                onPress={() => onChange('standard')}
                disabled={changeTier.isPending}
                accessibilityRole="button"
                accessibilityLabel="Switch to the standard tier"
              >
                <View style={styles.ctaSecondary}>
                  {pendingTier === 'standard' ? (
                    <ActivityIndicator color={theme.colors.ink.soft} />
                  ) : (
                    <Text style={styles.ctaSecondaryText}>Switch to Standard</Text>
                  )}
                </View>
              </Pressable>
            ) : (
              <Pressable
                onPress={() => onChange('premium')}
                disabled={changeTier.isPending}
                accessibilityRole="button"
                accessibilityLabel="Upgrade to premium"
              >
                <View style={styles.ctaPrimary}>
                  {pendingTier === 'premium' ? (
                    <ActivityIndicator color={theme.colors.paper} />
                  ) : (
                    <Text style={styles.ctaPrimaryText}>Upgrade to Premium</Text>
                  )}
                </View>
              </Pressable>
            )}
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  scroll: { padding: theme.spacing[4], paddingBottom: theme.spacing[8], gap: theme.spacing[4] },

  hero: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[1],
    padding: theme.spacing[5],
    alignItems: 'center',
    gap: theme.spacing[2],
  },
  heroActive: { backgroundColor: theme.colors.herb.DEFAULT },
  heroIcon: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: theme.colors.herb.tint,
    alignItems: 'center',
    justifyContent: 'center',
  },
  heroTitle: { fontFamily: 'Geist-Bold', fontSize: 22, color: theme.colors.ink.DEFAULT, textAlign: 'center' },
  heroTitleActive: { color: theme.colors.paper },
  heroSub: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.soft, textAlign: 'center' },
  heroSubActive: { color: theme.colors.paper },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: -theme.spacing[2],
    marginTop: theme.spacing[1],
  },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[1],
    padding: theme.spacing[4],
    gap: theme.spacing[3],
  },
  perkRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[3] },
  perkText: { flex: 1, fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.DEFAULT },

  priceRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingBottom: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  priceInterval: { fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.DEFAULT, textTransform: 'capitalize' },
  priceSaving: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: theme.colors.herb.DEFAULT, marginTop: 2 },
  priceAmount: { fontFamily: 'Inter-SemiBold', fontSize: 18, color: theme.colors.ink.DEFAULT, fontVariant: ['tabular-nums'] },
  priceNote: { fontFamily: 'Inter', fontSize: 12, lineHeight: 16, color: theme.colors.ink.muted },

  ctaWrap: { marginTop: theme.spacing[2] },
  ctaPrimary: {
    backgroundColor: theme.colors.herb.DEFAULT,
    borderRadius: theme.radius.md,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaPrimaryText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.paper },
  ctaSecondary: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
  },
  ctaSecondaryText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.soft },
});
