import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { CalendarDays, ChevronLeft } from 'lucide-react-native';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';

// Android ripple tints — translucent tokens, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.coral.DEFAULT}14`;
import {
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
  istTodayIso,
} from '../components/chef/WeeklyMenuDishCard';
import { MealPlanBookRow } from '../components/chef/MealPlanBookRow';

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

// IST (UTC+5:30, no DST) date helpers so the client's bookable-day set matches the
// server, which interprets each YYYY-MM-DD as IST-midnight and applies the 12h lead
// from there. Using device-local midnight instead offered days the server then
// rejected as "too soon" (client/server lead-time mismatch).
const IST_OFFSET_MS = 5.5 * 60 * 60 * 1000;

// The IST calendar date `offsetDays` from `now`: its YYYY-MM-DD and the real (UTC)
// instant of that IST midnight.
function istDay(now: Date, offsetDays: number): { iso: string; midnightMs: number } {
  const wall = new Date(now.getTime() + IST_OFFSET_MS);
  wall.setUTCDate(wall.getUTCDate() + offsetDays);
  const y = wall.getUTCFullYear();
  const m = String(wall.getUTCMonth() + 1).padStart(2, '0');
  const d = String(wall.getUTCDate()).padStart(2, '0');
  const iso = `${y}-${m}-${d}`;
  return { iso, midnightMs: Date.parse(`${iso}T00:00:00+05:30`) };
}

// Human label ("Mon, 28 Jul") for an ISO date, rendered in IST so it matches the iso.
function istDayLabel(iso: string): string {
  return new Date(`${iso}T12:00:00+05:30`).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    timeZone: 'Asia/Kolkata',
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

  // The horizon window (tomorrow .. +HORIZON_DAYS) for the per-date menu, in IST.
  const window = useMemo(() => {
    const now = new Date();
    return { from: istDay(now, 1).iso, to: istDay(now, HORIZON_DAYS).iso };
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
    const now = new Date();
    const cutoffMs = now.getTime() + LEAD_MS;
    const bySlotVariant = (a: BookableCell, b: BookableCell) =>
      a.slot === b.slot
        ? a.variant.localeCompare(b.variant)
        : a.slot.localeCompare(b.slot);

    for (let i = 1; i <= HORIZON_DAYS; i++) {
      const { iso, midnightMs } = istDay(now, i);
      // IST-midnight within the 12h lead → the server rejects the day, so don't
      // offer it (matches the server's mealPlanLeadTime cutoff).
      if (midnightMs < cutoffMs) continue;
      // Weekday of THIS calendar date (IST), for the weekly-template match — a date
      // is a fixed weekday regardless of device tz.
      const dow = new Date(`${iso}T00:00:00Z`).getUTCDay();

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
        // Weekly template fallback. Forward the cell's combo fields (#192 thali/
        // combo) so a weekly combo renders as one on the card, and — like the
        // per-date branch — float combos ahead of à-la-carte within a slot.
        cells = weekly
          .filter((it) => it.dayOfWeek === dow)
          .map((it) => ({
            slot: it.slot,
            variant: it.variant,
            name: it.name,
            price: it.price,
            isCombo: it.isCombo,
            comboComponents: it.comboComponents,
            imageUrl: it.imageUrl,
            description: it.description,
          }))
          .sort((a, b) =>
            a.isCombo === b.isCombo ? bySlotVariant(a, b) : a.isCombo ? -1 : 1,
          );
      }
      if (cells.length > 0) {
        out.push({ date: iso, label: istDayLabel(iso), cells });
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
        onSuccess: () => {
          // Payment now happens AFTER the chef responds and the customer approves
          // (on the meal-plan detail screen) — NOT here. Sending the request charges
          // nothing; the chef reviews the days, then you approve and pay to lock it in.
          Alert.alert(
            'Request sent',
            'Your chef will review the days and confirm what they can cook. When they respond, you approve and pay to lock it in — nothing is charged yet.',
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
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: ICON_RIPPLE, borderless: true }}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Plan your week</Text>
        <View style={{ width: 24 }} />
      </View>

      {isLoading ? (
        <View style={styles.scroll}>
          <View style={[styles.skeletonLine, { width: '80%', height: 14, marginBottom: 20 }]} />
          {[0, 1].map((i) => (
            <View key={i} style={styles.daySection}>
              <View style={[styles.skeletonLine, { width: 120, height: 16, marginBottom: 12 }]} />
              <View style={{ flexDirection: 'row', gap: 12 }}>
                <View style={styles.skeletonCard} />
                <View style={styles.skeletonCard} />
              </View>
            </View>
          ))}
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
            accessibilityLabel="Retry loading the weekly menu"
            android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View style={[styles.retryBtn, pressed && Platform.OS === 'ios' && styles.retryBtnPressed]}>
                <Text style={styles.retryText}>Retry</Text>
              </View>
            )}
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
            {dates.map((d) => {
              // Group the day's cells by slot so "pick one per slot" reads
              // clearly (lunch first, then dinner). Combos are already floated
              // ahead within each slot by the `dates` builder.
              const groups = (['lunch', 'dinner'] as MealSlot[])
                .map((slot) => ({
                  slot,
                  label: slot === 'lunch' ? 'Lunch' : 'Dinner',
                  cells: d.cells.filter((c) => c.slot === slot),
                }))
                .filter((g) => g.cells.length > 0);
              return (
                <View key={d.date} style={styles.daySection}>
                  <WeeklyMenuDayHeader title={d.label} isToday={d.date === todayIso} />
                  <View style={styles.dayCard}>
                    {groups.map((g, gi) => (
                      <View key={g.slot}>
                        <View
                          style={[styles.slotHeader, gi > 0 && styles.slotHeaderDivided]}
                        >
                          <Text style={styles.slotLabel}>{g.label}</Text>
                          {g.cells.length > 1 ? (
                            <Text style={styles.slotHint}>pick one</Text>
                          ) : null}
                        </View>
                        {g.cells.map((cell, ci) => {
                          const sel = selection[selKey(d.date, cell.slot)];
                          const active = isSameCell(sel, cell);
                          return (
                            <MealPlanBookRow
                              key={cell.dailyMenuItemId ?? `${cell.slot}-${cell.variant}`}
                              name={cell.name}
                              variant={cell.variant}
                              price={cell.price}
                              isCombo={cell.isCombo}
                              comboComponents={cell.comboComponents}
                              selected={active}
                              divided={ci > 0}
                              onPress={() => toggle(d.date, cell)}
                            />
                          );
                        })}
                      </View>
                    ))}
                  </View>
                </View>
              );
            })}
          </ScrollView>

          {/* Sticky CTA bar per spec §2.5 — white, top hairline + shadow[2],
              coral filled, radius 8, 52pt. Distinct disabled state. */}
          <View style={styles.footer}>
            <View style={styles.footerSummary}>
              <View style={styles.footerRow}>
                <Text style={styles.footerCount}>
                  {selected.length} meal{selected.length === 1 ? '' : 's'} · food subtotal
                </Text>
                <Text style={styles.footerTotal}>₹{total.toFixed(0)}</Text>
              </View>
              <Text style={styles.footerNote}>
                GST &amp; delivery shown before you pay
              </Text>
            </View>
            <Pressable
              onPress={submit}
              disabled={selected.length === 0 || create.isPending}
              accessibilityRole="button"
              accessibilityLabel={
                selected.length === 0
                  ? 'Select meals to continue'
                  : `Request ${selected.length} meal${selected.length === 1 ? '' : 's'}`
              }
              android_ripple={
                selected.length === 0 || create.isPending
                  ? undefined
                  : { color: CANVAS_RIPPLE, borderless: false }
              }
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.cta,
                    (selected.length === 0 || create.isPending) && styles.ctaDisabled,
                    pressed &&
                      Platform.OS === 'ios' &&
                      selected.length > 0 &&
                      !create.isPending &&
                      styles.ctaPressed,
                  ]}
                >
                  {create.isPending ? (
                    <ActivityIndicator color={customerColors.canvas} />
                  ) : (
                    <Text style={styles.ctaText}>
                      {selected.length === 0
                        ? 'Select meals to continue'
                        : `Request ${selected.length} meal${selected.length === 1 ? '' : 's'}`}
                    </Text>
                  )}
                </View>
              )}
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
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: 24,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
  },
  retryBtnPressed: { backgroundColor: customerColors.coral.tint },
  retryText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.coral.pressed,
    textAlign: 'center',
  },
  scroll: { paddingHorizontal: 16, paddingTop: 16, paddingBottom: 24 },
  skeletonLine: { borderRadius: 4, backgroundColor: customerColors.hairline },
  skeletonCard: {
    width: 148,
    height: 176,
    borderRadius: 12,
    backgroundColor: customerColors.surface.soft,
  },
  intro: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    lineHeight: 20,
    marginBottom: 20,
  },
  daySection: { marginBottom: 20 },
  // Each day is one hairline-bordered group; rows inside are separated by
  // hairlines (chrome-light — no per-row card borders). overflow:hidden clips
  // the coral selected-row tint to the rounded corners.
  dayCard: {
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
    overflow: 'hidden',
  },
  slotHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 14,
    paddingTop: 12,
    paddingBottom: 6,
  },
  slotHeaderDivided: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
  },
  slotLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.4,
    textTransform: 'uppercase',
    color: customerColors.charcoal.soft,
  },
  slotHint: {
    fontFamily: 'Inter',
    fontSize: 11,
    color: customerColors.charcoal.soft,
  },
  footer: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    gap: 12,
    shadowColor: customerTheme.shadow[2].shadowColor,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: customerTheme.shadow[2].shadowOpacity,
    shadowRadius: customerTheme.shadow[2].shadowRadius,
    elevation: customerTheme.shadow[2].elevation,
  },
  footerSummary: { gap: 2 },
  footerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'baseline',
    gap: 12,
  },
  footerCount: {
    flexShrink: 1,
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
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
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaPressed: { backgroundColor: customerColors.coral.pressed },
  ctaDisabled: { backgroundColor: customerColors.charcoal.soft, opacity: 0.5 },
  ctaText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas },
});
