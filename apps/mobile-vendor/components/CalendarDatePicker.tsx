import { useMemo, useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { ChevronLeft, ChevronRight } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';

const WEEKDAYS = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];
const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

function toISO(year: number, month: number, day: number): string {
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

interface CalendarDatePickerProps {
  value: string;
  onChange: (iso: string) => void;
  maxDate?: Date;
  minDate?: Date;
}

export function CalendarDatePicker({ value, onChange, maxDate, minDate }: CalendarDatePickerProps) {
  const selected = /^\d{4}-\d{2}-\d{2}$/.test(value) ? value : null;
  const initial = selected ? new Date(`${selected}T12:00:00`) : new Date();
  const [year, setYear] = useState(initial.getFullYear());
  const [month, setMonth] = useState(initial.getMonth());

  const { leadingBlanks, daysInMonth } = useMemo(() => {
    const first = new Date(year, month, 1);
    return {
      leadingBlanks: first.getDay(),
      daysInMonth: new Date(year, month + 1, 0).getDate(),
    };
  }, [year, month]);

  function shiftMonth(delta: number): void {
    const next = new Date(year, month + delta, 1);
    setYear(next.getFullYear());
    setMonth(next.getMonth());
  }

  function isDisabled(day: number): boolean {
    const d = new Date(year, month, day, 12);
    if (maxDate && d > maxDate) return true;
    if (minDate && d < minDate) return true;
    return false;
  }

  const cells: (number | null)[] = [
    ...Array.from({ length: leadingBlanks }, () => null),
    ...Array.from({ length: daysInMonth }, (_, i) => i + 1),
  ];
  while (cells.length % 7 !== 0) cells.push(null);

  const canGoNext = !maxDate || new Date(year, month + 1, 1) <= maxDate;

  return (
    <View style={styles.root}>
      <View style={styles.header}>
        <Pressable
          onPress={() => shiftMonth(-1)}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Previous month"
        >
          {({ pressed }) => (
            <View style={[styles.navBtn, pressed && { opacity: 0.6 }]}>
              <ChevronLeft size={18} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.monthLabel}>
          {MONTHS[month]} {year}
        </Text>
        <Pressable
          onPress={() => shiftMonth(1)}
          disabled={!canGoNext}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Next month"
        >
          {({ pressed }) => (
            <View style={[styles.navBtn, (pressed || !canGoNext) && { opacity: !canGoNext ? 0.25 : 0.6 }]}>
              <ChevronRight size={18} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
      </View>

      <View style={styles.grid}>
        {WEEKDAYS.map((w, i) => (
          <View key={`w${i}`} style={styles.cell}>
            <Text style={styles.weekday}>{w}</Text>
          </View>
        ))}
        {cells.map((day, i) => {
          if (day === null) return <View key={`b${i}`} style={styles.cell} />;
          const iso = toISO(year, month, day);
          const active = iso === selected;
          const disabled = isDisabled(day);
          return (
            <View key={iso} style={styles.cell}>
              <Pressable
                onPress={() => onChange(iso)}
                disabled={disabled}
                accessibilityRole="button"
                accessibilityLabel={`${day} ${MONTHS[month]} ${year}`}
                accessibilityState={{ selected: active, disabled }}
              >
                <View style={[styles.day, active && styles.dayActive]}>
                  <Text
                    style={[
                      styles.dayText,
                      active && styles.dayTextActive,
                      disabled && styles.dayTextDisabled,
                    ]}
                  >
                    {day}
                  </Text>
                </View>
              </Pressable>
            </View>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    padding: theme.spacing[3],
    gap: theme.spacing[2],
  },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  navBtn: {
    width: 32,
    height: 32,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.bone,
  },
  monthLabel: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  grid: { flexDirection: 'row', flexWrap: 'wrap' },
  cell: {
    width: `${100 / 7}%`,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 2,
  },
  weekday: { fontFamily: 'Inter-Medium', fontSize: 11, color: theme.colors.ink.muted },
  day: {
    width: 36,
    height: 36,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: theme.radius.full,
  },
  dayActive: { backgroundColor: theme.colors.herb.DEFAULT },
  dayText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  dayTextActive: { color: theme.colors.paper, fontFamily: 'Inter-SemiBold' },
  dayTextDisabled: { color: theme.colors.mist.strong },
});
