import { useEffect, useMemo, useState } from 'react';
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
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import {
  useVendorDashboard,
  useToggleAcceptingOrders,
} from '../../hooks/useVendorDashboard';
import {
  useVendorPendingOrders,
  useOrderAction,
  type Order,
} from '../../hooks/useVendorOrders';
import {
  useExpiringDocuments,
  describeDocumentType,
} from '../../hooks/useExpiringDocuments';
import { useActionRequiredAdminRequests } from '../../hooks/useAdminRequests';
import { useAuthStore } from '../../store/auth-store';
import { PendingOrderCard } from '../../components/vendor/PendingOrderCard';

// Pull a human-readable display name out of the user object. Falls back
// from `name` → Gmail handle → "Chef". Used by the Zone A command bar.
function deriveDisplayName(
  user: { name?: string; email?: string } | null | undefined,
): string {
  if (user?.name && user.name.trim().length > 0) return user.name.trim();
  const email = user?.email ?? '';
  if (email.includes('@')) {
    const local = email.split('@')[0]?.split('+')[0] ?? '';
    if (local.length > 0) {
      return local
        .split(/[._-]/)
        .filter(Boolean)
        .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
        .join(' ');
    }
  }
  return 'Chef';
}

// Time-of-day greeting for the two-line command bar (UI-V2 spec §4).
function deriveGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return 'Good morning';
  if (hour < 17) return 'Good afternoon';
  return 'Good evening';
}

function formatMinutesAgo(iso: string): string {
  const created = new Date(iso).getTime();
  if (Number.isNaN(created)) return '';
  const mins = Math.max(0, Math.floor((Date.now() - created) / 60_000));
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

const IN_FLIGHT_STATUSES = new Set<Order['status']>([
  'accepted',
  'preparing',
  'ready',
]);

const STATUS_LABEL: Record<string, string> = {
  accepted: 'Accepted',
  preparing: 'Preparing',
  ready: 'Ready',
};

// Status chip palette (UI-V2 spec §2): tint background + darker text of
// the same hue. Persimmon stays reserved for `ready` — the one in-flight
// state that needs the chef's eye.
const STATUS_CHIP: Record<string, { bg: string; text: string }> = {
  accepted: {
    bg: theme.colors.info.tint,
    text: theme.colors.info.DEFAULT,
  },
  preparing: {
    bg: theme.colors.amber.tint,
    text: theme.colors.ink.DEFAULT,
  },
  ready: {
    bg: theme.colors.herb.tint,
    text: theme.colors.herb.soft,
  },
};

// A chef glancing for 2s with wet hands needs to register pending count,
// kitchen status, and any "ready" order. Past-tense numbers (earnings,
// rating) come last. See agent design notes — pending wins the top.
export default function DashboardScreen() {
  const user = useAuthStore((s) => s.user);
  const {
    data: dashboard,
    isLoading,
    refetch,
    isError,
    error,
  } = useVendorDashboard();
  const { data: pendingResp, refetch: refetchPending } =
    useVendorPendingOrders();
  const toggleMutation = useToggleAcceptingOrders();
  const { triggerAction, isLoading: orderActionLoading } = useOrderAction();
  const { data: expiringDocsData } = useExpiringDocuments();
  const { data: actionRequests } = useActionRequiredAdminRequests();

  // User-initiated pull-to-refresh only — avoids the stuck-spinner bug when
  // React Query's isRefetching fires for background refetches (focus, mutation
  // invalidation, stale-time expiry).
  const [isPulling, setIsPulling] = useState(false);
  async function onPullRefresh(): Promise<void> {
    setIsPulling(true);
    try {
      await Promise.all([refetch(), refetchPending()]);
    } finally {
      setIsPulling(false);
    }
  }

  const expiringDocs = expiringDocsData?.documents ?? [];

  const displayName = deriveDisplayName(
    user as { name?: string; email?: string } | null,
  );

  const pendingOrders = pendingResp?.orders ?? [];
  const inFlightOrders = useMemo(
    () =>
      (dashboard?.recentOrders ?? []).filter((o) =>
        IN_FLIGHT_STATUSES.has(o.status as Order['status']),
      ),
    [dashboard?.recentOrders],
  );

  // Last order timestamp across pending + recent. Drives the dead-screen
  // reassurance copy: a chef staring at an empty screen at 3am needs to
  // know it's quiet, not that the platform broke.
  const lastOrderIso = useMemo(() => {
    const all: string[] = [
      ...pendingOrders.map((o) => o.createdAt),
      ...(dashboard?.recentOrders ?? []).map((o) => o.createdAt),
    ];
    if (all.length === 0) return null;
    return all.reduce((a, b) =>
      new Date(a).getTime() > new Date(b).getTime() ? a : b,
    );
  }, [pendingOrders, dashboard?.recentOrders]);

  const minutesSinceLastOrder = useMemo(() => {
    if (!lastOrderIso) return null;
    const t = new Date(lastOrderIso).getTime();
    if (Number.isNaN(t)) return null;
    return Math.floor((Date.now() - t) / 60_000);
  }, [lastOrderIso]);

  // 404 with "Chef profile not found" → user signed in but never
  // completed onboarding. Auto-route to the wizard. Preserved from v2.
  useEffect(() => {
    if (!isError) return;
    const status = (error as { response?: { data?: { error?: string } } } | null)
      ?.response?.data?.error;
    if (status === 'Chef profile not found') {
      router.replace('/(onboarding)/personal-info');
    }
  }, [isError, error]);

  if (isError) {
    return (
      <SafeAreaView style={styles.errorScreen}>
        <Text style={styles.errorTitle}>Couldn't load your dashboard</Text>
        <Text style={styles.errorBody}>
          Check your connection and try again.
        </Text>
        <Pressable onPress={() => refetch()} style={styles.errorPrimary}>
          <Text style={styles.errorPrimaryText}>Retry</Text>
        </Pressable>
        <Pressable
          onPress={() => router.replace('/(onboarding)/personal-info')}
          style={styles.errorSecondary}
        >
          <Text style={styles.errorSecondaryText}>Complete onboarding</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  const acceptingOrders = dashboard?.acceptingOrders ?? false;
  const isSurge = pendingOrders.length > 3;
  const isQuiet =
    !isLoading &&
    pendingOrders.length === 0 &&
    inFlightOrders.length === 0 &&
    (minutesSinceLastOrder === null || minutesSinceLastOrder > 120);

  const hasAlerts =
    (actionRequests?.length ?? 0) > 0 || expiringDocs.length > 0;

  // Today card visibility — hidden when the chef has truly zero history
  // (new install): `₹0 | 0 orders | 0.0★` reads as a broken screen.
  const showToday =
    !isLoading &&
    ((dashboard?.todayOrders ?? 0) > 0 || (dashboard?.totalReviews ?? 0) > 0);

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl
            refreshing={isPulling}
            onRefresh={onPullRefresh}
            tintColor={theme.colors.ink.DEFAULT}
          />
        }
      >
        {/* Zone A — Command bar. Two-line greeting left (caption + name),
            Open/Closed pill right. The pill is high-contrast so a closed
            kitchen reads as obviously bordered, never as a soft toggle
            that's easy to skip. */}
        <View style={styles.commandBar}>
          <View style={styles.commandGreetingBlock}>
            <Text style={styles.commandGreeting}>{deriveGreeting()}</Text>
            <Text style={styles.commandName} numberOfLines={1}>
              {displayName}
            </Text>
          </View>
          <Pressable
            onPress={() => toggleMutation.mutate(!acceptingOrders)}
            disabled={toggleMutation.isPending || isLoading}
            accessibilityRole="button"
            accessibilityLabel={
              acceptingOrders
                ? 'Kitchen is open. Tap to close.'
                : 'Kitchen is closed. Tap to open.'
            }
          >
            {({ pressed }) => (
              // Visual layer on an inner View. iOS occasionally drops
              // backgroundColor + borderWidth from Pressable with a
              // function-based `style` prop — same trick the shared
              // <Button> primitive uses.
              <View
                style={[
                  styles.statusButton,
                  acceptingOrders
                    ? styles.statusButtonOpen
                    : styles.statusButtonClosed,
                  pressed && { opacity: 0.85 },
                  (toggleMutation.isPending || isLoading) && { opacity: 0.5 },
                ]}
              >
                <View
                  style={[
                    styles.statusDot,
                    {
                      backgroundColor: acceptingOrders
                        ? theme.colors.herb.DEFAULT
                        : theme.colors.ink.muted,
                    },
                  ]}
                />
                <Text
                  style={[
                    styles.statusButtonLabel,
                    {
                      color: acceptingOrders
                        ? theme.colors.paper
                        : theme.colors.ink.DEFAULT,
                    },
                  ]}
                >
                  {acceptingOrders ? 'Open' : 'Closed'}
                </Text>
              </View>
            )}
          </Pressable>
        </View>

        {/* Zone B' — single merged ACTION REQUIRED stack (UI-V2 spec §7).
            Admin requests first (an admin explicitly asked the chef for
            something — fresher / higher-friction), then doc expiry (a
            lapsed FSSAI license hides the kitchen from customers). */}
        {hasAlerts && (
          <View style={styles.alertSection}>
            <Text style={styles.sectionLabel}>ACTION REQUIRED</Text>
            <View style={styles.alertCards}>
              {(actionRequests ?? []).map((req) => (
                <Pressable
                  key={req.id}
                  onPress={() => router.push('/admin-requests')}
                  accessibilityRole="button"
                  accessibilityLabel={`Admin needs information for ${req.title}. Tap to view.`}
                >
                  {({ pressed }) => (
                    <View
                      style={[styles.alertCard, pressed && { opacity: 0.85 }]}
                    >
                      <View style={styles.alertRowTop}>
                        <View
                          style={[
                            styles.alertDot,
                            {
                              backgroundColor:
                                theme.colors.destructive.DEFAULT,
                            },
                          ]}
                        />
                        <Text style={styles.alertLabel} numberOfLines={1}>
                          Admin needs info
                        </Text>
                        <Text style={styles.alertCta}>View request →</Text>
                      </View>
                      <Text style={styles.alertBody} numberOfLines={1}>
                        {req.title}
                      </Text>
                    </View>
                  )}
                </Pressable>
              ))}
              {expiringDocs.map((doc) => (
                <Pressable
                  key={doc.id}
                  onPress={() => router.push('/(onboarding)/documents')}
                  accessibilityRole="button"
                  accessibilityLabel={`${describeDocumentType(doc.type)} expires in ${doc.daysUntilExpiry} days. Tap to re-upload.`}
                >
                  {({ pressed }) => (
                    <View
                      style={[styles.alertCard, pressed && { opacity: 0.85 }]}
                    >
                      <View style={styles.alertRowTop}>
                        <View
                          style={[
                            styles.alertDot,
                            {
                              backgroundColor:
                                doc.daysUntilExpiry <= 7
                                  ? theme.colors.destructive.DEFAULT
                                  : theme.colors.amber.DEFAULT,
                            },
                          ]}
                        />
                        <Text style={styles.alertLabel} numberOfLines={1}>
                          {doc.daysUntilExpiry <= 0
                            ? 'Expired'
                            : `Expires in ${doc.daysUntilExpiry} day${doc.daysUntilExpiry === 1 ? '' : 's'}`}
                        </Text>
                        <Text style={styles.alertCta}>Re-upload →</Text>
                      </View>
                      <Text style={styles.alertBody} numberOfLines={1}>
                        {describeDocumentType(doc.type)} expires on{' '}
                        {new Date(doc.expiryDate).toLocaleDateString('en-IN', {
                          day: 'numeric',
                          month: 'short',
                          year: 'numeric',
                        })}
                        . Re-upload to keep your kitchen visible.
                      </Text>
                    </View>
                  )}
                </Pressable>
              ))}
            </View>
          </View>
        )}

        {/* Zone B — Action queue. The hero when there's anything to act on.
            White shadowed cards on the bone canvas — each row demands a
            decision. */}
        {isLoading ? (
          <View style={styles.actionList}>
            <Skeleton
              height={120}
              style={{ borderRadius: theme.radius.DEFAULT }}
            />
            <Skeleton
              height={120}
              style={{
                borderRadius: theme.radius.DEFAULT,
                marginTop: theme.spacing[2],
              }}
            />
          </View>
        ) : pendingOrders.length > 0 ? (
          <View>
            {isSurge && (
              <View style={styles.surgeBanner}>
                <View
                  style={[
                    styles.statusDot,
                    { backgroundColor: theme.colors.herb.DEFAULT },
                  ]}
                />
                <Text style={styles.surgeBannerLabel}>
                  {pendingOrders.length} orders awaiting acceptance
                </Text>
              </View>
            )}
            <View style={styles.actionList}>
              {pendingOrders.slice(0, 3).map((order) => (
                <PendingOrderCard
                  key={order.id}
                  order={order}
                  disabled={orderActionLoading}
                  onOpenDetail={() => router.push(`/orders/${order.id}`)}
                  onAccept={() => triggerAction(order.id, 'accepted')}
                  onReject={() => triggerAction(order.id, 'rejected')}
                />
              ))}
              {pendingOrders.length > 3 && (
                <Pressable
                  onPress={() => router.push('/(tabs)/orders')}
                  hitSlop={6}
                  style={styles.seeMoreInline}
                  accessibilityRole="link"
                >
                  <Text style={styles.seeMoreInlineLabel}>
                    +{pendingOrders.length - 3} more — open queue
                  </Text>
                </Pressable>
              )}
            </View>
          </View>
        ) : null}

        {/* Zone C — In progress. One white group card of rows with inset
            hairlines (UI-V2 spec §1/§7). The visual demotion vs pending is
            intentional: pending = hero cards (act), in-progress = rows in
            a group card (just be aware). */}
        {!isLoading && inFlightOrders.length > 0 && (
          <View style={styles.section}>
            <Text style={styles.sectionLabel}>IN PROGRESS</Text>
            <View style={styles.groupCard}>
              <View style={styles.groupCardInner}>
                {inFlightOrders.slice(0, 5).map((order, idx, arr) => (
                  <InFlightRow
                    key={order.id}
                    order={order}
                    isLast={idx === arr.length - 1}
                    onPress={() => router.push(`/orders/${order.id}`)}
                  />
                ))}
              </View>
            </View>
          </View>
        )}

        {/* Zone D — TODAY stat card. Inline after IN PROGRESS for an active
            screen; bottom-anchored ONLY in the quiet state (below). */}
        {!isQuiet && showToday && (
          <View style={styles.section}>
            <Text style={styles.sectionLabel}>TODAY</Text>
            <TodayStatCard
              earnings={dashboard?.todayEarnings ?? 0}
              orders={dashboard?.todayOrders ?? 0}
              rating={dashboard?.rating ?? 0}
            />
          </View>
        )}

        {/* Quiet state — dead-screen reassurance pushed to the visible
            bottom of the viewport via marginTop:auto on the contentContainer's
            flexGrow:1. A chef staring at an empty screen at 3am needs to
            know it's quiet, not that the platform broke. */}
        {isQuiet && (
          <View style={styles.bottomZone}>
            <View style={styles.quietBlock}>
              <Text style={styles.quietHeadline}>
                {acceptingOrders ? 'Quiet right now' : 'Kitchen is closed'}
              </Text>
              <Text style={styles.quietBody}>
                {acceptingOrders
                  ? lastOrderIso
                    ? `Last order ${formatMinutesAgo(lastOrderIso)}. You're open and visible to customers.`
                    : 'You’re open and visible to customers. New orders will appear here.'
                  : 'Tap the Closed button above to start receiving orders.'}
              </Text>
            </View>
            {showToday && (
              <View>
                <Text style={styles.sectionLabel}>TODAY</Text>
                <TodayStatCard
                  earnings={dashboard?.todayEarnings ?? 0}
                  orders={dashboard?.todayOrders ?? 0}
                  rating={dashboard?.rating ?? 0}
                />
              </View>
            )}
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

interface TodayStatCardProps {
  earnings: number;
  orders: number;
  rating: number;
}

// "TODAY" stat card — white paper card on the bone canvas, three columns
// with caption labels under each value (UI-V2 spec §7).
function TodayStatCard({ earnings, orders, rating }: TodayStatCardProps) {
  return (
    <View style={todayStyles.card}>
      <View style={todayStyles.col}>
        <Text style={todayStyles.earnings}>₹{earnings.toFixed(0)}</Text>
        <Text style={todayStyles.colLabel}>Earnings</Text>
      </View>
      <View style={todayStyles.col}>
        <Text style={todayStyles.value}>{orders}</Text>
        <Text style={todayStyles.colLabel}>Orders</Text>
      </View>
      <View style={todayStyles.col}>
        <Text style={todayStyles.value}>{rating.toFixed(1)}★</Text>
        <Text style={todayStyles.colLabel}>Rating</Text>
      </View>
    </View>
  );
}

interface InFlightRowProps {
  // Narrow shape — works for both `Order` (pending) and `RecentOrder`
  // (dashboard summary), which omits items/deliveryAddress.
  order: {
    id: string;
    customerName: string;
    total: number;
    status: string;
    createdAt: string;
  };
  isLast: boolean;
  onPress: () => void;
}

function InFlightRow({ order, isLast, onPress }: InFlightRowProps) {
  const chip = STATUS_CHIP[order.status] ?? {
    bg: theme.colors.mist.DEFAULT,
    text: theme.colors.ink.soft,
  };
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
    >
      {({ pressed }) => (
        // Inner-View carries layout. iOS strips flex/bg from Pressable
        // when style prop returns an array — inner-View pattern is safe.
        <View
          style={[
            rowStyles.root,
            pressed && { backgroundColor: theme.colors.bone },
          ]}
        >
          <View style={[rowStyles.inner, !isLast && rowStyles.innerBorder]}>
            <View style={[rowStyles.chip, { backgroundColor: chip.bg }]}>
              <Text style={[rowStyles.chipLabel, { color: chip.text }]}>
                {STATUS_LABEL[order.status] ?? order.status}
              </Text>
            </View>
            <View style={rowStyles.nameBlock}>
              <Text style={rowStyles.name} numberOfLines={1}>
                {order.customerName}
              </Text>
              <Text style={rowStyles.meta}>
                {formatMinutesAgo(order.createdAt)}
              </Text>
            </View>
            <Text style={rowStyles.total}>₹{order.total.toFixed(0)}</Text>
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  scrollContent: {
    flexGrow: 1,
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[6],
  },
  bottomZone: {
    marginTop: 'auto',
  },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[2],
  },

  // Zone A — Command bar
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[5],
    gap: theme.spacing[3],
  },
  commandGreetingBlock: {
    flex: 1,
  },
  commandGreeting: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[0.5],
  },
  commandName: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.3,
  },
  statusButton: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.full,
    minHeight: 40,
    minWidth: 92,
    justifyContent: 'center',
    borderWidth: 1,
    ...theme.shadow[1],
  },
  statusButtonOpen: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderColor: theme.colors.ink.DEFAULT,
  },
  statusButtonClosed: {
    backgroundColor: theme.colors.paper,
    borderColor: theme.colors.mist.strong,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  statusButtonLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    letterSpacing: 0.2,
  },

  // Zone B' — merged ACTION REQUIRED alert stack
  alertSection: {
    marginBottom: theme.spacing[6],
  },
  alertCards: {
    gap: theme.spacing[2],
  },
  alertCard: {
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    gap: theme.spacing[1],
  },
  alertRowTop: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
  },
  alertDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
  },
  alertLabel: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: 0.1,
  },
  alertCta: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.DEFAULT,
    letterSpacing: 0.1,
  },
  alertBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 19,
  },

  // Zone B — Action queue + surge banner + see-more link
  surgeBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.herb.tint,
    marginBottom: theme.spacing[3],
  },
  surgeBannerLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.soft,
    letterSpacing: 0.1,
  },
  actionList: { gap: theme.spacing[2] },
  seeMoreInline: {
    alignSelf: 'flex-start',
    paddingVertical: theme.spacing[2],
    marginTop: theme.spacing[1],
  },
  seeMoreInlineLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.DEFAULT,
  },

  // Zone C — In progress group card
  section: {
    marginTop: theme.spacing[6],
  },
  // Shadow lives on the outer card; the inner wrapper clips pressed-row
  // backgrounds to the rounded corners (iOS overflow:hidden would clip
  // the shadow if both lived on one View).
  groupCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[1],
  },
  groupCardInner: {
    borderRadius: theme.radius.lg,
    overflow: 'hidden',
  },

  // Dead-screen reassurance — bottom-anchored via parent bottomZone
  quietBlock: {
    paddingBottom: theme.spacing[6],
  },
  quietHeadline: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.2,
    marginBottom: theme.spacing[2],
  },
  quietBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
    maxWidth: 320,
  },

  // Error state — preserved from v2, canvas bg per UI-V2 spec
  errorScreen: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    backgroundColor: theme.colors.bone,
  },
  errorTitle: {
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
  },
  errorPrimary: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorPrimaryText: {
    color: theme.colors.paper,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
  },
  errorSecondary: {
    marginTop: theme.spacing[3],
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorSecondaryText: {
    color: theme.colors.herb.DEFAULT,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
  },
});

const todayStyles = StyleSheet.create({
  card: {
    flexDirection: 'row',
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  col: {
    flex: 1,
    alignItems: 'flex-start',
  },
  earnings: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    lineHeight: 28,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  value: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 18,
    lineHeight: 28,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  colLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[1],
  },
});

const rowStyles = StyleSheet.create({
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
