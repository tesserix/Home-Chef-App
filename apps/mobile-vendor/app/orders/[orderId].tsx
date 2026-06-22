import { useEffect, useMemo, useState } from 'react';
import {
  ActionSheetIOS,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import * as FileSystem from 'expo-file-system/legacy';
import * as Sharing from 'expo-sharing';
import * as SecureStore from 'expo-secure-store';
import * as ImagePicker from 'expo-image-picker';
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
  type FulfillmentType,
} from '../../hooks/useOrderDetail';
import {
  useOrderAction,
  useUpdateOrderStatus,
  useUploadOrderPhoto,
  type OrderPhotoKind,
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

// Pickup orders have no driver leg, so the terminal `delivered` status reads
// "Collected" and the "ready" state means "ready for the customer to collect".
function statusLabelFor(
  status: OrderDetailStatus,
  fulfillment: FulfillmentType,
): string {
  if (fulfillment === 'pickup') {
    if (status === 'delivered' || status === 'picked_up') return 'Collected';
  }
  return STATUS_LABEL[status] ?? status;
}

// Status chip palette per UI-V2-SPEC §2: tint bg + dark text of same hue.
interface StatusChipColors {
  bg: string;
  text: string;
}

const STATUS_CHIP: Record<OrderDetailStatus, StatusChipColors> = {
  pending: { bg: theme.colors.mist.DEFAULT, text: theme.colors.ink.DEFAULT },
  preparing: { bg: theme.colors.mist.DEFAULT, text: theme.colors.ink.DEFAULT },
  ready: { bg: theme.colors.success.tint, text: theme.colors.success.soft },
  accepted: { bg: theme.colors.info.tint, text: theme.colors.info.DEFAULT },
  picked_up: { bg: theme.colors.mist.DEFAULT, text: theme.colors.diet.veg },
  delivered: { bg: theme.colors.mist.DEFAULT, text: theme.colors.diet.veg },
  cancelled: {
    bg: theme.colors.destructive.tint,
    text: theme.colors.destructive.DEFAULT,
  },
  rejected: {
    bg: theme.colors.destructive.tint,
    text: theme.colors.destructive.DEFAULT,
  },
};

const STATUS_CHIP_FALLBACK: StatusChipColors = {
  bg: theme.colors.mist.DEFAULT,
  text: theme.colors.ink.soft,
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

// ---- Sub-components -----------------------------------------------------------

interface CommandBarProps {
  orderNumber?: string;
  status?: OrderDetailStatus;
  fulfillmentType?: FulfillmentType;
  onBack: () => void;
}

function CommandBar({
  orderNumber,
  status,
  fulfillmentType = 'delivery',
  onBack,
}: CommandBarProps) {
  const chip = status ? (STATUS_CHIP[status] ?? STATUS_CHIP_FALLBACK) : null;
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
        {status && chip ? (
          <View style={styles.commandStatusRow}>
            <View style={[styles.statusChip, { backgroundColor: chip.bg }]}>
              <Text style={[styles.statusChipLabel, { color: chip.text }]}>
                {statusLabelFor(status, fulfillmentType)}
              </Text>
            </View>
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
  fulfillmentType: FulfillmentType;
  orderId: string;
  customerName: string;
  total: number;
  disabled: boolean;
  /** ISO timestamp of the last status transition — used to compute
   *  waiting-for-driver elapsed time in the `ready` state. */
  updatedAt?: string;
  onAccept: () => void;
  onReject: () => void;
  onMarkPreparing: () => void;
  onMarkReady: () => void;
  onMarkHandedOver: () => void;
  onMarkOutForDelivery: () => void;
  onMarkDelivered: () => void;
  onCancel: () => void;
}

function FooterActions({
  status,
  fulfillmentType,
  orderId,
  customerName,
  total,
  disabled,
  updatedAt,
  onAccept,
  onReject,
  onMarkPreparing,
  onMarkReady,
  onMarkHandedOver,
  onMarkOutForDelivery,
  onMarkDelivered,
  onCancel,
}: FooterActionsProps) {
  const isPickup = fulfillmentType === 'pickup';
  const isChefDelivery = fulfillmentType === 'chef_delivery';
  // Ticks every 45 s so the "ready" caption's elapsed counter updates.
  // Only runs when status === 'ready'; clears on status change or unmount.
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (status !== 'ready') return;
    const id = setInterval(() => setNow(Date.now()), 45_000);
    return () => clearInterval(id);
  }, [status]);

  const readyElapsed = (() => {
    if (status !== 'ready') return '';
    const ts = updatedAt;
    if (!ts) return '';
    const mins = Math.max(0, Math.floor((now - new Date(ts).getTime()) / 60_000));
    return mins < 1 ? '' : ` · ${mins}m`;
  })();
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
          accessibilityRole="button"
          accessibilityLabel={`Reject order from ${customerName}`}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.rejectBtn,
                pressed && { backgroundColor: theme.colors.bone },
                disabled && { opacity: 0.4 },
              ]}
            >
              <Text style={styles.rejectLabel}>Reject</Text>
            </View>
          )}
        </Pressable>
        <Pressable
          onPress={onAccept}
          disabled={disabled}
          style={styles.flex1}
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
          style={styles.flex1}
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
          style={styles.flex1}
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

  // status === 'ready'
  //  • Pickup: the customer comes to collect, so the chef completes the order
  //    by tapping "Mark handed over" (ready → delivered, relabeled "Collected").
  //  • Delivery: no chef transition — a 3PL driver collects. Show the elapsed
  //    wait time so the chef knows how long the driver is taking.
  if (status === 'ready') {
    if (isPickup) {
      return (
        <View style={styles.footer}>
          <Pressable
            onPress={onMarkHandedOver}
            disabled={disabled}
            style={styles.flex1}
            accessibilityRole="button"
            accessibilityLabel={`Mark order handed over to ${customerName}`}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.primaryBtnFull,
                  pressed && { opacity: 0.85 },
                  disabled && { opacity: 0.4 },
                ]}
              >
                <Text style={styles.primaryLabel}>Mark handed over</Text>
              </View>
            )}
          </Pressable>
          {cancelLink}
        </View>
      );
    }
    if (isChefDelivery) {
      // The chef delivers themselves → they advance the order out for delivery.
      return (
        <View style={styles.footer}>
          <Pressable
            onPress={onMarkOutForDelivery}
            disabled={disabled}
            style={styles.flex1}
            accessibilityRole="button"
            accessibilityLabel="Mark order out for delivery"
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.primaryBtnFull,
                  pressed && { opacity: 0.85 },
                  disabled && { opacity: 0.4 },
                ]}
              >
                <Text style={styles.primaryLabel}>Out for delivery</Text>
              </View>
            )}
          </Pressable>
          {cancelLink}
        </View>
      );
    }
    return (
      <View style={[styles.footer, styles.footerCaptionWrap]}>
        <Text style={styles.footerCaption}>{`Waiting for driver to pick up${readyElapsed}.`}</Text>
        {cancelLink}
      </View>
    );
  }

  // status === 'picked_up' — for chef_delivery the chef is en route and
  // completes the order on arrival. (3PL orders are driver-controlled here.)
  if (status === 'picked_up' && isChefDelivery) {
    return (
      <View style={styles.footer}>
        <Pressable
          onPress={onMarkDelivered}
          disabled={disabled}
          style={styles.flex1}
          accessibilityRole="button"
          accessibilityLabel={`Mark order delivered to ${customerName}`}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.primaryBtnFull,
                pressed && { opacity: 0.85 },
                disabled && { opacity: 0.4 },
              ]}
            >
              <Text style={styles.primaryLabel}>Mark delivered</Text>
            </View>
          )}
        </Pressable>
      </View>
    );
  }

  // Delivered — chef can download the GST invoice for their books +
  // forward it to the customer if asked. PDF is generated server-side
  // by services.GenerateOrderInvoicePDF and streamed via /chef/orders/
  // :orderId/invoice.pdf.
  if (status === 'delivered') {
    return (
      <View style={[styles.footer, styles.footerCaptionWrap]}>
        <Text style={styles.footerCaption}>
          {isPickup ? 'Collected by the customer.' : 'Delivered to customer.'}
        </Text>
        <Pressable
          onPress={() => downloadInvoice(orderId)}
          hitSlop={6}
          accessibilityRole="button"
          accessibilityLabel="Download invoice PDF"
          style={styles.cancelLinkWrap}
        >
          <Text style={styles.invoiceLinkLabel}>Download invoice (PDF)</Text>
        </Pressable>
      </View>
    );
  }

  const caption: Partial<Record<OrderDetailStatus, string>> = {
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
  const uploadPhoto = useUploadOrderPhoto();
  const cancelOrder = useCancelOrder(orderId);
  const cancelItem = useCancelOrderItem(orderId);
  const { show: showToast } = useToast();

  // A lifecycle photo is REQUIRED before these transitions: the chef captures
  // the prepared dish (ready) / the handoff (handover), it uploads, and only
  // then does the status advance. A cancel or a failed upload leaves the order
  // in its current state so the kitchen is never blocked — the chef just
  // retries. Camera-first (it's proof), falling back to the library if the
  // camera permission is denied or unavailable.
  async function captureAndAdvance(
    kind: OrderPhotoKind,
    nextStatus: 'ready' | 'delivered',
  ): Promise<void> {
    if (!order || uploadPhoto.isPending || updateStatus.isPending) return;

    let asset: ImagePicker.ImagePickerAsset | undefined;
    const cam = await ImagePicker.requestCameraPermissionsAsync();
    if (cam.granted) {
      const shot = await ImagePicker.launchCameraAsync({
        mediaTypes: ['images'],
        quality: 0.6,
      });
      if (shot.canceled) return;
      asset = shot.assets[0];
    } else {
      const lib = await ImagePicker.launchImageLibraryAsync({
        mediaTypes: ['images'],
        quality: 0.6,
      });
      if (lib.canceled) return;
      asset = lib.assets[0];
    }
    if (!asset?.uri) return;

    try {
      await uploadPhoto.mutateAsync({ orderId: order.id, kind, uri: asset.uri });
      await updateStatus.mutateAsync({ orderId: order.id, status: nextStatus });
    } catch {
      showToast({
        message: 'Could not upload the photo. Please try again.',
        tone: 'error',
      });
    }
  }

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
  const isPickup = order.fulfillmentType === 'pickup';
  const isChefDelivery = order.fulfillmentType === 'chef_delivery';

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <CommandBar
        orderNumber={order.orderNumber}
        status={order.status}
        fulfillmentType={order.fulfillmentType}
        onBack={handleBack}
      />

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* CUSTOMER section — first name only. Phone + direct call/message are
            intentionally not shown: the rider handles pickup→delivery, and
            withholding contact prevents off-platform arrangements. */}
        <SectionLabel>CUSTOMER</SectionLabel>
        <View style={styles.card}>
          <View style={styles.customerRow}>
            <View style={styles.customerTextBlock}>
              <Text style={styles.customerName} numberOfLines={1}>
                {order.customerName || 'Customer'}
              </Text>
            </View>
          </View>
        </View>

        {/* ITEMS section */}
        <SectionLabel>ITEMS</SectionLabel>
        <View style={styles.card}>
          <View style={styles.cardClip}>
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
        </View>

        {/* FULFILLMENT section.
            • Pickup: the customer collects from the chef's kitchen — no delivery
              area to show; the chef just needs to know it's a pickup.
            • Delivery: area only (city/state), never the full street address. The
              exact address + navigation notes go to the rider, not the chef, so
              customer and chef can't arrange off-platform delivery. */}
        {isPickup ? (
          <>
            <SectionLabel>PICKUP</SectionLabel>
            <View style={styles.card}>
              <View style={[styles.cardClip, styles.addressGroup]}>
                <Text style={styles.addressLine}>
                  Customer collects from your kitchen
                </Text>
                <Text style={styles.areaReassurance}>
                  The customer will come to you. Tap “Mark handed over” once
                  they’ve collected the order.
                </Text>
              </View>
            </View>
          </>
        ) : isChefDelivery ? (
          // Chef self-delivery: the chef delivers, so they DO get the full
          // address + phone to reach the customer's door (scoped exception to
          // the area-only privacy rule).
          <>
            <SectionLabel>DELIVER TO</SectionLabel>
            <View style={styles.card}>
              <View style={[styles.cardClip, styles.addressGroup]}>
                <Text style={styles.addressLine}>
                  {addressLines.length > 0
                    ? addressLines.join('\n')
                    : 'Address on file'}
                </Text>
                {order.customerPhone ? (
                  <Text style={styles.areaReassurance}>
                    {order.customerName || 'Customer'} · {order.customerPhone}
                  </Text>
                ) : null}
                <Text style={styles.areaReassurance}>
                  You’re delivering this order yourself.
                </Text>
              </View>
            </View>
          </>
        ) : (
          <>
            <SectionLabel>DELIVERY AREA</SectionLabel>
            <View style={styles.card}>
              <View style={[styles.cardClip, styles.addressGroup]}>
                <Text style={styles.addressLine}>
                  {addressLines.length > 0
                    ? addressLines.join(' · ')
                    : 'Delivery area on file'}
                </Text>
                <Text style={styles.areaReassurance}>
                  Your rider will collect and deliver this order.
                </Text>
              </View>
            </View>
          </>
        )}

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
        <View style={styles.card}>
          {([
            ['Ordered', timing.orderedAt],
            ['Accepted', timing.acceptedAt],
            ['Prepared', timing.preparedAt],
            ['Picked up', timing.pickedUpAt],
            [isPickup ? 'Collected' : 'Delivered', timing.deliveredAt],
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
        <View style={styles.card}>
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
        fulfillmentType={order.fulfillmentType}
        orderId={order.id}
        customerName={order.customerName || 'this customer'}
        total={pricing.total}
        disabled={
          actionLoading ||
          updateStatus.isPending ||
          uploadPhoto.isPending ||
          cancelOrder.isPending
        }
        updatedAt={order.timing.preparedAt ?? order.timing.orderedAt}
        onAccept={() => triggerAction(order.id, 'accepted')}
        onReject={() => triggerAction(order.id, 'rejected')}
        onMarkPreparing={() =>
          updateStatus.mutate({ orderId: order.id, status: 'preparing' })
        }
        onMarkReady={() => captureAndAdvance('ready', 'ready')}
        onMarkHandedOver={() => captureAndAdvance('handover', 'delivered')}
        onMarkOutForDelivery={() =>
          updateStatus.mutate({ orderId: order.id, status: 'picked_up' })
        }
        onMarkDelivered={() =>
          updateStatus.mutate({ orderId: order.id, status: 'delivered' })
        }
        onCancel={openCancelSheet}
      />
    </SafeAreaView>
  );
}

// downloadInvoice fetches the chef-side PDF invoice with the chef's
// auth token, saves it to a cache file, and opens the system share
// sheet so the chef can save to Files / forward to the customer /
// email it. Side-stepped having to embed the token in a URL by using
// FileSystem.downloadAsync with an explicit Authorization header.
async function downloadInvoice(orderId: string): Promise<void> {
  try {
    const token = await SecureStore.getItemAsync('access_token');
    if (!token) {
      Alert.alert('Sign in required', 'Sign in again to download invoices.');
      return;
    }
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? '';
    const url = `${apiBase}/chef/orders/${orderId}/invoice.pdf`;
    const target = `${FileSystem.cacheDirectory}invoice-${orderId}.pdf`;
    const dl = await FileSystem.downloadAsync(url, target, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (dl.status !== 200) {
      Alert.alert('Could not download invoice', `Server returned ${dl.status}.`);
      return;
    }
    if (await Sharing.isAvailableAsync()) {
      await Sharing.shareAsync(dl.uri, { mimeType: 'application/pdf', dialogTitle: 'Invoice' });
    } else {
      Alert.alert('Saved', `Invoice saved to ${dl.uri}`);
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Download failed.';
    Alert.alert('Could not download invoice', msg);
  }
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
  root: { flex: 1, backgroundColor: theme.colors.bone },
  flex1: { flex: 1 },

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
    marginTop: theme.spacing[1],
  },
  // Status chip (UI-V2-SPEC §2) — tint bg pill with dark same-hue text.
  statusChip: {
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 3,
    borderRadius: theme.radius.full,
  },
  statusChipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
  },
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
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[2],
  },

  // White group card on the bone canvas (UI-V2-SPEC §1)
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[4],
    ...theme.shadow[1],
  },
  // Inner clip layer — keeps row backgrounds (pressed/cancelled fills)
  // inside the card radius without clipping the outer shadow on iOS.
  cardClip: {
    borderRadius: theme.radius.lg,
    overflow: 'hidden',
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
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 22,
  },
  areaReassurance: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
    marginTop: 4,
  },

  // SPECIAL INSTRUCTIONS
  instructionsCallout: {
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[4],
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
  },
  instructionsText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
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
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
  },
  retryLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },

  // Footer — white action bar lifted off the canvas with a top shadow
  // (UI-V2-SPEC §6-style elevation, no hairline).
  footer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[5],
    backgroundColor: theme.colors.paper,
    shadowColor: theme.colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: 0.06,
    shadowRadius: 12,
    elevation: 8,
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
  // Ghost Reject button (~96 wide) beside the full-flex Accept primary
  // (UI-V2-SPEC §3).
  rejectBtn: {
    width: 96,
    minHeight: 52,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.paper,
    alignItems: 'center',
    justifyContent: 'center',
  },
  rejectLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: 0.1,
  },
  // Cancel-order link — herb text link, no underline (UI-V2-SPEC §3).
  cancelLinkWrap: {
    alignSelf: 'center',
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
  },
  cancelLinkLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
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
    letterSpacing: 0.2,
  },
  invoiceLinkLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    marginTop: 4,
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
    borderRadius: theme.radius.md,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
  },
  primaryBtnFull: {
    flex: 1,
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    minHeight: 52,
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
