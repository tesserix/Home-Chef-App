// Customer notification center. Renders the shared notification feed
// (@homechef/mobile-shared/hooks) with the customer's coral design system, keeps
// it live over the WebSocket, and deep-links each notification to the thing it's
// about (an order, a chef, a meal plan).

import { useCallback } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, Stack } from 'expo-router';
import {
  ArrowLeft,
  BadgeCheck,
  Bell,
  CalendarDays,
  Package,
  Star,
  Truck,
  Wallet,
  type LucideIcon,
} from 'lucide-react-native';
import { EmptyState } from '@homechef/mobile-shared';
import {
  useNotificationList,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  useNotificationSocket,
  parseNotificationData,
  type AppNotification,
} from '@homechef/mobile-shared/hooks';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useAuthStore } from '../store/auth-store';
import { api } from '../lib/api';

function relativeTime(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return '';
  const mins = Math.floor((Date.now() - then) / 60_000);
  if (mins < 1) return 'now';
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h`;
  const days = Math.floor(hrs / 24);
  if (days < 7) return `${days}d`;
  return new Date(iso).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
}

/** A leading icon per notification kind, so the feed scans at a glance. */
function iconFor(n: AppNotification): LucideIcon {
  const t = `${n.type} ${n.title}`.toLowerCase();
  if (t.includes('review') || t.includes('rating') || t.includes('star')) return Star;
  if (t.includes('meal') || t.includes('tiffin') || t.includes('plan')) return CalendarDays;
  if (t.includes('payout') || t.includes('payment') || t.includes('refund') || t.includes('wallet'))
    return Wallet;
  if (t.includes('approv') || t.includes('verif')) return BadgeCheck;
  if (t.includes('deliver') || t.includes('way')) return Truck;
  if (t.includes('order')) return Package;
  return Bell;
}

/** Route a tapped notification to the screen it references. */
function destinationFor(n: AppNotification): string | null {
  const d = parseNotificationData(n);
  const orderId = d.order_id ?? d.orderId;
  if (orderId) return `/order/${orderId}`;
  const planId = d.meal_plan_id ?? d.mealPlanId ?? d.planId;
  if (planId) return `/meal-plans/${planId}`;
  const chefId = d.chef_id ?? d.chefId;
  if (chefId) return `/chef/${chefId}`;
  return null;
}

export default function NotificationsScreen() {
  const { data: notifications = [], isLoading, refetch, isRefetching } =
    useNotificationList(api);
  const markRead = useMarkNotificationRead(api);
  const markAll = useMarkAllNotificationsRead(api);

  useNotificationSocket({
    apiBaseUrl: process.env.EXPO_PUBLIC_API_URL,
    getToken: () => useAuthStore.getState().accessToken,
  });

  const hasUnread = notifications.some((n) => !n.isRead);

  const onPressRow = useCallback(
    (n: AppNotification) => {
      if (!n.isRead) markRead.mutate(n.id);
      const dest = destinationFor(n);
      if (dest) router.push(dest as never);
    },
    [markRead],
  );

  const renderItem = useCallback(
    ({ item }: { item: AppNotification }) => {
      const Icon = iconFor(item);
      return (
        <Pressable
          onPress={() => onPressRow(item)}
          accessibilityRole="button"
          accessibilityLabel={`${item.title}. ${item.message}${item.isRead ? '' : '. Unread'}`}
          android_ripple={{ color: `${customerColors.charcoal.DEFAULT}0F` }}
          style={({ pressed }) => [styles.rowPressable, pressed && { opacity: 0.7 }]}
        >
          {/* Layout on a plain View — a Pressable with a function-style prop
              drops flex props on this Fabric build (icon/body/dot would stack). */}
          <View style={[styles.row, !item.isRead && styles.rowUnread]}>
            <View style={[styles.iconCircle, !item.isRead && styles.iconCircleUnread]}>
              <Icon
                size={18}
                color={item.isRead ? customerColors.charcoal.soft : customerColors.coral.DEFAULT}
              />
            </View>
            <View style={styles.body}>
              <View style={styles.topLine}>
                <Text style={styles.title} numberOfLines={1}>
                  {item.title}
                </Text>
                <Text style={styles.time}>{relativeTime(item.createdAt)}</Text>
              </View>
              <Text style={styles.message} numberOfLines={2}>
                {item.message}
              </Text>
            </View>
            {!item.isRead ? <View style={styles.unreadDot} /> : null}
          </View>
        </Pressable>
      );
    },
    [onPressRow],
  );

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <Stack.Screen options={{ headerShown: false }} />
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          style={styles.backBtn}
        >
          <ArrowLeft size={22} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Notifications</Text>
        {hasUnread ? (
          <Pressable
            onPress={() => markAll.mutate()}
            accessibilityRole="button"
            accessibilityLabel="Mark all notifications read"
            hitSlop={8}
            style={styles.markAllBtn}
          >
            <Text style={styles.markAllText}>Mark all read</Text>
          </Pressable>
        ) : (
          <View style={styles.markAllBtn} />
        )}
      </View>

      {isLoading ? (
        <View style={styles.center}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : (
        <FlatList
          data={notifications}
          keyExtractor={(n) => n.id}
          renderItem={renderItem}
          onRefresh={refetch}
          refreshing={isRefetching}
          contentContainerStyle={
            notifications.length === 0 ? styles.emptyWrap : styles.listContent
          }
          ListEmptyComponent={
            <EmptyState
              icon={<Bell size={40} color={customerColors.charcoal.soft} />}
              title="No notifications yet"
              body="Order updates, chef news and meal-plan reminders will show up here."
            />
          }
        />
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    gap: 8,
  },
  backBtn: { padding: 4 },
  headerTitle: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
  },
  markAllBtn: { minWidth: 90, alignItems: 'flex-end' },
  markAllText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.coral.pressed,
  },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  emptyWrap: { flexGrow: 1, justifyContent: 'center' },
  listContent: { paddingHorizontal: 16, paddingBottom: 32 },

  rowPressable: { marginTop: 6 },
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 10,
    paddingVertical: 9,
    paddingHorizontal: 10,
    borderRadius: 14,
    backgroundColor: customerColors.surface.soft,
  },
  rowUnread: { backgroundColor: customerColors.coral.tint },
  iconCircle: {
    width: 34,
    height: 34,
    borderRadius: 17,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.surface.DEFAULT,
  },
  iconCircleUnread: { backgroundColor: customerColors.surface.DEFAULT },
  body: { flex: 1, gap: 3, paddingTop: 1 },
  topLine: { flexDirection: 'row', alignItems: 'baseline', gap: 8 },
  title: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  time: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  message: {
    fontFamily: 'Inter',
    fontSize: 13,
    lineHeight: 18,
    color: customerColors.charcoal.soft,
  },
  unreadDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: customerColors.coral.DEFAULT,
    marginTop: 4,
  },
});
