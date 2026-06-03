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
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import {
  useVendorPendingOrders,
  useVendorOrderHistory,
  useOrderAction,
  type Order,
} from '../../hooks/useVendorOrders';
import { PendingOrderCard } from '../../components/vendor/PendingOrderCard';
import { UndoSnackbar } from '../../components/vendor/UndoSnackbar';

type ActiveTab = 'queue' | 'history';

const HISTORY_STATUS_LABEL: Record<string, string> = {
  delivered: 'Delivered',
  picked_up: 'Picked up',
  cancelled: 'Cancelled',
  rejected: 'Rejected',
  accepted: 'Accepted',
  preparing: 'Preparing',
  ready: 'Ready',
  pending: 'Pending',
};

const HISTORY_STATUS_DOT: Record<string, string> = {
  delivered: theme.colors.diet.veg, // green — happy path
  picked_up: theme.colors.diet.veg,
  cancelled: theme.colors.destructive.DEFAULT,
  rejected: theme.colors.destructive.DEFAULT,
  pending: theme.colors.amber.DEFAULT,
  accepted: theme.colors.info.DEFAULT,
  preparing: theme.colors.amber.DEFAULT,
  ready: theme.colors.herb.DEFAULT,
};

function formatMinutesAgo(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return '';
  const mins = Math.max(0, Math.floor((Date.now() - t) / 60_000));
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

// Bucket history orders into Today / Yesterday / weekday-date headers.
// Server-side `dateLabel` is a follow-up backend ask — until then we do
// it client-side, accepting that page boundaries can produce duplicate
// headers on infinite scroll. Acceptable for now; documented in the
// backend asks list.
function dateBucketFor(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return 'Unknown';
  const today = new Date();
  const startOfDay = (dt: Date) =>
    new Date(dt.getFullYear(), dt.getMonth(), dt.getDate()).getTime();
  const diffDays = Math.floor(
    (startOfDay(today) - startOfDay(d)) / (1000 * 60 * 60 * 24),
  );
  if (diffDays === 0) return 'Today';
  if (diffDays === 1) return 'Yesterday';
  return d.toLocaleDateString('en-IN', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

// ----- Queue tab (live pending orders) -----------------------------------

function QueueTab() {
  const { data, isLoading, isRefetching, refetch } = useVendorPendingOrders();
  const { triggerAction, handleUndo, pendingUndo, isLoading: actionLoading } =
    useOrderAction();
  const orders = data?.orders ?? [];
  const isSurge = orders.length > 3;

  if (isLoading) {
    return (
      <View style={styles.skeletonStack}>
        <Skeleton
          height={140}
          style={{ borderRadius: theme.radius.DEFAULT }}
        />
        <Skeleton
          height={140}
          style={{
            borderRadius: theme.radius.DEFAULT,
            marginTop: theme.spacing[2],
          }}
        />
      </View>
    );
  }

  return (
    <View style={{ flex: 1 }}>
      <FlatList
        data={orders}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.queueList}
        ListHeaderComponent={
          isSurge ? (
            <View style={styles.surgeBanner}>
              <View
                style={[
                  styles.statusDot,
                  { backgroundColor: theme.colors.herb.DEFAULT },
                ]}
              />
              <Text style={styles.surgeBannerLabel}>
                {orders.length} orders awaiting acceptance
              </Text>
            </View>
          ) : null
        }
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
            tintColor={theme.colors.ink.DEFAULT}
          />
        }
        renderItem={({ item }) => (
          <View style={styles.queueCardWrap}>
            <PendingOrderCard
              order={item}
              showInstructions
              disabled={actionLoading}
              onOpenDetail={() => router.push(`/orders/${item.id}`)}
              onAccept={() => triggerAction(item.id, 'accepted')}
              onReject={() => triggerAction(item.id, 'rejected')}
            />
          </View>
        )}
        ListEmptyComponent={<QueueEmpty />}
      />
      <UndoSnackbar pendingUndo={pendingUndo} onUndo={handleUndo} />
    </View>
  );
}

function QueueEmpty() {
  // Reuse `lastOrderIso` style messaging from the dashboard's quiet block.
  // The hook doesn't return last-known data here, so the copy is generic.
  return (
    <View style={styles.emptyBlock}>
      <Text style={styles.emptyHeadline}>Queue is clear</Text>
      <Text style={styles.emptyBody}>
        New orders arrive here automatically. We poll every 10 seconds.
      </Text>
    </View>
  );
}

// ----- History tab (paginated, grouped by date) --------------------------

interface HistoryListItem {
  type: 'header' | 'order';
  key: string;
  bucket?: string;
  order?: Order;
}

function HistoryTab() {
  const [page, setPage] = useState(1);
  const { data, isLoading, isRefetching, refetch } =
    useVendorOrderHistory(page);
  const orders = data?.orders ?? [];
  const hasMore = data ? page * (data.limit ?? 20) < data.total : false;

  // Group orders into a flat list of [header, order, order, header, ...]
  // entries. Sorted by createdAt desc within each bucket; buckets ordered
  // by recency. Pure derivation — no mutation.
  const listItems = useMemo<HistoryListItem[]>(() => {
    const groups = new Map<string, Order[]>();
    for (const o of orders) {
      const bucket = dateBucketFor(o.createdAt);
      const list = groups.get(bucket);
      if (list) list.push(o);
      else groups.set(bucket, [o]);
    }
    const out: HistoryListItem[] = [];
    for (const [bucket, ords] of groups) {
      out.push({ type: 'header', key: `h-${bucket}`, bucket });
      for (const o of ords) {
        out.push({ type: 'order', key: o.id, order: o });
      }
    }
    return out;
  }, [orders]);

  if (isLoading) {
    return (
      <View style={styles.skeletonStack}>
        <Skeleton height={20} style={{ width: 120, marginBottom: 12 }} />
        <Skeleton height={56} style={{ marginBottom: 8 }} />
        <Skeleton height={56} style={{ marginBottom: 8 }} />
        <Skeleton height={56} />
      </View>
    );
  }

  return (
    <FlatList
      data={listItems}
      keyExtractor={(item) => item.key}
      contentContainerStyle={styles.historyList}
      refreshControl={
        <RefreshControl
          refreshing={isRefetching}
          onRefresh={() => {
            setPage(1);
            refetch();
          }}
          tintColor={theme.colors.ink.DEFAULT}
        />
      }
      onEndReachedThreshold={0.3}
      onEndReached={() => {
        if (hasMore) setPage((p) => p + 1);
      }}
      renderItem={({ item }) =>
        item.type === 'header' ? (
          <Text style={styles.dateHeader}>
            {(item.bucket ?? '').toUpperCase()}
          </Text>
        ) : item.order ? (
          <HistoryRow order={item.order} />
        ) : null
      }
      ListEmptyComponent={
        <View style={styles.emptyBlock}>
          <Text style={styles.emptyHeadline}>No orders yet</Text>
          <Text style={styles.emptyBody}>
            Once customers start ordering, you'll see them here.
          </Text>
        </View>
      }
    />
  );
}

interface HistoryRowProps {
  order: Order;
}

function HistoryRow({ order }: HistoryRowProps) {
  return (
    <Pressable
      onPress={() => router.push(`/orders/${order.id}`)}
      accessibilityRole="button"
      accessibilityLabel={`Open order details for ${order.customerName}`}
    >
      {({ pressed }) => (
        // Inner-View pattern — keeps flex layout under iOS Pressable bug.
        <View
          style={[
            historyRowStyles.root,
            pressed && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View
            style={[
              historyRowStyles.dot,
              {
                backgroundColor:
                  HISTORY_STATUS_DOT[order.status] ?? theme.colors.ink.muted,
              },
            ]}
          />
          <View style={historyRowStyles.nameBlock}>
            <Text style={historyRowStyles.name} numberOfLines={1}>
              {order.customerName}
            </Text>
            <Text style={historyRowStyles.meta}>
              {HISTORY_STATUS_LABEL[order.status] ?? order.status} ·{' '}
              {formatMinutesAgo(order.createdAt)}
            </Text>
          </View>
          <Text style={historyRowStyles.total}>₹{order.total.toFixed(0)}</Text>
        </View>
      )}
    </Pressable>
  );
}

// ----- Screen shell -------------------------------------------------------

export default function OrdersScreen() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('queue');

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar */}
      <View style={styles.commandBar}>
        <Text style={styles.commandTitle}>Orders</Text>
      </View>

      {/* Bare-text segmented control — two labels with an ink underline
          on the active one. Lower visual weight than the prior bg-mist
          pill, which is the right call for a control set once per session. */}
      <View style={styles.tabBar}>
        <TabLabel
          label="Queue"
          active={activeTab === 'queue'}
          onPress={() => setActiveTab('queue')}
        />
        <TabLabel
          label="History"
          active={activeTab === 'history'}
          onPress={() => setActiveTab('history')}
        />
      </View>

      {activeTab === 'queue' ? <QueueTab /> : <HistoryTab />}
    </SafeAreaView>
  );
}

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

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },

  // Zone A — Command bar (matches dashboard)
  commandBar: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  // Tab bar
  tabBar: {
    flexDirection: 'row',
    paddingHorizontal: theme.spacing[4],
    gap: theme.spacing[6],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  // Queue list
  queueList: {
    padding: theme.spacing[4],
    paddingBottom: theme.spacing[10],
    gap: theme.spacing[2],
  },
  queueCardWrap: {
    marginBottom: 0, // FlatList gap handles spacing
  },

  // Surge banner (reused from dashboard pattern)
  surgeBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.DEFAULT,
    backgroundColor: theme.colors.herb.tint,
    marginBottom: theme.spacing[3],
  },
  surgeBannerLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.soft,
    letterSpacing: 0.1,
  },
  statusDot: { width: 8, height: 8, borderRadius: 4 },

  // Skeletons + History
  skeletonStack: {
    padding: theme.spacing[4],
  },
  historyList: {
    padding: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },
  dateHeader: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[4],
    marginBottom: theme.spacing[2],
  },

  // Empty states
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

const historyRowStyles = StyleSheet.create({
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
  nameBlock: { flex: 1 },
  name: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  meta: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 1,
  },
  total: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
});
