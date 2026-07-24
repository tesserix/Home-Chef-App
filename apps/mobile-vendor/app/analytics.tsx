import { useState } from 'react';
import {
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
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { DashboardStatsCard } from '../components/vendor/DashboardStatsCard';
import {
  useChefAnalytics,
  useSubscriptionMetrics,
  useDemandForecast,
  type AnalyticsPeriod,
  type PopularItem,
  type Trend,
} from '../hooks/useChefAnalytics';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const PERIODS: { label: string; value: AnalyticsPeriod }[] = [
  { label: '7 Days', value: '7d' },
  { label: '30 Days', value: '30d' },
  { label: '90 Days', value: '90d' },
];

function inr(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

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
      accessibilityLabel={label}
      hitSlop={7}
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
            {typeof item.percentage === 'number' ? `${item.percentage}% of orders` : ''}
          </Text>
        </View>
      </View>
    </View>
  );
}

// Hand-rolled bar chart (no charting dependency) for a revenue/orders trend.
function TrendBars({ trend }: { trend: Trend }) {
  const max = Math.max(1, ...trend.data);
  const labelEvery = Math.max(1, Math.ceil(trend.data.length / 6));
  return (
    <View style={styles.barsRow}>
      {trend.data.map((v, i) => (
        <View key={`${i}-${trend.labels[i] ?? ''}`} style={styles.barCol}>
          <View style={styles.barTrack}>
            <View style={[styles.barFill, { height: `${Math.max(3, (v / max) * 100)}%` }]} />
          </View>
          <Text style={styles.barLabel} numberOfLines={1}>
            {i % labelEvery === 0 ? (trend.labels[i] ?? '') : ''}
          </Text>
        </View>
      ))}
    </View>
  );
}

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

export default function AnalyticsScreen() {
  const [period, setPeriod] = useState<AnalyticsPeriod>('7d');
  const { data, isLoading, isError, isRefetching, refetch } =
    useChefAnalytics(period);
  const { data: subs } = useSubscriptionMetrics();
  const { data: forecast } = useDemandForecast();

  const summary = data?.summary;
  const hasData =
    (summary?.orders ?? 0) > 0 ||
    (summary?.revenue ?? 0) > 0 ||
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
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            // Inner-View pattern — visual styles on the View, never a
            // function-style array on the Pressable (iOS drops them).
            <View
              style={[
                styles.backButton,
                pressed && Platform.OS === 'ios' && { opacity: 0.6 },
              ]}
            >
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
          <Pressable
            onPress={() => refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry"
            android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.errorButton,
                  pressed && Platform.OS === 'ios' && { opacity: 0.85 },
                ]}
              >
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
          {/* Summary — 2×2 stat grid (#228): revenue, orders, AOV, repeat rate. */}
          <View style={styles.statGrid}>
            <View style={styles.statRow}>
              <DashboardStatsCard
                title="Revenue"
                value={inr(summary?.revenue ?? 0)}
                subtitle={
                  summary && summary.prevRevenue > 0
                    ? `${summary.revenue >= summary.prevRevenue ? '▲' : '▼'} vs last period`
                    : undefined
                }
              />
              <DashboardStatsCard title="Orders" value={summary?.orders ?? 0} />
            </View>
            <View style={styles.statRow}>
              <DashboardStatsCard title="Avg order" value={inr(summary?.aov ?? 0)} />
              <DashboardStatsCard
                title="Repeat customers"
                value={`${summary?.repeatRate ?? 0}%`}
                subtitle="ordered 2+ times"
              />
            </View>
          </View>

          {/* Revenue trend (#228) — hand-rolled bars, no charting dependency. */}
          {(data?.revenueTrends?.data?.length ?? 0) > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionLabel}>REVENUE TREND</Text>
              <View style={styles.chartCard}>
                <TrendBars trend={data!.revenueTrends} />
              </View>
            </View>
          )}

          {/* Tomorrow's demand forecast (#230) */}
          {forecast && forecast.totalExpected > 0 && (
            <View style={styles.section}>
              <Text style={styles.sectionLabel}>TOMORROW'S DEMAND</Text>
              <View style={styles.forecastCard}>
                <Text style={styles.forecastBig}>
                  ~{forecast.totalExpected} meals
                </Text>
                <Text style={styles.forecastSub}>
                  {forecast.subscriptionMeals} subscription ({forecast.subscriptionLunch} lunch ·{' '}
                  {forecast.subscriptionDinner} dinner) + ~{forecast.alaCarteForecast} à-la-carte
                </Text>
                {forecast.likelyDishes.length > 0 && (
                  <Text style={styles.forecastDishes}>
                    Likely: {forecast.likelyDishes.map((d) => `${d.name} (~${d.expected})`).join(', ')}
                  </Text>
                )}
                <Text style={styles.forecastBasis}>{forecast.basis}</Text>
              </View>
            </View>
          )}

          {/* Subscription health (#229) */}
          {subs && (subs.activePlans > 0 || subs.subscribers > 0) && (
            <View style={styles.section}>
              <Text style={styles.sectionLabel}>SUBSCRIPTIONS</Text>
              <View style={styles.statRow}>
                <DashboardStatsCard title="Active plans" value={subs.activePlans} />
                <DashboardStatsCard title="Churn" value={`${subs.churnRate}%`} />
                <DashboardStatsCard
                  title="Adherence"
                  value={`${subs.adherenceRate}%`}
                  subtitle="days delivered"
                />
              </View>
            </View>
          )}

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

  // Summary 2×2 stat grid (#228)
  statGrid: { gap: theme.spacing[3] },
  statRow: { flexDirection: 'row', gap: theme.spacing[3] },

  // Revenue trend bars (#228)
  chartCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  barsRow: { flexDirection: 'row', alignItems: 'flex-end', height: 120, gap: 3 },
  barCol: { flex: 1, alignItems: 'center', height: '100%', justifyContent: 'flex-end' },
  barTrack: { width: '100%', flex: 1, justifyContent: 'flex-end' },
  barFill: {
    width: '70%',
    alignSelf: 'center',
    // Vendor reconciliation: persimmon → ink (charts stay untouched
    // structurally; only the token backing the fill colour moves).
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: 3,
    minHeight: 3,
  },
  barLabel: {
    fontFamily: 'Inter',
    fontSize: 9,
    color: theme.colors.ink.muted,
    marginTop: 4,
  },

  // Demand forecast card (#230) — canvas+cards white surface (spec §1).
  // Was a persimmon-tint highlight; vendor reconciliation drops tint
  // callouts that aren't operational-positive status down to the plain
  // card treatment used elsewhere on this screen.
  forecastCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  forecastBig: { fontFamily: 'Geist-Bold', fontSize: 26, color: theme.colors.ink.DEFAULT, fontVariant: ['tabular-nums'] },
  forecastSub: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft, marginTop: 4, fontVariant: ['tabular-nums'] },
  forecastDishes: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft, marginTop: 6 },
  forecastBasis: { fontFamily: 'Inter', fontSize: 11, color: theme.colors.ink.muted, marginTop: 6 },
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
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
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
