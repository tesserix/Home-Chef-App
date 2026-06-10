import { useState } from 'react';
import {
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
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
    >
      {/* Inner-View pattern — visual styles live on the View, not the
          Pressable, to dodge the iOS function-style style drop. */}
      <View style={[tabStyles.segment, active && tabStyles.segmentActive]}>
        <Text style={[tabStyles.label, active && tabStyles.labelActive]}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

interface PopularItemRowProps {
  item: PopularItem;
  rank: number;
  isLast: boolean;
}

// One row of the white "group card" (UI-V2-SPEC §1/§9) — inset hairline
// separators (left-padded by the row, hairline on the inner block) and the
// last row drops its rule so the card edge stays clean.
function PopularItemRow({ item, rank, isLast }: PopularItemRowProps) {
  return (
    <View style={rowStyles.root}>
      <View style={[rowStyles.inner, !isLast && rowStyles.innerBorder]}>
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
      {/* Zone A — Command bar: back chevron + title on the bone canvas */}
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
        >
          {({ pressed }) => (
            // Inner-View pattern — visual styles on the View, never a
            // function-style array on the Pressable (iOS drops them).
            <View style={[styles.backButton, pressed && { opacity: 0.6 }]}>
              <ChevronLeft
                size={22}
                color={theme.colors.ink.DEFAULT}
                strokeWidth={2}
              />
            </View>
          )}
        </Pressable>

        <Text style={styles.commandTitle}>Analytics</Text>
      </View>

      {/* Period tabs — iOS-style segmented control (UI-V2-SPEC §5) */}
      <View style={styles.segmentTrack}>
        {PERIODS.map((p) => (
          <TabLabel
            key={p.value}
            label={p.label}
            active={period === p.value}
            onPress={() => setPeriod(p.value)}
          />
        ))}
      </View>

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
          <Pressable onPress={() => refetch()} accessibilityRole="button">
            {({ pressed }) => (
              <View style={[styles.errorButton, pressed && { opacity: 0.85 }]}>
                <Text style={styles.errorButtonText}>Retry</Text>
              </View>
            )}
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
          {/* Hero — revenue + orders on a white paper card (UI-V2-SPEC §1).
              Same visual grammar as the dashboard TODAY stat card so the
              chef reads the figures fluently. */}
          <View style={styles.summaryCard}>
            <View style={styles.summaryCol}>
              <Text style={styles.summaryRevenue}>
                ₹{(data?.totalRevenue ?? 0).toLocaleString('en-IN')}
              </Text>
              <Text style={styles.summaryColLabel}>Revenue</Text>
            </View>
            <View style={styles.summaryCol}>
              <Text style={styles.summaryOrders}>
                {data?.totalOrders ?? 0}
              </Text>
              <Text style={styles.summaryColLabel}>
                {(data?.totalOrders ?? 0) === 1 ? 'Order' : 'Orders'}
              </Text>
            </View>
          </View>

          {/* Popular items section — white group card */}
          {(data?.popularItems?.length ?? 0) > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionLabel}>POPULAR ITEMS</Text>
              <View style={styles.groupCard}>
                <View style={styles.groupCardInner}>
                  {data!.popularItems.slice(0, 10).map((item, index) => (
                    <PopularItemRow
                      key={item.name}
                      item={item}
                      rank={index + 1}
                      isLast={
                        index === Math.min(data!.popularItems.length, 10) - 1
                      }
                    />
                  ))}
                </View>
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
  root: { flex: 1, backgroundColor: theme.colors.bone },

  // Command bar — back chevron + title, on the bone canvas
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
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

  // Segmented control track (UI-V2-SPEC §5)
  segmentTrack: {
    flexDirection: 'row',
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[2],
    backgroundColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: 3,
    minHeight: 40,
  },

  // Hero — white stat card (UI-V2-SPEC §1)
  summaryCard: {
    flexDirection: 'row',
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    marginTop: theme.spacing[2],
    ...theme.shadow[1],
  },
  summaryCol: {
    flex: 1,
    alignItems: 'flex-start',
  },
  summaryRevenue: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.4,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  summaryOrders: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.4,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  summaryColLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[1],
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
  // White group card for the popular-items rows (UI-V2-SPEC §1)
  groupCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[1],
  },
  groupCardInner: {
    borderRadius: theme.radius.lg,
    overflow: 'hidden',
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

const rowStyles = StyleSheet.create({
  // Outer row carries only the inset left padding; the hairline lives on
  // the inner block so the separator reads as inset (UI-V2-SPEC §1).
  root: {
    paddingLeft: theme.spacing[4],
  },
  inner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 56,
    paddingVertical: theme.spacing[2],
    paddingRight: theme.spacing[4],
  },
  innerBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
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
