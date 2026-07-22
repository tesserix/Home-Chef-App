import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { CalendarDays, ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  mealPlanAdvanceBreakdown,
  useChefDailyMenu,
  useChefWeeklyMenu,
  useCreateMealPlan,
  type MealSlot,
  type MealVariant,
  type WeeklyMenuItem,
} from '../hooks/useMealPlans';
import { bookingEmptyState } from '../lib/booking-empty-state';
import {
  WeeklyMenuDayHeader,
  WeeklyMenuDishCard,
  istTodayIso,
} from '../components/chef/WeeklyMenuDishCard';

const HORIZON_DAYS = 14; // how far ahead a customer can pre-book
const LEAD_MS = 12 * 60 * 60 * 1000; // server's booking lead time (mealPlanLeadTime)

// A bookable choice for one (date, slot) — sourced from the chef's per-date menu
// (#405/#406, incl. combos) when published for that date, else the weekly cell.
interface BookableCell {
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  price: number;
  dailyMenuItemId?: string; // present for per-date items
  isCombo?: boolean;
  comboComponents?: string[];
  // Presentation-only extras for the photo-forward card — never part of the
  // selection identity (that stays dailyMenuItemId / variant).
  imageUrl?: string;
  description?: string;
}

interface Selected {
  date: string; // YYYY-MM-DD
  slot: MealSlot;
  variant: MealVariant;
  price: number;
  name: string;
  dailyMenuItemId?: string;
}

function isoDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function dayLabel(d: Date): string {
  return d.toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

const selKey = (date: string, slot: MealSlot) => `${date}-${slot}`;

// Book a tiffin plan (#196): the customer picks days over the next two weeks
// from one chef's published weekly menu, choosing a slot + veg/nonveg per day,
// then submits a single advance request → POST /meal-plans. The chef then
// accepts all or cherry-picks; a trim comes back here for approval.
export default function BookMealPlanScreen() {
  const { chefId } = useLocalSearchParams<{ chefId: string }>();
  const { data: menu, isLoading, isError, refetch } = useChefWeeklyMenu(chefId);

  // The horizon window (tomorrow .. +HORIZON_DAYS) for the per-date menu.
  const window = useMemo(() => {
    const base = new Date();
    const from = new Date(base);
    from.setDate(base.getDate() + 1);
    const to = new Date(base);
    to.setDate(base.getDate() + HORIZON_DAYS);
    return { from: isoDate(from), to: isoDate(to) };
  }, []);
  const { data: daily } = useChefDailyMenu(chefId, window.from, window.to);
  const create = useCreateMealPlan();

  const [selection, setSelection] = useState<Record<string, Selected>>({});

  // Upcoming dates (from tomorrow) the chef cooks. Each date's cells come from
  // the chef's PUBLISHED per-date menu when it has one (dynamic dishes + combos,
  // #405/#406); otherwise they fall back to the fixed weekly template.
  const dates = useMemo(() => {
    const weekly = menu?.items ?? [];
    const dailyByDate = new Map(
      (daily?.days ?? []).map((d) => [d.date, d.items]),
    );
    if (weekly.length === 0 && dailyByDate.size === 0) return [];
    const out: { date: string; label: string; cells: BookableCell[] }[] = [];
    const base = new Date();
    const cutoff = base.getTime() + LEAD_MS;
    const bySlotVariant = (a: BookableCell, b: BookableCell) =>
      a.slot === b.slot
        ? a.variant.localeCompare(b.variant)
        : a.slot.localeCompare(b.slot);

    for (let i = 1; i <= HORIZON_DAYS; i++) {
      const d = new Date(base);
      d.setDate(base.getDate() + i);
      d.setHours(0, 0, 0, 0);
      if (d.getTime() < cutoff) continue;
      const iso = isoDate(d);

      let cells: BookableCell[];
      const dayItems = dailyByDate.get(iso);
      if (dayItems && dayItems.length > 0) {
        // Per-date menu: combos first (the default choice), then à-la-carte.
        cells = dayItems
          .map((it) => ({
            slot: it.slot,
            variant: it.variant,
            name: it.name,
            price: it.price,
            dailyMenuItemId: it.id,
            isCombo: it.isCombo,
            comboComponents: it.comboComponents,
            imageUrl: it.imageUrl,
            description: it.description,
          }))
          .sort((a, b) =>
            a.isCombo === b.isCombo ? bySlotVariant(a, b) : a.isCombo ? -1 : 1,
          );
      } else {
        cells = weekly
          .filter((it) => it.dayOfWeek === d.getDay())
          .map((it) => ({
            slot: it.slot,
            variant: it.variant,
            name: it.name,
            price: it.price,
            imageUrl: it.imageUrl,
            description: it.description,
          }))
          .sort(bySlotVariant);
      }
      if (cells.length > 0) {
        out.push({ date: iso, label: dayLabel(d), cells });
      }
    }
    return out;
  }, [menu, daily]);

  // Bookable dates are the authority here, not the weekly menu. A chef who
  // publishes only per-date menus (#405 — the ones carrying combos) still has
  // a grid to show; gating on the weekly menu hid it behind "No weekly menu".
  const emptyState = bookingEmptyState(Boolean(menu?.isPublished), dates.length);

  const selected = Object.values(selection);
  const total = selected.reduce((s, x) => s + x.price, 0);

  // "Today" affordance — IST, never the device timezone. The horizon starts
  // tomorrow, so this only shows for devices running behind IST.
  const todayIso = istTodayIso();

  // One choice per (date, slot). Tapping the selected cell again clears it.
  // Cell identity is the per-date item id when present, else the weekly variant.
  function isSameCell(cur: Selected | undefined, cell: BookableCell): boolean {
    if (!cur) return false;
    return cell.dailyMenuItemId
      ? cur.dailyMenuItemId === cell.dailyMenuItemId
      : !cur.dailyMenuItemId && cur.variant === cell.variant;
  }

  function toggle(date: string, cell: BookableCell) {
    const key = selKey(date, cell.slot);
    setSelection((prev) => {
      const next = { ...prev };
      if (isSameCell(next[key], cell)) {
        delete next[key];
      } else {
        next[key] = {
          date,
          slot: cell.slot,
          variant: cell.variant,
          price: cell.price,
          name: cell.name,
          dailyMenuItemId: cell.dailyMenuItemId,
        };
      }
      return next;
    });
  }

  function submit() {
    if (!chefId || selected.length === 0) return;
    create.mutate(
      {
        chefId,
        days: selected.map((s) => ({
          date: s.date,
          slot: s.slot,
          variant: s.variant,
          ...(s.dailyMenuItemId ? { dailyMenuItemId: s.dailyMenuItemId } : {}),
        })),
      },
      {
        onSuccess: (created) => {
          // Escrow on but the server couldn't attach a payment order — don't
          // silently proceed as if the advance were collected.
          if (created.paymentError) {
            Alert.alert('Payment unavailable', created.paymentError);
            return;
          }
          // Escrow on: collect the FULL advance before the chef is notified.
          if (created.razorpayOrderId) {
            // #402: charge the SERVER total (food + GST + per-day delivery), not the
            // food-only selection sum, and show the exact breakdown before checkout so
            // the customer never pays more than what's displayed.
            const b = mealPlanAdvanceBreakdown(created.mealPlan);
            const goToPay = () =>
              router.push({
                pathname: '/payment/checkout',
                params: {
                  kind: 'mealplan',
                  mealPlanId: created.mealPlan.id,
                  razorpayOrderId: created.razorpayOrderId!,
                  razorpayKeyId: created.razorpayKeyId ?? '',
                  amount: String(b.amountPaise), // paise — the actual charge
                  currency: created.mealPlan.currency ?? 'INR',
                },
              });
            Alert.alert(
              `Pay ₹${b.total.toFixed(0)} advance`,
              `Food ₹${b.food.toFixed(2)}\nGST ₹${b.gst.toFixed(2)}\nDelivery ₹${b.delivery.toFixed(2)}\n\nTotal ₹${b.total.toFixed(2)}`,
              [
                { text: 'Back', style: 'cancel' },
                { text: 'Pay now', onPress: goToPay },
              ],
            );
            return;
          }
          // Escrow off: unpaid handshake — chef reviews and confirms the days.
          Alert.alert(
            'Request sent',
            'Your chef will review the days and confirm. You’ll be notified when they respond.',
            [{ text: 'OK', onPress: () => router.replace('/meal-plans' as never) }],
          );
        },
        onError: () =>
          Alert.alert(
            'Could not send',
            'Some days may be too soon or unavailable. Please adjust and try again.',
          ),
      },
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={8} accessibilityLabel="Go back">
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Plan your week</Text>
        <View style={{ width: 24 }} />
      </View>

      {isLoading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : isError ? (
        // The request failed (network/server) — don't masquerade as "no menu",
        // which would tell the customer the chef has nothing when they might.
        <View style={styles.centered}>
          <CalendarDays size={40} color={customerColors.charcoal.soft} strokeWidth={1.5} />
          <Text style={styles.emptyTitle}>Couldn&apos;t load the menu</Text>
          <Text style={styles.emptyText}>
            Something went wrong fetching this chef&apos;s weekly menu. Check your
            connection and try again.
          </Text>
          <Pressable
            onPress={() => void refetch()}
            accessibilityRole="button"
            style={styles.retryBtn}
          >
            <Text style={styles.retryText}>Retry</Text>
          </Pressable>
        </View>
      ) : emptyState === 'no-menu' ? (
        <View style={styles.centered}>
          <CalendarDays size={40} color={customerColors.charcoal.soft} strokeWidth={1.5} />
          <Text style={styles.emptyTitle}>No weekly menu yet</Text>
          <Text style={styles.emptyText}>
            This chef hasn&apos;t published a tiffin menu you can pre-book. Try
            ordering individual dishes instead.
          </Text>
        </View>
      ) : emptyState === 'no-dates' ? (
        <View style={styles.centered}>
          <CalendarDays size={40} color={customerColors.charcoal.soft} strokeWidth={1.5} />
          <Text style={styles.emptyTitle}>No upcoming days to book</Text>
          <Text style={styles.emptyText}>
            This chef&apos;s weekly menu has no upcoming days open for pre-booking
            right now. Please check back soon.
          </Text>
        </View>
      ) : (
        <>
          <ScrollView contentContainerStyle={styles.scroll}>
            <Text style={styles.intro}>
              Pick the meals you want over the next two weeks. Choose veg or
              non-veg for each — send your plan and the chef confirms the days
              they can cook.
            </Text>
            {dates.map((d) => (
              <View key={d.date} style={styles.daySection}>
                <WeeklyMenuDayHeader title={d.label} isToday={d.date === todayIso} />
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  style={styles.dayRow}
                  contentContainerStyle={styles.dayRowContent}
                >
                  {d.cells.map((cell) => {
                    const sel = selection[selKey(d.date, cell.slot)];
                    const active = isSameCell(sel, cell);
                    return (
                      <WeeklyMenuDishCard
                        key={cell.dailyMenuItemId ?? `${cell.slot}-${cell.variant}`}
                        name={cell.name}
                        slot={cell.slot}
                        variant={cell.variant}
                        price={cell.price}
                        imageUrl={cell.imageUrl}
                        description={cell.description}
                        isCombo={cell.isCombo}
                        comboComponents={cell.comboComponents}
                        selectable
                        selected={active}
                        onPress={() => toggle(d.date, cell)}
                      />
                    );
                  })}
                </ScrollView>
              </View>
            ))}
          </ScrollView>

          <View style={styles.footer}>
            <View style={styles.footerSummary}>
              <Text style={styles.footerCount}>
                {selected.length} meal{selected.length === 1 ? '' : 's'} · food
                subtotal
              </Text>
              <Text style={styles.footerTotal}>₹{total.toFixed(0)}</Text>
              <Text style={styles.footerNote}>
                GST &amp; delivery shown before you pay
              </Text>
            </View>
            <Pressable
              onPress={submit}
              disabled={selected.length === 0 || create.isPending}
              accessibilityRole="button"
            >
              <View
                style={[
                  styles.cta,
                  (selected.length === 0 || create.isPending) && styles.ctaDisabled,
                ]}
              >
                {create.isPending ? (
                  <ActivityIndicator color="#fff" />
                ) : (
                  <Text style={styles.ctaText}>
                    {selected.length === 0
                      ? 'Select meals to continue'
                      : `Request ${selected.length} meal${selected.length === 1 ? '' : 's'}`}
                  </Text>
                )}
              </View>
            </Pressable>
          </View>
        </>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 18,
    color: customerColors.charcoal.DEFAULT,
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 32,
    gap: 8,
  },
  emptyTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
    marginTop: 8,
  },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 20,
  },
  retryBtn: {
    marginTop: 16,
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
  },
  retryText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.coral.pressed,
  },
  scroll: { paddingHorizontal: 16, paddingBottom: 24 },
  intro: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    lineHeight: 20,
    marginBottom: 20,
  },
  daySection: { marginBottom: 24 },
  // Day rows bleed to the screen edge so a peeking card invites the scroll.
  // flexGrow: 0 — RN's ScrollView base style is flexGrow: 1, so a horizontal row
  // grows into free vertical space unless pinned (see orders.tsx filterRow).
  dayRow: { marginHorizontal: -16, flexGrow: 0 },
  dayRowContent: { paddingHorizontal: 16, gap: 12 },
  footer: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    gap: 12,
  },
  footerSummary: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  footerCount: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },
  footerTotal: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  footerNote: {
    fontFamily: 'Inter',
    fontSize: 11,
    color: customerColors.charcoal.soft,
  },
  cta: {
    height: 52,
    borderRadius: 12,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaDisabled: { backgroundColor: customerColors.charcoal.soft, opacity: 0.5 },
  ctaText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: '#fff' },
});
