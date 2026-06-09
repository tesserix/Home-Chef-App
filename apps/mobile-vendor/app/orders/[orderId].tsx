import { useMemo } from 'react';
import {
  ActionSheetIOS,
  Alert,
  Linking,
  Platform,
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
import { Skeleton, useToast } from '@homechef/mobile-shared/ui';
import { DietIcon } from '../../components/vendor/DietIcon';
import {
  useOrderDetail,
  type OrderDetail,
  type OrderDetailStatus,
} from '../../hooks/useOrderDetail';
import {
  useOrderAction,
  useUpdateOrderStatus,
} from '../../hooks/useVendorOrders';
import {
  useCancelOrder,
  useCancelOrderItem,
  CANCEL_REASON_LABEL,
  type CancelReason,
} from '../../hooks/useCancelOrder';

// Statuses where the chef can still cancel + refund. picked_up onward
// is out of the chef's hands — cancellation becomes a customer-support
// concern. Matches the backend's `cancellableStatuses` allowlist in
// apps/api/handlers/chef_order_cancel.go.
const CANCELLABLE_STATUSES: ReadonlySet<OrderDetailStatus> = new Set([
  'accepted',
  'preparing',
  'ready',
]);

// CANCEL_REASONS preserves a stable display order across iOS/Android
// action sheets and the destructive Android Alert dialog.
const CANCEL_REASONS: CancelReason[] = [
  'out_of_ingredient',
  'equipment_failure',
  'customer_request',
  'other',
];

// ---- Status display maps -------------------------------------------------------

const STATUS_LABEL: Record<OrderDetailStatus, string> = {
  pending: 'New order',
  accepted: 'Accepted',
  preparing: 'Preparing',
  ready: 'Ready for pickup',
  picked_up: 'Out for delivery',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
  rejected: 'Rejected',
};

const STATUS_DOT: Record<OrderDetailStatus, string> = {
  pending: theme.colors.amber.DEFAULT,
  accepted: theme.colors.info.DEFAULT,
  preparing: theme.colors.amber.DEFAULT,
  ready: theme.colors.herb.DEFAULT,
  picked_up: theme.colors.info.DEFAULT,
  delivered: theme.colors.diet.veg,
  cancelled: theme.colors.destructive.DEFAULT,
  rejected: theme.colors.destructive.DEFAULT,
};

// ---- Helpers ------------------------------------------------------------------

function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleString('en-IN', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
    hour12: true,
  });
}

function formatAddressLines(addr: OrderDetail['deliveryAddress']): string[] {
  if (typeof addr === 'string') {
    return addr.split(/,\s*/).filter(Boolean);
  }
  if (addr && typeof addr === 'object') {
    const a = addr as {
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
    ].filter((l): l is string => !!l && l.trim().length > 0);
  }
  return [];
}

function mapsUrl(query: string): string {
  const encoded = encodeURIComponent(query);
  if (Platform.OS === 'ios') {
    return `http://maps.apple.com/?q=${encoded}`;
  }
  return `https://www.google.com/maps/search/?q=${encoded}`;
}

// ---- Sub-components -----------------------------------------------------------

interface CommandBarProps {
  orderNumber?: string;
  status?: OrderDetailStatus;
  onBack: () => void;
}

function CommandBar({ orderNumber, status, onBack }: CommandBarProps) {
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
            <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} strokeWidth={2} />
          </View>
        )}
      </Pressable>
      <View style={styles.commandTitleBlock}>
        <Text style={styles.commandTitle} numberOfLines={1}>
          {orderNumber ? `#${orderNumber}` : 'Order'}
        </Text>
        {status ? (
          <View style={styles.commandStatusRow}>
            <View
              style={[
                styles.statusDot,
                { backgroundColor: STATUS_DOT[status] ?? theme.colors.ink.muted },
              ]}
            />
            <Text style={styles.commandStatusLabel}>
              {STATUS_LABEL[status] ?? status}
            </Text>
          </View>
        ) : null}
      </View>
      <View style={styles.commandSpacer} />
    </View>
  );
}

interface SectionLabelProps {
  children: string;
}

function SectionLabel({ children }: SectionLabelProps) {
  return <Text style={styles.sectionLabel}>{children}</Text>;
}

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
    <View style={[styles.totalRow, hasBorderBottom && styles.rowBorderBottom]}>
      <Text style={[styles.totalLabel, emphasis && styles.totalLabelStrong]}>
        {label}
      </Text>
      <Text style={[styles.totalValue, emphasis && styles.totalValueStrong]}>
        ₹{value.toLocaleString('en-IN', { minimumFractionDigits: 0 })}
      </Text>
    </View>
  );
}

interface FooterActionsProps {
  status: OrderDetailStatus;
  orderId: string;
  customerName: string;
  total: number;
  disabled: boolean;
  onAccept: () => void;
  onReject: () => void;
  onMarkPreparing: () => void;
  onMarkReady: () => void;
  onCancel: () => void;
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
  onCancel,
}: FooterActionsProps) {
  // Renders the destructive secondary action below the primary
  // status-transition button when the order is cancellable. Kept as
  // a link instead of a same-row button so the primary action stays
  // the visual hero.
  const cancelLink = CANCELLABLE_STATUSES.has(status) ? (
    <Pressable
      onPress={onCancel}
      disabled={disabled}
      hitSlop={8}
      accessibilityRole="button"
      accessibilityLabel="Cancel this order"
      style={styles.cancelLinkWrap}
    >
      <Text style={styles.cancelLinkLabel}>Cancel order</Text>
    </Pressable>
  ) : null;
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
          accessibilityRole="button"
          accessibilityLabel={`Accept ₹${total.toFixed(0)} order from ${customerName}`}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.primaryBtn,
                pressed && { opacity: 0.85 },
                disabled && { opacity: 0.4 },
              ]}
            >
              <Text style={styles.primaryLabel}>Accept</Text>
            </View>
          )}
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
          accessibilityRole="button"
        >
          {({ pressed }) => (
            <View
              style={[
                styles.primaryBtnFull,
                pressed && { opacity: 0.85 },
                disabled && { opacity: 0.4 },
              ]}
            >
              <Text style={styles.primaryLabel}>Mark preparing</Text>
            </View>
          )}
        </Pressable>
        {cancelLink}
      </View>
    );
  }

  if (status === 'preparing') {
    return (
      <View style={styles.footer}>
        <Pressable
          onPress={onMarkReady}
          disabled={disabled}
          accessibilityRole="button"
        >
          {({ pressed }) => (
            <View
              style={[
                styles.primaryBtnFull,
                pressed && { opacity: 0.85 },
                disabled && { opacity: 0.4 },
              ]}
            >
              <Text style={styles.primaryLabel}>Mark ready for pickup</Text>
            </View>
          )}
        </Pressable>
        {cancelLink}
      </View>
    );
  }

  // status === 'ready' — chef has prepped, no more transitions for them;
  // we still let them cancel until the driver picks up.
  if (status === 'ready' && cancelLink) {
    return <View style={styles.footer}>{cancelLink}</View>;
  }

  const caption: Partial<Record<OrderDetailStatus, string>> = {
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

// ---- Loading skeleton ---------------------------------------------------------

function DetailSkeleton() {
  return (
    <View style={styles.skeletonPad}>
      <Skeleton height={32} style={{ width: 160, marginBottom: 8 }} />
      <Skeleton height={16} style={{ width: 100, marginBottom: 32 }} />
      <Skeleton height={14} style={{ width: 80, marginBottom: 12 }} />
      <Skeleton height={44} style={{ marginBottom: 2 }} />
      <Skeleton height={44} style={{ marginBottom: 24 }} />
      <Skeleton height={14} style={{ width: 80, marginBottom: 12 }} />
      <Skeleton height={44} style={{ marginBottom: 2 }} />
      <Skeleton height={44} />
    </View>
  );
}

// ---- Main screen -------------------------------------------------------------

export default function OrderDetailScreen() {
  const { orderId } = useLocalSearchParams<{ orderId: string }>();
  const { data: order, isLoading, isError, refetch } = useOrderDetail(orderId);
  const { triggerAction, isLoading: actionLoading } = useOrderAction();
  const updateStatus = useUpdateOrderStatus();
  const cancelOrder = useCancelOrder(orderId);
  const cancelItem = useCancelOrderItem(orderId);
  const { show: showToast } = useToast();

  // Two-step destructive flow: pick a reason, then confirm. iOS gets the
  // native action sheet (familiar + dismissable by swipe); Android gets
  // an Alert with N buttons (the closest cross-platform analog without
  // pulling in a sheet library). Both end at submitCancel() which fires
  // the mutation + shows a success/error toast.
  function submitCancel(reason: CancelReason): void {
    cancelOrder.mutate(reason, {
      onSuccess: () => {
        showToast({
          message: 'Order cancelled. Customer is being refunded.',
          tone: 'success',
        });
      },
      onError: () => {
        showToast({
          message: 'Could not cancel. Try again.',
          tone: 'error',
        });
      },
    });
  }

  // Per-line cancel. Mirrors the order-level cancel flow but targets a
  // single OrderItem — backend partial-refunds that line + recomputes
  // totals; order status stays in prep so the chef continues with the
  // remaining items.
  function submitItemCancel(itemId: string, itemName: string, reason: CancelReason): void {
    cancelItem.mutate(
      { itemId, reason },
      {
        onSuccess: () => {
          showToast({
            message: `${itemName} marked unavailable. Customer is being refunded.`,
            tone: 'success',
          });
        },
        onError: () => {
          showToast({
            message: 'Could not cancel this item. Try again.',
            tone: 'error',
          });
        },
      },
    );
  }

  function openItemCancelSheet(itemId: string, itemName: string): void {
    if (Platform.OS === 'ios') {
      ActionSheetIOS.showActionSheetWithOptions(
        {
          title: `Can't fulfill "${itemName}"?`,
          message:
            'The customer is refunded for this item only. The rest of the order continues.',
          options: [
            ...CANCEL_REASONS.map((r) => CANCEL_REASON_LABEL[r]),
            'Keep item',
          ],
          cancelButtonIndex: CANCEL_REASONS.length,
          destructiveButtonIndex: CANCEL_REASONS.length - 1,
        },
        (index) => {
          if (index < 0 || index >= CANCEL_REASONS.length) return;
          submitItemCancel(itemId, itemName, CANCEL_REASONS[index]!);
        },
      );
      return;
    }
    Alert.alert(
      `Can't fulfill "${itemName}"?`,
      'The customer is refunded for this item only.',
      [
        { text: 'Keep item', style: 'cancel' },
        {
          text: "Can't fulfill",
          style: 'destructive',
          onPress: () =>
            promptCancelReasonAndroid((r) => submitItemCancel(itemId, itemName, r)),
        },
      ],
    );
  }

  function openCancelSheet(): void {
    if (Platform.OS === 'ios') {
      ActionSheetIOS.showActionSheetWithOptions(
        {
          title: 'Cancel order',
          message: 'The customer will be refunded in full.',
          options: [
            ...CANCEL_REASONS.map((r) => CANCEL_REASON_LABEL[r]),
            'Keep order',
          ],
          cancelButtonIndex: CANCEL_REASONS.length,
          destructiveButtonIndex: CANCEL_REASONS.length - 1,
        },
        (index) => {
          if (index < 0 || index >= CANCEL_REASONS.length) return;
          submitCancel(CANCEL_REASONS[index]!);
        },
      );
      return;
    }
    // Android Alert max 3 buttons reliably — split into a 2-step prompt:
    // first confirm intent, then a follow-up to pick a reason.
    Alert.alert(
      'Cancel this order?',
      'The customer will be refunded in full. Pick a reason on the next screen.',
      [
        { text: 'Keep order', style: 'cancel' },
        {
          text: 'Cancel order',
          style: 'destructive',
          onPress: () => promptCancelReasonAndroid(submitCancel),
        },
      ],
    );
  }

  const addressLines = useMemo(
    () => (order ? formatAddressLines(order.deliveryAddress) : []),
    [order],
  );

  const mapsQuery = useMemo(
    () => addressLines.join(', '),
    [addressLines],
  );

  function handleBack(): void {
    if (router.canGoBack()) router.back();
    else router.replace('/(tabs)/orders');
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <CommandBar onBack={handleBack} />
        <DetailSkeleton />
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
            Check your connection and try again.
          </Text>
          <Pressable
            onPress={() => refetch()}
            accessibilityRole="button"
          >
            {({ pressed }) => (
              <View style={[styles.retryBtn, pressed && { opacity: 0.85 }]}>
                <Text style={styles.retryLabel}>Retry</Text>
              </View>
            )}
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  const { timing, pricing } = order;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <CommandBar
        orderNumber={order.orderNumber}
        status={order.status}
        onBack={handleBack}
      />

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* CUSTOMER section */}
        <SectionLabel>CUSTOMER</SectionLabel>
        <View style={styles.hairlineGroup}>
          <View style={styles.customerRow}>
            <View style={styles.customerTextBlock}>
              <Text style={styles.customerName} numberOfLines={1}>
                {order.customerName || 'Customer'}
              </Text>
              {order.customerPhone ? (
                <Text style={styles.customerPhone} numberOfLines={1}>
                  {order.customerPhone}
                </Text>
              ) : null}
            </View>
            {order.customerPhone ? (
              <View style={styles.contactButtons}>
                <Pressable
                  onPress={() => Linking.openURL(`tel:${order.customerPhone}`)}
                  accessibilityRole="button"
                  accessibilityLabel={`Call ${order.customerName}`}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.contactBtn,
                        pressed && { backgroundColor: theme.colors.bone },
                      ]}
                    >
                      <Text style={styles.contactBtnLabel}>Call</Text>
                    </View>
                  )}
                </Pressable>
                <Pressable
                  onPress={() => Linking.openURL(`sms:${order.customerPhone}`)}
                  accessibilityRole="button"
                  accessibilityLabel={`Message ${order.customerName}`}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.contactBtn,
                        styles.contactBtnSecondary,
                        pressed && { opacity: 0.7 },
                      ]}
                    >
                      <Text style={styles.contactBtnLabelSecondary}>
                        Message
                      </Text>
                    </View>
                  )}
                </Pressable>
              </View>
            ) : null}
          </View>
        </View>

        {/* ITEMS section */}
        <SectionLabel>ITEMS</SectionLabel>
        <View style={styles.hairlineGroup}>
          {order.items.length === 0 ? (
            <Text style={styles.bodyMuted}>No items recorded</Text>
          ) : (
            order.items.map((item, idx) => {
              // A line is per-line-cancellable while the whole order is
              // cancellable AND this specific line hasn't been struck
              // already. Backend rejects late attempts; this just avoids
              // surfacing a button that would error.
              const itemCancellable =
                CANCELLABLE_STATUSES.has(order.status) && !item.isCancelled;
              return (
                <View
                  key={item.id || `${item.name}-${idx}`}
                  style={[
                    styles.itemRow,
                    idx < order.items.length - 1 && styles.rowBorderBottom,
                    item.isCancelled && styles.itemRowCancelled,
                  ]}
                >
                  <DietIcon
                    kind={
                      item.isVeg === true
                        ? 'veg'
                        : item.isVeg === false
                          ? 'non-veg'
                          : 'unknown'
                    }
                    size={12}
                  />
                  <View style={styles.itemBody}>
                    <Text
                      style={[
                        styles.itemName,
                        item.isCancelled && styles.itemNameCancelled,
                      ]}
                      numberOfLines={2}
                    >
                      {item.quantity > 1
                        ? `${item.quantity} × ${item.name}`
                        : item.name}
                    </Text>
                    {item.specialInstructions ? (
                      <Text style={styles.itemNote} numberOfLines={2}>
                        {item.specialInstructions}
                      </Text>
                    ) : null}
                    {item.isCancelled ? (
                      <Text style={styles.itemCancelledBadge}>
                        Refunded ₹
                        {(item.refundAmount ?? 0).toLocaleString('en-IN', {
                          maximumFractionDigits: 2,
                        })}
                      </Text>
                    ) : (
                      <Text style={styles.itemUnitPrice}>
                        ₹{item.unitPrice.toLocaleString('en-IN')} each
                      </Text>
                    )}
                    {itemCancellable ? (
                      <Pressable
                        onPress={() => openItemCancelSheet(item.id, item.name)}
                        disabled={cancelItem.isPending}
                        hitSlop={6}
                        accessibilityRole="button"
                        accessibilityLabel={`Mark ${item.name} as unfulfillable`}
                        style={styles.itemCancelLinkWrap}
                      >
                        <Text style={styles.itemCancelLinkLabel}>
                          Can't fulfill this item
                        </Text>
                      </Pressable>
                    ) : null}
                  </View>
                  <Text
                    style={[
                      styles.itemLineTotal,
                      item.isCancelled && styles.itemLineTotalCancelled,
                    ]}
                  >
                    ₹{item.lineTotal.toLocaleString('en-IN')}
                  </Text>
                </View>
              );
            })
          )}
        </View>

        {/* DELIVERY ADDRESS section */}
        {addressLines.length > 0 ? (
          <>
            <SectionLabel>DELIVERY ADDRESS</SectionLabel>
            <Pressable
              onPress={() =>
                mapsQuery ? Linking.openURL(mapsUrl(mapsQuery)) : undefined
              }
              accessibilityRole="button"
              accessibilityLabel="Open in Maps"
              disabled={!mapsQuery}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.hairlineGroup,
                    styles.addressGroup,
                    pressed && { backgroundColor: theme.colors.bone },
                  ]}
                >
                  {addressLines.map((line) => (
                    <Text key={line} style={styles.addressLine}>
                      {line}
                    </Text>
                  ))}
                  {mapsQuery ? (
                    <Text style={styles.mapsLink}>Open in Maps →</Text>
                  ) : null}
                </View>
              )}
            </Pressable>
            {order.deliveryInstructions ? (
              <View style={styles.deliveryInstructionsWrap}>
                <Text style={styles.deliveryInstructionsText}>
                  {order.deliveryInstructions}
                </Text>
              </View>
            ) : null}
          </>
        ) : null}

        {/* SPECIAL INSTRUCTIONS */}
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

        {/* TIMING section */}
        <SectionLabel>TIMING</SectionLabel>
        <View style={styles.hairlineGroup}>
          {([
            ['Ordered', timing.orderedAt],
            ['Accepted', timing.acceptedAt],
            ['Prepared', timing.preparedAt],
            ['Picked up', timing.pickedUpAt],
            ['Delivered', timing.deliveredAt],
          ] as [string, string | null | undefined][])
            .filter(([, ts]) => !!ts)
            .map(([label, ts], idx, arr) => (
              <View
                key={label}
                style={[
                  styles.timingRow,
                  idx < arr.length - 1 && styles.rowBorderBottom,
                ]}
              >
                <Text style={styles.timingLabel}>{label}</Text>
                <Text style={styles.timingValue}>
                  {formatDateTime(ts ?? '')}
                </Text>
              </View>
            ))}
        </View>

        {/* PRICING section */}
        <SectionLabel>PRICING</SectionLabel>
        <View style={styles.hairlineGroup}>
          <TotalRow label="Subtotal" value={pricing.subtotal} />
          {pricing.deliveryFee > 0 ? (
            <TotalRow label="Delivery fee" value={pricing.deliveryFee} />
          ) : null}
          {pricing.serviceFee > 0 ? (
            <TotalRow label="Service fee" value={pricing.serviceFee} />
          ) : null}
          {pricing.tax > 0 ? (
            <TotalRow label="Tax" value={pricing.tax} />
          ) : null}
          {pricing.chefTip > 0 ? (
            <TotalRow label="Tip" value={pricing.chefTip} />
          ) : null}
          <TotalRow
            label="Total"
            value={pricing.total}
            emphasis
            hasBorderBottom={false}
          />
        </View>

        {/* Bottom padding for footer */}
        <View style={{ height: theme.spacing[10] }} />
      </ScrollView>

      <FooterActions
        status={order.status}
        orderId={order.id}
        customerName={order.customerName || 'this customer'}
        total={pricing.total}
        disabled={actionLoading || updateStatus.isPending || cancelOrder.isPending}
        onAccept={() => triggerAction(order.id, 'accepted')}
        onReject={() => triggerAction(order.id, 'rejected')}
        onMarkPreparing={() =>
          updateStatus.mutate({ orderId: order.id, status: 'preparing' })
        }
        onMarkReady={() =>
          updateStatus.mutate({ orderId: order.id, status: 'ready' })
        }
        onCancel={openCancelSheet}
      />
    </SafeAreaView>
  );
}

// promptCancelReasonAndroid emulates an action sheet with a chained
// Alert for the reason picker. We split out a function so the cancel
// flow inside the screen stays readable.
function promptCancelReasonAndroid(submit: (r: CancelReason) => void): void {
  // First two reasons + "Other" / "More…" because Android Alert
  // caps reliably at 3 buttons. "More" chains a second prompt with
  // the remaining options.
  Alert.alert('Why?', '', [
    {
      text: CANCEL_REASON_LABEL.out_of_ingredient,
      onPress: () => submit('out_of_ingredient'),
    },
    {
      text: CANCEL_REASON_LABEL.equipment_failure,
      onPress: () => submit('equipment_failure'),
    },
    {
      text: 'More…',
      onPress: () =>
        Alert.alert('Why?', '', [
          {
            text: CANCEL_REASON_LABEL.customer_request,
            onPress: () => submit('customer_request'),
          },
          {
            text: CANCEL_REASON_LABEL.other,
            onPress: () => submit('other'),
          },
          { text: 'Back', style: 'cancel' },
        ]),
    },
  ]);
}

// ---- Styles ------------------------------------------------------------------

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
  commandTitleBlock: { flex: 1 },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    lineHeight: 28,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },
  commandStatusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    marginTop: 2,
  },
  commandStatusLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  statusDot: { width: 6, height: 6, borderRadius: 3 },
  commandSpacer: { width: 32 },

  // Scroll
  scroll: { flex: 1 },
  scrollContent: { paddingBottom: theme.spacing[10] },

  // Section label
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[5],
    paddingBottom: theme.spacing[2],
  },

  // Hairline group wrapper
  hairlineGroup: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  // Shared bottom hairline for rows inside a group
  rowBorderBottom: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  // CUSTOMER
  customerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 56,
    gap: theme.spacing[3],
  },
  customerTextBlock: { flex: 1 },
  customerName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  customerPhone: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    marginTop: 2,
    fontVariant: ['tabular-nums'],
  },
  contactButtons: {
    flexDirection: 'row',
    gap: theme.spacing[2],
  },
  contactBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    minHeight: 36,
    minWidth: 60,
    alignItems: 'center',
    justifyContent: 'center',
  },
  contactBtnSecondary: {
    backgroundColor: theme.colors.paper,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
  },
  contactBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },
  contactBtnLabelSecondary: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: 0.2,
  },

  // ITEMS
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
  itemUnitPrice: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 3,
    fontVariant: ['tabular-nums'],
  },
  itemLineTotal: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    paddingTop: 2,
  },

  // DELIVERY ADDRESS
  addressGroup: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  addressLine: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 22,
  },
  mapsLink: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.DEFAULT,
    marginTop: theme.spacing[2],
    letterSpacing: 0.1,
  },
  deliveryInstructionsWrap: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[3],
  },
  deliveryInstructionsText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
  },

  // SPECIAL INSTRUCTIONS
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

  // TIMING
  timingRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 40,
    gap: theme.spacing[3],
  },
  timingLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  timingValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    textAlign: 'right',
    flex: 1,
  },

  // PRICING / totals
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

  // Fallback body
  bodyMuted: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },

  // Loading skeleton
  skeletonPad: {
    padding: theme.spacing[4],
    gap: theme.spacing[2],
  },

  // Error state
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
  // Cancel-order destructive link — secondary affordance below the
  // primary status-transition button. Underlined so it reads as an
  // action, not just labelled text; coloured destructive to flag the
  // tone before tap.
  cancelLinkWrap: {
    alignSelf: 'center',
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
  },
  cancelLinkLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.destructive.DEFAULT,
    textDecorationLine: 'underline',
  },
  // Per-line cancel — smaller, lower-affordance link inside the item
  // row. The whole-order cancel is the loud one; this is for the chef
  // mid-prep who only needs to drop one line.
  itemCancelLinkWrap: {
    marginTop: 4,
    paddingVertical: 4,
    alignSelf: 'flex-start',
  },
  itemCancelLinkLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
    textDecorationLine: 'underline',
    letterSpacing: 0.2,
  },
  // Cancelled-line presentation — dimmed background, strikethrough on
  // text + price so the chef visually parses "this line is dead".
  itemRowCancelled: {
    opacity: 0.6,
    backgroundColor: theme.colors.bone,
  },
  itemNameCancelled: {
    textDecorationLine: 'line-through',
    color: theme.colors.ink.muted,
  },
  itemLineTotalCancelled: {
    textDecorationLine: 'line-through',
    color: theme.colors.ink.muted,
  },
  itemCancelledBadge: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
    marginTop: 2,
    letterSpacing: 0.2,
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
});
