import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Image,
  Linking,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import * as ImagePicker from 'expo-image-picker';
import * as Haptics from 'expo-haptics';
import { Camera, X } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { CalendarDatePicker } from './CalendarDatePicker';
import {
  EXPENSE_CATEGORIES,
  useExpenseMutations,
  useUploadExpenseReceipt,
  type ExpenseCategory,
} from '../hooks/useChefExpenses';
import { useKitchenMarket } from '../hooks/useKitchenMarket';
import { currencySymbol } from '../lib/format';

const MAX_RECEIPT_BYTES = 5 * 1024 * 1024;

type DateChoice = 'today' | 'yesterday' | 'custom';

function isoDaysAgo(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}

function fmtChipDate(iso: string): string {
  return new Date(`${iso}T12:00:00`).toLocaleDateString('en-IN', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
}

interface ExpenseQuickAddProps {
  orderId?: string;
  showDatePicker?: boolean;
  onSaved?: () => void;
}

export function ExpenseQuickAdd({ orderId, showDatePicker = true, onSaved }: ExpenseQuickAddProps) {
  const symbol = currencySymbol(useKitchenMarket()?.currency);
  const { create } = useExpenseMutations();
  const uploadReceipt = useUploadExpenseReceipt();

  const [category, setCategory] = useState<ExpenseCategory>('ingredients');
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [dateChoice, setDateChoice] = useState<DateChoice>('today');
  const [customDate, setCustomDate] = useState(isoDaysAgo(0));
  const [calendarOpen, setCalendarOpen] = useState(false);
  const [receiptUri, setReceiptUri] = useState<string | null>(null);

  const busy = create.isPending || uploadReceipt.isPending;

  function resolvedDate(): string {
    if (dateChoice === 'today') return isoDaysAgo(0);
    if (dateChoice === 'yesterday') return isoDaysAgo(1);
    return customDate;
  }

  async function pickReceipt(kind: 'camera' | 'library'): Promise<void> {
    const perm =
      kind === 'camera'
        ? await ImagePicker.requestCameraPermissionsAsync()
        : await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) {
      Alert.alert(
        kind === 'camera' ? 'Camera access needed' : 'Photo access needed',
        'Allow access in Settings to attach the bill.',
        [
          { text: 'Not now', style: 'cancel' },
          { text: 'Open Settings', onPress: () => void Linking.openSettings() },
        ],
      );
      return;
    }
    const opts: ImagePicker.ImagePickerOptions = { mediaTypes: ['images'], quality: 0.6 };
    const result =
      kind === 'camera'
        ? await ImagePicker.launchCameraAsync(opts)
        : await ImagePicker.launchImageLibraryAsync(opts);
    const asset = result.canceled ? null : result.assets[0];
    if (!asset) return;
    if (typeof asset.fileSize === 'number' && asset.fileSize > MAX_RECEIPT_BYTES) {
      Alert.alert('Photo too large', 'Please use a photo under 5 MB.');
      return;
    }
    setReceiptUri(asset.uri);
  }

  function onAttachReceipt(): void {
    Alert.alert('Attach bill / receipt', undefined, [
      { text: 'Take photo', onPress: () => void pickReceipt('camera') },
      { text: 'Choose from library', onPress: () => void pickReceipt('library') },
      { text: 'Cancel', style: 'cancel' },
    ]);
  }

  async function onSave(): Promise<void> {
    const value = Number(amount);
    if (!Number.isFinite(value) || value <= 0) {
      Alert.alert('Enter a valid amount', `The expense amount must be more than ${symbol}0.`);
      return;
    }
    const expenseDate = resolvedDate();
    if (!/^\d{4}-\d{2}-\d{2}$/.test(expenseDate)) {
      Alert.alert('Check the date', 'Use the YYYY-MM-DD format, e.g. 2026-07-15.');
      return;
    }
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    let receiptPath: string | undefined;
    if (receiptUri) {
      try {
        receiptPath = await uploadReceipt.mutateAsync(receiptUri);
      } catch {
        Alert.alert('Receipt upload failed', 'Saving the expense without the bill — you can re-add it later.');
      }
    }
    create.mutate(
      { category, amount: value, note: note.trim() || undefined, expenseDate, orderId, receiptPath },
      {
        onSuccess: () => {
          setAmount('');
          setNote('');
          setReceiptUri(null);
          setDateChoice('today');
          onSaved?.();
        },
        onError: (err) =>
          Alert.alert('Could not save', err instanceof Error ? err.message : 'Please try again.'),
      },
    );
  }

  return (
    <View style={styles.root}>
      <View style={styles.chipWrap}>
        {EXPENSE_CATEGORIES.map((c) => {
          const active = category === c.value;
          return (
            <Pressable
              key={c.value}
              onPress={() => setCategory(c.value)}
              accessibilityRole="radio"
              accessibilityState={{ selected: active }}
            >
              <View style={[styles.chip, active && styles.chipActive]}>
                <Text style={[styles.chipText, active && styles.chipTextActive]}>{c.label}</Text>
              </View>
            </Pressable>
          );
        })}
      </View>

      <TextInput
        style={styles.input}
        placeholder={`Amount (${symbol})`}
        placeholderTextColor={theme.colors.ink.muted}
        keyboardType="decimal-pad"
        value={amount}
        onChangeText={setAmount}
        accessibilityLabel="Expense amount in rupees"
      />

      {showDatePicker ? (
        <View style={styles.chipWrap}>
          {(['today', 'yesterday', 'custom'] as const).map((d) => {
            const active = dateChoice === d;
            const label =
              d === 'today'
                ? `Today · ${fmtChipDate(isoDaysAgo(0))}`
                : d === 'yesterday'
                  ? `Yesterday · ${fmtChipDate(isoDaysAgo(1))}`
                  : active
                    ? fmtChipDate(customDate)
                    : 'Other date';
            return (
              <Pressable
                key={d}
                onPress={() => {
                  setDateChoice(d);
                  setCalendarOpen(d === 'custom' ? dateChoice !== 'custom' || !calendarOpen : false);
                }}
                accessibilityRole="radio"
                accessibilityState={{ selected: active }}
              >
                <View style={[styles.chip, active && styles.chipActive]}>
                  <Text style={[styles.chipText, active && styles.chipTextActive]}>{label}</Text>
                </View>
              </Pressable>
            );
          })}
        </View>
      ) : null}
      {showDatePicker && dateChoice === 'custom' && calendarOpen ? (
        <CalendarDatePicker
          value={customDate}
          onChange={(iso) => {
            setCustomDate(iso);
            setCalendarOpen(false);
          }}
          maxDate={new Date()}
        />
      ) : null}

      <TextInput
        style={styles.input}
        placeholder="Note (optional)"
        placeholderTextColor={theme.colors.ink.muted}
        value={note}
        onChangeText={setNote}
        maxLength={500}
        accessibilityLabel="Expense note"
      />

      {receiptUri ? (
        <View style={styles.receiptRow}>
          <Image source={{ uri: receiptUri }} style={styles.receiptThumb} />
          <Text style={styles.receiptText} numberOfLines={1}>
            Bill attached
          </Text>
          <Pressable
            onPress={() => setReceiptUri(null)}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Remove attached bill"
          >
            <X size={16} color={theme.colors.ink.muted} />
          </Pressable>
        </View>
      ) : (
        <Pressable onPress={onAttachReceipt} accessibilityRole="button" accessibilityLabel="Attach bill or receipt photo">
          {({ pressed }) => (
            <View style={[styles.attachBtn, pressed && { opacity: 0.7 }]}>
              <Camera size={16} color={theme.colors.ink.soft} />
              <Text style={styles.attachText}>Attach bill / receipt (recommended)</Text>
            </View>
          )}
        </Pressable>
      )}

      <Pressable
        onPress={() => void onSave()}
        disabled={busy}
        accessibilityRole="button"
        accessibilityLabel="Save expense"
      >
        {({ pressed }) => (
          <View style={[styles.saveBtn, pressed && { opacity: 0.85 }, busy && styles.saveBtnDisabled]}>
            {busy ? <ActivityIndicator size="small" color={theme.colors.paper} /> : null}
            <Text style={styles.saveText}>{busy ? 'Saving…' : 'Save expense'}</Text>
          </View>
        )}
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: theme.spacing[3] },
  chipWrap: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: {
    borderRadius: theme.radius.full,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: 6,
    minHeight: 32,
    justifyContent: 'center',
  },
  chipActive: {
    backgroundColor: theme.colors.herb.DEFAULT,
    borderColor: theme.colors.herb.DEFAULT,
  },
  chipText: { fontFamily: 'Inter-Medium', fontSize: 13, color: theme.colors.ink.soft },
  chipTextActive: { color: theme.colors.paper },
  input: {
    minHeight: 44,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.paper,
    paddingHorizontal: theme.spacing[3],
    fontFamily: 'Inter',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
  },
  attachBtn: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderStyle: 'dashed',
    borderColor: theme.colors.mist.strong,
  },
  attachText: { fontFamily: 'Inter-Medium', fontSize: 14, color: theme.colors.ink.soft },
  receiptRow: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  receiptThumb: { width: 40, height: 40, borderRadius: theme.radius.sm, backgroundColor: theme.colors.bone },
  receiptText: { flex: 1, fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft },
  saveBtn: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.ink.DEFAULT,
    paddingHorizontal: theme.spacing[5],
  },
  saveBtnDisabled: { backgroundColor: theme.colors.mist.strong },
  saveText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.paper },
});
