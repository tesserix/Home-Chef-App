// My tiffin subscriptions (#283): list + manage (pause / resume / skip / cancel)
// and a quick adherence + fulfillment glance. Orders are placed automatically by
// the platform at the chef's cutoff — no daily tapping.

import {
  ActivityIndicator,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAlert } from '@homechef/mobile-shared/ui';
import {
  isPastMealDay,
  isSkippableMealDay,
  istCalendarDate,
  subscriptionBillingLine,
} from '@homechef/mobile-shared/utils';
import { router } from 'expo-router';
import { AlertCircle, ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { friendlyErrorMessage } from '../lib/errors';
import { useChef } from '../hooks/useChefs';
import {
  useMealSubscriptions,
  useMealSubAction,
  useMealFulfillments,
  type MealFulfillment,
  type MealSubscription,
} from '../hooks/useMealSubscription';

// Android ripple tints — translucent tokens, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;
const CORAL_GHOST_RIPPLE = `${customerColors.coral.DEFAULT}14`;

// How many upcoming days to surface per subscription. Enough to cover "I'm away
// later this week" without turning the card into a calendar.
const UPCOMING_LIMIT = 5;

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

// ─── Loading skeleton — matches SubCard proportions ──────────────────────────

function SkeletonCard() {
  return (
    <View style={styles.card}>
      <View style={styles.cardTop}>
        <View style={[styles.skeletonLine, { width: '50%', height: 15 }]} />
        <View style={[styles.skeletonChip, { width: 64, height: 20 }]} />
      </View>
      <View style={[styles.skeletonLine, { width: '70%', height: 13, marginTop: 8 }]} />
    </View>
  );
}

// ─── Error state (R8) ─────────────────────────────────────────────────────────

function ErrorState({ onRetry, retrying }: { onRetry: () => void; retrying: boolean }) {
  return (
    <View style={styles.centered}>
      <View style={styles.emptyIconWrap}>
        <AlertCircle size={28} color={customerColors.charcoal.soft} />
      </View>
      <Text style={styles.errorTitle}>Couldn&apos;t load your subscriptions</Text>
      <Text style={styles.muted}>
        This is a connection problem — your subscriptions and billing are unaffected.
      </Text>
      <Pressable
        onPress={onRetry}
        accessibilityRole="button"
        accessibilityLabel="Retry loading subscriptions"
        android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
      >
        {({ pressed }) => (
          <View style={[styles.retryBtn, pressed && Platform.OS === 'ios' && styles.pressedSurface]}>
            <Text style={styles.retryText}>{retrying ? 'Retrying…' : 'Try again'}</Text>
          </View>
        )}
      </Pressable>
    </View>
  );
}

export default function SubscriptionsScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useMealSubscriptions();
  const subs = data?.data ?? [];

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
        <Text style={styles.headerTitle}>My subscriptions</Text>
        <View style={{ width: 26 }} />
      </View>

      {isLoading ? (
        <View style={styles.scroll}>
          <SkeletonCard />
          <SkeletonCard />
        </View>
      ) : isError ? (
        // A failed fetch used to fall through to the empty state below — so a
        // network blip told a PAYING customer their subscription did not exist.
        // Alarming, and a support call. Say what actually happened and offer a retry.
        <ErrorState onRetry={() => void refetch()} retrying={isRefetching} />
      ) : subs.length === 0 ? (
        <View style={styles.centered}>
          <Text style={styles.muted}>No tiffin subscriptions yet.</Text>
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {subs.map((s) => (
            <SubCard key={s.id} sub={s} />
          ))}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function SubCard({ sub }: { sub: MealSubscription }) {
  const { showAlert } = useAlert();
  const action = useMealSubAction();
  const { data: fulfil } = useMealFulfillments(sub.id);
  const adherence = fulfil?.adherence;
  const active = sub.status === 'active';
  const paused = sub.status === 'paused';
  const terminal = sub.status === 'cancelled';
  const { data: chef } = useChef(sub.chefId);
  const chefName = chef?.data?.name;
  const billingLine = subscriptionBillingLine(sub);

  function run(a: 'pause' | 'resume' | 'cancel') {
    const confirm = a === 'cancel';
    const go = () => action.mutate({ id: sub.id, action: a });
    if (confirm) {
      showAlert('Cancel subscription?', 'You can resubscribe anytime.', [
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
    showAlert(
      'Skip this meal?',
      `${fmtDay(f.date)} · ${f.slot === 'lunch' ? 'Lunch' : 'Dinner'}. You won't be charged for it — the credit applies to your next cycle.`,
      [
        { text: 'Keep it', style: 'cancel' },
        {
          text: 'Skip',
          onPress: () =>
            action.mutate(
              // f.date arrives as an RFC3339 UTC instant (the API's default
              // time.Time JSON encoding of the fulfillment's IST-midnight
              // moment) -- the server's Skip handler only accepts a bare
              // YYYY-MM-DD, so this must be converted, not forwarded as-is
              // (see #696: this was the actual reason Skip never worked).
              { id: sub.id, action: 'skip', date: istCalendarDate(new Date(f.date)) },
              {
                onError: (err) =>
                  showAlert(
                    "Couldn't skip this meal",
                    // The server is still authoritative — this fallback only
                    // shows if its error body doesn't parse. Skip only renders
                    // for rows isSkippableMealDay already approved, so the only
                    // realistic failure is a race where IST midnight passed
                    // between screen load and tap; describe that, not a cutoff
                    // this handler never checks.
                    friendlyErrorMessage(
                      err,
                      'That day has already started, so it can no longer be skipped.',
                    ),
                  ),
              },
            ),
        },
      ],
    );
  }

  // Listing and Skip-eligibility are two different questions. Today's scheduled
  // meal is still the most relevant row on the screen and must stay listed —
  // only past days drop off. Whether Skip renders for a given row is decided
  // separately per-row below, via isSkippableMealDay, which mirrors the
  // server's Skip handler exactly (apps/api/handlers/meal_subscription.go:
  // d.After(todayIST)) so today never offers a control the server would reject,
  // regardless of the customer's device timezone (#696).
  const upcoming = (fulfil?.data ?? [])
    .filter((f) => f.status === 'scheduled' && !isPastMealDay(f.date))
    .sort((a, b) => a.date.localeCompare(b.date))
    .slice(0, UPCOMING_LIMIT);

  return (
    <View style={styles.card}>
      <View style={styles.cardTop}>
        <View style={styles.cardTitleCol}>
          {/* Three subscriptions to three kitchens rendered identically, so
              Cancel was a guess about which one (#1042). */}
          {chefName ? (
            <Text style={styles.cardChef} numberOfLines={1}>
              {chefName}
            </Text>
          ) : null}
          <Text style={styles.cardTitle}>
            {sub.slots.map((x) => (x === 'lunch' ? 'Lunch' : 'Dinner')).join(' + ')} ·{' '}
            {sub.variant === 'veg' ? 'Veg' : 'Non-veg'}
          </Text>
        </View>
        {/* Status chip per spec §2.7 — active/trial reads as in-progress
            (coral-tint), paused/past_due neutral, cancelled neutral. */}
        <View
          style={[
            styles.badge,
            (active || sub.status === 'trialing') && styles.badgeActive,
          ]}
        >
          <Text
            style={[
              styles.badgeText,
              (active || sub.status === 'trialing') && styles.badgeTextActive,
            ]}
          >
            {STATUS_LABEL[sub.status]}
          </Text>
        </View>
      </View>
      <Text style={styles.sub}>
        {sub.days.length} days/week · {sub.cadence === 'monthly' ? 'Monthly' : 'Weekly'} ·{' '}
        {money(sub.cycleAmount)}
      </Text>
      {billingLine ? <Text style={styles.billing}>{billingLine}</Text> : null}
      {sub.creditBalance > 0 ? (
        <Text style={styles.credit}>{money(sub.creditBalance)} credit applies to your next cycle</Text>
      ) : null}
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
          <Text style={styles.upcomingNote}>Skip by the end of the day before.</Text>
          {upcoming.map((f) => (
            <View key={f.id} style={styles.dayRow}>
              <View style={styles.dayInfo}>
                <Text style={styles.dayDate}>{fmtDay(f.date)}</Text>
                <Text style={styles.dayMeal} numberOfLines={1}>
                  {f.slot === 'lunch' ? 'Lunch' : 'Dinner'}
                  {f.dishName ? ` · ${f.dishName}` : ''}
                </Text>
              </View>
              {isSkippableMealDay(f.date) ? (
                <ActionBtn
                  label="Skip"
                  onPress={() => skipDay(f)}
                  pending={action.isPending}
                  accessibilityLabel={`Skip ${fmtDay(f.date)} ${f.slot === 'lunch' ? 'lunch' : 'dinner'}`}
                />
              ) : (
                // Today's meal is listed for visibility but its IST calendar
                // day has already started, so it can't be skipped — a quiet
                // label instead of dead space, matching the fallback error copy.
                <Text style={styles.dayLocked}>Already started</Text>
              )}
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
      android_ripple={
        pending ? undefined : { color: danger ? CORAL_GHOST_RIPPLE : GHOST_RIPPLE, borderless: false }
      }
    >
      {({ pressed }) => (
        <View
          style={[
            styles.actionBtn,
            danger && styles.actionBtnDanger,
            pending && styles.actionPending,
            pressed && Platform.OS === 'ios' && !pending && styles.pressedSurface,
          ]}
        >
          <Text style={[styles.actionText, danger && styles.actionTextDanger]}>{label}</Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  // White-first canvas per surface model §1 — cards separate by hairline, not
  // a grey page background.
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  emptyIconWrap: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 12,
  },
  scroll: { padding: 16, gap: 12 },
  muted: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, textAlign: 'center' },
  card: {
    backgroundColor: customerColors.canvas,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    padding: 16,
    gap: 6,
  },
  cardTop: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  cardTitleCol: { flex: 1 },
  cardChef: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  cardTitle: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  billing: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
    marginTop: 2,
  },
  sub: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, fontVariant: ['tabular-nums'] },
  credit: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.coral.pressed,
    fontVariant: ['tabular-nums'],
  },
  adherence: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, fontVariant: ['tabular-nums'] },
  // Status chip — tint bg + dark text of same family (spec §2.7 / §2 item 7).
  badge: { borderRadius: 9999, paddingHorizontal: 10, paddingVertical: 3, backgroundColor: customerColors.surface.soft },
  badgeActive: { backgroundColor: customerColors.coral.tint },
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
  pressedSurface: { backgroundColor: customerColors.surface.soft },
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
  upcomingNote: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    marginBottom: 4,
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
  dayLocked: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    paddingHorizontal: 4,
  },
  actionBtnDanger: { borderColor: customerColors.coral.DEFAULT },
  actionText: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: customerColors.charcoal.DEFAULT },
  actionTextDanger: { color: customerColors.coral.pressed },
  skeletonLine: { borderRadius: 4, backgroundColor: customerColors.hairline },
  skeletonChip: { borderRadius: 9999, backgroundColor: customerColors.surface.soft },
});
