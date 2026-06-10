import { useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
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

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Period = 'week' | 'month' | 'year';

interface PopularItem {
  name: string;
  orders: number;
  // The backend (apps/api/handlers/chefs.go:1089) returns `percentage` —
  // share of the chef's total order count — and does NOT return `revenue`.
  // `revenue` is kept here as optional for forward compatibility if the
  // API later adds it; until then we render the percentage instead.
  percentage?: number;
  revenue?: number;
}

interface DailyRevenue {
  date: string;
  revenue: number;
}

interface AnalyticsResponse {
  period: string;
  totalOrders: number;
  totalRevenue: number;
  popularItems: PopularItem[];
  // dailyRevenue retained for future sparkline once API ships a
  // previousPeriodRevenue baseline and we have ≥8 gap-free data points.
  dailyRevenue: DailyRevenue[];
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

function useAnalytics(period: Period) {
  return useQuery<AnalyticsResponse>({
    queryKey: ['chef', 'analytics', period],
    queryFn: () =>
      api
        .get<AnalyticsResponse>(`/chef/analytics?period=${period}`)
        .then((r) => r.data),
    staleTime: 60_000,
  });
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const PERIODS: { label: string; value: Period }[] = [
  { label: 'Week', value: 'week' },
  { label: 'Month', value: 'month' },
  { label: 'Year', value: 'year' },
];

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

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

interface PopularItemRowProps {
  item: PopularItem;
  rank: number;
  isLast: boolean;
}

function PopularItemRow({ item, rank, isLast }: PopularItemRowProps) {
  return (
    <View style={[rowStyles.root, isLast && rowStyles.rootLast]}>
      <Text style={rowStyles.rank}>#{rank}</Text>
      <Text style={rowStyles.name} numberOfLines={1}>
        {item.name}
      </Text>
      <View style={rowStyles.right}>
        <Text style={rowStyles.orders}>{item.orders} orders</Text>
        <Text style={rowStyles.revenue}>
          {typeof item.revenue === 'number'
            ? `₹${item.revenue.toLocaleString('en-IN')}`
            : typeof item.percentage === 'number'
              ? `${item.percentage}% of orders`
              : ''}
        </Text>
      </View>
    </View>
  );
}

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

export default function AnalyticsScreen() {
  const [period, setPeriod] = useState<Period>('week');
  const { data, isLoading, isError, isRefetching, refetch } =
    useAnalytics(period);

  const hasData =
    (data?.totalOrders ?? 0) > 0 ||
    (data?.totalRevenue ?? 0) > 0 ||
    (data?.popularItems?.length ?? 0) > 0;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar: back chevron left, title, period tabs right */}
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          style={({ pressed }) => [
            styles.backButton,
            pressed && { opacity: 0.6 },
          ]}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
        >
          <ChevronLeft
            size={22}
            color={theme.colors.ink.DEFAULT}
            strokeWidth={2}
          />
        </Pressable>

        <Text style={styles.commandTitle}>Analytics</Text>

        <View style={styles.periodTabs}>
          {PERIODS.map((p) => (
            <TabLabel
              key={p.value}
              label={p.label}
              active={period === p.value}
              onPress={() => setPeriod(p.value)}
            />
          ))}
        </View>
      </View>

      {/* Hairline below command bar — matches Orders screen tabBar rule */}
      <View style={styles.commandBarRule} />

      {/* Body states */}
      {isLoading ? (
        <View style={styles.skeletonStack}>
          {/* Summary strip skeleton */}
          <Skeleton height={36} style={styles.skeletonSummary} />
          {/* Section label skeleton */}
          <Skeleton height={12} style={styles.skeletonSectionLabel} />
          {/* Row skeletons */}
          <Skeleton height={44} style={styles.skeletonRow} />
          <Skeleton height={44} style={styles.skeletonRow} />
          <Skeleton height={44} style={styles.skeletonRow} />
        </View>
      ) : isError ? (
        <View style={styles.errorBlock}>
          <Text style={styles.errorHeadline}>Couldn't load analytics</Text>
          <Text style={styles.errorBody}>
            Check your connection and try again.
          </Text>
          <Pressable
            onPress={() => refetch()}
            style={({ pressed }) => [
              styles.errorButton,
              pressed && { opacity: 0.85 },
            ]}
            accessibilityRole="button"
          >
            <Text style={styles.errorButtonText}>Retry</Text>
          </Pressable>
        </View>
      ) : !hasData ? (
        <View style={styles.emptyBlock}>
          <Text style={styles.emptyHeadline}>Nothing here yet</Text>
          <Text style={styles.emptyBody}>
            Analytics appear after your first completed order.
          </Text>
        </View>
      ) : (
        <ScrollView
          style={styles.scrollView}
          contentContainerStyle={styles.scrollContent}
          showsVerticalScrollIndicator={false}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={() => refetch()}
              tintColor={theme.colors.ink.DEFAULT}
            />
          }
        >
          {/* Summary strip — two tabular figures baseline-aligned,
              hairline divider between them. Same visual grammar as the
              dashboard today-strip so the chef reads them fluently. */}
          <View style={styles.summaryStrip}>
            <Text style={styles.summaryRevenue}>
              ₹{(data?.totalRevenue ?? 0).toLocaleString('en-IN')}
            </Text>
            <Text style={styles.summaryDivider}>|</Text>
            <Text style={styles.summaryOrders}>
              {data?.totalOrders ?? 0}{' '}
              {(data?.totalOrders ?? 0) === 1 ? 'order' : 'orders'}
            </Text>
          </View>

          {/* Popular items section */}
          {(data?.popularItems?.length ?? 0) > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionLabel}>POPULAR ITEMS</Text>
              <View>
                {data!.popularItems.slice(0, 10).map((item, index) => (
                  <PopularItemRow
                    key={item.name}
                    item={item}
                    rank={index + 1}
                    isLast={index === Math.min(data!.popularItems.length, 10) - 1}
                  />
                ))}
              </View>
            </View>
          )}

          {/* Chart placeholder — intentionally empty.
              Render a hairline sparkline here once:
              1. API returns `previousPeriodRevenue` for a delta baseline.
              2. We have ≥8 data points with no zero-revenue gaps.
              3. react-native-svg is added to package.json.
              Until then, the numbers in the summary strip are the chart. */}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },

  // Command bar
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[2],
    gap: theme.spacing[3],
  },
  backButton: {
    minWidth: 28,
    minHeight: 44,
    justifyContent: 'center',
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    flex: 1,
  },
  periodTabs: {
    flexDirection: 'row',
    gap: theme.spacing[4],
    alignItems: 'flex-end',
  },
  commandBarRule: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
  },

  // Summary strip (matches dashboard todayStrip)
  summaryStrip: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: theme.spacing[3],
    paddingTop: theme.spacing[5],
    paddingBottom: theme.spacing[5],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  summaryRevenue: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 30,
    letterSpacing: -0.4,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  summaryDivider: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.mist.strong,
  },
  summaryOrders: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    fontVariant: ['tabular-nums'],
  },

  // Section
  section: {
    marginTop: theme.spacing[6],
  },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[2],
  },

  // Scroll container
  scrollView: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },

  // Loading skeletons
  skeletonStack: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[5],
  },
  skeletonSummary: {
    borderRadius: theme.radius.sm,
    marginBottom: theme.spacing[6],
  },
  skeletonSectionLabel: {
    width: 100,
    borderRadius: theme.radius.sm,
    marginBottom: theme.spacing[3],
  },
  skeletonRow: {
    borderRadius: theme.radius.sm,
    marginBottom: theme.spacing[1],
  },

  // Error state (matches Orders screen pattern)
  errorBlock: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
  },
  errorHeadline: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
    marginBottom: theme.spacing[2],
  },
  errorBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    marginBottom: theme.spacing[6],
    lineHeight: 22,
  },
  errorButton: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorButtonText: {
    color: theme.colors.paper,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
  },

  // Empty state (matches Orders queueEmpty + dashboard quietBlock voice)
  emptyBlock: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[10],
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
    maxWidth: 320,
  },
});

const tabStyles = StyleSheet.create({
  root: {
    paddingTop: theme.spacing[1],
    paddingBottom: theme.spacing[2],
    alignItems: 'center',
    minHeight: 44,
    justifyContent: 'flex-end',
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
    width: '100%',
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
  // Last row drops its bottom hairline so the section doesn't double-rule
  // against whatever follows it.
  rootLast: {
    borderBottomWidth: 0,
  },
  rank: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.1,
    width: 24,
    textAlign: 'left',
  },
  name: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  right: {
    alignItems: 'flex-end',
  },
  orders: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    fontVariant: ['tabular-nums'],
  },
  revenue: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    marginTop: 1,
  },
});
