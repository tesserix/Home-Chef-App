import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { useDockClearance } from '../../components/navigation/Dock';
import {
  useVendorPendingOrders,
  useVendorOrderHistory,
  useOrderAction,
  type Order,
} from '../../hooks/useVendorOrders';
import { PendingOrderCard } from '../../components/vendor/PendingOrderCard';
import { UndoSnackbar } from '../../components/vendor/UndoSnackbar';

type ActiveTab = 'new' | 'history';

// Maps order status to its key under the `orders.status` i18n namespace.
const HISTORY_STATUS_KEY: Record<string, string> = {
  delivered: 'delivered',
  picked_up: 'pickedUp',
  cancelled: 'cancelled',
  rejected: 'rejected',
  accepted: 'accepted',
  preparing: 'preparing',
  ready: 'ready',
  pending: 'pending',
};

// Status chip palette per UI-V2-SPEC §2: tint bg + dark text of same hue.
interface StatusChipColors {
  bg: string;
  text: string;
}

const HISTORY_STATUS_CHIP: Record<string, StatusChipColors> = {
  delivered: { bg: theme.colors.mist.DEFAULT, text: theme.colors.diet.veg },
  picked_up: { bg: theme.colors.mist.DEFAULT, text: theme.colors.diet.veg },
  cancelled: {
    bg: theme.colors.destructive.tint,
    text: theme.colors.destructive.DEFAULT,
  },
  rejected: {
    bg: theme.colors.destructive.tint,
    text: theme.colors.destructive.DEFAULT,
  },
  pending: { bg: theme.colors.mist.DEFAULT, text: theme.colors.ink.DEFAULT },
  preparing: { bg: theme.colors.mist.DEFAULT, text: theme.colors.ink.DEFAULT },
  ready: { bg: theme.colors.success.tint, text: theme.colors.success.soft },
  accepted: { bg: theme.colors.info.tint, text: theme.colors.info.DEFAULT },
};

const STATUS_CHIP_FALLBACK: StatusChipColors = {
  bg: theme.colors.mist.DEFAULT,
  text: theme.colors.ink.soft,
};

// Statuses that appear in History (not in Active/Cooking or New).
// A chef-delivery order at `picked_up` means the CHEF is out delivering — still
// active, so it must NOT fall into History until `delivered`. Only 3PL/pickup
// `picked_up` (rider has it, out of the chef's hands) is History.
function isHistoryOrder(o: { status: string; fulfillmentType?: string }): boolean {
  if (o.status === 'delivered' || o.status === 'cancelled' || o.status === 'rejected') {
    return true;
  }
  if (o.status === 'picked_up') return o.fulfillmentType !== 'chef_delivery';
  return false;
}

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
function dateBucketFor(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return 'Unknown';
  const today = new Date();
  const startOfDay = (dt: Date) =>
    new Date(dt.getFullYear(), dt.getMonth(), dt.getDate()).getTime();
  const diffDays = Math.floor(
    (startOfDay(today) - startOfDay(d)) / (1000 * 60 * 60 * 24),
  );
  if (diffDays === 0) return 'orders.today';
  if (diffDays === 1) return 'orders.yesterday';
  return d.toLocaleDateString('en-IN', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
  });
}

// ----- New tab (live pending orders, accept/reject) --------------------------

function NewTab() {
  const { t } = useTranslation();
  const { data, isLoading, refetch } = useVendorPendingOrders();
  const { triggerAction, handleUndo, pendingUndo, isLoading: actionLoading } =
    useOrderAction();
  const orders = data?.orders ?? [];
  const isSurge = orders.length > 3;
  const dockClearance = useDockClearance();

  // Pull-to-refresh gated to user action only. See QueueTab rationale.
  const [isPulling, setIsPulling] = useState(false);
  async function onPullRefresh(): Promise<void> {
    setIsPulling(true);
    try {
      await refetch();
    } finally {
      setIsPulling(false);
    }
  }

  if (isLoading) {
    return (
      <View style={styles.skeletonStack}>
        <Skeleton height={140} style={{ borderRadius: theme.radius.lg }} />
        <Skeleton
          height={140}
          style={{
            borderRadius: theme.radius.lg,
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
        contentContainerStyle={[styles.tabList, { paddingBottom: dockClearance }]}
        ListHeaderComponent={
          isSurge ? (
            <View style={styles.surgeBanner}>
              <View
                style={[
                  styles.statusDot,
                  { backgroundColor: theme.colors.amber.DEFAULT },
                ]}
              />
              <Text style={styles.surgeBannerLabel}>
                {t('orders.ordersAwaiting', { count: orders.length })}
              </Text>
            </View>
          ) : null
        }
        refreshControl={
          <RefreshControl
            refreshing={isPulling}
            onRefresh={onPullRefresh}
            tintColor={theme.colors.ink.DEFAULT}
          />
        }
        renderItem={({ item }) => (
          <PendingOrderCard
            order={item}
            showInstructions
            disabled={actionLoading}
            onOpenDetail={() => router.push(`/orders/${item.id}`)}
            onAccept={() => triggerAction(item.id, 'accepted')}
            onReject={() => triggerAction(item.id, 'rejected')}
          />
        )}
        ItemSeparatorComponent={() => <View style={{ height: theme.spacing[2] }} />}
        ListEmptyComponent={<NewEmpty />}
      />
      <UndoSnackbar pendingUndo={pendingUndo} onUndo={handleUndo} />
    </View>
  );
}

function NewEmpty() {
  const { t } = useTranslation();
  return (
    <View style={styles.emptyBlock}>
      <Text style={styles.emptyHeadline}>{t('orders.queueClear')}</Text>
      <Text style={styles.emptyBody}>{t('orders.queueClearBody')}</Text>
    </View>
  );
}

// ----- History tab (delivered/cancelled/rejected, paginated, date-grouped) ---

interface HistoryListItem {
  type: 'header' | 'order';
  key: string;
  bucket?: string;
  order?: Order;
  first?: boolean;
  last?: boolean;
}

function HistoryTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const { data, isLoading, refetch } = useVendorOrderHistory(page);

  // Filter to past/terminal statuses only — active orders appear in the
  // Active tab instead. Client-side filter avoids an extra API parameter.
  const orders = useMemo(
    () => (data?.orders ?? []).filter(isHistoryOrder),
    [data?.orders],
  );
  const hasMore = data ? page * (data.limit ?? 20) < data.total : false;
  const dockClearance = useDockClearance();

  const [isPulling, setIsPulling] = useState(false);
  async function onPullRefresh(): Promise<void> {
    setIsPulling(true);
    try {
      await refetch();
    } finally {
      setIsPulling(false);
    }
  }

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
      ords.forEach((o, idx) => {
        out.push({
          type: 'order',
          key: o.id,
          order: o,
          first: idx === 0,
          last: idx === ords.length - 1,
        });
      });
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
      contentContainerStyle={[styles.historyList, { paddingBottom: dockClearance }]}
      refreshControl={
        <RefreshControl
          refreshing={isPulling}
          onRefresh={() => {
            setPage(1);
            void onPullRefresh();
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
            {(item.bucket?.startsWith('orders.')
              ? t(item.bucket)
              : (item.bucket ?? '')
            ).toUpperCase()}
          </Text>
        ) : item.order ? (
          <HistoryRow
            order={item.order}
            first={item.first ?? false}
            last={item.last ?? false}
          />
        ) : null
      }
      ListFooterComponent={
        hasMore ? (
          <View style={styles.loadMoreRow}>
            <ActivityIndicator size="small" color={theme.colors.ink.muted} />
          </View>
        ) : null
      }
      ListEmptyComponent={
        <View style={styles.emptyBlock}>
          <Text style={styles.emptyHeadline}>{t('orders.noOrdersYet')}</Text>
          <Text style={styles.emptyBody}>{t('orders.noOrdersBody')}</Text>
        </View>
      }
    />
  );
}

interface HistoryRowProps {
  order: Order;
  first: boolean;
  last: boolean;
}

// Each row is one segment of a white "group card" (UI-V2-SPEC §1/§9).
function HistoryRow({ order, first, last }: HistoryRowProps) {
  const { t } = useTranslation();
  const chip = HISTORY_STATUS_CHIP[order.status] ?? STATUS_CHIP_FALLBACK;
  const statusKey = HISTORY_STATUS_KEY[order.status];
  return (
    <View
      style={[
        historyRowStyles.cardSegment,
        first && historyRowStyles.cardTop,
        last && historyRowStyles.cardBottom,
      ]}
    >
      <View
        style={[
          historyRowStyles.clip,
          first && historyRowStyles.cardTop,
          last && historyRowStyles.cardBottom,
        ]}
      >
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
                style={[historyRowStyles.chip, { backgroundColor: chip.bg }]}
              >
                <Text
                  style={[historyRowStyles.chipLabel, { color: chip.text }]}
                >
                  {statusKey ? t(`orders.status.${statusKey}`) : order.status}
                </Text>
              </View>
              <View style={historyRowStyles.nameBlock}>
                <Text style={historyRowStyles.name} numberOfLines={1}>
                  {order.customerName}
                </Text>
                <Text style={historyRowStyles.meta}>
                  {formatMinutesAgo(order.createdAt)}
                </Text>
              </View>
              <Text style={historyRowStyles.total}>
                ₹{order.total.toFixed(0)}
              </Text>
            </View>
          )}
        </Pressable>
        {!last && <View style={historyRowStyles.separator} />}
      </View>
    </View>
  );
}

// ----- Segmented control ------------------------------------------------------

interface TabLabelProps {
  label: string;
  badge?: number;
  active: boolean;
  onPress: () => void;
}

function TabLabel({ label, badge, active, onPress }: TabLabelProps) {
  return (
    <Pressable
      onPress={onPress}
      style={tabStyles.root}
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
    >
      {/* Inner-View pattern — visual styles live on View to dodge iOS
          function-style style drop. */}
      <View style={[tabStyles.segment, active && tabStyles.segmentActive]}>
        <View style={tabStyles.labelRow}>
          <Text style={[tabStyles.label, active && tabStyles.labelActive]}>
            {label}
          </Text>
          {badge != null && badge > 0 ? (
            <View style={tabStyles.badge}>
              <Text style={tabStyles.badgeLabel}>
                {badge > 9 ? '9+' : String(badge)}
              </Text>
            </View>
          ) : null}
        </View>
      </View>
    </Pressable>
  );
}

// ----- Screen shell -----------------------------------------------------------

export default function OrdersScreen() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<ActiveTab>('new');

  // Badge count for the New segment so the chef knows there are orders waiting
  // without having to switch tabs.
  const { data: pendingData } = useVendorPendingOrders();
  const newCount = pendingData?.orders.length ?? 0;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar */}
      <View style={styles.commandBar}>
        <Text style={styles.commandTitle}>{t('orders.title')}</Text>
      </View>

      {/* Two-segment control: New / History.
          Active/in-progress orders are managed from the Dashboard "In Progress"
          section, which shows the full lifecycle stepper. */}
      <View style={styles.segmentTrack}>
        <TabLabel
          label={t('orders.new')}
          badge={newCount}
          active={activeTab === 'new'}
          onPress={() => setActiveTab('new')}
        />
        <TabLabel
          label={t('orders.history')}
          active={activeTab === 'history'}
          onPress={() => setActiveTab('history')}
        />
      </View>

      {activeTab === 'new' ? <NewTab /> : <HistoryTab />}
    </SafeAreaView>
  );
}

// ----- Styles -----------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },

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

  // Two-segment control track (UI-V2-SPEC §5)
  segmentTrack: {
    flexDirection: 'row',
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[2],
    backgroundColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: 3,
    minHeight: 40,
  },

  // Tab list (New + Active use this)
  tabList: {
    padding: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },

  // Surge banner (reused from dashboard pattern)
  surgeBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.amber.tint,
    marginBottom: theme.spacing[3],
  },
  surgeBannerLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
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
    marginTop: theme.spacing[6],
    marginBottom: theme.spacing[2],
  },

  // Infinite-scroll load-more indicator
  loadMoreRow: {
    alignItems: 'center',
    paddingVertical: theme.spacing[4],
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
    flex: 1,
  },
  segment: {
    flex: 1,
    minHeight: 34,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: 9,
  },
  segmentActive: {
    backgroundColor: theme.colors.paper,
    ...theme.shadow[1],
  },
  labelRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
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
  // Small count badge — ink pill with paper numerals.
  badge: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.full,
    minWidth: 16,
    height: 16,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 3,
  },
  badgeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    color: theme.colors.paper,
    fontVariant: ['tabular-nums'],
    lineHeight: 14,
  },
});

const historyRowStyles = StyleSheet.create({
  cardSegment: {
    backgroundColor: theme.colors.paper,
    ...theme.shadow[1],
  },
  cardTop: {
    borderTopLeftRadius: theme.radius.lg,
    borderTopRightRadius: theme.radius.lg,
  },
  cardBottom: {
    borderBottomLeftRadius: theme.radius.lg,
    borderBottomRightRadius: theme.radius.lg,
  },
  clip: {
    overflow: 'hidden',
    backgroundColor: theme.colors.paper,
  },
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 56,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
  },
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4],
  },
  chip: {
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 3,
    borderRadius: theme.radius.full,
  },
  chipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
  },
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
