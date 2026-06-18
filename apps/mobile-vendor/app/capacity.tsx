import { useEffect, useState } from 'react';
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
import { theme } from '@homechef/mobile-shared/theme';
import { Button } from '@homechef/mobile-shared/ui';
import {
  useCapacitySettings,
  useSetItemCapacity,
  useUpdateCapacitySettings,
} from '../hooks/useCapacity';
import { useVendorMenu } from '../hooks/useVendorMenu';

const HHMM = /^([01]\d|2[0-3]):[0-5]\d$/;

// Capacity & cutoff controls (#48): per-meal order cutoffs + auto-sold-out, and
// per-dish daily caps with today's remaining/sold counts.
export default function CapacityScreen() {
  const { data: settings, isLoading } = useCapacitySettings();
  const updateSettings = useUpdateCapacitySettings();
  const setItemCap = useSetItemCapacity();
  const { data: menu } = useVendorMenu();

  const [cutoffEnabled, setCutoffEnabled] = useState(false);
  const [lunch, setLunch] = useState('');
  const [dinner, setDinner] = useState('');
  const [autoSoldOut, setAutoSoldOut] = useState(true);
  const [hydrated, setHydrated] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [capDraft, setCapDraft] = useState('');

  useEffect(() => {
    if (hydrated || !settings) return;
    setCutoffEnabled(settings.cutoffEnabled);
    setLunch(settings.lunchCutoff ?? '');
    setDinner(settings.dinnerCutoff ?? '');
    setAutoSoldOut(settings.autoSoldOut);
    setHydrated(true);
  }, [settings, hydrated]);

  function saveSettings() {
    for (const [label, v] of [['Lunch', lunch], ['Dinner', dinner]] as const) {
      if (v !== '' && !HHMM.test(v)) {
        Alert.alert('Invalid time', `${label} cutoff must be HH:MM (24h), e.g. 10:00.`);
        return;
      }
    }
    updateSettings.mutate(
      { cutoffEnabled, lunchCutoff: lunch, dinnerCutoff: dinner, autoSoldOut },
      {
        onSuccess: () => Alert.alert('Saved', 'Your capacity settings are updated.'),
        onError: () => Alert.alert('Could not save', 'Please try again.'),
      },
    );
  }

  function startEdit(itemId: string, current?: number | null) {
    setEditingId(itemId);
    setCapDraft(current != null && current > 0 ? String(current) : '');
  }

  function saveCap(itemId: string) {
    const trimmed = capDraft.trim();
    const cap = trimmed === '' ? null : Math.max(0, parseInt(trimmed, 10) || 0);
    setItemCap.mutate(
      { itemId, dailyCapacity: cap },
      { onError: () => Alert.alert('Could not save', 'Please try again.') },
    );
    setEditingId(null);
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator color={theme.colors.herb.DEFAULT} />
      </SafeAreaView>
    );
  }

  const items = menu?.items ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={8} accessibilityLabel="Go back">
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Capacity & cutoffs</Text>
        <View style={{ width: 24 }} />
      </View>

      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        {/* Cutoffs */}
        <View style={styles.card}>
          <View style={styles.rowBetween}>
            <View style={{ flex: 1 }}>
              <Text style={styles.cardTitle}>Order cutoffs</Text>
              <Text style={styles.caption}>
                Auto-close ordering once the day's cutoffs pass.
              </Text>
            </View>
            <Switch
              value={cutoffEnabled}
              onValueChange={setCutoffEnabled}
              trackColor={{ true: theme.colors.herb.DEFAULT }}
            />
          </View>

          {cutoffEnabled ? (
            <>
              <CutoffRow label="Lunch cutoff" value={lunch} onChange={setLunch} />
              <CutoffRow label="Dinner cutoff" value={dinner} onChange={setDinner} />
            </>
          ) : null}
        </View>

        {/* Auto sold-out */}
        <View style={styles.card}>
          <View style={styles.rowBetween}>
            <View style={{ flex: 1 }}>
              <Text style={styles.cardTitle}>Auto sold-out</Text>
              <Text style={styles.caption}>
                Hide a dish automatically when its daily cap is reached.
              </Text>
            </View>
            <Switch
              value={autoSoldOut}
              onValueChange={setAutoSoldOut}
              trackColor={{ true: theme.colors.herb.DEFAULT }}
            />
          </View>
        </View>

        <Button
          label="Save settings"
          variant="primary"
          loading={updateSettings.isPending}
          onPress={saveSettings}
        />

        {/* Per-dish daily caps */}
        <Text style={styles.sectionLabel}>Daily caps per dish</Text>
        <View style={styles.card}>
          {items.length === 0 ? (
            <Text style={styles.caption}>Add menu items first to set daily caps.</Text>
          ) : (
            items.map((it, i) => {
              const capped = it.dailyCapacity != null && it.dailyCapacity > 0;
              const editing = editingId === it.id;
              return (
                <View key={it.id} style={[styles.itemRow, i < items.length - 1 && styles.itemDivider]}>
                  <View style={{ flex: 1 }}>
                    <Text style={styles.itemName} numberOfLines={1}>{it.name}</Text>
                    <Text style={styles.itemSub}>
                      {capped
                        ? it.soldOut
                          ? `Sold out (cap ${it.dailyCapacity})`
                          : `${it.remainingToday ?? it.dailyCapacity} left today · cap ${it.dailyCapacity}`
                        : 'Unlimited'}
                    </Text>
                  </View>
                  {editing ? (
                    <View style={styles.editRow}>
                      <TextInput
                        style={styles.capInput}
                        placeholder="∞"
                        placeholderTextColor={theme.colors.ink.muted}
                        value={capDraft}
                        onChangeText={(t) => setCapDraft(t.replace(/[^0-9]/g, '').slice(0, 4))}
                        keyboardType="number-pad"
                        autoFocus
                      />
                      <Pressable onPress={() => saveCap(it.id)} hitSlop={8}>
                        <Text style={styles.edit}>Save</Text>
                      </Pressable>
                    </View>
                  ) : (
                    <Pressable onPress={() => startEdit(it.id, it.dailyCapacity)} hitSlop={8}>
                      <Text style={styles.edit}>Edit</Text>
                    </Pressable>
                  )}
                </View>
              );
            })
          )}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

function CutoffRow({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (s: string) => void;
}) {
  return (
    <View style={styles.cutoffRow}>
      <Text style={styles.cutoffLabel}>{label}</Text>
      <TextInput
        style={styles.timeInput}
        placeholder="HH:MM"
        placeholderTextColor={theme.colors.ink.muted}
        value={value}
        onChangeText={(t) => onChange(t.replace(/[^0-9:]/g, '').slice(0, 5))}
        keyboardType="numbers-and-punctuation"
        maxLength={5}
      />
    </View>
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
  scroll: { padding: theme.spacing[4], gap: theme.spacing[4], paddingBottom: theme.spacing[10] },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  rowBetween: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[3] },
  cardTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT },
  caption: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.muted, marginTop: 2 },
  cutoffRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: theme.spacing[3],
  },
  cutoffLabel: { fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.DEFAULT },
  timeInput: {
    width: 90,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
    paddingVertical: theme.spacing[2],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
  },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    letterSpacing: 0.3,
    textTransform: 'uppercase',
    color: theme.colors.ink.soft,
  },
  itemRow: { flexDirection: 'row', alignItems: 'center', paddingVertical: theme.spacing[3] },
  itemDivider: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  itemName: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  itemSub: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.muted, marginTop: 2 },
  edit: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.herb.DEFAULT },
  editRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[3] },
  capInput: {
    width: 56,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
    paddingVertical: theme.spacing[1],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
  },
});
