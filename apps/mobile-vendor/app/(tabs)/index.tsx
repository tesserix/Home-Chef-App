import { useEffect, useMemo, useState } from 'react';
import {
  ActionSheetIOS,
  Alert,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import Animated, {
  cancelAnimation,
  Easing,
  FadeInDown,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withRepeat,
  withTiming,
} from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import {
  useVendorDashboard,
  useToggleAcceptingOrders,
  usePauseReceiving,
  useResumeReceiving,
  type PauseMinutes,
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

// Time-of-day greeting key for the two-line command bar (UI-V2 spec §4).
function deriveGreetingKey(): string {
  const hour = new Date().getHours();
  if (hour < 12) return 'dashboard.goodMorning';
  if (hour < 17) return 'dashboard.goodAfternoon';
  return 'dashboard.goodEvening';
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

const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);
const PULSE_EASING = Easing.bezier(0.4, 0, 0.2, 1);

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
    bg: theme.colors.mist.DEFAULT,
    text: theme.colors.ink.DEFAULT,
  },
  ready: {
    bg: theme.colors.success.tint,
    text: theme.colors.success.soft,
  },
};

// A chef glancing for 2s with wet hands needs to register pending count,
// kitchen status, and any "ready" order. Past-tense numbers (earnings,
// rating) come last. See agent design notes — pending wins the top.
export default function DashboardScreen() {
  const { t } = useTranslation();
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
  const pauseMutation = usePauseReceiving();
  const resumeMutation = useResumeReceiving();
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

  // Live pulse on the Open pill's dot — a quiet "we're live" heartbeat.
  // 900 ms state-change easing, suspended under reduced motion. Must run
  // before the isError early return (hooks rule).
  const acceptingOrders = dashboard?.acceptingOrders ?? false;
  const reduceMotion = useReducedMotion();
  const pulse = useSharedValue(1);
  useEffect(() => {
    if (acceptingOrders && !reduceMotion) {
      pulse.value = withRepeat(
        withTiming(0.35, { duration: 900, easing: PULSE_EASING }),
        -1,
        true,
      );
    } else {
      cancelAnimation(pulse);
      pulse.value = 1;
    }
  }, [acceptingOrders, reduceMotion, pulse]);
  const pulseStyle = useAnimatedStyle(() => ({ opacity: pulse.value }));

  // Timed-pause state: closed AND a future reopen time → show "Back HH:MM".
  const pausedUntil = dashboard?.pausedUntil
    ? new Date(dashboard.pausedUntil)
    : null;
  const isPaused =
    !acceptingOrders &&
    pausedUntil !== null &&
    !Number.isNaN(pausedUntil.getTime()) &&
    pausedUntil.getTime() > Date.now();
  const statusBusy =
    toggleMutation.isPending ||
    pauseMutation.isPending ||
    resumeMutation.isPending ||
    isLoading;
  const statusLabel = acceptingOrders
    ? 'Open'
    : isPaused
      ? `Back ${pausedUntil!.toLocaleTimeString('en-IN', { hour: 'numeric', minute: '2-digit' })}`
      : 'Closed';

  // Tapping the status pill opens Open / Close / Pause {15,30,60} as a native
  // sheet (iOS) or chained alert (Android) — the same pattern orders.tsx uses.
  function openStatusMenu(): void {
    const options: { label: string; action: () => void }[] = acceptingOrders
      ? [
          { label: 'Close kitchen', action: () => toggleMutation.mutate(false) },
          { label: 'Pause 15 min', action: () => pauseMutation.mutate(15) },
          { label: 'Pause 30 min', action: () => pauseMutation.mutate(30) },
          { label: 'Pause 60 min', action: () => pauseMutation.mutate(60) },
        ]
      : [{ label: 'Open kitchen', action: () => resumeMutation.mutate() }];

    if (Platform.OS === 'ios') {
      ActionSheetIOS.showActionSheetWithOptions(
        {
          title: 'Kitchen status',
          options: [...options.map((o) => o.label), 'Cancel'],
          cancelButtonIndex: options.length,
        },
        (i) => {
          if (i < options.length) options[i]!.action();
        },
      );
    } else {
      Alert.alert('Kitchen status', undefined, [
        ...options.map((o) => ({ text: o.label, onPress: o.action })),
        { text: 'Cancel', style: 'cancel' as const },
      ]);
    }
  }

  if (isError) {
    return (
      <SafeAreaView style={styles.errorScreen}>
        <Text style={styles.errorTitle}>{t('dashboard.errorTitle')}</Text>
        <Text style={styles.errorBody}>
          {t('dashboard.errorBody')}
        </Text>
        <Pressable onPress={() => refetch()} style={styles.errorPrimary}>
          <Text style={styles.errorPrimaryText}>{t('common.retry')}</Text>
        </Pressable>
        <Pressable
          onPress={() => router.replace('/(onboarding)/personal-info')}
          style={styles.errorSecondary}
        >
          <Text style={styles.errorSecondaryText}>{t('dashboard.completeOnboarding')}</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

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
        {/* Zone A — Hero header. Dark ink card: greeting + name, persimmon
            Open pill with a live pulse dot, and today's numbers in light
            numerals on near-black. The one statement piece on the screen —
            everything below it is calm white-on-bone. */}
        <Animated.View
          style={styles.hero}
          entering={
            reduceMotion
              ? undefined
              : FadeInDown.duration(250).easing(ENTRANCE_EASING)
          }
        >
          <View style={styles.heroTopRow}>
            <View style={styles.heroGreetingBlock}>
              <Text style={styles.heroGreeting}>{t(deriveGreetingKey())}</Text>
              <Text style={styles.heroName} numberOfLines={1}>
                {displayName}
              </Text>
            </View>
            <Pressable
              onPress={openStatusMenu}
              disabled={statusBusy}
              accessibilityRole="button"
              accessibilityLabel={
                acceptingOrders
                  ? t('dashboard.kitchenAccessibilityOpen')
                  : isPaused
                    ? t('dashboard.kitchenAccessibilityPaused', {
                        time: statusLabel.replace('Back ', ''),
                      })
                    : t('dashboard.kitchenAccessibilityClosed')
              }
            >
              {({ pressed }) => (
                // Visual layer on an inner View. iOS occasionally drops
                // backgroundColor + borderWidth from Pressable with a
                // function-based `style` prop — same trick the shared
                // <Button> primitive uses.
                <View
                  style={[
                    styles.statusPill,
                    acceptingOrders
                      ? styles.statusPillOpen
                      : styles.statusPillClosed,
                    pressed && { opacity: 0.85 },
                    statusBusy && {
                      opacity: 0.5,
                    },
                  ]}
                >
                  <Animated.View
                    style={[
                      styles.statusDot,
                      {
                        backgroundColor: acceptingOrders
                          ? theme.colors.paper
                          : theme.colors.ink.muted,
                      },
                      acceptingOrders && pulseStyle,
                    ]}
                  />
                  <Text
                    style={[
                      styles.statusPillLabel,
                      {
                        color: acceptingOrders
                          ? theme.colors.paper
                          : theme.colors.mist.DEFAULT,
                      },
                    ]}
                  >
                    {statusLabel}
                  </Text>
                </View>
              )}
            </Pressable>
          </View>
          {showToday && (
            <View style={styles.heroStatsRow}>
              <View style={styles.heroStatMain}>
                <Text style={styles.heroEarnings}>
                  ₹{(dashboard?.todayEarnings ?? 0).toFixed(0)}
                </Text>
                <Text style={styles.heroStatLabel}>{t('dashboard.todaysEarnings')}</Text>
              </View>
              <View style={styles.heroStatCol}>
                <Text style={styles.heroStatValue}>
                  {dashboard?.todayOrders ?? 0}
                </Text>
                <Text style={styles.heroStatLabel}>{t('dashboard.orders')}</Text>
              </View>
              <View style={styles.heroStatCol}>
                <Text style={styles.heroStatValue}>
                  {(dashboard?.rating ?? 0).toFixed(1)}★
                </Text>
                <Text style={styles.heroStatLabel}>{t('dashboard.rating')}</Text>
              </View>
            </View>
          )}
        </Animated.View>

        {/* Zone B' — single merged ACTION REQUIRED stack (UI-V2 spec §7).
            Admin requests first (an admin explicitly asked the chef for
            something — fresher / higher-friction), then doc expiry (a
            lapsed FSSAI license hides the kitchen from customers). */}
        {hasAlerts && (
          <Animated.View
            style={styles.alertSection}
            entering={
              reduceMotion
                ? undefined
                : FadeInDown.delay(60).duration(250).easing(ENTRANCE_EASING)
            }
          >
            <Text style={styles.sectionLabel}>{t('dashboard.actionRequired')}</Text>
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
                          {t('dashboard.adminNeedsInfo')}
                        </Text>
                        <Text style={styles.alertCta}>{t('dashboard.viewRequest')}</Text>
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
                            ? t('dashboard.expired')
                            : doc.daysUntilExpiry === 1
                              ? t('dashboard.expiresInDay', {
                                  count: doc.daysUntilExpiry,
                                })
                              : t('dashboard.expiresInDays', {
                                  count: doc.daysUntilExpiry,
                                })}
                        </Text>
                        <Text style={styles.alertCta}>{t('dashboard.reupload')}</Text>
                      </View>
                      <Text style={styles.alertBody} numberOfLines={1}>
                        {t('dashboard.docExpiresOn', {
                          document: describeDocumentType(doc.type),
                          date: new Date(doc.expiryDate).toLocaleDateString(
                            'en-IN',
                            {
                              day: 'numeric',
                              month: 'short',
                              year: 'numeric',
                            },
                          ),
                        })}
                      </Text>
                    </View>
                  )}
                </Pressable>
              ))}
            </View>
          </Animated.View>
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
                    { backgroundColor: theme.colors.amber.DEFAULT },
                  ]}
                />
                <Text style={styles.surgeBannerLabel}>
                  {t('dashboard.ordersAwaiting', { count: pendingOrders.length })}
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
                    {t('dashboard.seeMore', { count: pendingOrders.length - 3 })}
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
          <Animated.View
            style={styles.section}
            entering={
              reduceMotion
                ? undefined
                : FadeInDown.delay(140).duration(250).easing(ENTRANCE_EASING)
            }
          >
            <Text style={styles.sectionLabel}>{t('dashboard.inProgress')}</Text>
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
          </Animated.View>
        )}

        {/* Quiet state — dead-screen reassurance pushed to the visible
            bottom of the viewport via marginTop:auto on the contentContainer's
            flexGrow:1. A chef staring at an empty screen at 3am needs to
            know it's quiet, not that the platform broke. (Today's numbers
            live in the hero, so this zone is reassurance copy only.) */}
        {isQuiet && (
          <View style={styles.bottomZone}>
            <View style={styles.quietBlock}>
              <Text style={styles.quietHeadline}>
                {acceptingOrders
                  ? t('dashboard.quietRightNow')
                  : t('dashboard.kitchenClosed')}
              </Text>
              <Text style={styles.quietBody}>
                {acceptingOrders
                  ? lastOrderIso
                    ? t('dashboard.quietLastOrder', {
                        ago: formatMinutesAgo(lastOrderIso),
                      })
                    : t('dashboard.quietOpen')
                  : t('dashboard.quietClosed')}
              </Text>
            </View>
          </View>
        )}
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

  // Zone A — Hero header (dark ink card)
  hero: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[5],
    marginTop: theme.spacing[3],
    marginBottom: theme.spacing[6],
    gap: theme.spacing[5],
    ...theme.shadow[2],
  },
  heroTopRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: theme.spacing[3],
  },
  heroGreetingBlock: {
    flex: 1,
  },
  heroGreeting: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[0.5],
  },
  heroName: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    color: theme.colors.paper,
    letterSpacing: -0.3,
  },
  statusPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.full,
    minHeight: 40,
    minWidth: 92,
    justifyContent: 'center',
  },
  statusPillOpen: {
    backgroundColor: theme.colors.success.DEFAULT,
  },
  statusPillClosed: {
    backgroundColor: 'transparent',
    borderWidth: 1,
    borderColor: theme.colors.ink.soft,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  statusPillLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    letterSpacing: 0.2,
  },
  heroStatsRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    gap: theme.spacing[5],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.ink.soft,
    paddingTop: theme.spacing[4],
  },
  heroStatMain: {
    flex: 1,
  },
  heroEarnings: {
    fontFamily: 'Geist-Bold',
    fontSize: 30,
    lineHeight: 34,
    letterSpacing: -0.5,
    color: theme.colors.paper,
    fontVariant: ['tabular-nums'],
  },
  heroStatCol: {
    alignItems: 'flex-start',
  },
  heroStatValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    lineHeight: 22,
    color: theme.colors.paper,
    fontVariant: ['tabular-nums'],
  },
  heroStatLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[1],
  },

  // Zone B' — merged ACTION REQUIRED alert stack
  alertSection: {
    marginBottom: theme.spacing[6],
  },
  alertCards: {
    gap: theme.spacing[2],
  },
  alertCard: {
    backgroundColor: theme.colors.bone,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
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
    color: theme.colors.ink.DEFAULT,
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
    backgroundColor: theme.colors.amber.tint,
    marginBottom: theme.spacing[3],
  },
  surgeBannerLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
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
    color: theme.colors.ink.DEFAULT,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
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
