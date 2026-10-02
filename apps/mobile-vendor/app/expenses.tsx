import { useState } from 'react';
import {
  Alert,
  Linking,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import {
  ChevronDown,
  ChevronLeft,
  ChevronUp,
  FileDown,
  Paperclip,
  Receipt,
  Trash2,
} from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { downloadAndSharePdf } from '../lib/download-pdf';
import { ExpenseQuickAdd } from '../components/ExpenseQuickAdd';
import {
  currentFyStartYear,
  expenseCategoryLabel,
  useChefExpenses,
  useExpenseMutations,
  useFYStatement,
} from '../hooks/useChefExpenses';
import { useKitchenMoney } from '../hooks/useKitchenMarket';

const INK_RIPPLE = `${theme.colors.ink.DEFAULT}14`;

function fmtShortDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
}

export default function ExpensesScreen() {
  const money = useKitchenMoney();
  const fy = currentFyStartYear();
  const { data, isLoading, refetch, isRefetching } = useChefExpenses();
  const { data: stmt } = useFYStatement(fy);
  const { remove } = useExpenseMutations();
  const [downloading, setDownloading] = useState(false);
  const [showBreakdown, setShowBreakdown] = useState(false);

  const expenses = data?.expenses ?? [];
  const fyEndShort = String((fy + 1) % 100).padStart(2, '0');

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
                  <Text style={styles.statValue}>{money(stmt.netEarnings)}</Text>
                </View>
                <View style={styles.stat}>
                  <Text style={styles.statLabel}>Expenses</Text>
                  <Text style={styles.statValue}>{money(stmt.totalExpenses)}</Text>
                </View>
                <View style={styles.stat}>
                  <Text style={styles.statLabel}>Net income</Text>
                  <Text style={[styles.statValue, { color: theme.colors.herb.DEFAULT }]}>
                    {money(stmt.netIncome)}
                  </Text>
                </View>
              </View>
              <Text style={styles.hint}>
                Income from {stmt.ordersCount} delivered orders minus your recorded expenses.
                Download the PDF for GST / income-tax filing.
              </Text>

              {/* Itemised breakdown, matching the vendor portal's annual
                  statement. The three headline figures above answer "how much",
                  but a chef reconciling with their accountant needs to see which
                  line the commission and TDS came off — otherwise the only way
                  to check the maths is to open the PDF. */}
              <Pressable
                onPress={() => setShowBreakdown((v) => !v)}
                accessibilityRole="button"
                accessibilityState={{ expanded: showBreakdown }}
                accessibilityLabel={
                  showBreakdown ? 'Hide statement breakdown' : 'Show statement breakdown'
                }
              >
                {({ pressed }) => (
                  <View style={[styles.disclosureRow, pressed && { opacity: 0.7 }]}>
                    <Text style={styles.disclosureText}>
                      {showBreakdown ? 'Hide breakdown' : 'Show full breakdown'}
                    </Text>
                    {showBreakdown ? (
                      <ChevronUp size={16} color={theme.colors.ink.muted} />
                    ) : (
                      <ChevronDown size={16} color={theme.colors.ink.muted} />
                    )}
                  </View>
                )}
              </Pressable>

              {showBreakdown ? (
                <View style={styles.breakdown}>
                  <Text style={styles.breakdownLabel}>
                    INCOME · {stmt.ordersCount} DELIVERED ORDERS
                  </Text>
                  <Line label="Food revenue" amount={stmt.foodRevenue} />
                  <Line label="GST collected from customers" amount={stmt.gstCollected} />
                  <Line label="Customer tips" amount={stmt.tips} />
                  <Line label="Gross receipts" amount={stmt.grossReceipts} strong />
                  <Line label="Platform commission" amount={stmt.platformCommission} negative />
                  <Line
                    label="GST on commission (ITC eligible)"
                    amount={
                      stmt.commissionCgst + stmt.commissionSgst + stmt.commissionIgst
                    }
                  />
                  <Line label="TDS withheld (194-O)" amount={stmt.tdsWithheld} negative />
                  <Line label="Net earnings from platform" amount={stmt.netEarnings} strong />

                  <Text style={[styles.breakdownLabel, { marginTop: theme.spacing[3] }]}>
                    EXPENSES · SELF-DECLARED
                  </Text>
                  {stmt.expenses.byCategory.length === 0 ? (
                    <Text style={styles.body}>
                      No expenses recorded in this financial year yet.
                    </Text>
                  ) : (
                    stmt.expenses.byCategory.map((c) => (
                      <Line
                        key={c.category}
                        label={`${expenseCategoryLabel(c.category)} (${c.count})`}
                        amount={c.amount}
                        negative
                      />
                    ))
                  )}
                  <Line label="Total expenses" amount={stmt.totalExpenses} strong />
                  <Line label="Net income (pre-tax)" amount={stmt.netIncome} strong highlight />
                </View>
              ) : null}
            </>
          ) : (
            <Skeleton style={{ height: 56, borderRadius: theme.radius.md }} />
          )}
          <Pressable
            onPress={() => void onDownloadStatement()}
            disabled={downloading}
            accessibilityRole="button"
            accessibilityLabel={`Download FY ${fy}-${fyEndShort} statement PDF`}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.primaryBtn,
                  pressed && { opacity: 0.85 },
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

        <View style={styles.card}>
          <View style={styles.cardHeader}>
            <Receipt size={18} color={theme.colors.ink.DEFAULT} />
            <Text style={styles.cardTitle}>Add expense</Text>
          </View>
          <ExpenseQuickAdd />
        </View>

        <View style={styles.card}>
          <View style={styles.cardHeader}>
            <Receipt size={18} color={theme.colors.ink.DEFAULT} />
            <Text style={styles.cardTitle}>Recorded expenses</Text>
          </View>
          {isLoading ? (
            <Skeleton style={{ height: 80, borderRadius: theme.radius.md }} />
          ) : expenses.length === 0 ? (
            <Text style={styles.body}>
              Nothing recorded yet. You can also log expenses straight from an order while you cook
              — they land here automatically.
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
                    {e.orderNumber ? ` · #${e.orderNumber}` : ''}
                  </Text>
                  <Text style={styles.expenseSub} numberOfLines={1}>
                    {fmtShortDate(e.expenseDate)}
                    {e.note ? ` · ${e.note}` : ''}
                  </Text>
                </View>
                {e.receiptUrl ? (
                  <Pressable
                    onPress={() => void Linking.openURL(e.receiptUrl!)}
                    hitSlop={8}
                    accessibilityRole="button"
                    accessibilityLabel="View attached bill"
                  >
                    {({ pressed }) => (
                      <View style={pressed ? { opacity: 0.6 } : null}>
                        <Paperclip size={15} color={theme.colors.ink.soft} />
                      </View>
                    )}
                  </Pressable>
                ) : null}
                <Text style={styles.expenseAmount}>{money(e.amount)}</Text>
                <Pressable
                  onPress={() =>
                    onDelete(e.id, `${expenseCategoryLabel(e.category)} ${money(e.amount)}`)
                  }
                  hitSlop={8}
                  accessibilityRole="button"
                  accessibilityLabel={`Delete ${expenseCategoryLabel(e.category)} expense of ${money(e.amount)}`}
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

/**
 * One labelled money line on the annual statement.
 *
 * `negative` prefixes a minus rather than colouring the figure red: these are
 * ordinary deductions on a statement, not errors, and red would read as alarm.
 */
function Line({
  label,
  amount,
  strong,
  negative,
  highlight,
}: {
  label: string;
  amount: number;
  strong?: boolean;
  negative?: boolean;
  highlight?: boolean;
}) {
  const money = useKitchenMoney();
  return (
    <View style={styles.lineRow}>
      <Text style={[styles.lineLabel, strong && styles.lineLabelStrong]} numberOfLines={2}>
        {label}
      </Text>
      <Text
        style={[
          styles.lineAmount,
          strong && styles.lineAmountStrong,
          highlight && { color: theme.colors.herb.DEFAULT },
        ]}
      >
        {negative ? '−' : ''}
        {money(amount)}
      </Text>
    </View>
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
  primaryBtnDisabled: { backgroundColor: theme.colors.mist.strong },
  disclosureRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    minHeight: 32,
  },
  disclosureText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: theme.colors.ink.DEFAULT,
  },
  breakdown: { gap: 0 },
  breakdownLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.8,
    color: theme.colors.ink.muted,
    paddingBottom: theme.spacing[1],
  },
  lineRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },
  lineLabel: { flex: 1, fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft },
  lineLabelStrong: { fontFamily: 'Inter-SemiBold', color: theme.colors.ink.DEFAULT },
  lineAmount: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  lineAmountStrong: { fontFamily: 'Inter-SemiBold', fontSize: 14 },
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
