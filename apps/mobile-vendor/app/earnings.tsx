import { useMemo, useState, type ReactNode } from 'react';
import {
  FlatList,
  Platform,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
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
import { payoutHoldMeta, statementStatusMeta } from '../lib/payout-hold';

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
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      {({ pressed }) => (
        // Inner-View pattern — visual styles live on the View, not the
        // Pressable, to dodge the iOS function-style style drop.
        <View
          style={[
            tabStyles.segment,
            active && tabStyles.segmentActive,
            pressed && Platform.OS === 'ios' && { opacity: 0.7 },
          ]}
        >
          <Text style={[tabStyles.label, active && tabStyles.labelActive]}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

// ----- Group-card segment shell (UI-V2-SPEC §1/§9) -----------------------------
// Each list row is one segment of a white group card. First-in-group gets top
// radii, last gets bottom radii; the shadow lives on the outer layer so the
// inner overflow-hidden clip (needed for the pressed-state bone fill to respect
// the radius) doesn't cut it off. Mirrors orders.tsx HistoryRow.

interface CardSegmentProps {
  first: boolean;
  last: boolean;
  children: ReactNode;
}

function CardSegment({ first, last, children }: CardSegmentProps) {
  return (
    <View
      style={[
        segmentStyles.shell,
        first && segmentStyles.top,
        last && segmentStyles.bottom,
      ]}
    >
      <View
        style={[
          segmentStyles.clip,
          first && segmentStyles.top,
          last && segmentStyles.bottom,
        ]}
      >
        {children}
        {!last && <View style={segmentStyles.separator} />}
      </View>
    </View>
  );
}

const segmentStyles = StyleSheet.create({
  shell: {
    backgroundColor: theme.colors.paper,
    ...theme.shadow[1],
  },
  top: {
    borderTopLeftRadius: theme.radius.lg,
    borderTopRightRadius: theme.radius.lg,
  },
  bottom: {
    borderBottomLeftRadius: theme.radius.lg,
    borderBottomRightRadius: theme.radius.lg,
  },
  clip: {
    overflow: 'hidden',
    backgroundColor: theme.colors.paper,
  },
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4], // inset — aligned to row content
  },
});

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
  const { t } = useTranslation();
  const hasBank =
    payoutMethod === 'bank_transfer' && bankAccountNumber.trim() !== '';
  const hasUpi = payoutMethod === 'upi' && upiId.trim() !== '';

  let label: string;
  let sublabel: string;
  if (hasBank) {
    label = bankAccountName
      ? `${bankAccountName} · ${bankAccountNumber}`
      : bankAccountNumber;
    sublabel = bankIFSC ? t('earnings.ifsc', { ifsc: bankIFSC }) : t('earnings.bankAccount');
  } else if (hasUpi) {
    label = upiId;
    sublabel = t('earnings.upi');
  } else {
    label = t('earnings.noPayoutAccount');
    sublabel = t('earnings.addPayoutHint');
  }

  return (
    <Pressable
      onPress={() => router.push('/payout')}
      accessibilityRole="button"
      accessibilityLabel={`Payout account: ${label}`}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            accountRowStyles.root,
            pressed && Platform.OS === 'ios' && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={accountRowStyles.textBlock}>
            <Text style={accountRowStyles.sectionLabel}>{t('earnings.payoutAccountLabel')}</Text>
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
  /** Totals row — emphasized label + Geist-Bold value (UI-V2-SPEC) */
  emphasized?: boolean;
  /** Last row in the card — drops the inset hairline */
  last?: boolean;
}

function BreakdownRow({
  label,
  value,
  secondary,
  accent,
  annotation,
  emphasized,
  last,
}: BreakdownRowProps) {
  return (
    <View style={[bkStyles.row, last && bkStyles.rowLast]}>
      <Text
        style={[bkStyles.rowLabel, emphasized && bkStyles.rowLabelEmphasized]}
        numberOfLines={1}
      >
        {label}
        {annotation ? (
          <Text style={bkStyles.rowAnnotation}> {annotation}</Text>
        ) : null}
      </Text>
      <Text
        style={[
          bkStyles.rowValue,
          secondary && bkStyles.rowValueSecondary,
          emphasized && bkStyles.rowValueEmphasized,
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
  rowLast: {
    borderBottomWidth: 0,
  },
  rowLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    flex: 1,
  },
  rowLabelEmphasized: {
    fontFamily: 'Inter-SemiBold',
    color: theme.colors.ink.DEFAULT,
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
  rowValueEmphasized: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
  },
  rowValueAccent: {
    color: theme.colors.ink.DEFAULT,
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
  const { t } = useTranslation();
  // #617 — escrow hold pill; empty labelKey (no hold / flags off) renders nothing.
  const hold = payoutHoldMeta(order.payoutHoldStatus);
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Open order ${order.orderNumber}`}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            orderRowStyles.root,
            pressed && Platform.OS === 'ios' && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={orderRowStyles.leftBlock}>
            <Text style={orderRowStyles.orderNumber} numberOfLines={1}>
              #{order.orderNumber}
            </Text>
            <Text style={orderRowStyles.date}>
              {fmtShortDate(order.completedAt)}
            </Text>
            {hold.labelKey ? (
              <View style={[orderRowStyles.holdPill, { backgroundColor: hold.bg }]}>
                <Text style={[orderRowStyles.holdPillText, { color: hold.fg }]}>
                  {t(hold.labelKey)}
                </Text>
              </View>
            ) : null}
          </View>
          <View style={orderRowStyles.rightBlock}>
            <Text style={orderRowStyles.netPayout}>{fmtInr(order.netPayout)}</Text>
            <Text style={orderRowStyles.gross}>
              {t('earnings.gross', { amount: fmtInr(order.gross) })}
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
  // Rows live inside white group cards (CardSegment) — separators are
  // rendered by the segment shell, not the row itself.
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 56,
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
  },
  leftBlock: { flex: 1 },
  orderNumber: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  // #617 — escrow hold / statement-status pill (tint bg + colored text, radius.full).
  holdPill: {
    alignSelf: 'flex-start',
    marginTop: 4,
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 2,
    borderRadius: theme.radius.full,
  },
  holdPillText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
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
  const { t } = useTranslation();
  // #617 — disbursement pill: paid (green) vs pending (amber).
  const status = statementStatusMeta(statement.status);
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Download settlement statement for ${fmtWeekRange(statement.weekStart, statement.weekEnd)}`}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            orderRowStyles.root,
            pressed && Platform.OS === 'ios' && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={orderRowStyles.leftBlock}>
            <Text style={orderRowStyles.orderNumber} numberOfLines={1}>
              {fmtWeekRange(statement.weekStart, statement.weekEnd)}
            </Text>
            <Text style={orderRowStyles.date}>
              {statement.ordersCount === 1
                ? t('earnings.ordersInPeriodOne', { count: statement.ordersCount })
                : t('earnings.ordersInPeriod', { count: statement.ordersCount })}
            </Text>
            <View style={[orderRowStyles.holdPill, { backgroundColor: status.bg }]}>
              <Text style={[orderRowStyles.holdPillText, { color: status.fg }]}>
                {t(status.labelKey)}
              </Text>
            </View>
          </View>
          <View style={orderRowStyles.rightBlock}>
            <Text style={orderRowStyles.netPayout}>
              {fmtInr(statement.netPayout)}
            </Text>
            <Text style={statementRowStyles.download}>{t('earnings.downloadPdfShort')}</Text>
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
    color: theme.colors.ink.DEFAULT, // persimmon accent
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
  const { t } = useTranslation();
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Download TDS certificate for ${fyLabel}`}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            accountRowStyles.root,
            pressed && Platform.OS === 'ios' && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={accountRowStyles.textBlock}>
            <Text style={accountRowStyles.sectionLabel}>{t('earnings.taxDocumentsLabel')}</Text>
            <Text style={accountRowStyles.value} numberOfLines={1}>
              {t('earnings.tdsCertificateFy', { fy: fyLabel })}
            </Text>
            <Text style={accountRowStyles.sub} numberOfLines={1}>
              {t('earnings.annualSummary')}
            </Text>
          </View>
          <Text style={statementRowStyles.download}>{t('earnings.downloadShort')}</Text>
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
  const { t } = useTranslation();
  const detail =
    refund.reason ||
    (refund.items && refund.items.length > 0
      ? refund.items.map((i) => i.name).join(', ')
      : t('earnings.refund'));
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Refund on order ${refund.orderNumber}, ${fmtInr(refund.amount)}`}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            orderRowStyles.root,
            pressed && Platform.OS === 'ios' && { backgroundColor: theme.colors.bone },
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
  | {
      type: 'orderEntry';
      order: EarningsBreakdownOrder;
      first: boolean;
      last: boolean;
    }
  | { type: 'refundsHeader' }
  | { type: 'refundEntry'; refund: RefundEntry; first: boolean; last: boolean }
  | { type: 'statementsHeader' }
  | {
      type: 'statementEntry';
      statement: WeeklyStatement;
      first: boolean;
      last: boolean;
    }
  | { type: 'taxRow' }
  | { type: 'payoutRow' }
  | { type: 'empty' };

// ----- Screen -----------------------------------------------------------------

export default function EarningsScreen() {
  const { t } = useTranslation();
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
      orders.forEach((order, idx) => {
        items.push({
          type: 'orderEntry',
          order,
          first: idx === 0,
          last: idx === orders.length - 1,
        });
      });
    }

    const refundList = refunds ?? [];
    if (refundList.length > 0) {
      items.push({ type: 'refundsHeader' });
      refundList.forEach((refund, idx) => {
        items.push({
          type: 'refundEntry',
          refund,
          first: idx === 0,
          last: idx === refundList.length - 1,
        });
      });
    }

    const stmts = statements ?? [];
    if (stmts.length > 0) {
      items.push({ type: 'statementsHeader' });
      stmts.forEach((statement, idx) => {
        items.push({
          type: 'statementEntry',
          statement,
          first: idx === 0,
          last: idx === stmts.length - 1,
        });
      });
    }

    items.push({ type: 'taxRow' });
    items.push({ type: 'payoutRow' });
    return items;
  }, [totals, orders, refunds, statements]);

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityLabel={t('earnings.goBack')}
            accessibilityRole="button"
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.backButton,
                  pressed && Platform.OS === 'ios' && { opacity: 0.6 },
                ]}
              >
                <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} />
              </View>
            )}
          </Pressable>
          <Text style={styles.commandTitle}>{t('earnings.title')}</Text>
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
        <Text style={styles.errorTitle}>{t('earnings.errorTitle')}</Text>
        <Text style={styles.errorBody}>
          {t('earnings.errorBody')}
        </Text>
        <Pressable
          onPress={() => refetchPayout()}
          accessibilityRole="button"
          android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.errorPrimary,
                pressed && Platform.OS === 'ios' && { opacity: 0.85 },
              ]}
            >
              <Text style={styles.errorPrimaryText}>{t('common.retry')}</Text>
            </View>
          )}
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
                ? t('earnings.heroThisWeek')
                : period === 'month'
                  ? t('earnings.heroThisMonth')
                  : t('earnings.heroAllTime')}
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

      case 'figures': {
        // #617 — real escrow figures. "Held" = net payout still in escrow for the
        // period (₹0 while the flags are off, so this reads as "nothing held" pre-
        // launch); "Last payout" = the most recent PAID weekly statement (the old
        // fields the backend never sent, so they always showed ₹0 / "None yet").
        const held = breakdown?.totals.held ?? 0;
        const lastPaid = (statements ?? []).find((s) => s.status === 'paid');
        return (
          <View style={styles.figuresStrip}>
            <View style={styles.figureItem}>
              <Text style={styles.figureLabel}>{t('earnings.figureHeld')}</Text>
              <Text style={styles.figureValue}>{fmtInr(held)}</Text>
            </View>
            <View style={styles.figureDivider} />
            {lastPaid ? (
              <View style={styles.figureItem}>
                <Text style={styles.figureLabel}>{t('earnings.figureLastPayout')}</Text>
                <Text style={styles.figureValue}>{fmtInr(lastPaid.netPayout)}</Text>
                <Text style={styles.figureSub}>
                  {lastPaid.paidAt
                    ? new Date(lastPaid.paidAt).toLocaleDateString('en-IN', {
                        day: 'numeric',
                        month: 'short',
                      })
                    : fmtWeekRange(lastPaid.weekStart, lastPaid.weekEnd)}
                </Text>
              </View>
            ) : (
              <View style={styles.figureItem}>
                <Text style={styles.figureLabel}>{t('earnings.figureLastPayout')}</Text>
                <Text style={styles.figureSub}>{t('earnings.noneYet')}</Text>
              </View>
            )}
          </View>
        );
      }

      case 'tabBar':
        return (
          <View style={styles.tabBar}>
            <TabLabel
              label={t('earnings.week')}
              active={period === 'week'}
              onPress={() => setPeriod('week')}
            />
            <TabLabel
              label={t('earnings.month')}
              active={period === 'month'}
              onPress={() => setPeriod('month')}
            />
            <TabLabel
              label={t('earnings.all')}
              active={period === 'all'}
              onPress={() => setPeriod('all')}
            />
          </View>
        );

      case 'breakdown':
        if (!totals || !rates) return null;
        return (
          <View style={styles.breakdownSection}>
            <Text style={styles.breakdownSectionLabel}>{t('earnings.breakdownLabel')}</Text>
            <View style={styles.breakdownGroup}>
              <BreakdownRow
                label={t('earnings.grossRevenue')}
                value={fmtInr(totals.grossRevenue)}
              />
              <BreakdownRow
                label={t('earnings.platformCommission')}
                annotation={`(${fmtPct(rates.platformCommission)})`}
                value={`− ${fmtInr(totals.platformCommission)}`}
                secondary
              />
              {gstInterState ? (
                <>
                  <BreakdownRow
                    label={t('earnings.gstIgst')}
                    annotation={`(${fmtPct(rates.gst)})`}
                    value={`− ${fmtInr(totals.igst)}`}
                    secondary
                  />
                </>
              ) : (
                <>
                  <BreakdownRow
                    label={t('earnings.gst')}
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
                label={t('earnings.tds')}
                annotation={`(${fmtPct(rates.tds)})`}
                value={`− ${fmtInr(totals.tds)}`}
                secondary
              />
              <BreakdownRow
                label={t('earnings.netPayout')}
                value={fmtInr(totals.netPayout)}
                accent={totals.netPayout > 0}
                emphasized
                last
              />
            </View>
            <Text style={styles.ordersCount}>
              {totals.ordersCount === 1
                ? t('earnings.ordersInPeriodOne', { count: totals.ordersCount })
                : t('earnings.ordersInPeriod', { count: totals.ordersCount })}
            </Text>
          </View>
        );

      case 'orderListHeader':
        return (
          <Text style={styles.dateHeader}>{t('earnings.orderHistoryHeader')}</Text>
        );

      case 'orderEntry':
        return (
          <CardSegment first={item.first} last={item.last}>
            <OrderHistoryRow
              order={item.order}
              onPress={() => router.push(`/orders/${item.order.orderId}`)}
            />
          </CardSegment>
        );

      case 'refundsHeader':
        return <Text style={styles.dateHeader}>{t('earnings.refundsHeader')}</Text>;

      case 'refundEntry':
        return (
          <CardSegment first={item.first} last={item.last}>
            <RefundRow
              refund={item.refund}
              onPress={() => router.push(`/orders/${item.refund.orderId}`)}
            />
          </CardSegment>
        );

      case 'statementsHeader':
        return <Text style={styles.dateHeader}>{t('earnings.weeklyStatementsHeader')}</Text>;

      case 'statementEntry':
        return (
          <CardSegment first={item.first} last={item.last}>
            <StatementRow
              statement={item.statement}
              onPress={() =>
                downloadAndSharePdf(
                  `/chef/statements/${item.statement.id}/statement.pdf`,
                  `statement-${item.statement.weekStart}.pdf`,
                )
              }
            />
          </CardSegment>
        );

      case 'taxRow':
        return (
          <View style={styles.payoutSection}>
            <View style={styles.singleCard}>
              <View style={styles.singleCardClip}>
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
            </View>
          </View>
        );

      case 'payoutRow':
        return (
          <View style={styles.payoutSection}>
            <View style={styles.singleCard}>
              <View style={styles.singleCardClip}>
                <PayoutAccountRow
                  payoutMethod={payoutData?.payoutMethod ?? ''}
                  bankAccountName={payoutData?.bankAccountName ?? ''}
                  bankAccountNumber={payoutData?.bankAccountNumber ?? ''}
                  bankIFSC={payoutData?.bankIFSC ?? ''}
                  upiId={payoutData?.upiId ?? ''}
                />
              </View>
            </View>
          </View>
        );

      case 'empty':
        return (
          <View style={styles.emptyBlock}>
            <Text style={styles.emptyHeadline}>{t('earnings.noOrdersInPeriod')}</Text>
            <Text style={styles.emptyBody}>
              {t('earnings.noOrdersInPeriodBody')}
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
          accessibilityLabel={t('earnings.goBack')}
          accessibilityRole="button"
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.backButton,
                pressed && Platform.OS === 'ios' && { opacity: 0.6 },
              ]}
            >
              <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle}>{t('earnings.title')}</Text>
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
  root: { flex: 1, backgroundColor: theme.colors.bone },
  errorRoot: {
    flex: 1,
    backgroundColor: theme.colors.bone,
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

  // Hero block — white paper card on the bone canvas (UI-V2-SPEC §1)
  heroBlock: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    marginTop: theme.spacing[1],
    marginBottom: theme.spacing[2],
    ...theme.shadow[1],
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

  // Figures strip — white stat card (pending / last payout)
  figuresStrip: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    marginBottom: theme.spacing[4],
    gap: theme.spacing[6],
    ...theme.shadow[1],
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

  // Period tabs — iOS-style segmented control track (UI-V2-SPEC §5)
  tabBar: {
    flexDirection: 'row',
    backgroundColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: 3,
    minHeight: 40,
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
  // Fee table lives inside a white card; rows keep their inset hairlines
  // via the card's horizontal padding (UI-V2-SPEC §1).
  breakdownGroup: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    paddingHorizontal: theme.spacing[4],
    ...theme.shadow[1],
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

  // Standalone single-row cards (tax docs, payout account)
  payoutSection: {
    marginTop: theme.spacing[6],
  },
  // Shadow on the outer shell; inner clip keeps the pressed-state bone
  // fill inside the rounded corners (matches the CardSegment pattern).
  singleCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[1],
  },
  singleCardClip: {
    borderRadius: theme.radius.lg,
    overflow: 'hidden',
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
    flex: 1,
  },
  segment: {
    flex: 1,
    minHeight: 34, // 40 track minus 3pt padding top/bottom
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: 9,
  },
  segmentActive: {
    backgroundColor: theme.colors.paper,
    ...theme.shadow[1],
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.1,
  },
  labelActive: {
    color: theme.colors.ink.DEFAULT,
  },
});

const accountRowStyles = StyleSheet.create({
  // Row inside a standalone white card — card shell carries radius/shadow.
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: theme.spacing[4],
    paddingHorizontal: theme.spacing[4],
    minHeight: 56,
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
