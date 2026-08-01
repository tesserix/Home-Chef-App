// Expenses & annual tax statement — the chef's own books. Gas refills,
// ingredient runs, utensils and other kitchen costs recorded here roll into
// the FY (Apr–Mar) statement, which nets them against platform earnings and
// downloads as a PDF for GST / income-tax filing.

import { useState } from 'react';
import {
  Alert,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft, FileDown, Receipt, Trash2 } from 'lucide-react-native';
import * as Haptics from 'expo-haptics';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { downloadAndSharePdf } from '../lib/download-pdf';
import {
  EXPENSE_CATEGORIES,
  currentFyStartYear,
  expenseCategoryLabel,
  useChefExpenses,
  useExpenseMutations,
  useFYStatement,
  type ExpenseCategory,
} from '../hooks/useChefExpenses';

const INK_RIPPLE = `${theme.colors.ink.DEFAULT}14`;

function fmtInr(value: number): string {
  return `₹${value.toLocaleString('en-IN', { minimumFractionDigits: 0, maximumFractionDigits: 2 })}`;
}

function todayISO(): string {
  const d = new Date();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}

function fmtShortDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
}

export default function ExpensesScreen() {
  const fy = currentFyStartYear();
  const { data, isLoading, refetch, isRefetching } = useChefExpenses();
  const { data: stmt } = useFYStatement(fy);
  const { create, remove } = useExpenseMutations();

  const [category, setCategory] = useState<ExpenseCategory>('ingredients');
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [date, setDate] = useState(todayISO());
  const [downloading, setDownloading] = useState(false);

  const expenses = data?.expenses ?? [];
  const fyEndShort = String((fy + 1) % 100).padStart(2, '0');

  function onAdd(): void {
    const value = Number(amount);
    if (!Number.isFinite(value) || value <= 0) {
      Alert.alert('Enter a valid amount', 'The expense amount must be more than ₹0.');
      return;
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) {
      Alert.alert('Check the date', 'Use the YYYY-MM-DD format, e.g. 2026-07-15.');
      return;
    }
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    create.mutate(
      { category, amount: value, note: note.trim() || undefined, expenseDate: date },
      {
        onSuccess: () => {
          setAmount('');
          setNote('');
          setDate(todayISO());
        },
        onError: (err) =>
          Alert.alert(
            'Could not save',
            err instanceof Error ? err.message : 'Please try again.',
          ),
      },
    );
  }

  function onDelete(id: string, label: string): void {
    Alert.alert('Delete expense?', `${label} will be removed from your books.`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: () =>
          remove.mutate(id, {
            onError: () => Alert.alert('Could not delete', 'Please try again.'),
          }),
      },
    ]);
  }

  async function onDownloadStatement(): Promise<void> {
    setDownloading(true);
    try {
      await downloadAndSharePdf(
        `/chef/tax/fy-statement.pdf?year=${fy}`,
        `fy-statement-FY${fy}-${fyEndShort}.pdf`,
      );
    } finally {
      setDownloading(false);
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: INK_RIPPLE, borderless: true }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : null}>
              <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.headerTitle}>Expenses & Tax</Text>
        <View style={{ width: 22 }} />
      </View>

      <ScrollView
        contentContainerStyle={styles.scroll}
        refreshControl={<RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} />}
        keyboardShouldPersistTaps="handled"
      >
        {/* FY statement summary + download */}
        <View style={styles.card}>
          <View style={styles.cardHeader}>
            <FileDown size={18} color={theme.colors.ink.DEFAULT} />
            <Text style={styles.cardTitle}>
              Annual statement · FY {fy}-{fyEndShort}
            </Text>
          </View>
          {stmt ? (
            <>
              <View style={styles.statRow}>
                <View style={styles.stat}>
                  <Text style={styles.statLabel}>Net earnings</Text>
                  <Text style={styles.statValue}>{fmtInr(stmt.netEarnings)}</Text>
                </View>
                <View style={styles.stat}>
                  <Text style={styles.statLabel}>Expenses</Text>
                  <Text style={styles.statValue}>{fmtInr(stmt.totalExpenses)}</Text>
                </View>
                <View style={styles.stat}>
                  <Text style={styles.statLabel}>Net income</Text>
                  <Text style={[styles.statValue, { color: theme.colors.herb.DEFAULT }]}>
                    {fmtInr(stmt.netIncome)}
                  </Text>
                </View>
              </View>
              <Text style={styles.hint}>
                Income from {stmt.ordersCount} delivered orders minus your recorded expenses.
                Download the PDF for GST / income-tax filing.
              </Text>
            </>
          ) : (
            <Skeleton style={{ height: 56, borderRadius: theme.radius.md }} />
          )}
          <Pressable
            onPress={() => void onDownloadStatement()}
            disabled={downloading}
            accessibilityRole="button"
            accessibilityLabel={`Download FY ${fy}-${fyEndShort} statement PDF`}
            android_ripple={{ color: `${theme.colors.paper}33` }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.primaryBtn,
                  pressed && styles.primaryBtnPressed,
                  downloading && styles.primaryBtnDisabled,
                ]}
              >
                <FileDown size={16} color={theme.colors.paper} />
                <Text style={styles.primaryBtnText}>
                  {downloading ? 'Preparing…' : 'Download FY statement (PDF)'}
                </Text>
              </View>
            )}
          </Pressable>
        </View>

        {/* Add expense */}
        <View style={styles.card}>
          <View style={styles.cardHeader}>
            <Receipt size={18} color={theme.colors.ink.DEFAULT} />
            <Text style={styles.cardTitle}>Add expense</Text>
          </View>
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
                    <Text style={[styles.chipText, active && styles.chipTextActive]}>
                      {c.label}
                    </Text>
                  </View>
                </Pressable>
              );
            })}
          </View>
          <TextInput
            style={styles.input}
            placeholder="Amount (₹)"
            placeholderTextColor={theme.colors.ink.muted}
            keyboardType="decimal-pad"
            value={amount}
            onChangeText={setAmount}
            accessibilityLabel="Expense amount in rupees"
          />
          <TextInput
            style={styles.input}
            placeholder="Date (YYYY-MM-DD)"
            placeholderTextColor={theme.colors.ink.muted}
            autoCapitalize="none"
            value={date}
            onChangeText={setDate}
            accessibilityLabel="Expense date"
          />
          <TextInput
            style={styles.input}
            placeholder="Note (optional)"
            placeholderTextColor={theme.colors.ink.muted}
            value={note}
            onChangeText={setNote}
            maxLength={500}
            accessibilityLabel="Expense note"
          />
          <Pressable
            onPress={onAdd}
            disabled={create.isPending}
            accessibilityRole="button"
            accessibilityLabel="Save expense"
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.primaryBtn,
                  pressed && styles.primaryBtnPressed,
                  create.isPending && styles.primaryBtnDisabled,
                ]}
              >
                <Text style={styles.primaryBtnText}>
                  {create.isPending ? 'Saving…' : 'Save expense'}
                </Text>
              </View>
            )}
          </Pressable>
        </View>

        {/* Expense list */}
        <View style={styles.card}>
          <View style={styles.cardHeader}>
            <Receipt size={18} color={theme.colors.ink.DEFAULT} />
            <Text style={styles.cardTitle}>Recorded expenses</Text>
          </View>
          {isLoading ? (
            <Skeleton style={{ height: 80, borderRadius: theme.radius.md }} />
          ) : expenses.length === 0 ? (
            <Text style={styles.body}>
              Nothing recorded yet. Gas, ingredients, utensils and other kitchen costs you add
              roll into your annual statement automatically.
            </Text>
          ) : (
            expenses.map((e, i) => (
              <View
                key={e.id}
                style={[styles.expenseRow, i === expenses.length - 1 && styles.expenseRowLast]}
              >
                <View style={{ flex: 1, minWidth: 0 }}>
                  <Text style={styles.expenseCat} numberOfLines={1}>
                    {expenseCategoryLabel(e.category)}
                  </Text>
                  <Text style={styles.expenseSub} numberOfLines={1}>
                    {fmtShortDate(e.expenseDate)}
                    {e.note ? ` · ${e.note}` : ''}
                  </Text>
                </View>
                <Text style={styles.expenseAmount}>{fmtInr(e.amount)}</Text>
                <Pressable
                  onPress={() => onDelete(e.id, `${expenseCategoryLabel(e.category)} ${fmtInr(e.amount)}`)}
                  hitSlop={8}
                  accessibilityRole="button"
                  accessibilityLabel={`Delete ${expenseCategoryLabel(e.category)} expense of ${fmtInr(e.amount)}`}
                >
                  {({ pressed }) => (
                    <View style={pressed ? { opacity: 0.6 } : null}>
                      <Trash2 size={16} color={theme.colors.destructive.DEFAULT} />
                    </View>
                  )}
                </Pressable>
              </View>
            ))
          )}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  headerTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: theme.colors.ink.DEFAULT,
  },
  scroll: { padding: theme.spacing[4], paddingBottom: 40, gap: theme.spacing[4] },

  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    gap: theme.spacing[3],
    ...theme.shadow[1],
  },
  cardHeader: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  cardTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT },
  body: { fontFamily: 'Inter', fontSize: 14, lineHeight: 20, color: theme.colors.ink.soft },
  hint: { fontFamily: 'Inter', fontSize: 12, lineHeight: 17, color: theme.colors.ink.muted },

  statRow: { flexDirection: 'row', gap: theme.spacing[3] },
  stat: { flex: 1 },
  statLabel: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.muted },
  statValue: {
    fontFamily: 'Geist-Bold',
    fontSize: 17,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    marginTop: 2,
  },

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

  primaryBtn: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.ink.DEFAULT,
    paddingHorizontal: theme.spacing[5],
  },
  primaryBtnPressed: { opacity: 0.85 },
  primaryBtnDisabled: { backgroundColor: theme.colors.mist.strong },
  primaryBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.paper },

  expenseRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  expenseRowLast: { borderBottomWidth: 0 },
  expenseCat: { fontFamily: 'Inter-Medium', fontSize: 15, color: theme.colors.ink.DEFAULT },
  expenseSub: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.muted, marginTop: 2 },
  expenseAmount: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
});
