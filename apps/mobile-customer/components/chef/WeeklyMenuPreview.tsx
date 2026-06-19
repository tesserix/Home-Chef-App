// Read-only weekly-menu preview on the chef detail screen (#1). Renders the
// chef's published fixed weekly menu — one dish per day per slot, with a veg /
// non-veg dot. Booking against it lives in the separate meal-plan flow; this is
// purely "here's what this chef cooks each day". Reusable + presentational: pass
// it the items, it groups by day and renders nothing if the menu is empty.

import { StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { WeeklyMenuItem } from '../../hooks/useMealPlans';

// Week starts Monday; Sunday (0) sorts last.
const DAY_ORDER = [1, 2, 3, 4, 5, 6, 0];
const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
const SLOT_ORDER: WeeklyMenuItem['slot'][] = ['lunch', 'dinner'];

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

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
            <Text style={styles.dayName}>{DAY_NAMES[day]}</Text>
            <View style={styles.cells}>
              {cells.map((c, i) => {
                const isVeg = c.variant === 'veg';
                return (
                  <View key={c.id ?? `${day}-${c.slot}-${c.variant}-${i}`} style={styles.cell}>
                    <View style={styles.cellTop}>
                      <View
                        style={[
                          styles.dot,
                          { backgroundColor: isVeg ? customerColors.success.DEFAULT : customerColors.destructive.DEFAULT },
                        ]}
                      />
                      <Text style={styles.cellSlot}>
                        {c.slot === 'lunch' ? 'Lunch' : 'Dinner'} · {isVeg ? 'Veg' : 'Non-veg'}
                      </Text>
                    </View>
                    <Text style={styles.cellName} numberOfLines={2}>
                      {c.name}
                    </Text>
                    {c.price > 0 ? <Text style={styles.cellPrice}>{money(c.price)}</Text> : null}
                  </View>
                );
              })}
            </View>
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { paddingHorizontal: 20, paddingTop: 16, gap: 4 },
  heading: { fontFamily: 'Geist-Bold', fontSize: 18, color: customerColors.charcoal.DEFAULT },
  sub: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginBottom: 8 },
  dayBlock: { marginTop: 8 },
  dayName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 8,
  },
  cells: { flexDirection: 'row', flexWrap: 'wrap', gap: 10 },
  cell: {
    width: 150,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    borderRadius: 12,
    padding: 12,
    gap: 4,
    backgroundColor: customerColors.canvas,
  },
  cellTop: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 2 },
  cellSlot: { fontFamily: 'Inter-SemiBold', fontSize: 11, color: customerColors.charcoal.soft },
  cellName: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  cellPrice: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
});
