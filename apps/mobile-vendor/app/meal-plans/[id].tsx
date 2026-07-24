import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { Button } from '@homechef/mobile-shared/ui';
import {
  useChefMealPlanRequests,
  useRespondMealPlan,
  type MealPlanDay,
} from '../../hooks/useMealPlans';

function dayLabel(d: MealPlanDay): string {
  return new Date(d.date).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

// Chef respond screen (#195): accept every day, or cherry-pick the days the chef
// can cook (the rest are declined). A trim routes the plan back to the customer
// for approval; accept-all confirms immediately. Mirrors RespondMealPlan (API).
export default function MealPlanRespondScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading } = useChefMealPlanRequests('pending_chef');
  const respond = useRespondMealPlan();

  const plan = useMemo(
    () => data?.data.find((p) => p.id === id),
    [data, id],
  );

  // Days the chef will NOT cook (excluded). Default: cook everything.
  const [excluded, setExcluded] = useState<Set<string>>(new Set());

  if (isLoading) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator color={theme.colors.ink.DEFAULT} />
      </SafeAreaView>
    );
  }

  if (!plan) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <Header />
        <View style={styles.centered}>
          <Text style={styles.muted}>
            This request is no longer pending — it may already be handled.
          </Text>
        </View>
      </SafeAreaView>
    );
  }

  const days = plan.days ?? [];
  const acceptedDays = days.filter((d) => !excluded.has(d.id));
  const acceptAll = excluded.size === 0;
  const acceptedTotal = acceptedDays.reduce((s, d) => s + (d.price ?? 0), 0);

  function toggle(dayId: string) {
    setExcluded((prev) => {
      const next = new Set(prev);
      if (next.has(dayId)) next.delete(dayId);
      else next.add(dayId);
      return next;
    });
  }

  function submit() {
    if (!plan) return;
    if (acceptedDays.length === 0) {
      Alert.alert(
        'Pick at least one day',
        'Accept at least one day, or decline the whole request from the requests list.',
      );
      return;
    }
    const confirmMsg = acceptAll
      ? `Accept all ${days.length} days? The customer still needs to approve & pay before it's confirmed.`
      : `Cook ${acceptedDays.length} of ${days.length} days? The customer must approve the change.`;
    Alert.alert('Confirm response', confirmMsg, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Confirm',
        onPress: () => {
          respond.mutate(
            {
              id: plan.id,
              acceptAll,
              acceptedDayIds: acceptAll ? [] : acceptedDays.map((d) => d.id),
            },
            {
              onSuccess: () => {
                Alert.alert(
                  'Sent for approval',
                  acceptAll
                    ? 'The customer will review and pay the advance to confirm.'
                    : 'The customer will review and approve your revised plan.',
                );
                router.back();
              },
              onError: () =>
                Alert.alert('Could not submit', 'Please try again.'),
            },
          );
        },
      },
    ]);
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <Header />
      <ScrollView contentContainerStyle={styles.scroll}>
        <Text style={styles.planNo}>{plan.mealPlanNumber}</Text>
        <Text style={styles.summary}>
          {days.length} day{days.length === 1 ? '' : 's'} requested · ₹
          {plan.total.toFixed(0)}
        </Text>
        <Text style={styles.hint}>
          Toggle off any day you can&apos;t cook. Leave all on to accept the
          full plan.
        </Text>

        <View style={styles.card}>
          {days.map((d, i) => {
            const on = !excluded.has(d.id);
            return (
              <View
                key={d.id}
                style={[styles.dayRow, i < days.length - 1 && styles.dayDivider]}
              >
                <View style={{ flex: 1 }}>
                  <Text style={[styles.dayDate, !on && styles.dimmed]}>
                    {dayLabel(d)}
                  </Text>
                  <View style={styles.dayMeta}>
                    <Text style={[styles.slot, !on && styles.dimmed]}>
                      {d.slot === 'lunch' ? 'Lunch' : 'Dinner'}
                    </Text>
                    <View
                      style={[
                        styles.variantDot,
                        {
                          borderColor:
                            d.variant === 'veg'
                              ? theme.colors.diet.veg
                              : theme.colors.diet.nonVeg,
                        },
                      ]}
                    >
                      <View
                        style={[
                          styles.variantDotInner,
                          {
                            backgroundColor:
                              d.variant === 'veg'
                                ? theme.colors.diet.veg
                                : theme.colors.diet.nonVeg,
                          },
                        ]}
                      />
                    </View>
                    <Text style={[styles.dish, !on && styles.dimmed]} numberOfLines={1}>
                      {d.dishName ?? '—'}
                    </Text>
                  </View>
                </View>
                <Text style={[styles.price, !on && styles.dimmed]}>
                  ₹{(d.price ?? 0).toFixed(0)}
                </Text>
                <Switch
                  value={on}
                  onValueChange={() => toggle(d.id)}
                  trackColor={{ true: theme.colors.ink.DEFAULT }}
                  style={styles.switch}
                  accessibilityLabel={`${dayLabel(d)} ${d.slot === 'lunch' ? 'lunch' : 'dinner'}, ${on ? 'cooking' : 'declined'}, tap to toggle`}
                />
              </View>
            );
          })}
        </View>
      </ScrollView>

      <View style={styles.footer}>
        <View style={styles.footerSummary}>
          <Text style={styles.footerLabel}>
            {acceptedDays.length} of {days.length} days
          </Text>
          <Text style={styles.footerTotal}>₹{acceptedTotal.toFixed(0)}</Text>
        </View>
        <Button
          label={acceptAll ? 'Accept all days' : `Confirm ${acceptedDays.length} days`}
          variant="primary"
          size="lg"
          loading={respond.isPending}
          onPress={submit}
        />
      </View>
    </SafeAreaView>
  );
}

function Header() {
  return (
    <View style={styles.header}>
      <Pressable
        onPress={() => router.back()}
        hitSlop={8}
        accessibilityRole="button"
        accessibilityLabel="Go back"
        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
      >
        {({ pressed }) => (
          <View style={pressed && Platform.OS === 'ios' && { opacity: 0.6 }}>
            <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
          </View>
        )}
      </Pressable>
      <Text style={styles.title}>Review request</Text>
      <View style={{ width: 24 }} />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: theme.spacing[6],
    backgroundColor: theme.colors.bone,
  },
  muted: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: theme.colors.ink.muted,
    textAlign: 'center',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: theme.colors.ink.DEFAULT,
  },
  scroll: { padding: theme.spacing[4], paddingBottom: theme.spacing[10] },
  planNo: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: theme.colors.ink.muted,
  },
  summary: {
    fontFamily: 'Geist-Bold',
    fontSize: 22,
    color: theme.colors.ink.DEFAULT,
    marginTop: 2,
    fontVariant: ['tabular-nums'],
  },
  hint: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.soft,
    marginTop: theme.spacing[2],
    marginBottom: theme.spacing[4],
    lineHeight: 18,
  },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    paddingHorizontal: theme.spacing[4],
    ...theme.shadow[1],
  },
  dayRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingVertical: theme.spacing[3],
  },
  dayDivider: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  dayDate: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
  },
  dayMeta: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    marginTop: 3,
  },
  slot: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft },
  variantDot: {
    width: 14,
    height: 14,
    borderRadius: 3,
    borderWidth: 1.5,
    alignItems: 'center',
    justifyContent: 'center',
  },
  variantDotInner: { width: 7, height: 7, borderRadius: 1.5 },
  dish: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.soft,
  },
  price: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  switch: { marginLeft: theme.spacing[1] },
  dimmed: { color: theme.colors.ink.muted, textDecorationLine: 'line-through' },
  footer: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[4],
    backgroundColor: theme.colors.paper,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    gap: theme.spacing[3],
  },
  footerSummary: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  footerLabel: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: theme.colors.ink.soft,
  },
  footerTotal: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
});
