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
  useChefWeeklyMenu,
  useCreateMealPlan,
  type MealSlot,
  type MealVariant,
  type WeeklyMenuItem,
} from '../hooks/useMealPlans';

const HORIZON_DAYS = 14; // how far ahead a customer can pre-book
const LEAD_MS = 12 * 60 * 60 * 1000; // server's booking lead time (mealPlanLeadTime)

interface Selected {
  date: string; // YYYY-MM-DD
  slot: MealSlot;
  variant: MealVariant;
  price: number;
  name: string;
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
  const create = useCreateMealPlan();

  const [selection, setSelection] = useState<Record<string, Selected>>({});

  // Upcoming dates (from tomorrow) that the chef cooks, each with its cells.
  const dates = useMemo(() => {
    const items = menu?.items ?? [];
    if (items.length === 0) return [];
    const out: { date: string; label: string; cells: WeeklyMenuItem[] }[] = [];
    const base = new Date();
    // Skip days the server's lead-time cutoff would reject (a late-evening
    // booking can't take tomorrow's meal). Mirrors mealPlanLeadTime on the API.
    const cutoff = base.getTime() + LEAD_MS;
    for (let i = 1; i <= HORIZON_DAYS; i++) {
      const d = new Date(base);
      d.setDate(base.getDate() + i);
      d.setHours(0, 0, 0, 0);
      if (d.getTime() < cutoff) continue;
      const cells = items
        .filter((it) => it.dayOfWeek === d.getDay())
        .sort((a, b) =>
          a.slot === b.slot
            ? a.variant.localeCompare(b.variant)
            : a.slot.localeCompare(b.slot),
        );
      if (cells.length > 0) {
        out.push({ date: isoDate(d), label: dayLabel(d), cells });
      }
    }
    return out;
  }, [menu]);

  const selected = Object.values(selection);
  const total = selected.reduce((s, x) => s + x.price, 0);

  function toggle(date: string, cell: WeeklyMenuItem) {
    const key = selKey(date, cell.slot);
    setSelection((prev) => {
      const next = { ...prev };
      const cur = next[key];
      if (cur && cur.variant === cell.variant) {
        delete next[key]; // tapping the same chip clears the slot
      } else {
        next[key] = {
          date,
          slot: cell.slot,
          variant: cell.variant,
          price: cell.price,
          name: cell.name,
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
            router.push({
              pathname: '/payment/checkout',
              params: {
                kind: 'mealplan',
                mealPlanId: created.mealPlan.id,
                razorpayOrderId: created.razorpayOrderId,
                razorpayKeyId: created.razorpayKeyId ?? '',
                amount: String(Math.round(total * 100)), // paise
                currency: created.mealPlan.currency ?? 'INR',
              },
            });
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
      ) : !menu?.isPublished ? (
        <View style={styles.centered}>
          <CalendarDays size={40} color={customerColors.charcoal.soft} strokeWidth={1.5} />
          <Text style={styles.emptyTitle}>No weekly menu yet</Text>
          <Text style={styles.emptyText}>
            This chef hasn&apos;t published a tiffin menu you can pre-book. Try
            ordering individual dishes instead.
          </Text>
        </View>
      ) : dates.length === 0 ? (
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
                <Text style={styles.dayLabel}>{d.label}</Text>
                <View style={styles.cellWrap}>
                  {d.cells.map((cell) => {
                    const sel = selection[selKey(d.date, cell.slot)];
                    const active = sel?.variant === cell.variant;
                    const accent =
                      cell.variant === 'veg'
                        ? customerColors.success.DEFAULT
                        : customerColors.destructive.DEFAULT;
                    return (
                      <Pressable
                        key={`${cell.slot}-${cell.variant}`}
                        onPress={() => toggle(d.date, cell)}
                        accessibilityRole="button"
                        accessibilityState={{ selected: active }}
                      >
                        <View
                          style={[styles.cell, active && styles.cellActive]}
                        >
                          <View style={styles.cellTop}>
                            <View style={[styles.dot, { backgroundColor: accent }]} />
                            <Text style={styles.cellSlot}>
                              {cell.slot === 'lunch' ? 'Lunch' : 'Dinner'} ·{' '}
                              {cell.variant === 'veg' ? 'Veg' : 'Non-veg'}
                            </Text>
                          </View>
                          <Text style={styles.cellName} numberOfLines={1}>
                            {cell.name}
                          </Text>
                          <Text style={styles.cellPrice}>₹{cell.price.toFixed(0)}</Text>
                        </View>
                      </Pressable>
                    );
                  })}
                </View>
              </View>
            ))}
          </ScrollView>

          <View style={styles.footer}>
            <View style={styles.footerSummary}>
              <Text style={styles.footerCount}>
                {selected.length} meal{selected.length === 1 ? '' : 's'} selected
              </Text>
              <Text style={styles.footerTotal}>₹{total.toFixed(0)}</Text>
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
  daySection: { marginBottom: 20 },
  dayLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 10,
  },
  cellWrap: { flexDirection: 'row', flexWrap: 'wrap', gap: 10 },
  cell: {
    width: 150,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    padding: 12,
    gap: 4,
  },
  cellActive: {
    borderColor: customerColors.coral.DEFAULT,
    backgroundColor: customerColors.coral.tint,
  },
  cellTop: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 2 },
  cellSlot: {
    fontFamily: 'Inter-Medium',
    fontSize: 12,
    color: customerColors.charcoal.soft,
  },
  cellName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
  },
  cellPrice: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
  },
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
