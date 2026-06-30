import { useEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { useFormDraft } from '@homechef/mobile-shared/hooks';
import { theme } from '@homechef/mobile-shared/theme';
import { Button } from '@homechef/mobile-shared/ui';
import {
  useSaveWeeklyMenu,
  useWeeklyMenu,
  weeklyMenuHole,
  type MealSlot,
  type MealVariant,
  type WeeklyMenuItem,
} from '../../hooks/useMealPlans';

// Day-of-week display order (Mon-first), mapped to the API's 0=Sun..6=Sat.
const DAYS: { dow: number; short: string; long: string }[] = [
  { dow: 1, short: 'Mon', long: 'Monday' },
  { dow: 2, short: 'Tue', long: 'Tuesday' },
  { dow: 3, short: 'Wed', long: 'Wednesday' },
  { dow: 4, short: 'Thu', long: 'Thursday' },
  { dow: 5, short: 'Fri', long: 'Friday' },
  { dow: 6, short: 'Sat', long: 'Saturday' },
  { dow: 0, short: 'Sun', long: 'Sunday' },
];

const SLOTS: { slot: MealSlot; label: string }[] = [
  { slot: 'lunch', label: 'Lunch' },
  { slot: 'dinner', label: 'Dinner' },
];

const VARIANTS: { variant: MealVariant; label: string; color: string }[] = [
  { variant: 'veg', label: 'Veg', color: theme.colors.diet.veg },
  { variant: 'nonveg', label: 'Non-veg', color: theme.colors.diet.nonVeg },
];

interface Cell {
  name: string;
  price: string;
}

const cellKey = (dow: number, slot: MealSlot, variant: MealVariant) =>
  `${dow}-${slot}-${variant}`;

// Weekly-menu editor (#192/#195): the fixed dishes a chef offers per
// day × slot × veg/nonveg. Customers pre-book against these cells (#196).
// Replace-all save mirrors PutWeeklyMenu.
export default function WeeklyMenuEditorScreen() {
  const { data, isLoading } = useWeeklyMenu();
  const save = useSaveWeeklyMenu();
  // Local backup of the in-progress grid — a pre-save safety net so an app
  // background/kill before the server "Save draft" doesn't lose the chef's edits.
  const { ready, draft, saveDraft, clearDraft } =
    useFormDraft<Record<string, Cell>>('weekly-menu-draft');

  const [selectedDow, setSelectedDow] = useState<number>(1);
  const [cells, setCells] = useState<Record<string, Cell>>({});
  const [published, setPublished] = useState(false);
  const [hydrated, setHydrated] = useState(false);

  // Hydrate once from the server payload; later refetches don't clobber edits.
  // Wait for the draft load too: an unsaved local backup (most recent edits)
  // wins over the server copy when present.
  useEffect(() => {
    if (hydrated || !data || !ready) return;
    if (draft && Object.keys(draft).length > 0) {
      setCells(draft);
    } else {
      const next: Record<string, Cell> = {};
      for (const it of data.items ?? []) {
        next[cellKey(it.dayOfWeek, it.slot, it.variant)] = {
          name: it.name ?? '',
          price: it.price ? String(it.price) : '',
        };
      }
      setCells(next);
    }
    setPublished(Boolean(data.isPublished));
    setHydrated(true);
  }, [data, hydrated, ready, draft]);

  // Persist the grid as a local backup on every change, once hydrated. Skip the
  // hydration-coincident run so we only back up the chef's own edits, not an
  // untouched server copy (which would otherwise always shadow fresh data).
  const persistArmed = useRef(false);
  useEffect(() => {
    if (!hydrated) return;
    if (!persistArmed.current) {
      persistArmed.current = true;
      return;
    }
    saveDraft(cells);
  }, [hydrated, cells, saveDraft]);

  const filledByDay = useMemo(() => {
    const set = new Set<number>();
    for (const key of Object.keys(cells)) {
      const c = cells[key];
      if (c && c.name.trim()) {
        const dow = Number(key.split('-')[0]);
        set.add(dow);
      }
    }
    return set;
  }, [cells]);

  function setCell(
    dow: number,
    slot: MealSlot,
    variant: MealVariant,
    patch: Partial<Cell>,
  ) {
    const key = cellKey(dow, slot, variant);
    setCells((prev) => {
      const base: Cell = prev[key] ?? { name: '', price: '' };
      return { ...prev, [key]: { ...base, ...patch } };
    });
  }

  // Copy convenience (#1): replicate the selected day's cells to every other day.
  function copyDayToAll() {
    setCells((prev) => {
      const next = { ...prev };
      for (const s of SLOTS) {
        for (const v of VARIANTS) {
          const src = prev[cellKey(selectedDow, s.slot, v.variant)];
          for (const d of DAYS) {
            if (d.dow === selectedDow) continue;
            next[cellKey(d.dow, s.slot, v.variant)] = src ? { ...src } : { name: '', price: '' };
          }
        }
      }
      return next;
    });
  }

  function buildItems(): WeeklyMenuItem[] {
    const items: WeeklyMenuItem[] = [];
    for (const key of Object.keys(cells)) {
      const c = cells[key];
      if (!c || !c.name.trim()) continue;
      const [dowStr, slot, variant] = key.split('-');
      items.push({
        dayOfWeek: Number(dowStr),
        slot: slot as MealSlot,
        variant: variant as MealVariant,
        name: c.name.trim(),
        price: Number.parseFloat(c.price) || 0,
      });
    }
    return items;
  }

  function onSave(nextPublished: boolean) {
    const items = buildItems();
    if (nextPublished) {
      // Match the API's complete-grid rule (#1) so the chef sees the real reason
      // (e.g. "Tue is missing its dinner dish") instead of a generic 400.
      const hole = weeklyMenuHole(items);
      if (hole) {
        Alert.alert('Finish the week first', hole);
        return;
      }
    }
    save.mutate(
      { isPublished: nextPublished, items },
      {
        onSuccess: () => {
          // Server now holds the grid — the local backup is redundant.
          clearDraft();
          setPublished(nextPublished);
          Alert.alert(
            'Saved',
            nextPublished
              ? 'Your weekly menu is live — customers can pre-book it.'
              : 'Saved as a draft (not visible to customers yet).',
          );
        },
        // Surface the server's reason (e.g. the validatePublishableGrid 400
        // message naming the missing day) instead of a generic failure.
        onError: (err) =>
          Alert.alert(
            nextPublished ? 'Could not publish' : 'Could not save',
            getServerErrorMessage(
              err,
              nextPublished
                ? 'Fill every offered day before publishing, then try again.'
                : 'Please try again.',
            ),
          ),
      },
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator color={theme.colors.herb.DEFAULT} />
      </SafeAreaView>
    );
  }

  const day = DAYS.find((d) => d.dow === selectedDow) ?? DAYS[0]!;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
        >
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Weekly menu</Text>
        <View style={{ width: 24 }} />
      </View>

      {/* Day selector */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.dayTabs}
      >
        {DAYS.map((d) => {
          const active = d.dow === selectedDow;
          return (
            <Pressable
              key={d.dow}
              onPress={() => setSelectedDow(d.dow)}
              accessibilityRole="button"
            >
              <View style={[styles.dayTab, active && styles.dayTabActive]}>
                <Text style={[styles.dayTabText, active && styles.dayTabTextActive]}>
                  {d.short}
                </Text>
                {filledByDay.has(d.dow) ? (
                  <View
                    style={[
                      styles.dot,
                      { backgroundColor: active ? theme.colors.paper : theme.colors.herb.DEFAULT },
                    ]}
                  />
                ) : (
                  <View style={styles.dotPlaceholder} />
                )}
              </View>
            </Pressable>
          );
        })}
      </ScrollView>

      <ScrollView
        contentContainerStyle={styles.scroll}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.dayHeadingRow}>
          <Text style={styles.dayHeading}>{day.long}</Text>
          <Pressable onPress={copyDayToAll} hitSlop={8} accessibilityRole="button" accessibilityLabel="Copy this day to all days">
            <Text style={styles.copyLink}>Copy to all days</Text>
          </Pressable>
        </View>
        {SLOTS.map((s) => (
          <View key={s.slot} style={styles.slotBlock}>
            <Text style={styles.slotLabel}>{s.label}</Text>
            {VARIANTS.map((v) => {
              const key = cellKey(day.dow, s.slot, v.variant);
              const c = cells[key];
              return (
                <View key={v.variant} style={styles.cellRow}>
                  <View style={[styles.variantTag, { borderColor: v.color }]}>
                    <View style={[styles.variantTagDot, { backgroundColor: v.color }]} />
                    <Text style={[styles.variantTagText, { color: v.color }]}>
                      {v.label}
                    </Text>
                  </View>
                  <TextInput
                    style={styles.nameInput}
                    placeholder="Dish name"
                    placeholderTextColor={theme.colors.ink.muted}
                    value={c?.name ?? ''}
                    onChangeText={(t) => setCell(day.dow, s.slot, v.variant, { name: t })}
                  />
                  <TextInput
                    style={styles.priceInput}
                    placeholder="₹0"
                    placeholderTextColor={theme.colors.ink.muted}
                    keyboardType="numeric"
                    value={c?.price ?? ''}
                    onChangeText={(t) =>
                      setCell(day.dow, s.slot, v.variant, {
                        price: t.replace(/[^0-9.]/g, ''),
                      })
                    }
                  />
                </View>
              );
            })}
          </View>
        ))}

        <View style={styles.publishRow}>
          <View style={{ flex: 1 }}>
            <Text style={styles.publishLabel}>Published</Text>
            <Text style={styles.publishCaption}>
              {published
                ? 'Live — customers can pre-book this menu'
                : 'Draft — not visible to customers'}
            </Text>
          </View>
          <Switch
            value={published}
            onValueChange={(v) => onSave(v)}
            trackColor={{ true: theme.colors.herb.DEFAULT }}
          />
        </View>
      </ScrollView>

      <View style={styles.footer}>
        <Button
          label={published ? 'Save changes' : 'Save draft'}
          variant="primary"
          loading={save.isPending}
          onPress={() => onSave(published)}
        />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  dayTabs: {
    paddingHorizontal: theme.spacing[4],
    gap: theme.spacing[2],
    paddingBottom: theme.spacing[3],
  },
  dayTab: {
    minWidth: 48,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.paper,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    alignItems: 'center',
    gap: 4,
  },
  dayTabActive: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderColor: theme.colors.ink.DEFAULT,
  },
  dayTabText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: theme.colors.ink.soft,
  },
  dayTabTextActive: { color: theme.colors.paper },
  dot: { width: 5, height: 5, borderRadius: 2.5 },
  dotPlaceholder: { width: 5, height: 5 },
  scroll: { paddingHorizontal: theme.spacing[4], paddingBottom: theme.spacing[10] },
  dayHeadingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: theme.spacing[3],
  },
  dayHeading: {
    fontFamily: 'Geist-Bold',
    fontSize: 18,
    color: theme.colors.ink.DEFAULT,
  },
  copyLink: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: theme.colors.herb.DEFAULT,
  },
  slotBlock: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    marginBottom: theme.spacing[3],
    ...theme.shadow[1],
  },
  slotLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    letterSpacing: 0.3,
    textTransform: 'uppercase',
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[3],
  },
  cellRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    marginBottom: theme.spacing[3],
  },
  variantTag: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 6,
    borderRadius: theme.radius.sm,
    borderWidth: 1,
    width: 78,
  },
  variantTagDot: { width: 8, height: 8, borderRadius: 2 },
  variantTagText: { fontFamily: 'Inter-SemiBold', fontSize: 12 },
  nameInput: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
  },
  priceInput: {
    width: 64,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[2],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    textAlign: 'center',
  },
  publishRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    marginTop: theme.spacing[2],
    ...theme.shadow[1],
  },
  publishLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: theme.colors.ink.DEFAULT,
  },
  publishCaption: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },
  footer: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[4],
    backgroundColor: theme.colors.paper,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },
});
