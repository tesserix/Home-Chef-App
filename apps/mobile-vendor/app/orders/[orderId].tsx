import { useMemo } from 'react';
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import {
  useVendorOrderDetail,
  useOrderAction,
  useUpdateOrderStatus,
  type Order,
} from '../../hooks/useVendorOrders';

const STATUS_LABEL: Record<Order['status'], string> = {
  pending: 'New order',
  accepted: 'Accepted',
  preparing: 'Preparing',
  ready: 'Ready for pickup',
  picked_up: 'Out for delivery',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
  rejected: 'Rejected',
};

const STATUS_DOT: Record<Order['status'], string> = {
  pending: theme.colors.amber.DEFAULT,
  accepted: theme.colors.info.DEFAULT,
  preparing: theme.colors.amber.DEFAULT,
  ready: theme.colors.herb.DEFAULT,
  picked_up: theme.colors.info.DEFAULT,
  delivered: theme.colors.diet.veg,
  cancelled: theme.colors.destructive.DEFAULT,
  rejected: theme.colors.destructive.DEFAULT,
};

function formatTimeAgo(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return '';
  const mins = Math.max(0, Math.floor((Date.now() - t) / 60_000));
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function formatAddress(addr: Order['deliveryAddress']): string[] {
  // Backend ships deliveryAddress as a structured object on the order
  // response but the old list path serializes it as a string. Handle both.
  if (typeof addr === 'string') {
    return addr.split(/,\s*/).filter(Boolean);
  }
  if (addr && typeof addr === 'object') {
    const a = addr as unknown as {
      line1?: string;
      line2?: string;
      city?: string;
      state?: string;
      postalCode?: string;
    };
    return [
      a.line1,
      a.line2,
      [a.city, a.state, a.postalCode].filter(Boolean).join(', '),
    ].filter((line): line is string => !!line && line.trim().length > 0);
  }
  return [];
}

export default function OrderDetailScreen() {
  const { orderId } = useLocalSearchParams<{ orderId: string }>();
  const { data: order, isLoading, isError, refetch } = useVendorOrderDetail(
    orderId,
  );
  const { triggerAction, isLoading: actionLoading } = useOrderAction();
  const updateStatus = useUpdateOrderStatus();

  const addressLines = useMemo(
    () => (order ? formatAddress(order.deliveryAddress) : []),
    [order],
  );

  function handleBack(): void {
    if (router.canGoBack()) router.back();
    else router.replace('/(tabs)/orders');
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <CommandBar onBack={handleBack} />
        <View style={styles.centered}>
          <ActivityIndicator color={theme.colors.ink.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError || !order) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <CommandBar onBack={handleBack} />
        <View style={styles.centered}>
          <Text style={styles.errorHeadline}>Couldn't load this order</Text>
          <Text style={styles.errorBody}>
            Open it again from the Orders queue or your history.
          </Text>
          <Pressable
            onPress={() => refetch()}
            style={({ pressed }) => [
              styles.retryBtn,
              pressed && { opacity: 0.85 },
            ]}
            accessibilityRole="button"
          >
            <Text style={styles.retryLabel}>Retry</Text>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <CommandBar onBack={handleBack} />

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* Hero — Geist name, total beside it, status dot + age below. */}
        <View style={styles.hero}>
          <View style={styles.heroTopRow}>
            <Text style={styles.heroName} numberOfLines={1}>
              {order.customerName || 'Customer'}
            </Text>
            <Text style={styles.heroTotal}>₹{order.total.toFixed(0)}</Text>
          </View>
          <View style={styles.heroMetaRow}>
            <View
              style={[
                styles.statusDot,
                {
                  backgroundColor:
                    STATUS_DOT[order.status] ?? theme.colors.ink.muted,
                },
              ]}
            />
            <Text style={styles.heroMeta}>
              {STATUS_LABEL[order.status] ?? order.status} ·{' '}
              {formatTimeAgo(order.createdAt)}
            </Text>
          </View>
        </View>

        {/* Items section */}
        <SectionLabel>ITEMS</SectionLabel>
        <View style={styles.itemsGroup}>
          {order.items.length === 0 ? (
            <Text style={styles.bodyMuted}>No items recorded</Text>
          ) : (
            order.items.map((item, idx) => (
              <View
                key={`${item.name}-${idx}`}
                style={[
                  styles.itemRow,
                  idx < order.items.length - 1 && styles.rowBorderBottom,
                ]}
              >
                <View style={styles.itemBody}>
                  <Text style={styles.itemName} numberOfLines={2}>
                    {item.quantity > 1 ? `${item.quantity} × ` : ''}
                    {item.name}
                  </Text>
                  {item.notes ? (
                    <Text style={styles.itemNote} numberOfLines={2}>
                      {item.notes}
                    </Text>
                  ) : null}
                </View>
                <Text style={styles.itemPrice}>
                  ₹{(item.subtotal ?? item.price * item.quantity).toFixed(0)}
                </Text>
              </View>
            ))
          )}
        </View>

        {/* Totals breakdown — only when the response includes the
            split. The list path doesn't ship subtotal/fees so we keep
            the section silent in that case. */}
        {order.subtotal !== undefined ? (
          <>
            <SectionLabel>TOTAL</SectionLabel>
            <View style={styles.totalsGroup}>
              <TotalRow label="Subtotal" value={order.subtotal} />
              {(order.discount ?? 0) > 0 ? (
                <TotalRow label="Discount" value={-(order.discount ?? 0)} />
              ) : null}
              {(order.deliveryFee ?? 0) > 0 ? (
                <TotalRow label="Delivery" value={order.deliveryFee ?? 0} />
              ) : null}
              {(order.serviceFee ?? 0) > 0 ? (
                <TotalRow label="Service" value={order.serviceFee ?? 0} />
              ) : null}
              {(order.tax ?? 0) > 0 ? (
                <TotalRow
                  label={order.taxName || 'Tax'}
                  value={order.tax ?? 0}
                />
              ) : null}
              {(order.tip ?? 0) > 0 ? (
                <TotalRow label="Tip" value={order.tip ?? 0} />
              ) : null}
              <TotalRow
                label="Total"
                value={order.total}
                emphasis
                hasBorderBottom={false}
              />
            </View>
          </>
        ) : null}

        {/* Delivery address */}
        {addressLines.length > 0 ? (
          <>
            <SectionLabel>DELIVERY ADDRESS</SectionLabel>
            <View style={styles.addressGroup}>
              {addressLines.map((line) => (
                <Text key={line} style={styles.addressLine}>
                  {line}
                </Text>
              ))}
            </View>
          </>
        ) : null}

        {/* Special instructions — persimmon tint callout matches the
            orders queue card's instructions pill. */}
        {order.specialInstructions ? (
          <>
            <SectionLabel>SPECIAL INSTRUCTIONS</SectionLabel>
            <View style={styles.instructionsCallout}>
              <Text style={styles.instructionsText}>
                {order.specialInstructions}
              </Text>
            </View>
          </>
        ) : null}
      </ScrollView>

      {/* Status-dependent footer actions */}
      <FooterActions
        status={order.status}
        orderId={order.id}
        customerName={order.customerName || 'this customer'}
        total={order.total}
        disabled={actionLoading || updateStatus.isPending}
        onAccept={() => triggerAction(order.id, 'accepted')}
        onReject={() => triggerAction(order.id, 'rejected')}
        onMarkPreparing={() =>
          updateStatus.mutate({ orderId: order.id, status: 'preparing' })
        }
        onMarkReady={() =>
          updateStatus.mutate({ orderId: order.id, status: 'ready' })
        }
      />
    </SafeAreaView>
  );
}

// ---- Command bar ------------------------------------------------------------

interface CommandBarProps {
  onBack: () => void;
}

function CommandBar({ onBack }: CommandBarProps) {
  return (
    <View style={styles.commandBar}>
      <Pressable
        onPress={onBack}
        hitSlop={8}
        accessibilityLabel="Go back"
        accessibilityRole="button"
      >
        {({ pressed }) => (
          <View style={[styles.backBtn, pressed && { opacity: 0.6 }]}>
            <ChevronLeft
              size={22}
              color={theme.colors.ink.DEFAULT}
              strokeWidth={2}
            />
          </View>
        )}
      </Pressable>
      <Text style={styles.commandTitle}>Order</Text>
      <View style={styles.commandSpacer} />
    </View>
  );
}

// ---- Section label ----------------------------------------------------------

interface SectionLabelProps {
  children: string;
}

function SectionLabel({ children }: SectionLabelProps) {
  return <Text style={styles.sectionLabel}>{children}</Text>;
}

// ---- Total row --------------------------------------------------------------

interface TotalRowProps {
  label: string;
  value: number;
  emphasis?: boolean;
  hasBorderBottom?: boolean;
}

function TotalRow({
  label,
  value,
  emphasis = false,
  hasBorderBottom = true,
}: TotalRowProps) {
  return (
    <View
      style={[
        styles.totalRow,
        hasBorderBottom && styles.rowBorderBottom,
      ]}
    >
      <Text style={[styles.totalLabel, emphasis && styles.totalLabelStrong]}>
        {label}
      </Text>
      <Text style={[styles.totalValue, emphasis && styles.totalValueStrong]}>
        {value < 0 ? '−' : ''}₹{Math.abs(value).toFixed(0)}
      </Text>
    </View>
  );
}

// ---- Footer actions ---------------------------------------------------------

interface FooterActionsProps {
  status: Order['status'];
  orderId: string;
  customerName: string;
  total: number;
  disabled: boolean;
  onAccept: () => void;
  onReject: () => void;
  onMarkPreparing: () => void;
  onMarkReady: () => void;
}

function FooterActions({
  status,
  customerName,
  total,
  disabled,
  onAccept,
  onReject,
  onMarkPreparing,
  onMarkReady,
}: FooterActionsProps) {
  if (status === 'pending') {
    return (
      <View style={styles.footer}>
        <Pressable
          onPress={onReject}
          disabled={disabled}
          hitSlop={8}
          style={styles.rejectBtn}
          accessibilityRole="button"
          accessibilityLabel={`Reject order from ${customerName}`}
        >
          <Text style={styles.rejectLabel}>Reject</Text>
        </Pressable>
        <Pressable
          onPress={onAccept}
          disabled={disabled}
          style={({ pressed }) => [
            styles.primaryBtn,
            pressed && { opacity: 0.85 },
            disabled && { opacity: 0.4 },
          ]}
          accessibilityRole="button"
          accessibilityLabel={`Accept ₹${total.toFixed(0)} order from ${customerName}`}
        >
          <Text style={styles.primaryLabel}>Accept</Text>
        </Pressable>
      </View>
    );
  }

  if (status === 'accepted') {
    return (
      <View style={styles.footer}>
        <Pressable
          onPress={onMarkPreparing}
          disabled={disabled}
          style={({ pressed }) => [
            styles.primaryBtnFull,
            pressed && { opacity: 0.85 },
            disabled && { opacity: 0.4 },
          ]}
          accessibilityRole="button"
        >
          <Text style={styles.primaryLabel}>Mark preparing</Text>
        </Pressable>
      </View>
    );
  }

  if (status === 'preparing') {
    return (
      <View style={styles.footer}>
        <Pressable
          onPress={onMarkReady}
          disabled={disabled}
          style={({ pressed }) => [
            styles.primaryBtnFull,
            pressed && { opacity: 0.85 },
            disabled && { opacity: 0.4 },
          ]}
          accessibilityRole="button"
        >
          <Text style={styles.primaryLabel}>Mark ready for pickup</Text>
        </Pressable>
      </View>
    );
  }

  // ready / picked_up / delivered / cancelled / rejected — no action.
  // Surface a one-line status caption so the footer doesn't read as empty.
  const caption: Record<string, string> = {
    ready: 'Waiting for driver to pick up.',
    picked_up: 'Out for delivery.',
    delivered: 'Delivered to customer.',
    cancelled: 'This order was cancelled.',
    rejected: 'You rejected this order.',
  };
  const text = caption[status];
  if (!text) return null;

  return (
    <View style={[styles.footer, styles.footerCaptionWrap]}>
      <Text style={styles.footerCaption}>{text}</Text>
    </View>
  );
}

// ---- Styles ----------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },

  // Command bar
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backBtn: {
    width: 32,
    height: 32,
    justifyContent: 'center',
    alignItems: 'flex-start',
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    flex: 1,
  },
  commandSpacer: { width: 32 },

  // Scroll
  scroll: { flex: 1 },
  scrollContent: { paddingBottom: theme.spacing[10] },

  // Hero
  hero: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[5],
  },
  heroTopRow: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: theme.spacing[3],
  },
  heroName: {
    flex: 1,
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },
  heroTotal: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    lineHeight: 28,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  heroMetaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    marginTop: theme.spacing[2],
  },
  heroMeta: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  statusDot: { width: 8, height: 8, borderRadius: 4 },

  // Section labels — same caption-spaced cap-grey treatment as settings
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[6],
    paddingBottom: theme.spacing[2],
  },

  // Items
  itemsGroup: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  itemRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
    gap: theme.spacing[3],
  },
  itemBody: { flex: 1 },
  itemName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 22,
  },
  itemNote: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    marginTop: 2,
    lineHeight: 19,
  },
  itemPrice: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },

  // Shared hairline row border
  rowBorderBottom: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  // Totals
  totalsGroup: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  totalRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 40,
  },
  totalLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  totalLabelStrong: {
    fontFamily: 'Inter-SemiBold',
    color: theme.colors.ink.DEFAULT,
  },
  totalValue: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    fontVariant: ['tabular-nums'],
  },
  totalValueStrong: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.2,
  },

  // Address
  addressGroup: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  addressLine: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 22,
  },

  // Special instructions
  instructionsCallout: {
    marginHorizontal: theme.spacing[4],
    backgroundColor: theme.colors.herb.tint,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
  },
  instructionsText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.herb.soft,
    lineHeight: 22,
  },

  // Body fallback
  bodyMuted: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },

  // Footer
  footer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[5],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
  },
  footerCaptionWrap: {
    justifyContent: 'center',
  },
  footerCaption: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    flex: 1,
  },
  rejectBtn: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    minHeight: 44,
    justifyContent: 'center',
  },
  rejectLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    textDecorationLine: 'underline',
  },
  primaryBtn: {
    flex: 1,
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
  },
  primaryBtnFull: {
    flex: 1,
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
  },
  primaryLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.3,
  },

  // Loading + error
  centered: {
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
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    lineHeight: 20,
    marginBottom: theme.spacing[4],
    maxWidth: 320,
  },
  retryBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  retryLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
});
