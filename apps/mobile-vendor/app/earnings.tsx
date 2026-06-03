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

// ----- Types ----------------------------------------------------------------

interface BankAccount {
  bankName: string;
  accountNumber: string;
  ifsc: string;
}

interface LastPayout {
  amount: number;
  date: string;
}

interface EarningsHistoryEntry {
  week: string;
  amount: number;
}

interface PayoutResponse {
  bankAccount: BankAccount | null;
  upiId: string | null;
  totalEarnings: number;
  pendingPayout: number;
  lastPayout: LastPayout | null;
  earningsHistory: EarningsHistoryEntry[];
}

type Period = 'week' | 'month' | 'all';

// ----- Hook -----------------------------------------------------------------

function useEarnings() {
  return useQuery<PayoutResponse>({
    queryKey: ['chef', 'payout'],
    queryFn: () => api.get<PayoutResponse>('/chef/payout').then((r) => r.data),
    staleTime: 60_000,
  });
}

// ----- Helpers --------------------------------------------------------------

function maskAccount(accountNumber: string): string {
  if (accountNumber.length <= 4) return accountNumber;
  return '****' + accountNumber.slice(-4);
}

// Filter `earningsHistory` client-side so we don't need a refetch on tab
// change. The API returns all history; we slice by period here.
function filterByPeriod(
  entries: EarningsHistoryEntry[],
  period: Period,
): EarningsHistoryEntry[] {
  if (period === 'all') return entries;
  const limit = period === 'week' ? 1 : 4;
  return entries.slice(0, limit);
}

// Sum amounts for a filtered set of entries.
function sumEntries(entries: EarningsHistoryEntry[]): number {
  return entries.reduce((acc, e) => acc + e.amount, 0);
}

// Format a week label like "25 May – 31 May" from the API's `week` string
// (which may already be formatted, e.g. "25 May – 31 May" or a date-range ISO).
// We pass it through as-is and let the API own the label format.
function formatWeekLabel(week: string): string {
  return week;
}

// ----- Sub-components -------------------------------------------------------

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
      <View
        style={[tabStyles.indicator, active && tabStyles.indicatorActive]}
      />
    </Pressable>
  );
}

interface HistoryRowProps {
  entry: EarningsHistoryEntry;
  isPaid: boolean;
}

function HistoryRow({ entry, isPaid }: HistoryRowProps) {
  return (
    <View style={rowStyles.root}>
      {/* Status dot: persimmon = paid (the only accent moment on this screen) */}
      <View
        style={[
          rowStyles.dot,
          {
            backgroundColor: isPaid
              ? theme.colors.herb.DEFAULT
              : theme.colors.ink.muted,
          },
        ]}
      />
      <View style={rowStyles.labelBlock}>
        <Text style={rowStyles.week} numberOfLines={1}>
          {formatWeekLabel(entry.week)}
        </Text>
        <Text style={rowStyles.status}>{isPaid ? 'Paid' : 'Pending'}</Text>
      </View>
      <Text style={rowStyles.amount}>
        ₹{entry.amount.toLocaleString('en-IN')}
      </Text>
    </View>
  );
}

// Render the payout account as a single hairline row (no card).
// Tappable to navigate to payout account settings screen.
interface PayoutAccountRowProps {
  bankAccount: BankAccount | null;
  upiId: string | null;
}

function PayoutAccountRow({ bankAccount, upiId }: PayoutAccountRowProps) {
  const label = bankAccount
    ? `${bankAccount.bankName} ****${bankAccount.accountNumber.slice(-4)}`
    : upiId
      ? upiId
      : 'No payout account';

  const sublabel = bankAccount
    ? bankAccount.ifsc
    : upiId
      ? 'UPI'
      : 'Tap to add a bank account or UPI ID';

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
            {sublabel ? (
              <Text style={accountRowStyles.sub} numberOfLines={1}>
                {sublabel}
              </Text>
            ) : null}
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

// ----- List item type -------------------------------------------------------

type ListItem =
  | { type: 'hero' }
  | { type: 'figures' }
  | { type: 'tabBar' }
  | { type: 'historyHeader' }
  | { type: 'entry'; entry: EarningsHistoryEntry; isPaid: boolean }
  | { type: 'payoutRow' }
  | { type: 'empty' };

// ----- Screen ---------------------------------------------------------------

export default function EarningsScreen() {
  const [period, setPeriod] = useState<Period>('week');
  const { data, isLoading, isError, refetch, isRefetching } = useEarnings();

  const filteredHistory = useMemo(
    () => filterByPeriod(data?.earningsHistory ?? [], period),
    [data?.earningsHistory, period],
  );

  const periodTotal = useMemo(
    () => sumEntries(filteredHistory),
    [filteredHistory],
  );

  // The most recent entry is "pending" (current week hasn't been paid out yet).
  // All others are considered paid. This is a reasonable heuristic until the
  // API exposes a per-entry `status` field.
  const listItems = useMemo<ListItem[]>(() => {
    const items: ListItem[] = [
      { type: 'hero' },
      { type: 'figures' },
      { type: 'tabBar' },
    ];
    if (filteredHistory.length === 0) {
      items.push({ type: 'empty' });
    } else {
      items.push({ type: 'historyHeader' });
      filteredHistory.forEach((entry, idx) => {
        items.push({ type: 'entry', entry, isPaid: idx > 0 });
      });
    }
    items.push({ type: 'payoutRow' });
    return items;
  }, [filteredHistory]);

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
          onPress={() => refetch()}
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
            <Text style={styles.heroAmount}>
              ₹{periodTotal.toLocaleString('en-IN')}
            </Text>
          </View>
        );

      case 'figures':
        return (
          <View style={styles.figuresStrip}>
            <View style={styles.figureItem}>
              <Text style={styles.figureLabel}>PENDING</Text>
              <Text style={styles.figureValue}>
                ₹{(data?.pendingPayout ?? 0).toLocaleString('en-IN')}
              </Text>
            </View>
            <View style={styles.figureDivider} />
            {data?.lastPayout ? (
              <View style={styles.figureItem}>
                <Text style={styles.figureLabel}>LAST PAYOUT</Text>
                <Text style={styles.figureValue}>
                  ₹{data.lastPayout.amount.toLocaleString('en-IN')}
                </Text>
                <Text style={styles.figureSub}>
                  {new Date(data.lastPayout.date).toLocaleDateString('en-IN', {
                    day: 'numeric',
                    month: 'short',
                  })}
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

      case 'historyHeader':
        return (
          <Text style={styles.dateHeader}>WEEKLY BREAKDOWN</Text>
        );

      case 'entry':
        return <HistoryRow entry={item.entry} isPaid={item.isPaid} />;

      case 'payoutRow':
        return (
          <View style={styles.payoutSection}>
            <PayoutAccountRow
              bankAccount={data?.bankAccount ?? null}
              upiId={data?.upiId ?? null}
            />
          </View>
        );

      case 'empty':
        return (
          <View style={styles.emptyBlock}>
            <Text style={styles.emptyHeadline}>No history yet</Text>
            <Text style={styles.emptyBody}>
              Completed orders will appear here once your first week closes.
            </Text>
          </View>
        );

      default:
        return null;
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Command bar — back button left, title center-left. No bg-bone ribbon. */}
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
        keyExtractor={(item, idx) =>
          item.type === 'entry' ? item.entry.week : `${item.type}-${idx}`
        }
        contentContainerStyle={styles.listContent}
        renderItem={renderItem}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
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

  // Command bar — same pattern as orders.tsx, no bg-bone ribbon
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

  // List content padding
  listContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },

  // Hero block — period total is the single commanding number
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

  // Figures strip — pending + last payout, hairline-divided
  figuresStrip: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    paddingTop: theme.spacing[4],
    paddingBottom: theme.spacing[4],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    gap: theme.spacing[6],
  },
  figureItem: {
    flex: 1,
  },
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

  // Tab bar — underline tabs, same as orders.tsx
  tabBar: {
    flexDirection: 'row',
    gap: theme.spacing[6],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
    marginBottom: 0,
  },

  // Date header for history section
  dateHeader: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[4],
    marginBottom: theme.spacing[2],
  },

  // Payout account section separator
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

const rowStyles = StyleSheet.create({
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 44,
    paddingVertical: theme.spacing[2],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  dot: { width: 8, height: 8, borderRadius: 4 },
  labelBlock: { flex: 1 },
  week: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  status: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 1,
  },
  amount: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
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
