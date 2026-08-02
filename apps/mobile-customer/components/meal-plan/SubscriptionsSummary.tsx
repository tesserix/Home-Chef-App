import { useMemo } from 'react';
import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import { CalendarCheck, ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  pickSubscriptionMealDay,
  subscriptionRowSummary,
} from '@homechef/mobile-shared/utils';

import {
  useMealFulfillments,
  useMealSubscriptions,
  type MealSubscription,
} from '../../hooks/useMealSubscription';

// SubscriptionsSummary — Plans-tab section for recurring tiffin subscriptions (#900).
// The tab previously rendered only <MealPlanList />, which reads one-off meal_plans and
// tells an active-subscription-only customer they have "no plans" at all. This adds one
// row per active/trialing subscription showing today's-or-next scheduled meal; tapping
// opens the existing /subscriptions screen where Skip/Pause/Cancel already live — no new
// inline actions are built here.

const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;

export function SubscriptionsSummary() {
  const { data, isLoading, isError, refetch } = useMealSubscriptions();

  if (isLoading) {
    return (
      <View style={styles.wrap}>
        <SkeletonRow />
      </View>
    );
  }

  if (isError) {
    return (
      <View style={styles.wrap}>
        <View style={styles.errorRow}>
          <Text style={styles.errorText}>Couldn't load your tiffin subscription</Text>
          <Pressable
            onPress={() => void refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry loading your subscription"
          >
            {({ pressed }) => (
              <View style={[styles.retryBtn, pressed && Platform.OS === 'ios' && styles.pressed]}>
                <Text style={styles.retryText}>Try again</Text>
              </View>
            )}
          </Pressable>
        </View>
      </View>
    );
  }

  const activeSubscriptions = (data?.data ?? []).filter(
    (s) => s.status === 'active' || s.status === 'trialing',
  );
  if (activeSubscriptions.length === 0) return null;

  return (
    <View style={styles.wrap}>
      {activeSubscriptions.map((sub) => (
        <SubscriptionRow key={sub.id} sub={sub} />
      ))}
    </View>
  );
}

function SubscriptionRow({ sub }: { sub: MealSubscription }) {
  const { data: fulfil, isLoading, isError } = useMealFulfillments(sub.id);
  const picked = useMemo(
    () => (fulfil?.data ? pickSubscriptionMealDay(fulfil.data) : null),
    [fulfil?.data],
  );
  const summary = subscriptionRowSummary(isLoading, isError, picked);

  return (
    <Pressable
      onPress={() => router.push('/subscriptions')}
      accessibilityRole="button"
      accessibilityLabel={`Manage your tiffin subscription — ${summary.replace(/^Tiffin · /, '')}`}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View style={[styles.card, pressed && Platform.OS === 'ios' && styles.pressed]}>
          <View style={styles.iconWrap}>
            <CalendarCheck size={18} color={customerColors.coral.DEFAULT} />
          </View>
          <Text style={styles.summary} numberOfLines={1}>
            {summary}
          </Text>
          <ChevronRight size={18} color={customerColors.charcoal.soft} />
        </View>
      )}
    </Pressable>
  );
}

function SkeletonRow() {
  return (
    <View style={[styles.card, styles.skeletonCard]}>
      <View style={[styles.skeletonLine, { width: '60%', height: 14 }]} />
      <View style={[styles.skeletonLine, { width: '40%', height: 12 }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { paddingHorizontal: 16, paddingTop: 8, gap: 12 },
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    backgroundColor: customerColors.canvas,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    padding: 16,
  },
  pressed: { backgroundColor: customerColors.surface.soft },
  skeletonCard: { flexDirection: 'column', alignItems: 'stretch', gap: 6 },
  iconWrap: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.coral.tint,
  },
  summary: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  errorRow: {
    backgroundColor: customerColors.canvas,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    padding: 16,
    gap: 12,
    alignItems: 'flex-start',
  },
  errorText: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  retryBtn: {
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: 20,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.hairline,
  },
  retryText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  skeletonLine: { borderRadius: 4, backgroundColor: customerColors.hairline },
});
