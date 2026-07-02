import { useEffect, useMemo, useState } from 'react';
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
import { ChevronLeft, Plus, Trash2 } from 'lucide-react-native';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { theme } from '@homechef/mobile-shared/theme';
import { Button } from '@homechef/mobile-shared/ui';
import {
  useMyDailyMenu,
  useSaveDailyMenu,
  type DailyMenuItemInput,
  type MealSlot,
  type MealVariant,
} from '../../hooks/useMealPlans';

// #405/#406 — per-DATE menu builder. Unlike the fixed weekly grid, each calendar
// date can hold MULTIPLE dishes per slot, and any dish can be a combo/thali
// (bundle) with a set price. Reuses the weekly-menu editor's look + save pattern.

const SLOTS: { slot: MealSlot; label: string }[] = [
  { slot: 'lunch', label: 'Lunch' },
  { slot: 'dinner', label: 'Dinner' },
];
const VARIANTS: { variant: MealVariant; label: string }[] = [
  { variant: 'veg', label: 'Veg' },
  { variant: 'nonveg', label: 'Non-veg' },
];

function ymd(d: Date): string {
  return d.toISOString().slice(0, 10);
}
function labelFor(iso: string): { dow: string; day: string } {
  const d = new Date(iso + 'T00:00:00');
  return {
    dow: d.toLocaleDateString('en-IN', { weekday: 'short' }),
    day: d.toLocaleDateString('en-IN', { day: 'numeric', month: 'short' }),
  };
}

function blankRow(sortOrder: number): DailyMenuItemInput {
  return {
    slot: 'lunch',
    variant: 'veg',
    name: '',
    price: 0,
    isCombo: false,
    comboComponents: [],
    sortOrder,
  };
}

export default function DailyMenuScreen() {
  // Next 14 bookable days (tomorrow onward, matching the customer horizon).
  const dates = useMemo(() => {
    const out: string[] = [];
    const base = new Date();
    for (let i = 1; i <= 14; i++) {
      const d = new Date(base);
      d.setDate(base.getDate() + i);
      out.push(ymd(d));
    }
    return out;
  }, []);
  const [selected, setSelected] = useState(dates[0]!);
  const { data, isLoading } = useMyDailyMenu(dates[0]!, dates[dates.length - 1]!);
  const save = useSaveDailyMenu();

  const [rows, setRows] = useState<DailyMenuItemInput[]>([]);
  const [isPublished, setIsPublished] = useState(false);

  // Seed the editor from the selected date's saved menu.
  useEffect(() => {
    const day = data?.days.find((d) => d.date === selected);
    setRows(
      (day?.items ?? []).map((it, i) => ({
        slot: it.slot,
        variant: it.variant,
        name: it.name,
        description: it.description,
        price: it.price,
        isCombo: it.isCombo,
        comboComponents: it.comboComponents ?? [],
        sortOrder: i,
      })),
    );
    setIsPublished(day?.isPublished ?? false);
  }, [selected, data]);

  function patchRow(i: number, patch: Partial<DailyMenuItemInput>) {
    setRows((prev) => prev.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  }

  function validate(): string | null {
    for (const r of rows) {
      if (!r.name.trim()) return 'Every dish needs a name.';
      if (r.price < 0) return 'Price cannot be negative.';
      if (r.isCombo && (r.comboComponents.filter(Boolean).length < 2 || r.price <= 0))
        return 'A combo needs at least two items and a set price.';
    }
    return null;
  }

  async function onSave(publish: boolean) {
    const err = validate();
    if (err) {
      Alert.alert('Check the menu', err);
      return;
    }
    if (publish && rows.length === 0) {
      Alert.alert('Add a dish', 'Add at least one dish before publishing.');
      return;
    }
    try {
      await save.mutateAsync({ date: selected, isPublished: publish, items: rows });
      setIsPublished(publish);
      Alert.alert(publish ? 'Published' : 'Saved', `${labelFor(selected).day} menu updated.`);
    } catch (e) {
      Alert.alert('Could not save', getServerErrorMessage(e, 'Please try again.'));
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={8} accessibilityLabel="Go back">
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Daily menu</Text>
        <View style={{ width: 24 }} />
      </View>

      {/* Date strip */}
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.dateStrip}>
        {dates.map((iso) => {
          const active = iso === selected;
          const l = labelFor(iso);
          return (
            <Pressable key={iso} onPress={() => setSelected(iso)} style={[styles.dateChip, active && styles.dateChipActive]}>
              <Text style={[styles.dateDow, active && styles.dateTextActive]}>{l.dow}</Text>
              <Text style={[styles.dateDay, active && styles.dateTextActive]}>{l.day}</Text>
            </Pressable>
          );
        })}
      </ScrollView>

      {isLoading ? (
        <ActivityIndicator style={{ marginTop: theme.spacing[6] }} color={theme.colors.ink.DEFAULT} />
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {rows.length === 0 ? (
            <Text style={styles.empty}>No dishes for this day yet. Add what you're cooking.</Text>
          ) : null}

          {rows.map((row, i) => (
            <View key={i} style={styles.card}>
              <View style={styles.cardTop}>
                <TextInput
                  value={row.name}
                  onChangeText={(t) => patchRow(i, { name: t })}
                  placeholder="Dish name"
                  placeholderTextColor={theme.colors.ink.muted}
                  style={styles.nameInput}
                />
                <Pressable onPress={() => setRows((p) => p.filter((_, idx) => idx !== i))} hitSlop={8} accessibilityLabel="Remove dish">
                  <Trash2 size={18} color={theme.colors.destructive.DEFAULT} />
                </Pressable>
              </View>

              <View style={styles.pillRow}>
                {SLOTS.map((s) => (
                  <Pressable key={s.slot} onPress={() => patchRow(i, { slot: s.slot })} style={[styles.pill, row.slot === s.slot && styles.pillActive]}>
                    <Text style={[styles.pillText, row.slot === s.slot && styles.pillTextActive]}>{s.label}</Text>
                  </Pressable>
                ))}
                {VARIANTS.map((v) => (
                  <Pressable key={v.variant} onPress={() => patchRow(i, { variant: v.variant })} style={[styles.pill, row.variant === v.variant && styles.pillActive]}>
                    <Text style={[styles.pillText, row.variant === v.variant && styles.pillTextActive]}>{v.label}</Text>
                  </Pressable>
                ))}
              </View>

              <View style={styles.priceRow}>
                <Text style={styles.priceLabel}>₹</Text>
                <TextInput
                  value={row.price ? String(row.price) : ''}
                  onChangeText={(t) => patchRow(i, { price: Number(t.replace(/[^0-9.]/g, '')) || 0 })}
                  placeholder="Price"
                  placeholderTextColor={theme.colors.ink.muted}
                  keyboardType="numeric"
                  style={styles.priceInput}
                />
                <Text style={styles.comboLabel}>Combo / Thali</Text>
                <Switch value={row.isCombo} onValueChange={(v) => patchRow(i, { isCombo: v })} />
              </View>

              {row.isCombo ? (
                <TextInput
                  value={row.comboComponents.join(', ')}
                  onChangeText={(t) => patchRow(i, { comboComponents: t.split(',').map((s) => s.trim()) })}
                  placeholder="Items in the combo (e.g. Rice, Dal, Sabji, Papad)"
                  placeholderTextColor={theme.colors.ink.muted}
                  style={styles.comboInput}
                />
              ) : null}
            </View>
          ))}

          <Pressable onPress={() => setRows((p) => [...p, blankRow(p.length)])} style={styles.addBtn} accessibilityLabel="Add a dish">
            <Plus size={18} color={theme.colors.herb.DEFAULT} />
            <Text style={styles.addBtnText}>Add dish</Text>
          </Pressable>

          <View style={styles.publishRow}>
            <Text style={styles.publishLabel}>{isPublished ? 'Published — customers can book this day' : 'Draft — not visible to customers'}</Text>
          </View>
        </ScrollView>
      )}

      <View style={styles.footer}>
        <View style={{ flex: 1 }}>
          <Button label="Save draft" variant="secondary" onPress={() => onSave(false)} loading={save.isPending} />
        </View>
        <View style={{ flex: 1 }}>
          <Button label="Publish" onPress={() => onSave(true)} loading={save.isPending} />
        </View>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  dateStrip: { paddingHorizontal: theme.spacing[4], gap: theme.spacing[2], paddingBottom: theme.spacing[2] },
  dateChip: {
    alignItems: 'center',
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.md,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    minWidth: 56,
  },
  dateChipActive: { backgroundColor: theme.colors.ink.DEFAULT, borderColor: theme.colors.ink.DEFAULT },
  dateDow: { fontFamily: 'Inter', fontSize: theme.typography.size.caption.size, color: theme.colors.ink.muted },
  dateDay: { fontFamily: 'Inter-SemiBold', fontSize: theme.typography.size.label.size, color: theme.colors.ink.DEFAULT },
  dateTextActive: { color: theme.colors.paper },
  scroll: { paddingHorizontal: theme.spacing[4], paddingBottom: theme.spacing[6], gap: theme.spacing[3] },
  empty: { fontFamily: 'Inter', color: theme.colors.ink.soft, marginTop: theme.spacing[4], textAlign: 'center' },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[3],
    gap: theme.spacing[2],
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  cardTop: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  nameInput: { flex: 1, fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT, paddingVertical: theme.spacing[1] },
  pillRow: { flexDirection: 'row', flexWrap: 'wrap', gap: theme.spacing[2] },
  pill: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[1],
    borderRadius: theme.radius.full,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  pillActive: { backgroundColor: theme.colors.herb.tint, borderColor: theme.colors.herb.DEFAULT },
  pillText: { fontFamily: 'Inter', fontSize: theme.typography.size.bodySm.size, color: theme.colors.ink.muted },
  pillTextActive: { color: theme.colors.herb.DEFAULT, fontFamily: 'Inter-SemiBold' },
  priceRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  priceLabel: { fontFamily: 'Inter-SemiBold', color: theme.colors.ink.DEFAULT },
  priceInput: { width: 80, fontFamily: 'Inter', color: theme.colors.ink.DEFAULT, fontVariant: ['tabular-nums'] },
  comboLabel: { marginLeft: 'auto', fontFamily: 'Inter', fontSize: theme.typography.size.bodySm.size, color: theme.colors.ink.soft },
  comboInput: {
    fontFamily: 'Inter',
    color: theme.colors.ink.DEFAULT,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    paddingTop: theme.spacing[2],
  },
  addBtn: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2], paddingVertical: theme.spacing[2] },
  addBtnText: { fontFamily: 'Inter-SemiBold', color: theme.colors.herb.DEFAULT },
  publishRow: { marginTop: theme.spacing[2] },
  publishLabel: { fontFamily: 'Inter', fontSize: theme.typography.size.bodySm.size, color: theme.colors.ink.soft },
  footer: { flexDirection: 'row', gap: theme.spacing[3], padding: theme.spacing[4], borderTopWidth: StyleSheet.hairlineWidth, borderTopColor: theme.colors.mist.DEFAULT },
});
