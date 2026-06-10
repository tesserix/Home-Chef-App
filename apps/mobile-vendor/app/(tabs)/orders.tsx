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
import { useTranslation } from 'react-i18next';
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
  pending: { bg: theme.colors.amber.tint, text: theme.colors.ink.DEFAULT },
  preparing: { bg: theme.colors.amber.tint, text: theme.colors.ink.DEFAULT },
  ready: { bg: theme.colors.success.tint, text: theme.colors.success.soft },
  accepted: { bg: theme.colors.info.tint, text: theme.colors.info.DEFAULT },
};

const STATUS_CHIP_FALLBACK: StatusChipColors = {
  bg: theme.colors.mist.DEFAULT,
  text: theme.colors.ink.soft,
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
// Returns either an i18n key ('orders.today' / 'orders.yesterday') for the
// relative buckets, or a pre-formatted locale date string for older buckets.
// Render-side decides whether to translate or display verbatim.
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

// ----- Queue tab (live pending orders) -----------------------------------

function QueueTab() {
  const { t } = useTranslation();
  const { data, isLoading, refetch } = useVendorPendingOrders();
  const { triggerAction, handleUndo, pendingUndo, isLoading: actionLoading } =
    useOrderAction();
  const orders = data?.orders ?? [];
  const isSurge = orders.length > 3;

  // Pull-to-refresh spinner gated to USER action only. React Query's
  // `isRefetching` also fires for background refetches (10s polling on
  // this hook) and would leave the spinner permanently visible.
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
        <Skeleton
          height={140}
          style={{ borderRadius: theme.radius.lg }}
        />
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
        contentContainerStyle={styles.queueList}
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
  const { t } = useTranslation();
  return (
    <View style={styles.emptyBlock}>
      <Text style={styles.emptyHeadline}>{t('orders.queueClear')}</Text>
      <Text style={styles.emptyBody}>
        {t('orders.queueClearBody')}
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
  // Position within the date bucket — drives group-card corner radii and
  // inset separators (style-based grouping keeps the flat-list virtualization).
  first?: boolean;
  last?: boolean;
}

function HistoryTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const { data, isLoading, refetch } = useVendorOrderHistory(page);
  const orders = data?.orders ?? [];
  const hasMore = data ? page * (data.limit ?? 20) < data.total : false;

  // User-initiated pull-to-refresh only (see QueueTab for the rationale).
  const [isPulling, setIsPulling] = useState(false);
  async function onPullRefresh(): Promise<void> {
    setIsPulling(true);
    try {
      await refetch();
    } finally {
      setIsPulling(false);
    }
  }

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
      contentContainerStyle={styles.historyList}
      refreshControl={
        <RefreshControl
          refreshing={isPulling}
          onRefresh={() => {
            setPage(1);
            onPullRefresh();
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
      ListEmptyComponent={
        <View style={styles.emptyBlock}>
          <Text style={styles.emptyHeadline}>{t('orders.noOrdersYet')}</Text>
          <Text style={styles.emptyBody}>
            {t('orders.noOrdersBody')}
          </Text>
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
// First-in-bucket gets top radii, last-in-bucket gets bottom radii; the
// shadow lives on an outer wrapper so the inner overflow-hidden clip (needed
// for the pressed-state bone fill to respect the radius) doesn't cut it off.
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
                <Text style={[historyRowStyles.chipLabel, { color: chip.text }]}>
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

// ----- Screen shell -------------------------------------------------------

export default function OrdersScreen() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<ActiveTab>('queue');

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar */}
      <View style={styles.commandBar}>
        <Text style={styles.commandTitle}>{t('orders.title')}</Text>
      </View>

      {/* iOS-style segmented control (UI-V2-SPEC §5) — mist track with a
          raised paper segment for the active tab. */}
      <View style={styles.segmentTrack}>
        <TabLabel
          label={t('orders.queue')}
          active={activeTab === 'queue'}
          onPress={() => setActiveTab('queue')}
        />
        <TabLabel
          label={t('orders.history')}
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
    marginTop: theme.spacing[6], // spacing between section groups (spec §1)
    marginBottom: theme.spacing[2], // gap above the group card
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

const historyRowStyles = StyleSheet.create({
  // Group-card segment shell (UI-V2-SPEC §1) — shadow on the outer layer,
  // overflow clip on the inner layer so the radius survives pressed-state bg.
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
    marginLeft: theme.spacing[4], // inset — aligned to row content
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
