// Read-only weekly-menu preview on the chef detail screen (#1). Renders the
// chef's published fixed weekly menu — one dish per day per slot — as a
// photo-forward horizontal row of dish cards per day (shared with the
// book-meal-plan screen via WeeklyMenuDishCard, so both surfaces speak one
// visual language). Booking against it lives in the separate meal-plan flow;
// this is purely "here's what this chef cooks each day". Reusable +
// presentational: pass it the items, it groups by day and renders nothing if
// the menu is empty.

import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { WeeklyMenuItem } from '../../hooks/useMealPlans';
import {
  WeeklyMenuDayHeader,
  WeeklyMenuDishCard,
  istTodayWeekday,
} from './WeeklyMenuDishCard';

// Week starts Monday; Sunday (0) sorts last.
const DAY_ORDER = [1, 2, 3, 4, 5, 6, 0];
const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
const SLOT_ORDER: WeeklyMenuItem['slot'][] = ['lunch', 'dinner'];

// Chef-detail section gutter — the day rows bleed to the screen edge and pad
// back in so a scrolled row peeks past the margin (appetite over alignment).
const GUTTER = 20;

export function WeeklyMenuPreview({ items }: { items: WeeklyMenuItem[] }) {
  if (!items || items.length === 0) return null;

  // Group by day → slot order, keeping only days that have cells.
  const byDay = new Map<number, WeeklyMenuItem[]>();
  for (const it of items) {
    const arr = byDay.get(it.dayOfWeek) ?? [];
    arr.push(it);
    byDay.set(it.dayOfWeek, arr);
  }
  const days = DAY_ORDER.filter((d) => byDay.has(d));
  const today = istTodayWeekday();

  return (
    <View style={styles.root}>
      <Text style={styles.heading}>This week&apos;s menu</Text>
      <Text style={styles.sub}>A fixed menu — every subscriber gets the same dish each day.</Text>

      {days.map((day) => {
        const cells = (byDay.get(day) ?? []).slice().sort((a, b) => {
          const s = SLOT_ORDER.indexOf(a.slot) - SLOT_ORDER.indexOf(b.slot);
          return s !== 0 ? s : a.variant.localeCompare(b.variant);
        });
        return (
          <View key={day} style={styles.dayBlock}>
            <WeeklyMenuDayHeader title={DAY_NAMES[day] ?? ''} isToday={day === today} />
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              style={styles.dayRow}
              contentContainerStyle={styles.dayRowContent}
            >
              {cells.map((c, i) => (
                <WeeklyMenuDishCard
                  key={c.id ?? `${day}-${c.slot}-${c.variant}-${i}`}
                  name={c.name}
                  slot={c.slot}
                  variant={c.variant}
                  price={c.price}
                  imageUrl={c.imageUrl}
                  description={c.description}
                />
              ))}
            </ScrollView>
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { paddingHorizontal: GUTTER, paddingTop: 16, gap: 4 },
  heading: { fontFamily: 'Geist-Bold', fontSize: 18, color: customerColors.charcoal.DEFAULT },
  sub: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginBottom: 8 },
  dayBlock: { marginTop: 12 },
  // Bleed the row to the screen edge so cards scroll under the margin.
  dayRow: { marginHorizontal: -GUTTER },
  dayRowContent: { paddingHorizontal: GUTTER, gap: 12 },
});
