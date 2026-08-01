import { useState } from 'react';
import { Alert, Pressable, StyleSheet, Text, View } from 'react-native';
import { ChevronDown, ChevronUp, Plus, Receipt, Trash2 } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { ExpenseQuickAdd } from './ExpenseQuickAdd';
import {
  expenseCategoryLabel,
  useChefExpenses,
  useExpenseMutations,
} from '../hooks/useChefExpenses';

function fmtInr(value: number): string {
  return `₹${value.toLocaleString('en-IN', { minimumFractionDigits: 0, maximumFractionDigits: 2 })}`;
}

export function OrderExpensesCard({ orderId }: { orderId: string }) {
  const { data } = useChefExpenses({ orderId });
  const { remove } = useExpenseMutations();
  const [adding, setAdding] = useState(false);

  const expenses = data?.expenses ?? [];
  const total = expenses.reduce((sum, e) => sum + e.amount, 0);

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

  return (
    <View style={styles.card}>
      <View style={styles.headerRow}>
        <View style={styles.headerLeft}>
          <Receipt size={16} color={theme.colors.ink.soft} />
          <Text style={styles.title}>Order expenses</Text>
          {expenses.length > 0 ? (
            <Text style={styles.totalText}>
              {expenses.length} · {fmtInr(total)}
            </Text>
          ) : null}
        </View>
        <Pressable
          onPress={() => setAdding((v) => !v)}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel={adding ? 'Close expense form' : 'Add expense for this order'}
        >
          {({ pressed }) => (
            <View style={[styles.addBtn, pressed && { opacity: 0.7 }]}>
              {adding ? (
                <ChevronUp size={16} color={theme.colors.ink.DEFAULT} />
              ) : (
                <>
                  <Plus size={14} color={theme.colors.ink.DEFAULT} />
                  <Text style={styles.addBtnText}>Add expense</Text>
                  <ChevronDown size={14} color={theme.colors.ink.muted} />
                </>
              )}
            </View>
          )}
        </Pressable>
      </View>

      {expenses.length === 0 && !adding ? (
        <Text style={styles.emptyText}>
          Bought ingredients or a gas refill for this order? Log it now — it counts against your
          tax-time statement.
        </Text>
      ) : null}

      {expenses.map((e, i) => (
        <View key={e.id} style={[styles.row, i === 0 && { borderTopWidth: 0 }]}>
          <View style={{ flex: 1, minWidth: 0 }}>
            <Text style={styles.rowLabel} numberOfLines={1}>
              {expenseCategoryLabel(e.category)}
              {e.note ? ` · ${e.note}` : ''}
            </Text>
          </View>
          <Text style={styles.rowAmount}>{fmtInr(e.amount)}</Text>
          <Pressable
            onPress={() => onDelete(e.id, `${expenseCategoryLabel(e.category)} ${fmtInr(e.amount)}`)}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel={`Delete ${expenseCategoryLabel(e.category)} expense`}
          >
            {({ pressed }) => (
              <View style={pressed ? { opacity: 0.6 } : null}>
                <Trash2 size={15} color={theme.colors.destructive.DEFAULT} />
              </View>
            )}
          </Pressable>
        </View>
      ))}

      {adding ? (
        <View style={styles.formWrap}>
          <ExpenseQuickAdd orderId={orderId} showDatePicker={false} onSaved={() => setAdding(false)} />
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    gap: theme.spacing[2],
    ...theme.shadow[1],
  },
  headerRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: 8 },
  headerLeft: { flexDirection: 'row', alignItems: 'center', gap: 8, flexShrink: 1 },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  totalText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.muted,
    fontVariant: ['tabular-nums'],
  },
  addBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    borderRadius: theme.radius.full,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: 6,
    minHeight: 32,
  },
  addBtnText: { fontFamily: 'Inter-Medium', fontSize: 13, color: theme.colors.ink.DEFAULT },
  emptyText: { fontFamily: 'Inter', fontSize: 13, lineHeight: 18, color: theme.colors.ink.muted },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    paddingVertical: theme.spacing[2],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },
  rowLabel: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.soft },
  rowAmount: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  formWrap: { paddingTop: theme.spacing[2] },
});
