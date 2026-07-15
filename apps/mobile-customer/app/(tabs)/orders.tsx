import React, { useCallback, useEffect, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { ShoppingBag } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useDockClearance } from '../../components/navigation/Dock';
import { ScreenTitle } from '../../components/shared/ScreenTitle';
import { useOrders } from '../../hooks/useOrderHistory';
import { OrderCard } from '../../components/orders/OrderCard';
import type { Order } from '../../types/customer';

// ─── Types ───────────────────────────────────────────────────────────────────

type StatusFilter = 'all' | 'active' | 'delivered' | 'cancelled';

const STATUS_FILTERS: { key: StatusFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'active', label: 'Active' },
  { key: 'delivered', label: 'Delivered' },
  { key: 'cancelled', label: 'Cancelled' },
];

// Maps filter key → API status param. 'active' covers multiple statuses;
// the backend handles it as a group filter. No business logic change.
const STATUS_MAP: Record<StatusFilter, string | undefined> = {
  all: undefined,
  active: 'active',
  delivered: 'delivered',
  cancelled: 'cancelled',
};

const PAGE_LIMIT = 20;

// ─── Loading skeleton — matches OrderCard proportions ────────────────────────

function SkeletonCard() {
  return (
    <View style={skeletonStyles.card}>
      <View style={skeletonStyles.topRow}>
        <View style={skeletonStyles.col}>
          <View style={[skeletonStyles.line, { width: '60%', height: 14 }]} />
          <View style={[skeletonStyles.line, { width: '35%', height: 11 }]} />
        </View>
        <View style={[skeletonStyles.chip, { width: 72, height: 22 }]} />
      </View>
      <View style={skeletonStyles.hairline} />
      <View style={skeletonStyles.bottomRow}>
        <View style={[skeletonStyles.line, { width: '50%', height: 12 }]} />
        <View style={[skeletonStyles.line, { width: '18%', height: 14 }]} />
      </View>
    </View>
  );
}

const skeletonStyles = StyleSheet.create({
  card: {
    marginHorizontal: 16,
    marginVertical: 5,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    paddingHorizontal: 16,
    paddingVertical: 14,
    backgroundColor: customerColors.canvas,
  },
  topRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: 10,
  },
  col: { flex: 1, gap: 6 },
  line: {
    borderRadius: 4,
    backgroundColor: customerColors.hairline,
  },
  chip: {
    borderRadius: 9999,
    backgroundColor: customerColors.surface.soft,
  },
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginVertical: 12,
  },
  bottomRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
});

// ─── Empty state — mirrors favorites empty state pattern exactly ──────────────

function EmptyState({ filter }: { filter: StatusFilter }) {
  const router = useRouter();

  const title =
    filter === 'all'
      ? 'No orders yet'
      : filter === 'active'
        ? 'No active orders'
        : filter === 'delivered'
          ? 'No delivered orders'
          : 'No cancelled orders';

  const body =
    filter === 'all'
      ? 'Browse home chefs and place your first order.'
      : 'Try switching to a different filter.';

  return (
    <View style={emptyStyles.container}>
      {/* Surface-soft icon circle — matches favorites empty state */}
      <View style={emptyStyles.iconCircle}>
        <ShoppingBag size={34} color={customerColors.charcoal.soft} />
      </View>
      <View style={emptyStyles.textGroup}>
        <Text style={emptyStyles.title}>{title}</Text>
        <Text style={emptyStyles.body}>{body}</Text>
      </View>
      {/* Coral "Browse chefs" CTA — visual styles on inner View per iOS bug */}
      {filter === 'all' && (
        <Pressable
          onPress={() => router.push('/(tabs)')}
          accessibilityRole="button"
          accessibilityLabel="Browse chefs"
        >
          <View style={emptyStyles.cta}>
            <Text style={emptyStyles.ctaLabel}>Browse chefs</Text>
          </View>
        </Pressable>
      )}
    </View>
  );
}

const emptyStyles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 32,
    paddingTop: 64,
    gap: 16,
  },
  iconCircle: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
  textGroup: {
    alignItems: 'center',
    gap: 8,
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
    textAlign: 'center',
    letterSpacing: -0.2,
  },
  body: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 20,
  },
  cta: {
    // Primary coral button — radius 8, min-height 52, Inter-SemiBold
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 32,
    paddingVertical: 14,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 8,
  },
  ctaLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.canvas,
  },
});

// ─── Main screen ─────────────────────────────────────────────────────────────

export default function OrdersScreen() {
  const dockClearance = useDockClearance();
  const [activeFilter, setActiveFilter] = useState<StatusFilter>('all');
  const [page, setPage] = useState(1);
  const [allOrders, setAllOrders] = useState<Order[]>([]);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const params = {
    page,
    limit: PAGE_LIMIT,
    status: STATUS_MAP[activeFilter],
  };

  const { data, isLoading, isRefetching, refetch } = useOrders(params);

  // Accumulate pages into a de-duped local list (immutable merge)
  useEffect(() => {
    if (data?.data) {
      if (page === 1) {
        setAllOrders(data.data);
      } else {
        setAllOrders((prev) => {
          const existingIds = new Set(prev.map((o) => o.id));
          const newItems = data.data.filter((o) => !existingIds.has(o.id));
          return [...prev, ...newItems];
        });
        setIsLoadingMore(false);
      }
    }
  }, [data, page]);

  function handleFilterChange(filter: StatusFilter) {
    setActiveFilter(filter);
    setPage(1);
    setAllOrders([]);
  }

  const handleRefresh = useCallback(() => {
    setPage(1);
    setAllOrders([]);
    void refetch();
  }, [refetch]);

  function handleLoadMore() {
    if (!data || isLoadingMore) return;
    const total = data.meta?.total ?? 0;
    if (allOrders.length < total) {
      setIsLoadingMore(true);
      setPage((prev) => prev + 1);
    }
  }

  // ── Filter chip row — Airbnb charcoal-underline style from home screen ──
  // Selected = charcoal text + 2px charcoal bottom underline; unselected =
  // charcoal-soft, no fill, no border. Matches the cuisine category bar.
  const renderFilterBar = () => (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={styles.filterRowContent}
      style={styles.filterRow}
      accessibilityRole="tablist"
    >
      {STATUS_FILTERS.map((f) => {
        const isActive = activeFilter === f.key;
        return (
          <Pressable
            key={f.key}
            onPress={() => handleFilterChange(f.key)}
            accessibilityRole="tab"
            accessibilityState={{ selected: isActive }}
            accessibilityLabel={`Filter orders: ${f.label}`}
          >
            <View style={[styles.filterChip, isActive && styles.filterChipActive]}>
              <Text
                style={[
                  styles.filterChipLabel,
                  isActive ? styles.filterChipLabelActive : styles.filterChipLabelDefault,
                ]}
              >
                {f.label}
              </Text>
            </View>
          </Pressable>
        );
      })}
    </ScrollView>
  );

  // ── Loading skeleton (first page only) ──
  if (isLoading && page === 1) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <ScreenTitle title="Orders" />
        {/* Filter bar skeleton */}
        {renderFilterBar()}
        {/* Skeleton cards */}
        <View style={{ paddingTop: 8 }}>
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <ScreenTitle title="Orders" />

      {/* ── Airbnb-style filter chip row ── */}
      {renderFilterBar()}

      {/* ── Order list ── */}
      <FlatList<Order>
        style={styles.list}
        data={allOrders}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <OrderCard order={item} />}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching && page === 1}
            onRefresh={handleRefresh}
            tintColor={customerColors.coral.DEFAULT}
          />
        }
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.3}
        contentContainerStyle={
          allOrders.length === 0
            ? styles.emptyContent
            : [styles.listContent, { paddingBottom: dockClearance }]
        }
        ListEmptyComponent={<EmptyState filter={activeFilter} />}
        ListFooterComponent={
          isLoadingMore ? (
            <ActivityIndicator
              size="small"
              color={customerColors.coral.DEFAULT}
              style={styles.footerLoader}
            />
          ) : null
        }
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: customerColors.canvas,
  },

  // ── Geist-Bold "Orders" header — matches favorites pattern (px-4, pt-3, pb-2)
  // ── Airbnb category-bar style filter chips ──
  // No fill, no pill border. Selected = charcoal text + 2px charcoal underline.
  filterRow: {
    // No background; content sits on white canvas
  },
  filterRowContent: {
    paddingHorizontal: 16,
    paddingBottom: 4,
    gap: 0,
  },
  filterChip: {
    paddingHorizontal: 14,
    paddingVertical: 10,
    // No background, no border — just text + optional underline
  },
  filterChipActive: {
    borderBottomWidth: 2,
    borderBottomColor: customerColors.charcoal.DEFAULT,
  },
  filterChipLabel: {
    fontFamily: 'Inter',
    fontSize: 14,
  },
  filterChipLabelActive: {
    color: customerColors.charcoal.DEFAULT,
    fontFamily: 'Inter-SemiBold',
  },
  filterChipLabelDefault: {
    color: customerColors.charcoal.soft,
  },

  // ── List layout ──
  // flex:1 so the FlatList fills the viewport and scrolls. Without it the list
  // is content-sized: short filters (Active/Delivered) fit and look fine, but
  // long ones (All/Cancelled) overflow and can't scroll — so they truncate and
  // onEndReached never fires (no load-more).
  list: {
    flex: 1,
  },
  listContent: {
    paddingTop: 8,
    paddingBottom: 32,
  },
  emptyContent: {
    flexGrow: 1,
  },
  footerLoader: {
    paddingVertical: 16,
  },
});
