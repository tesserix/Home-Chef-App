// My tiffin subscriptions (#283): list + manage (pause / resume / skip / cancel)
// and a quick adherence + fulfillment glance. Orders are placed automatically by
// the platform at the chef's cutoff — no daily tapping.

import { ActivityIndicator, Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { friendlyErrorMessage } from '../lib/errors';
import {
  useMealSubscriptions,
  useMealSubAction,
  useMealFulfillments,
  type MealFulfillment,
  type MealSubscription,
} from '../hooks/useMealSubscription';

// How many upcoming days to surface per subscription. Enough to cover "I'm away
// later this week" without turning the card into a calendar.
const UPCOMING_LIMIT = 5;

function startOfToday(): number {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d.getTime();
}

function fmtDay(iso: string): string {
  const d = new Date(iso);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diff = Math.round((new Date(iso).setHours(0, 0, 0, 0) - today.getTime()) / 86400000);
  if (diff === 0) return 'Today';
  if (diff === 1) return 'Tomorrow';
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' });
}

const STATUS_LABEL: Record<MealSubscription['status'], string> = {
  trialing: 'Trial',
  active: 'Active',
  paused: 'Paused',
  past_due: 'Payment due',
  cancelled: 'Cancelled',
};

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

export default function SubscriptionsScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useMealSubscriptions();
  const subs = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={10} accessibilityLabel="Go back">
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>My subscriptions</Text>
        <View style={{ width: 26 }} />
      </View>

      {isLoading ? (
        <View style={styles.centered}><ActivityIndicator color={customerColors.coral.DEFAULT} /></View>
      ) : isError ? (
        // A failed fetch used to fall through to the empty state below — so a
        // network blip told a PAYING customer their subscription did not exist.
        // Alarming, and a support call. Say what actually happened and offer a retry.
        <View style={styles.centered}>
          <Text style={styles.errorTitle}>Couldn&apos;t load your subscriptions</Text>
          <Text style={styles.muted}>
            This is a connection problem — your subscriptions and billing are unaffected.
          </Text>
          <Pressable
            onPress={() => void refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry loading subscriptions"
            style={({ pressed }) => [styles.retryBtn, pressed && styles.pressed]}
          >
            <Text style={styles.retryText}>{isRefetching ? 'Retrying…' : 'Try again'}</Text>
          </Pressable>
        </View>
      ) : subs.length === 0 ? (
        <View style={styles.centered}><Text style={styles.muted}>No tiffin subscriptions yet.</Text></View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {subs.map((s) => <SubCard key={s.id} sub={s} />)}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function SubCard({ sub }: { sub: MealSubscription }) {
  const action = useMealSubAction();
  const { data: fulfil } = useMealFulfillments(sub.id);
  const adherence = fulfil?.adherence;
  const active = sub.status === 'active';
  const paused = sub.status === 'paused';
  const terminal = sub.status === 'cancelled';

  function run(a: 'pause' | 'resume' | 'cancel') {
    const confirm = a === 'cancel';
    const go = () => action.mutate({ id: sub.id, action: a });
    if (confirm) {
      Alert.alert('Cancel subscription?', 'You can resubscribe anytime.', [
        { text: 'Keep', style: 'cancel' },
        { text: 'Cancel it', style: 'destructive', onPress: go },
      ]);
    } else {
      go();
    }
  }

  // Skip a single day — the #1 tiffin support request ("I'm away Thursday").
  // The endpoint and the hook already supported this; nothing ever called it, so
  // every skip was a phone call.
  function skipDay(f: MealFulfillment) {
    Alert.alert(
      'Skip this meal?',
      `${fmtDay(f.date)} · ${f.slot === 'lunch' ? 'Lunch' : 'Dinner'}. You won't be charged for it — the credit applies to your next cycle.`,
      [
        { text: 'Keep it', style: 'cancel' },
        {
          text: 'Skip',
          onPress: () =>
            action.mutate(
              { id: sub.id, action: 'skip', date: f.date },
              {
                onError: (err) =>
                  Alert.alert(
                    "Couldn't skip this meal",
                    // The server is authoritative on the cutoff; surfacing its
                    // reason beats guessing at the customer ("it MAY be too
                    // close" is what the meal-plan screen says, and it reads as
                    // an app that doesn't know its own rules).
                    friendlyErrorMessage(err, 'It may already be past the cutoff for that day.'),
                  ),
              },
            ),
        },
      ],
    );
  }

  // Only future, still-scheduled days can be skipped. Past/placed/delivered days
  // are shown for context but carry no action — offering a control that always
  // fails is worse than offering none.
  const upcoming = (fulfil?.data ?? [])
    .filter((f) => f.status === 'scheduled' && new Date(f.date).getTime() >= startOfToday())
    .sort((a, b) => a.date.localeCompare(b.date))
    .slice(0, UPCOMING_LIMIT);

  return (
    <View style={styles.card}>
      <View style={styles.cardTop}>
        <Text style={styles.cardTitle}>{sub.slots.map((x) => (x === 'lunch' ? 'Lunch' : 'Dinner')).join(' + ')} · {sub.variant === 'veg' ? 'Veg' : 'Non-veg'}</Text>
        <View style={[styles.badge, active && styles.badgeActive, paused && styles.badgePaused]}>
          <Text style={[styles.badgeText, active && styles.badgeTextActive]}>{STATUS_LABEL[sub.status]}</Text>
        </View>
      </View>
      <Text style={styles.sub}>{sub.days.length} days/week · {sub.cadence === 'monthly' ? 'Monthly' : 'Weekly'} · {money(sub.cycleAmount)}</Text>
      {sub.creditBalance > 0 ? <Text style={styles.credit}>{money(sub.creditBalance)} credit applies to your next cycle</Text> : null}
      {adherence ? (
        <Text style={styles.adherence}>
          {adherence.delivered} delivered · {adherence.skipped} skipped · {adherence.missed} missed
        </Text>
      ) : null}

      {/* Upcoming meals with a per-day Skip. This is the control the whole screen
          exists for — a subscriber's routine question is "not Thursday", not
          "cancel everything". */}
      {!terminal && upcoming.length > 0 && (
        <View style={styles.upcoming}>
          <Text style={styles.upcomingLabel}>Upcoming</Text>
          {upcoming.map((f) => (
            <View key={f.id} style={styles.dayRow}>
              <View style={styles.dayInfo}>
                <Text style={styles.dayDate}>{fmtDay(f.date)}</Text>
                <Text style={styles.dayMeal} numberOfLines={1}>
                  {f.slot === 'lunch' ? 'Lunch' : 'Dinner'}
                  {f.dishName ? ` · ${f.dishName}` : ''}
                </Text>
              </View>
              <ActionBtn
                label="Skip"
                onPress={() => skipDay(f)}
                pending={action.isPending}
                accessibilityLabel={`Skip ${fmtDay(f.date)} ${f.slot === 'lunch' ? 'lunch' : 'dinner'}`}
              />
            </View>
          ))}
        </View>
      )}

      {!terminal && (
        <View style={styles.actions}>
          {active && <ActionBtn label="Pause" onPress={() => run('pause')} pending={action.isPending} />}
          {paused && <ActionBtn label="Resume" onPress={() => run('resume')} pending={action.isPending} />}
          <ActionBtn label="Cancel" danger onPress={() => run('cancel')} pending={action.isPending} />
        </View>
      )}
    </View>
  );
}

function ActionBtn({
  label,
  onPress,
  danger,
  pending,
  accessibilityLabel,
}: {
  label: string;
  onPress: () => void;
  danger?: boolean;
  pending?: boolean;
  accessibilityLabel?: string;
}) {
  return (
    <Pressable
      onPress={onPress}
      // Cancel was double-tappable: no disabled state and no accessibilityState
      // while the mutation was in flight, so an impatient second tap fired a
      // second cancel. Disabling in-flight is the fix, and the state also has to
      // reach screen readers, not just sighted users.
      disabled={pending}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityState={{ disabled: !!pending, busy: !!pending }}
      style={({ pressed }) => [pressed && !pending && styles.pressed, pending && styles.actionPending]}
    >
      <View style={[styles.actionBtn, danger && styles.actionBtnDanger]}>
        <Text style={[styles.actionText, danger && styles.actionTextDanger]}>{label}</Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.surface.soft },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16, paddingVertical: 12 },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  scroll: { padding: 16, gap: 12 },
  muted: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  card: { backgroundColor: customerColors.canvas, borderRadius: 12, padding: 16, gap: 6 },
  cardTop: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  cardTitle: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT, flex: 1 },
  sub: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  credit: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: customerColors.coral.pressed },
  adherence: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft },
  badge: { borderRadius: 9999, paddingHorizontal: 10, paddingVertical: 3, backgroundColor: customerColors.surface.soft },
  badgeActive: { backgroundColor: customerColors.coral.tint },
  badgePaused: { backgroundColor: customerColors.surface.soft },
  badgeText: { fontFamily: 'Inter-SemiBold', fontSize: 11, color: customerColors.charcoal.soft },
  badgeTextActive: { color: customerColors.coral.pressed },
  actions: { flexDirection: 'row', gap: 8, marginTop: 8 },
  // minHeight 44 is the WCAG 2.2 / HIG target floor. These were ~34px (13px text
  // + 8px padding) — Pause/Resume/Cancel, on a screen about money, below the
  // minimum. Padding alone can't be trusted to reach it once text scales.
  actionBtn: {
    borderWidth: 1,
    borderColor: customerColors.hairline,
    borderRadius: 8,
    paddingHorizontal: 16,
    paddingVertical: 8,
    minHeight: 44,
    justifyContent: 'center',
  },
  actionPending: { opacity: 0.5 },
  pressed: { opacity: 0.7 },
  errorTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 6,
    textAlign: 'center',
  },
  retryBtn: {
    marginTop: 16,
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: 20,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.hairline,
  },
  retryText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  upcoming: { marginTop: 12, gap: 4 },
  upcomingLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    marginBottom: 2,
  },
  dayRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
    paddingVertical: 4,
  },
  dayInfo: { flex: 1 },
  dayDate: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  dayMeal: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft },
  actionBtnDanger: { borderColor: customerColors.coral.DEFAULT },
  actionText: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: customerColors.charcoal.DEFAULT },
  actionTextDanger: { color: customerColors.coral.pressed },
});
