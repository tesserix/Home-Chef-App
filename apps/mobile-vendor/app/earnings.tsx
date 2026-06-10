import { useMemo, useState } from 'react';
import {
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { api } from '../lib/api';
import { downloadAndSharePdf } from '../lib/download-pdf';
import {
  useEarningsBreakdown,
  type BreakdownPeriod,
  type EarningsBreakdownOrder,
} from '../hooks/useEarningsBreakdown';
import {
  useWeeklyStatements,
  type WeeklyStatement,
} from '../hooks/useWeeklyStatements';
import { useRefunds, type RefundEntry } from '../hooks/useRefunds';

// ----- Types (payout account) -------------------------------------------------

interface LastPayout {
  amount: number;
  date: string;
}

interface PayoutResponse {
  payoutMethod: 'bank_transfer' | 'upi' | '';
  bankAccountName: string;
  bankAccountNumber: string;
  bankIFSC: string;
  upiId: string;
  razorpayConnected: boolean;
  stripeConnected: boolean;
  pendingPayout?: number;
  lastPayout?: LastPayout | null;
}

type Period = 'week' | 'month' | 'all';

/** Map the UI period tab to the API's `period` query param. */
function toApiPeriod(p: Period): BreakdownPeriod {
  if (p === 'all') return 'cycle';
  return p;
}

// ----- Payout hook ------------------------------------------------------------

function usePayoutDetails() {
  return useQuery<PayoutResponse>({
    queryKey: ['chef', 'payout'],
    queryFn: () => api.get<PayoutResponse>('/chef/payout').then((r) => r.data),
    staleTime: 60_000,
  });
}

// ----- Formatting helpers -----------------------------------------------------

function fmtInr(value: number): string {
  return `₹${value.toLocaleString('en-IN', { minimumFractionDigits: 0 })}`;
}

function fmtPct(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`;
}

function fmtShortDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
}

// ----- Sub-components ---------------------------------------------------------

interface TabLabelProps {
  label: string;
  active: boolean;
  onPress: () => void;
}

function TabLabel({ label, active, onPress }: TabLabelProps) {
  return (
    <Pressable
      onPress={onPress}
      style={tabStyles.root}
      hitSlop={6}
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
    >
      <Text style={[tabStyles.label, active && tabStyles.labelActive]}>
        {label}
      </Text>
      <View style={[tabStyles.indicator, active && tabStyles.indicatorActive]} />
    </Pressable>
  );
}

interface PayoutAccountRowProps {
  payoutMethod: PayoutResponse['payoutMethod'];
  bankAccountName: string;
  bankAccountNumber: string;
  bankIFSC: string;
  upiId: string;
}

function PayoutAccountRow({
  payoutMethod,
  bankAccountName,
  bankAccountNumber,
  bankIFSC,
  upiId,
}: PayoutAccountRowProps) {
  const hasBank =
    payoutMethod === 'bank_transfer' && bankAccountNumber.trim() !== '';
  const hasUpi = payoutMethod === 'upi' && upiId.trim() !== '';

  let label: string;
  let sublabel: string;
  if (hasBank) {
    label = bankAccountName
      ? `${bankAccountName} · ${bankAccountNumber}`
      : bankAccountNumber;
    sublabel = bankIFSC ? `IFSC ${bankIFSC}` : 'Bank account';
  } else if (hasUpi) {
    label = upiId;
    sublabel = 'UPI';
  } else {
    label = 'No payout account';
    sublabel = 'Tap to add a bank account or UPI ID';
  }

  return (
    <Pressable
      onPress={() => router.push('/payout')}
      accessibilityRole="button"
      accessibilityLabel={`Payout account: ${label}`}
    >
      {({ pressed }) => (
        <View
          style={[
            accountRowStyles.root,
            pressed && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={accountRowStyles.textBlock}>
            <Text style={accountRowStyles.sectionLabel}>PAYOUT ACCOUNT</Text>
            <Text style={accountRowStyles.value} numberOfLines={1}>
              {label}
            </Text>
            <Text style={accountRowStyles.sub} numberOfLines={1}>
              {sublabel}
            </Text>
          </View>
          <ChevronLeft
            size={16}
            color={theme.colors.ink.muted}
            style={{ transform: [{ rotate: '180deg' }] }}
          />
        </View>
      )}
    </Pressable>
  );
}

// ---- Breakdown section -------------------------------------------------------

interface BreakdownRowProps {
  label: string;
  value: string;
  /** When true, renders the value in a muted/secondary style */
  secondary?: boolean;
  /** When true, renders the value in persimmon (positive net payout) */
  accent?: boolean;
  /** The right-side annotation shown in brackets after the label */
  annotation?: string;
}

function BreakdownRow({
  label,
  value,
  secondary,
  accent,
  annotation,
}: BreakdownRowProps) {
  return (
    <View style={bkStyles.row}>
      <Text style={bkStyles.rowLabel} numberOfLines={1}>
        {label}
        {annotation ? (
          <Text style={bkStyles.rowAnnotation}> {annotation}</Text>
        ) : null}
      </Text>
      <Text
        style={[
          bkStyles.rowValue,
          secondary && bkStyles.rowValueSecondary,
          accent && bkStyles.rowValueAccent,
        ]}
      >
        {value}
      </Text>
    </View>
  );
}

const bkStyles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
    minHeight: 40,
  },
  rowLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    flex: 1,
  },
  rowAnnotation: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  rowValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    textAlign: 'right',
  },
  rowValueSecondary: {
    color: theme.colors.ink.muted,
    fontFamily: 'Inter',
  },
  rowValueAccent: {
    color: theme.colors.herb.DEFAULT,
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
  },
});

// ---- Order history row (tappable) -------------------------------------------

interface OrderHistoryRowProps {
  order: EarningsBreakdownOrder;
  onPress: () => void;
}

function OrderHistoryRow({ order, onPress }: OrderHistoryRowProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Open order ${order.orderNumber}`}
    >
      {({ pressed }) => (
        <View
          style={[
            orderRowStyles.root,
            pressed && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={orderRowStyles.leftBlock}>
            <Text style={orderRowStyles.orderNumber} numberOfLines={1}>
              #{order.orderNumber}
            </Text>
            <Text style={orderRowStyles.date}>
              {fmtShortDate(order.completedAt)}
            </Text>
          </View>
          <View style={orderRowStyles.rightBlock}>
            <Text style={orderRowStyles.netPayout}>{fmtInr(order.netPayout)}</Text>
            <Text style={orderRowStyles.gross}>
              Gross {fmtInr(order.gross)}
            </Text>
          </View>
          <ChevronLeft
            size={14}
            color={theme.colors.ink.muted}
            style={{ transform: [{ rotate: '180deg' }] }}
          />
        </View>
      )}
    </Pressable>
  );
}

const orderRowStyles = StyleSheet.create({
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 44,
    paddingVertical: theme.spacing[2],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  leftBlock: { flex: 1 },
  orderNumber: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  date: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },
  rightBlock: { alignItems: 'flex-end' },
  netPayout: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  gross: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 1,
    fontVariant: ['tabular-nums'],
  },
});

// ---- Statement row (tappable → downloads the weekly settlement PDF) ----------

/** Format a [weekStart, weekEnd) ISO range as "1 Jun – 7 Jun" (weekEnd is exclusive). */
function fmtWeekRange(weekStart: string, weekEnd: string): string {
  const start = new Date(`${weekStart}T00:00:00`);
  const end = new Date(`${weekEnd}T00:00:00`);
  end.setDate(end.getDate() - 1); // weekEnd is the exclusive next Monday
  const opts: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'short' };
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return weekStart;
  }
  return `${start.toLocaleDateString('en-IN', opts)} – ${end.toLocaleDateString('en-IN', opts)}`;
}

interface StatementRowProps {
  statement: WeeklyStatement;
  onPress: () => void;
}

function StatementRow({ statement, onPress }: StatementRowProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Download settlement statement for ${fmtWeekRange(statement.weekStart, statement.weekEnd)}`}
    >
      {({ pressed }) => (
        <View
          style={[
            orderRowStyles.root,
            pressed && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={orderRowStyles.leftBlock}>
            <Text style={orderRowStyles.orderNumber} numberOfLines={1}>
              {fmtWeekRange(statement.weekStart, statement.weekEnd)}
            </Text>
            <Text style={orderRowStyles.date}>
              {statement.ordersCount} order
              {statement.ordersCount === 1 ? '' : 's'}
            </Text>
          </View>
          <View style={orderRowStyles.rightBlock}>
            <Text style={orderRowStyles.netPayout}>
              {fmtInr(statement.netPayout)}
            </Text>
            <Text style={statementRowStyles.download}>Download PDF</Text>
          </View>
          <ChevronLeft
            size={14}
            color={theme.colors.ink.muted}
            style={{ transform: [{ rotate: '180deg' }] }}
          />
        </View>
      )}
    </Pressable>
  );
}

const statementRowStyles = StyleSheet.create({
  download: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.herb.DEFAULT, // persimmon accent
    marginTop: 1,
  },
});

// ---- Tax document row (downloads the annual TDS certificate) -----------------

/** Current Indian financial year label, e.g. "FY 2026-27" (FY starts 1 Apr). */
function currentFyLabel(): string {
  const now = new Date();
  const fyStart = now.getMonth() >= 3 ? now.getFullYear() : now.getFullYear() - 1;
  const endShort = String((fyStart + 1) % 100).padStart(2, '0');
  return `FY ${fyStart}-${endShort}`;
}

interface TaxDocumentRowProps {
  fyLabel: string;
  onPress: () => void;
}

function TaxDocumentRow({ fyLabel, onPress }: TaxDocumentRowProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Download TDS certificate for ${fyLabel}`}
    >
      {({ pressed }) => (
        <View
          style={[
            accountRowStyles.root,
            pressed && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={accountRowStyles.textBlock}>
            <Text style={accountRowStyles.sectionLabel}>TAX DOCUMENTS</Text>
            <Text style={accountRowStyles.value} numberOfLines={1}>
              TDS certificate · {fyLabel}
            </Text>
            <Text style={accountRowStyles.sub} numberOfLines={1}>
              Annual summary (Section 194-O)
            </Text>
          </View>
          <Text style={statementRowStyles.download}>Download</Text>
        </View>
      )}
    </Pressable>
  );
}

// ---- Refund row (tappable → opens the order) ---------------------------------

interface RefundRowProps {
  refund: RefundEntry;
  onPress: () => void;
}

function RefundRow({ refund, onPress }: RefundRowProps) {
  const detail =
    refund.reason ||
    (refund.items && refund.items.length > 0
      ? refund.items.map((i) => i.name).join(', ')
      : 'Refund');
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Refund on order ${refund.orderNumber}, ${fmtInr(refund.amount)}`}
    >
      {({ pressed }) => (
        <View
          style={[
            orderRowStyles.root,
            pressed && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={orderRowStyles.leftBlock}>
            <Text style={orderRowStyles.orderNumber} numberOfLines={1}>
              #{refund.orderNumber}
            </Text>
            <Text style={orderRowStyles.date} numberOfLines={1}>
              {detail} · {fmtShortDate(refund.refundedAt)}
            </Text>
          </View>
          <View style={orderRowStyles.rightBlock}>
            <Text style={refundRowStyles.amount}>
              − {fmtInr(refund.amount)}
            </Text>
          </View>
          <ChevronLeft
            size={14}
            color={theme.colors.ink.muted}
            style={{ transform: [{ rotate: '180deg' }] }}
          />
        </View>
      )}
    </Pressable>
  );
}

const refundRowStyles = StyleSheet.create({
  amount: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.destructive.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
});

// ----- List item type ---------------------------------------------------------

type ListItem =
  | { type: 'hero' }
  | { type: 'figures' }
  | { type: 'tabBar' }
  | { type: 'breakdown' }
  | { type: 'orderListHeader' }
  | { type: 'orderEntry'; order: EarningsBreakdownOrder }
  | { type: 'refundsHeader' }
  | { type: 'refundEntry'; refund: RefundEntry }
  | { type: 'statementsHeader' }
  | { type: 'statementEntry'; statement: WeeklyStatement }
  | { type: 'taxRow' }
  | { type: 'payoutRow' }
  | { type: 'empty' };

// ----- Screen -----------------------------------------------------------------

export default function EarningsScreen() {
  const [period, setPeriod] = useState<Period>('week');
  const apiPeriod = toApiPeriod(period);

  // Payout account details (bank/UPI + pending/last payout)
  const {
    data: payoutData,
    isLoading: payoutLoading,
    isError: payoutError,
    refetch: refetchPayout,
  } = usePayoutDetails();

  // Earnings breakdown (new endpoint)
  const {
    data: breakdown,
    isLoading: breakdownLoading,
    refetch: refetchBreakdown,
  } = useEarningsBreakdown(apiPeriod);

  // Weekly settlement statements (issued by the backend cron)
  const { data: statements, refetch: refetchStatements } =
    useWeeklyStatements();

  // Refund history (one entry per refunded order)
  const { data: refunds, refetch: refetchRefunds } = useRefunds();

  const isLoading = payoutLoading || breakdownLoading;
  const isError = payoutError;

  // User-initiated pull-to-refresh only — avoids the stuck-spinner bug
  // when React Query's isRefetching goes true for background refetches.
  const [isPulling, setIsPulling] = useState(false);
  async function onPullRefresh(): Promise<void> {
    setIsPulling(true);
    try {
      await Promise.all([
        refetchPayout(),
        refetchBreakdown(),
        refetchStatements(),
        refetchRefunds(),
      ]);
    } finally {
      setIsPulling(false);
    }
  }

  const totals = breakdown?.totals;
  const orders = breakdown?.orders ?? [];
  const rates = breakdown?.rates;

  // Period total for the hero: net payout from breakdown, fallback to 0
  const periodTotal = totals?.netPayout ?? 0;

  // GST display helper: show CGST+SGST combined when intra-state, IGST when
  // inter-state (IGST > 0 signals inter-state supply).
  const gstInterState = (totals?.igst ?? 0) > 0;
  const gstCombined = (totals?.cgst ?? 0) + (totals?.sgst ?? 0);

  const listItems = useMemo<ListItem[]>(() => {
    const items: ListItem[] = [
      { type: 'hero' },
      { type: 'figures' },
      { type: 'tabBar' },
    ];

    if (totals) {
      items.push({ type: 'breakdown' });
    }

    if (orders.length === 0) {
      items.push({ type: 'empty' });
    } else {
      items.push({ type: 'orderListHeader' });
      orders.forEach((order) => {
        items.push({ type: 'orderEntry', order });
      });
    }

    const refundList = refunds ?? [];
    if (refundList.length > 0) {
      items.push({ type: 'refundsHeader' });
      refundList.forEach((refund) => {
        items.push({ type: 'refundEntry', refund });
      });
    }

    const stmts = statements ?? [];
    if (stmts.length > 0) {
      items.push({ type: 'statementsHeader' });
      stmts.forEach((statement) => {
        items.push({ type: 'statementEntry', statement });
      });
    }

    items.push({ type: 'taxRow' });
    items.push({ type: 'payoutRow' });
    return items;
  }, [totals, orders, statements]);

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            style={styles.backButton}
            accessibilityLabel="Go back"
            accessibilityRole="button"
          >
            <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} />
          </Pressable>
          <Text style={styles.commandTitle}>Earnings</Text>
        </View>
        <View style={styles.skeletonPad}>
          <Skeleton height={48} style={{ width: 180, marginBottom: 16 }} />
          <Skeleton height={20} style={{ width: 240, marginBottom: 32 }} />
          <Skeleton height={56} style={{ marginBottom: 8 }} />
          <Skeleton height={56} style={{ marginBottom: 8 }} />
          <Skeleton height={56} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView style={styles.errorRoot} edges={['top', 'left', 'right']}>
        <Text style={styles.errorTitle}>Couldn't load earnings</Text>
        <Text style={styles.errorBody}>
          Check your connection and try again.
        </Text>
        <Pressable
          onPress={() => refetchPayout()}
          style={styles.errorPrimary}
          accessibilityRole="button"
        >
          <Text style={styles.errorPrimaryText}>Retry</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  function renderItem({ item }: { item: ListItem }) {
    switch (item.type) {
      case 'hero':
        return (
          <View style={styles.heroBlock}>
            <Text style={styles.heroLabel}>
              {period === 'week'
                ? 'THIS WEEK'
                : period === 'month'
                  ? 'THIS MONTH'
                  : 'ALL TIME'}
            </Text>
            <Text style={styles.heroAmount}>{fmtInr(periodTotal)}</Text>
            {breakdown && (
              <Text style={styles.heroCycleDates}>
                {fmtShortDate(breakdown.cycleStart)} –{' '}
                {fmtShortDate(breakdown.cycleEnd)}
              </Text>
            )}
          </View>
        );

      case 'figures':
        return (
          <View style={styles.figuresStrip}>
            <View style={styles.figureItem}>
              <Text style={styles.figureLabel}>PENDING</Text>
              <Text style={styles.figureValue}>
                {fmtInr(payoutData?.pendingPayout ?? 0)}
              </Text>
            </View>
            <View style={styles.figureDivider} />
            {payoutData?.lastPayout ? (
              <View style={styles.figureItem}>
                <Text style={styles.figureLabel}>LAST PAYOUT</Text>
                <Text style={styles.figureValue}>
                  {fmtInr(payoutData.lastPayout.amount)}
                </Text>
                <Text style={styles.figureSub}>
                  {new Date(payoutData.lastPayout.date).toLocaleDateString(
                    'en-IN',
                    { day: 'numeric', month: 'short' },
                  )}
                </Text>
              </View>
            ) : (
              <View style={styles.figureItem}>
                <Text style={styles.figureLabel}>LAST PAYOUT</Text>
                <Text style={styles.figureSub}>None yet</Text>
              </View>
            )}
          </View>
        );

      case 'tabBar':
        return (
          <View style={styles.tabBar}>
            <TabLabel
              label="Week"
              active={period === 'week'}
              onPress={() => setPeriod('week')}
            />
            <TabLabel
              label="Month"
              active={period === 'month'}
              onPress={() => setPeriod('month')}
            />
            <TabLabel
              label="All"
              active={period === 'all'}
              onPress={() => setPeriod('all')}
            />
          </View>
        );

      case 'breakdown':
        if (!totals || !rates) return null;
        return (
          <View style={styles.breakdownSection}>
            <Text style={styles.breakdownSectionLabel}>BREAKDOWN</Text>
            <View style={styles.breakdownGroup}>
              <BreakdownRow
                label="Gross revenue"
                value={fmtInr(totals.grossRevenue)}
              />
              <BreakdownRow
                label="Platform commission"
                annotation={`(${fmtPct(rates.platformCommission)})`}
                value={`− ${fmtInr(totals.platformCommission)}`}
                secondary
              />
              {gstInterState ? (
                <>
                  <BreakdownRow
                    label="GST (IGST)"
                    annotation={`(${fmtPct(rates.gst)})`}
                    value={`− ${fmtInr(totals.igst)}`}
                    secondary
                  />
                </>
              ) : (
                <>
                  <BreakdownRow
                    label="GST"
                    annotation={`(${fmtPct(rates.gst)})`}
                    value={`− ${fmtInr(gstCombined)}`}
                    secondary
                  />
                  {gstCombined > 0 && (
                    <View style={bkStyles.row}>
                      <Text style={styles.gstSubRow}>
                        CGST {fmtInr(totals.cgst)} + SGST {fmtInr(totals.sgst)}
                      </Text>
                    </View>
                  )}
                </>
              )}
              <BreakdownRow
                label="TDS"
                annotation={`(${fmtPct(rates.tds)})`}
                value={`− ${fmtInr(totals.tds)}`}
                secondary
              />
              <BreakdownRow
                label="Net payout"
                value={fmtInr(totals.netPayout)}
                accent={totals.netPayout > 0}
              />
            </View>
            <Text style={styles.ordersCount}>
              {totals.ordersCount} order{totals.ordersCount === 1 ? '' : 's'}{' '}
              in period
            </Text>
          </View>
        );

      case 'orderListHeader':
        return (
          <Text style={styles.dateHeader}>ORDER HISTORY</Text>
        );

      case 'orderEntry':
        return (
          <OrderHistoryRow
            order={item.order}
            onPress={() => router.push(`/orders/${item.order.orderId}`)}
          />
        );

      case 'refundsHeader':
        return <Text style={styles.dateHeader}>REFUNDS</Text>;

      case 'refundEntry':
        return (
          <RefundRow
            refund={item.refund}
            onPress={() => router.push(`/orders/${item.refund.orderId}`)}
          />
        );

      case 'statementsHeader':
        return <Text style={styles.dateHeader}>WEEKLY STATEMENTS</Text>;

      case 'statementEntry':
        return (
          <StatementRow
            statement={item.statement}
            onPress={() =>
              downloadAndSharePdf(
                `/chef/statements/${item.statement.id}/statement.pdf`,
                `statement-${item.statement.weekStart}.pdf`,
              )
            }
          />
        );

      case 'taxRow':
        return (
          <View style={styles.payoutSection}>
            <TaxDocumentRow
              fyLabel={currentFyLabel()}
              onPress={() =>
                downloadAndSharePdf(
                  '/chef/tax/certificate',
                  `tds-certificate-${currentFyLabel().replace(/\s/g, '')}.pdf`,
                )
              }
            />
          </View>
        );

      case 'payoutRow':
        return (
          <View style={styles.payoutSection}>
            <PayoutAccountRow
              payoutMethod={payoutData?.payoutMethod ?? ''}
              bankAccountName={payoutData?.bankAccountName ?? ''}
              bankAccountNumber={payoutData?.bankAccountNumber ?? ''}
              bankIFSC={payoutData?.bankIFSC ?? ''}
              upiId={payoutData?.upiId ?? ''}
            />
          </View>
        );

      case 'empty':
        return (
          <View style={styles.emptyBlock}>
            <Text style={styles.emptyHeadline}>No orders in period</Text>
            <Text style={styles.emptyBody}>
              Completed orders will appear here after your first payout cycle
              closes.
            </Text>
          </View>
        );

      default:
        return null;
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          style={styles.backButton}
          accessibilityLabel="Go back"
          accessibilityRole="button"
        >
          <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.commandTitle}>Earnings</Text>
      </View>

      <FlatList
        data={listItems}
        keyExtractor={(item, idx) => {
          if (item.type === 'orderEntry') return item.order.orderId;
          if (item.type === 'statementEntry') return `stmt-${item.statement.id}`;
          if (item.type === 'refundEntry') return `refund-${item.refund.orderId}`;
          return `${item.type}-${idx}`;
        }}
        contentContainerStyle={styles.listContent}
        renderItem={renderItem}
        refreshControl={
          <RefreshControl
            refreshing={isPulling}
            onRefresh={onPullRefresh}
            tintColor={theme.colors.ink.DEFAULT}
          />
        }
        showsVerticalScrollIndicator={false}
      />
    </SafeAreaView>
  );
}

// ----- Styles ---------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  errorRoot: {
    flex: 1,
    backgroundColor: theme.colors.paper,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
  },

  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backButton: {
    minWidth: 44,
    minHeight: 44,
    alignItems: 'flex-start',
    justifyContent: 'center',
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  listContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },

  // Hero block
  heroBlock: {
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[5],
  },
  heroLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[2],
  },
  heroAmount: {
    fontFamily: 'Geist-Bold',
    fontSize: 44,
    lineHeight: 48,
    letterSpacing: -1,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  heroCycleDates: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 4,
    fontVariant: ['tabular-nums'],
  },

  // Figures strip
  figuresStrip: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    paddingTop: theme.spacing[4],
    paddingBottom: theme.spacing[4],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    gap: theme.spacing[6],
  },
  figureItem: { flex: 1 },
  figureLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
    marginBottom: 4,
  },
  figureValue: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    lineHeight: 24,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  figureSub: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },
  figureDivider: {
    width: StyleSheet.hairlineWidth,
    alignSelf: 'stretch',
    backgroundColor: theme.colors.mist.DEFAULT,
  },

  // Tab bar
  tabBar: {
    flexDirection: 'row',
    gap: theme.spacing[6],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
    marginBottom: 0,
  },

  // Breakdown section
  breakdownSection: {
    marginTop: theme.spacing[5],
  },
  breakdownSectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[2],
  },
  breakdownGroup: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },
  gstSubRow: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    paddingBottom: theme.spacing[2],
    flex: 1,
  },
  ordersCount: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[2],
  },

  // Date header for order list
  dateHeader: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[5],
    marginBottom: theme.spacing[2],
  },

  // Payout account section
  payoutSection: {
    marginTop: theme.spacing[6],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },

  // Empty state
  emptyBlock: {
    paddingTop: theme.spacing[6],
    paddingBottom: theme.spacing[4],
  },
  emptyHeadline: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.2,
    marginBottom: theme.spacing[2],
  },
  emptyBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
    maxWidth: 300,
  },

  // Loading skeleton
  skeletonPad: {
    padding: theme.spacing[4],
  },

  // Error state
  errorTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
    letterSpacing: -0.2,
    marginBottom: theme.spacing[2],
  },
  errorBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    marginBottom: theme.spacing[6],
  },
  errorPrimary: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  errorPrimaryText: {
    color: theme.colors.paper,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
  },
});

const tabStyles = StyleSheet.create({
  root: {
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[2],
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.1,
    paddingBottom: 6,
  },
  labelActive: {
    color: theme.colors.ink.DEFAULT,
  },
  indicator: {
    height: 2,
    backgroundColor: 'transparent',
    borderRadius: 1,
  },
  indicatorActive: {
    backgroundColor: theme.colors.ink.DEFAULT,
  },
});

const accountRowStyles = StyleSheet.create({
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: theme.spacing[4],
    minHeight: 44,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  textBlock: { flex: 1 },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
    marginBottom: 4,
  },
  value: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  sub: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },
});
