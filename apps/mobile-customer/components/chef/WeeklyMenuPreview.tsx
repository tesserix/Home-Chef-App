// Read-only weekly-menu preview on the chef detail screen (#1) — COMPACT,
// one-day-at-a-time. Instead of stacking all seven day sections (which made the
// screen scroll forever), a slim day-selector chip row picks a single day
// (defaulting to Today in IST) and a veg/non-veg chip row filters it; only that
// day's dishes render below as photo-forward WeeklyMenuDishCards. Booking still
// lives in the separate meal-plan flow. Same export + props as before — pass
// the items, it renders nothing if the menu is empty.

import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { DietIcon } from '@homechef/mobile-shared/ui';
import type { WeeklyMenuItem } from '../../hooks/useMealPlans';
import { WeeklyMenuDishCard, istTodayWeekday } from './WeeklyMenuDishCard';

// Week starts Monday; Sunday (0) sorts last.
const DAY_ORDER = [1, 2, 3, 4, 5, 6, 0];
const DAY_NAMES = [
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
];
const DAY_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const SLOT_ORDER: WeeklyMenuItem['slot'][] = ['lunch', 'dinner'];

// Chef-detail section gutter — the dish row bleeds to the screen edge and pads
// back in so a scrolled card peeks past the margin (appetite over alignment).
const GUTTER = 20;

type DietFilter = 'all' | 'veg' | 'nonveg';

const DIET_FILTERS: { key: DietFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'veg', label: 'Veg' },
  { key: 'nonveg', label: 'Non-veg' },
];

interface ChipProps {
  label: string;
  active: boolean;
  onPress: () => void;
  accessibilityLabel: string;
  /** FSSAI diet mark inside the Veg / Non-veg chips. */
  dietKind?: 'veg' | 'non-veg';
}

// Shared chip for the day selector and the diet filter. Active = coral tint +
// coral text (selected-chip accent); inactive = hairline outline. Visual +
// layout styles live on the inner View (iOS Pressable style-function bug).
function Chip({ label, active, onPress, accessibilityLabel, dietKind }: ChipProps) {
  return (
    <Pressable
      onPress={onPress}
      // Chip is visually slim; hitSlop tops the touch target up to ≥44pt.
      hitSlop={{ top: 8, bottom: 8 }}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      accessibilityState={{ selected: active }}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.chip,
            active && styles.chipActive,
            pressed && styles.chipPressed,
          ]}
        >
          {dietKind ? <DietIcon kind={dietKind} size={12} /> : null}
          <Text style={[styles.chipLabel, active && styles.chipLabelActive]}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

export function WeeklyMenuPreview({ items }: { items: WeeklyMenuItem[] }) {
  // null = "no explicit pick yet" → fall back to Today / first available day.
  const [selectedDay, setSelectedDay] = useState<number | null>(null);
  const [diet, setDiet] = useState<DietFilter>('all');

  if (!items || items.length === 0) return null;

  // Group by day, keeping only days that have cells.
  const byDay = new Map<number, WeeklyMenuItem[]>();
  for (const it of items) {
    const arr = byDay.get(it.dayOfWeek) ?? [];
    byDay.set(it.dayOfWeek, [...arr, it]);
  }
  const days = DAY_ORDER.filter((d) => byDay.has(d));
  const today = istTodayWeekday();

  // Default to Today (IST); if today has no dishes, the first available day.
  const fallbackDay = days.includes(today) ? today : (days[0] ?? today);
  const activeDay =
    selectedDay !== null && byDay.has(selectedDay) ? selectedDay : fallbackDay;

  const dayCells = (byDay.get(activeDay) ?? []).slice().sort((a, b) => {
    const s = SLOT_ORDER.indexOf(a.slot) - SLOT_ORDER.indexOf(b.slot);
    return s !== 0 ? s : a.variant.localeCompare(b.variant);
  });
  const filteredCells =
    diet === 'all'
      ? dayCells
      : dayCells.filter((c) =>
          diet === 'veg' ? c.variant === 'veg' : c.variant !== 'veg',
        );

  const activeDayName =
    activeDay === today ? 'today' : (DAY_NAMES[activeDay] ?? '');

  return (
    <View style={styles.root}>
      <Text style={styles.heading}>This week&apos;s menu</Text>
      <Text style={styles.sub}>Fixed menu — same dish for every subscriber.</Text>

      {/* Day selector — Today + the weekdays that have dishes. */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        style={styles.chipRowBleed}
        contentContainerStyle={styles.chipRowContent}
      >
        {days.map((day) => (
          <Chip
            key={day}
            label={day === today ? 'Today' : (DAY_SHORT[day] ?? '')}
            active={day === activeDay}
            onPress={() => setSelectedDay(day)}
            accessibilityLabel={`Show ${DAY_NAMES[day] ?? ''} dishes`}
          />
        ))}
      </ScrollView>

      {/* Veg / non-veg filter. */}
      <View style={styles.dietRow}>
        {DIET_FILTERS.map((f) => (
          <Chip
            key={f.key}
            label={f.label}
            active={diet === f.key}
            onPress={() => setDiet(f.key)}
            accessibilityLabel={
              f.key === 'all' ? 'Show all dishes' : `Show ${f.label} dishes`
            }
            dietKind={
              f.key === 'veg' ? 'veg' : f.key === 'nonveg' ? 'non-veg' : undefined
            }
          />
        ))}
      </View>

      {/* Only the selected day's dishes. */}
      {filteredCells.length === 0 ? (
        <Text style={styles.emptyDay}>
          No {diet === 'veg' ? 'veg' : 'non-veg'} dishes on {activeDayName}.
        </Text>
      ) : (
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          style={styles.dishRowBleed}
          contentContainerStyle={styles.dishRowContent}
        >
          {filteredCells.map((c, i) => (
            <WeeklyMenuDishCard
              key={c.id ?? `${activeDay}-${c.slot}-${c.variant}-${i}`}
              name={c.name}
              slot={c.slot}
              variant={c.variant}
              price={c.price}
              imageUrl={c.imageUrl}
              description={c.description}
            />
          ))}
        </ScrollView>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { paddingHorizontal: GUTTER, paddingTop: 16 },
  heading: {
    fontFamily: 'Geist-Bold',
    fontSize: 18,
    color: customerColors.charcoal.DEFAULT,
  },
  sub: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    marginTop: 2,
    marginBottom: 12,
  },

  // ---- Chips ----
  // Bleed the day row to the screen edge so chips scroll under the margin.
  // flexGrow: 0 — RN's ScrollView base style is flexGrow: 1, so a horizontal row
  // grows into free vertical space unless pinned (see orders.tsx filterRow).
  chipRowBleed: { marginHorizontal: -GUTTER, flexGrow: 0 },
  chipRowContent: { paddingHorizontal: GUTTER, gap: 8 },
  dietRow: { flexDirection: 'row', gap: 8, marginTop: 10 },
  chip: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
    minHeight: 32,
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
  },
  chipActive: {
    borderColor: customerColors.coral.DEFAULT,
    backgroundColor: customerColors.coral.tint,
  },
  chipPressed: {
    opacity: 0.7,
  },
  chipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    letterSpacing: 0.1,
    color: customerColors.charcoal.soft,
  },
  chipLabelActive: {
    color: customerColors.coral.pressed,
  },

  // ---- Selected day's dishes ----
  dishRowBleed: { marginHorizontal: -GUTTER, marginTop: 14, flexGrow: 0 },
  dishRowContent: { paddingHorizontal: GUTTER, gap: 12 },
  emptyDay: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    marginTop: 14,
  },
});
