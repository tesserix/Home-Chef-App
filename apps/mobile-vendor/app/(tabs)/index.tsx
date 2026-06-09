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

// Status dot palette. Persimmon is reserved for `ready` — the one
// in-flight state that needs the chef's eye (food is plated, driver
// hasn't arrived). Everything else is muted info/amber.
const STATUS_DOT: Record<string, string> = {
  accepted: theme.colors.info.DEFAULT,
  preparing: theme.colors.amber.DEFAULT,
  ready: theme.colors.herb.DEFAULT,
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
        {/* Zone A — Command bar. Name left, Open/Closed control right.
            No greeting. The status control is a sized button (not a pill)
            so a closed kitchen reads as obviously bordered, never as a
            soft toggle that's easy to skip. */}
        <View style={styles.commandBar}>
          <Text style={styles.commandName} numberOfLines={1}>
            {displayName}
          </Text>
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

        {/* Zone B'' — Admin requests awaiting a chef response. Rendered ABOVE
            doc expiry because info_requested is fresher / higher-friction:
            an admin has explicitly asked the chef for something. */}
        {(actionRequests?.length ?? 0) > 0 && (
          <View style={styles.expirySection}>
            <Text style={styles.expirySectionLabel}>ACTION REQUIRED</Text>
            <View style={styles.expiryCards}>
              {actionRequests!.map((req) => (
                <Pressable
                  key={req.id}
                  onPress={() => router.push('/admin-requests')}
                  accessibilityRole="button"
                  accessibilityLabel={`Admin needs information for ${req.title}. Tap to view.`}
                >
                  {({ pressed }) => (
                    <View style={[styles.expiryCard, pressed && { opacity: 0.85 }]}>
                      <View style={styles.expiryCardTop}>
                        <View
                          style={[
                            styles.expiryDot,
                            { backgroundColor: theme.colors.destructive.DEFAULT },
                          ]}
                        />
                        <Text style={styles.expiryDaysLabel}>Admin needs info</Text>
                      </View>
                      <Text style={styles.expiryDocType} numberOfLines={2}>
                        {req.title}
                      </Text>
                      <Text style={styles.expiryCtaLink}>View request →</Text>
                    </View>
                  )}
                </Pressable>
              ))}
            </View>
          </View>
        )}

        {/* Zone B' — Document expiry alerts. Rendered above the order queue
            because a lapsed FSSAI license hides the kitchen from customers —
            it's actionable even when there are no orders. */}
        {expiringDocs.length > 0 && (
          <View style={styles.expirySection}>
            <Text style={styles.expirySectionLabel}>ACTION REQUIRED</Text>
            <View style={styles.expiryCards}>
              {expiringDocs.map((doc) => (
                <Pressable
                  key={doc.id}
                  onPress={() => router.push('/(onboarding)/documents')}
                  accessibilityRole="button"
                  accessibilityLabel={`${describeDocumentType(doc.type)} expires in ${doc.daysUntilExpiry} days. Tap to re-upload.`}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.expiryCard,
                        pressed && { opacity: 0.85 },
                      ]}
                    >
                      <View style={styles.expiryCardTop}>
                        <View
                          style={[
                            styles.expiryDot,
                            {
                              backgroundColor:
                                doc.daysUntilExpiry <= 7
                                  ? theme.colors.destructive.DEFAULT
                                  : theme.colors.amber.DEFAULT,
                            },
                          ]}
                        />
                        <Text style={styles.expiryDaysLabel}>
                          {doc.daysUntilExpiry <= 0
                            ? 'Expired'
                            : `Expires in ${doc.daysUntilExpiry} day${doc.daysUntilExpiry === 1 ? '' : 's'}`}
                        </Text>
                      </View>
                      <Text style={styles.expiryDocType} numberOfLines={1}>
                        {describeDocumentType(doc.type)} expires on{' '}
                        {new Date(doc.expiryDate).toLocaleDateString('en-IN', {
                          day: 'numeric',
                          month: 'short',
                          year: 'numeric',
                        })}
                        . Re-upload to keep your kitchen visible.
                      </Text>
                      <Text style={styles.expiryCtaLink}>Re-upload →</Text>
                    </View>
                  )}
                </Pressable>
              ))}
            </View>
          </View>
        )}

        {/* Zone B — Action queue. The hero when there's anything to act on.
            Renders as filled bone cards because each row demands a decision. */}
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

        {/* Zone C — In progress. Borderless hairline rows, not cards.
            The visual demotion is intentional: pending = cards (act),
            in-progress = rows (just be aware). */}
        {!isLoading && inFlightOrders.length > 0 && (
          <View style={styles.section}>
            <Text style={styles.sectionLabel}>IN PROGRESS</Text>
            <View>
              {inFlightOrders.slice(0, 5).map((order) => (
                <InFlightRow
                  key={order.id}
                  order={order}
                  onPress={() => router.push(`/orders/${order.id}`)}
                />
              ))}
            </View>
          </View>
        )}

        {/* Bottom zone — pushed to the visible bottom of the viewport
            via marginTop:auto on the contentContainer's flexGrow:1.
            Holds the dead-screen reassurance + today summary. When
            there's active content above, this sits naturally after it. */}
        <View style={styles.bottomZone}>
          {isQuiet && (
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
          )}

          {/* Today strip. Hidden when the chef has truly zero history
              (new install) — `₹0 | 0 orders | 0.0★` reads as a broken
              screen, not a calm one. */}
          {!isLoading &&
            ((dashboard?.todayOrders ?? 0) > 0 ||
              (dashboard?.totalReviews ?? 0) > 0) && (
              <View style={styles.todayStrip}>
                <Text style={styles.todayEarnings}>
                  ₹{(dashboard?.todayEarnings ?? 0).toFixed(0)}
                </Text>
                <Text style={styles.todayDivider}>|</Text>
                <Text style={styles.todayStat}>
                  {dashboard?.todayOrders ?? 0} order
                  {(dashboard?.todayOrders ?? 0) === 1 ? '' : 's'}
                </Text>
                <Text style={styles.todayDivider}>|</Text>
                <Text style={styles.todayStat}>
                  {(dashboard?.rating ?? 0).toFixed(1)}★
                </Text>
              </View>
            )}
        </View>
      </ScrollView>
    </SafeAreaView>
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
  onPress: () => void;
}

function InFlightRow({ order, onPress }: InFlightRowProps) {
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
      <View
        style={[
          rowStyles.dot,
          {
            backgroundColor: STATUS_DOT[order.status] ?? theme.colors.ink.muted,
          },
        ]}
      />
      <View style={rowStyles.nameBlock}>
        <Text style={rowStyles.name} numberOfLines={1}>
          {order.customerName}
        </Text>
        <Text style={rowStyles.meta}>
          {STATUS_LABEL[order.status] ?? order.status} ·{' '}
          {formatMinutesAgo(order.createdAt)}
        </Text>
      </View>
      <Text style={rowStyles.total}>₹{order.total.toFixed(0)}</Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  scrollContent: {
    flexGrow: 1,
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[6],
  },
  bottomZone: {
    marginTop: 'auto',
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
  commandName: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.1,
  },
  statusButton: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.DEFAULT,
    minHeight: 36,
    minWidth: 92,
    justifyContent: 'center',
    borderWidth: 1,
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

  // Zone B' — Document expiry alerts
  expirySection: {
    marginBottom: theme.spacing[4],
  },
  expirySectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[2],
  },
  expiryCards: {
    gap: theme.spacing[2],
  },
  expiryCard: {
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[1],
    // Use borderWidth instead of hairlineWidth so the amber tint card edge
    // is always visible on paper bg.
    borderWidth: 1,
    borderColor: theme.colors.amber.DEFAULT,
  },
  expiryCardTop: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
  },
  expiryDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
  },
  expiryDaysLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: 0.1,
  },
  expiryDocType: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 19,
  },
  expiryCtaLink: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.DEFAULT,
    letterSpacing: 0.1,
    marginTop: theme.spacing[1],
  },

  // Zone B — Action queue + surge banner + see-more link + empty line
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
  actionList: { gap: theme.spacing[2] },
  seeMoreInline: {
    alignSelf: 'flex-start',
    paddingVertical: theme.spacing[2],
    marginTop: theme.spacing[1],
  },
  seeMoreInlineLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },
  // Zone C — In progress section
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

  // Zone D — Today strip (bottom-anchored)
  todayStrip: {
    paddingTop: theme.spacing[4],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: theme.spacing[3],
  },
  todayEarnings: {
    fontFamily: 'Geist-Bold',
    fontSize: 22,
    lineHeight: 24,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  todayDivider: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.mist.strong,
  },
  todayStat: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    fontVariant: ['tabular-nums'],
  },

  // Error state — preserved from v2
  errorScreen: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    backgroundColor: theme.colors.paper,
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
    color: theme.colors.ink.DEFAULT,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    textDecorationLine: 'underline',
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
