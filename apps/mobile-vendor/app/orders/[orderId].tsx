import { useEffect, useMemo, useState } from 'react';
import {
  ActionSheetIOS,
  Alert,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import * as FileSystem from 'expo-file-system/legacy';
import * as Sharing from 'expo-sharing';
import * as SecureStore from 'expo-secure-store';
import * as ImagePicker from 'expo-image-picker';
import * as Device from 'expo-device';
import * as Haptics from 'expo-haptics';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { Check, ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView, Skeleton, useToast } from '@homechef/mobile-shared/ui';
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
import {
  useReportDeliveryFailure,
  DELIVERY_FAILURE_REASONS,
  DELIVERY_FAILURE_REASON_LABEL,
  type DeliveryFailureReason,
} from '../../hooks/useReportDeliveryFailure';

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

// Carrier-aware status wording. The generic STATUS_LABEL only fits pickup
// orders; delivery/chef-delivery need their own copy so a chef-delivery order
// never reads "awaiting pickup"/"awaiting driver":
//  • pickup: terminal reads "Collected", ready reads "Ready for pickup".
//  • chef_delivery: ready reads "Ready", picked_up reads "Out for delivery".
//  • delivery (3PL): ready reads "Ready · awaiting rider", picked_up reads
//    "Picked up by rider".
function statusLabelFor(
  status: OrderDetailStatus,
  fulfillment: FulfillmentType,
): string {
  if (fulfillment === 'pickup') {
    if (status === 'delivered' || status === 'picked_up') return 'Collected';
    if (status === 'ready') return 'Ready for pickup';
  }
  if (status === 'ready') {
    if (fulfillment === 'chef_delivery') return 'Ready';
    if (fulfillment === 'delivery') return 'Ready · awaiting rider';
  }
  if (status === 'picked_up') {
    if (fulfillment === 'chef_delivery') return 'Out for delivery';
    if (fulfillment === 'delivery') return 'Picked up by rider';
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

// R7: light haptic on every chef-initiated status advance (Mark preparing,
// Mark ready, Out for delivery, Mark handed over, Mark delivered). Accept/
// reject already get their own (medium) impact from useOrderAction — this
// covers the rest of the lifecycle. Guarded with .catch: impactAsync can
// reject on a simulator with no haptic engine, and that must never surface
// as an unhandled rejection or block the actual status mutation.
function fireHaptic(): void {
  Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light).catch(() => {});
}

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
        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
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

interface ChipProps {
  label: string;
  selected: boolean;
  onPress: () => void;
  /** Fill the row evenly with its siblings (the offset row). */
  grow?: boolean;
}

/**
 * Single-select chip for the propose-a-time control (#715). Kept local next to
 * SectionLabel/TotalRow — same convention as the rest of this screen's small
 * pieces. 44px min height so it clears the vendor touch-target floor.
 */
function Chip({ label, selected, onPress, grow }: ChipProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="radio"
      accessibilityState={{ selected }}
      accessibilityLabel={label}
      style={[styles.chip, grow && styles.chipGrow, selected && styles.chipSelected]}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      <Text style={[styles.chipText, selected && styles.chipTextSelected]}>{label}</Text>
    </Pressable>
  );
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

interface TimelineStep {
  label: string;
  timestamp?: string | null;
}

interface OrderStatusTimelineProps {
  steps: TimelineStep[];
}

/**
 * Read-only status timeline (Task 9): Ordered → Accepted → Preparing →
 * Ready → Out for delivery → Delivered/Collected, each row stamped with the
 * real transition time. Purely a receipt — nothing here is pressable and
 * nothing mutates order state.
 *
 * Only steps the order data already carries are rendered; a step with no
 * timestamp is omitted entirely rather than shown as a greyed "upcoming"
 * placeholder (unlike ActiveOrderCard's stepper, this is a history log, not
 * a progress indicator — the backend has no "started preparing" timestamp
 * today, so "Preparing" never renders, and that is correct, not a bug).
 */
function OrderStatusTimeline({ steps }: OrderStatusTimelineProps) {
  const known = steps.filter((s): s is TimelineStep & { timestamp: string } => !!s.timestamp);
  if (known.length === 0) {
    return <Text style={styles.bodyMuted}>No timeline yet</Text>;
  }
  return (
    <View accessibilityRole="none">
      {known.map((step, idx) => {
        const isLast = idx === known.length - 1;
        const isReady = step.label === 'Ready';
        const dotColor = isReady ? theme.colors.success.DEFAULT : theme.colors.ink.DEFAULT;
        return (
          <View key={step.label} style={styles.timelineRow}>
            <View style={styles.timelineRail}>
              <View style={[styles.timelineDot, { backgroundColor: dotColor }]}>
                <Check size={10} color={theme.colors.paper} strokeWidth={3} />
              </View>
              {!isLast && <View style={styles.timelineConnector} />}
            </View>
            <View style={[styles.timelineContent, !isLast && styles.rowBorderBottom]}>
              <Text style={styles.timelineLabel}>{step.label}</Text>
              <Text style={styles.timelineTimestamp}>
                {formatDateTime(step.timestamp)}
              </Text>
            </View>
          </View>
        );
      })}
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
  /** An open delivery-failure review — closes the order off (no actions, an
   *  "under review" caption) until an admin confirms fault (#393). */
  deliveryFailureReported: boolean;
  /** ISO timestamp of the last status transition — used to compute
   *  waiting-for-driver elapsed time in the `ready` state. */
  updatedAt?: string;
  /** True when this is a `delivery` order the chef could self-deliver — the
   *  Mark-Ready footer then offers the I'll-deliver vs hand-to-rider choice. */
  canSelfDeliver: boolean;
  /** True when a 3PL provider is live — i.e. "hand to a rider" is a real option.
   *  When false the rider buttons/links are hidden (3PL dark, chef-only). */
  riderAvailable: boolean;
  /** True when the drop is beyond the chef's self-delivery radius — visually
   *  recommends "Hand to a rider" at the carrier choice. */
  overRange: boolean;
  selfDeliveryDistanceKm: number;
  selfDeliveryMaxDistanceKm: number;
  onAccept: () => void;
  onReject: () => void;
  onMarkPreparing: () => void;
  onMarkReady: () => void;
  /** Mark Ready carrying the chef's carrier choice (self-delivery chefs). */
  onReadyCarrier: (carrier: 'chef_delivery' | 'delivery') => void;
  /** Flip deliver↔rider while the order is still `ready` (no new photo). */
  onSwitchCarrier: (carrier: 'chef_delivery' | 'delivery') => void;
  onMarkHandedOver: () => void;
  onMarkOutForDelivery: () => void;
  onMarkDelivered: () => void;
  onCancel: () => void;
  /** Self-delivery chef reports they couldn't deliver (chef_delivery only). */
  onReportDeliveryFailure: () => void;
}

function FooterActions({
  status,
  fulfillmentType,
  orderId,
  customerName,
  total,
  disabled,
  updatedAt,
  canSelfDeliver,
  riderAvailable,
  overRange,
  selfDeliveryDistanceKm,
  selfDeliveryMaxDistanceKm,
  onAccept,
  onReject,
  onMarkPreparing,
  onMarkReady,
  onReadyCarrier,
  onSwitchCarrier,
  onMarkHandedOver,
  onMarkOutForDelivery,
  onMarkDelivered,
  onCancel,
  onReportDeliveryFailure,
  deliveryFailureReported,
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
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      <Text style={styles.cancelLinkLabel}>Cancel order</Text>
    </Pressable>
  ) : null;
  // "Couldn't deliver" — only a self-delivery chef, and only once they're
  // actually OUT FOR DELIVERY (status 'picked_up'/en route). At 'ready' the food
  // hasn't left the kitchen — that's what "Cancel order" is for; you can't fail a
  // delivery you haven't started. Quiet destructive link so the primary
  // "Mark delivered" action stays the hero.
  const deliveryFailLink =
    isChefDelivery && status === 'picked_up' ? (
      <Pressable
        onPress={onReportDeliveryFailure}
        disabled={disabled}
        hitSlop={8}
        accessibilityRole="button"
        accessibilityLabel="Report that you couldn't deliver this order"
        style={styles.cancelLinkWrap}
        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
      >
        <Text style={styles.cancelLinkLabel}>Couldn&apos;t deliver this order</Text>
      </Pressable>
    ) : null;

  // An open delivery-failure review closes the order off for the chef (#393):
  // no status actions until an admin confirms fault and resolves the payout.
  if (deliveryFailureReported) {
    return (
      <View style={[styles.footer, styles.footerCaptionWrap]}>
        <Text style={styles.footerCaption}>
          Delivery failure reported — our team is reviewing it.
        </Text>
      </View>
    );
  }
  if (status === 'pending') {
    return (
      <View style={styles.footer}>
        <Pressable
          onPress={onReject}
          disabled={disabled}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel={`Reject order from ${customerName}`}
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
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
          android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
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
          accessibilityLabel="Mark preparing"
          android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
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
    // Self-deliverable delivery order: the chef chooses, at Mark Ready, whether
    // to deliver it themselves or hand it to a rider. Over-range orders emphasise
    // "Hand to a rider" (primary) and de-emphasise "I'll deliver" (outline).
    if (canSelfDeliver) {
      // Only emphasise "hand to a rider" when a rider is actually available (3PL
      // live). With 3PL dark the rider button is hidden entirely and "I'll
      // deliver" is the single primary action.
      const emphasizeRider = overRange && riderAvailable;
      return (
        <View style={[styles.footer, styles.footerColumn]}>
          {emphasizeRider ? (
            <Text style={styles.carrierHint}>
              {`This drop is ${selfDeliveryDistanceKm.toFixed(1)} km away — beyond your ${selfDeliveryMaxDistanceKm} km range. Handing it to a rider is recommended.`}
            </Text>
          ) : null}
          <Pressable
            onPress={() => onReadyCarrier('chef_delivery')}
            disabled={disabled}
            accessibilityRole="button"
            accessibilityLabel="Mark ready, I will deliver it myself"
            android_ripple={{
              color: emphasizeRider
                ? `${theme.colors.ink.DEFAULT}14`
                : `${theme.colors.paper}33`,
              borderless: false,
            }}
          >
            {({ pressed }) => (
              <View
                style={[
                  emphasizeRider ? styles.carrierBtnSecondary : styles.carrierBtnPrimary,
                  pressed && { opacity: 0.85 },
                  disabled && { opacity: 0.4 },
                ]}
              >
                <Text
                  style={emphasizeRider ? styles.carrierLabelSecondary : styles.primaryLabel}
                >
                  Ready · I&apos;ll deliver
                </Text>
              </View>
            )}
          </Pressable>
          {riderAvailable ? (
            <Pressable
              onPress={() => onReadyCarrier('delivery')}
              disabled={disabled}
              accessibilityRole="button"
              accessibilityLabel="Mark ready, hand to a rider"
              android_ripple={{
                color: emphasizeRider
                  ? `${theme.colors.paper}33`
                  : `${theme.colors.ink.DEFAULT}14`,
                borderless: false,
              }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    emphasizeRider ? styles.carrierBtnPrimary : styles.carrierBtnSecondary,
                    pressed && { opacity: 0.85 },
                    disabled && { opacity: 0.4 },
                  ]}
                >
                  <Text
                    style={emphasizeRider ? styles.primaryLabel : styles.carrierLabelSecondary}
                  >
                    Ready · Hand to a rider
                  </Text>
                </View>
              )}
            </Pressable>
          ) : null}
          {/* Photo-required Ready step (Task 9): both carrier choices route
              through captureAndAdvanceWithCarrier, which opens the camera
              before advancing — name the requirement up front. */}
          <Text style={styles.photoCaption}>
            You&apos;ll attach a photo of the prepared order
          </Text>
          {cancelLink}
        </View>
      );
    }
    return (
      <View style={[styles.footer, styles.footerColumn]}>
        <Pressable
          onPress={onMarkReady}
          disabled={disabled}
          accessibilityRole="button"
          accessibilityLabel={isPickup ? 'Mark ready for pickup' : 'Mark ready'}
          android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.primaryBtnFull,
                pressed && { opacity: 0.85 },
                disabled && { opacity: 0.4 },
              ]}
            >
              <Text style={styles.primaryLabel}>
                {isPickup ? 'Mark ready for pickup' : 'Mark ready'}
              </Text>
            </View>
          )}
        </Pressable>
        {/* Photo-required Ready step (Task 9): onMarkReady routes through
            captureAndAdvance, which opens the camera before advancing. */}
        <Text style={styles.photoCaption}>
          You&apos;ll attach a photo of the prepared order
        </Text>
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
            android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
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
      // Until they're en route they can still hand it to a rider instead.
      return (
        <View style={[styles.footer, styles.footerColumn]}>
          <Pressable
            onPress={onMarkOutForDelivery}
            disabled={disabled}
            accessibilityRole="button"
            accessibilityLabel="Mark order out for delivery"
            android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.carrierBtnPrimary,
                  pressed && { opacity: 0.85 },
                  disabled && { opacity: 0.4 },
                ]}
              >
                <Text style={styles.primaryLabel}>Out for delivery</Text>
              </View>
            )}
          </Pressable>
          {riderAvailable ? (
            <Pressable
              onPress={() => onSwitchCarrier('delivery')}
              disabled={disabled}
              hitSlop={8}
              accessibilityRole="button"
              accessibilityLabel="Hand this order to a rider instead"
              style={styles.switchLinkWrap}
              android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
            >
              <Text style={styles.switchLinkLabel}>Hand to a rider instead</Text>
            </Pressable>
          ) : null}
          {cancelLink}
        </View>
      );
    }
    // 3PL delivery, still `ready` → waiting on a rider. A self-delivering chef
    // can take it over themselves before anyone is en route.
    return (
      <View style={[styles.footer, styles.footerColumn, styles.footerCaptionWrap]}>
        <Text style={styles.footerCaption}>{`Waiting for a rider to pick up${readyElapsed}.`}</Text>
        {canSelfDeliver ? (
          <Pressable
            onPress={() => onSwitchCarrier('chef_delivery')}
            disabled={disabled}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Deliver this order yourself instead"
            style={styles.switchLinkWrap}
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
          >
            <Text style={styles.switchLinkLabel}>I&apos;ll deliver this instead</Text>
          </Pressable>
        ) : null}
        {cancelLink}
      </View>
    );
  }

  // status === 'picked_up' — for chef_delivery the chef is en route and
  // completes the order on arrival. (3PL orders are driver-controlled here.)
  if (status === 'picked_up' && isChefDelivery) {
    return (
      <View style={[styles.footer, styles.footerColumn]}>
        <Pressable
          onPress={onMarkDelivered}
          disabled={disabled}
          accessibilityRole="button"
          accessibilityLabel={`Mark order delivered to ${customerName}`}
          android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
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
        {deliveryFailLink}
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
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
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
  // Home-tiffin scheduling (#709): at accept the chef confirms the customer's
  // requested time (null) or bumps it by a preset when the kitchen needs longer.
  const [proposeOffsetMin, setProposeOffsetMin] = useState<number | null>(null);
  // Chef's delivery-fee choice at accept (#703). Empty = charge the customer the
  // recommended amount as-is; a number (0..charged) lowers it + refunds the diff.
  const [deliveryFeeInput, setDeliveryFeeInput] = useState('');
  const updateStatus = useUpdateOrderStatus();
  const uploadPhoto = useUploadOrderPhoto();
  const cancelOrder = useCancelOrder(orderId);
  const cancelItem = useCancelOrderItem(orderId);
  const reportFailure = useReportDeliveryFailure(orderId);
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
    const asset = await pickReadyPhoto();
    if (!asset?.uri) return;

    try {
      await uploadPhoto.mutateAsync({ orderId: order.id, kind, uri: asset.uri });
      await updateStatus.mutateAsync({ orderId: order.id, status: nextStatus });
      fireHaptic();
    } catch {
      showToast({
        message: 'Could not upload the photo. Please try again.',
        tone: 'error',
      });
    }
  }

  // Mark Ready WITH a carrier choice (self-delivering chefs only): capture the
  // ready photo, then advance to `ready` carrying the chef's carrier decision
  // ('chef_delivery' = I'll deliver, 'delivery' = hand to a rider). Shares the
  // camera/library capture with captureAndAdvance via pickReadyPhoto().
  async function captureAndAdvanceWithCarrier(
    carrier: 'chef_delivery' | 'delivery',
  ): Promise<void> {
    if (!order || uploadPhoto.isPending || updateStatus.isPending) return;
    const asset = await pickReadyPhoto();
    if (!asset?.uri) return;
    try {
      await uploadPhoto.mutateAsync({
        orderId: order.id,
        kind: 'ready',
        uri: asset.uri,
      });
      await updateStatus.mutateAsync({
        orderId: order.id,
        status: 'ready',
        carrier,
      });
      fireHaptic();
    } catch {
      showToast({
        message: 'Could not update the order. Please try again.',
        tone: 'error',
      });
    }
  }

  // Re-send `ready` with the other carrier to flip deliver↔rider while the
  // order is still on the chef's side (before anyone is en route). No new photo
  // — the ready photo already exists; only the carrier flips server-side.
  async function switchReadyCarrier(
    carrier: 'chef_delivery' | 'delivery',
  ): Promise<void> {
    if (!order || updateStatus.isPending) return;
    try {
      await updateStatus.mutateAsync({
        orderId: order.id,
        status: 'ready',
        carrier,
      });
    } catch {
      showToast({
        message: 'Could not switch the carrier. Please try again.',
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

  // "Couldn't deliver" (#393) — a self-delivery chef reports a failed drop.
  // Two-step like cancel: pick a reason, then it fires. This does NOT refund
  // instantly — it opens an admin review and freezes the payout until fault is
  // decided, so the copy says so plainly.
  function submitDeliveryFailure(reason: DeliveryFailureReason): void {
    reportFailure.mutate(reason, {
      onSuccess: () => {
        showToast({
          message: "Reported. We'll review it and sort out the payment.",
          tone: 'success',
        });
      },
      onError: () => {
        showToast({
          message: "Couldn't report that right now. Try again.",
          tone: 'error',
        });
      },
    });
  }

  function openDeliveryFailureSheet(): void {
    if (Platform.OS === 'ios') {
      ActionSheetIOS.showActionSheetWithOptions(
        {
          title: "Couldn't deliver this order?",
          message:
            "We'll review what happened. Your payout for this order is held until our team confirms it — you won't be paid or charged automatically.",
          options: [
            ...DELIVERY_FAILURE_REASONS.map((r) => DELIVERY_FAILURE_REASON_LABEL[r]),
            'Keep trying',
          ],
          cancelButtonIndex: DELIVERY_FAILURE_REASONS.length,
          destructiveButtonIndex: DELIVERY_FAILURE_REASONS.length - 1,
        },
        (index) => {
          if (index < 0 || index >= DELIVERY_FAILURE_REASONS.length) return;
          submitDeliveryFailure(DELIVERY_FAILURE_REASONS[index]!);
        },
      );
      return;
    }
    Alert.alert(
      "Couldn't deliver this order?",
      "We'll review it and your payout is held until our team decides. Pick a reason next.",
      [
        { text: 'Keep trying', style: 'cancel' },
        {
          text: "Couldn't deliver",
          style: 'destructive',
          onPress: () => promptDeliveryFailureReasonAndroid(submitDeliveryFailure),
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
            accessibilityLabel="Retry loading order"
            android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
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

  // Soft self-delivery distance warning: the chef set a comfort radius and this
  // drop is farther than it. Informational only — the chef can still deliver
  // (proceed via the normal flow) or decline (cancel → refund). Hidden once the
  // order is no longer actionable. maxDistanceKm of 0 = chef set no radius.
  const isActiveStatus =
    order.status !== 'delivered' &&
    order.status !== 'cancelled' &&
    order.status !== 'rejected';
  const overSelfDeliveryRange =
    isChefDelivery &&
    isActiveStatus &&
    order.selfDeliveryMaxDistanceKm > 0 &&
    order.selfDeliveryDistanceKm > order.selfDeliveryMaxDistanceKm;

  // A still-`delivery` order the chef COULD self-deliver. Gate on the chef's
  // self-delivery CAPABILITY flag — NOT on distance data, which is 0 when the
  // chef set no comfort radius / coords are missing (the old heuristic hid the
  // carrier choice for exactly those chefs). The distance fields below stay
  // purely for the soft over-range nudge.
  const canSelfDeliver =
    order.fulfillmentType === 'delivery' && order.offersSelfDelivery;
  const overReadyRange =
    canSelfDeliver &&
    order.selfDeliveryMaxDistanceKm > 0 &&
    order.selfDeliveryDistanceKm > order.selfDeliveryMaxDistanceKm;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <CommandBar
        orderNumber={order.orderNumber}
        status={order.status}
        fulfillmentType={order.fulfillmentType}
        onBack={handleBack}
      />

      <KeyboardAwareScrollView
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
                        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
                      >
                        <Text style={styles.itemCancelLinkLabel}>
                          Can't make this?
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
            {overSelfDeliveryRange ? (
              <View
                style={styles.rangeWarning}
                accessibilityRole="alert"
              >
                <Text style={styles.rangeWarningTitle}>
                  {`This address is ${order.selfDeliveryDistanceKm.toFixed(1)} km away — beyond your ${order.selfDeliveryMaxDistanceKm} km range.`}
                </Text>
                <Text style={styles.rangeWarningBody}>
                  Still happy to deliver? Carry on below. If it’s too far, tap
                  Cancel to refund the customer.
                </Text>
              </View>
            ) : null}
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

        {/* HOME-TIFFIN SCHEDULING (#709): the customer's requested time. For a
            pending order the chef confirms it or proposes a later one; once
            accepted, the confirmed/proposed time is shown. */}
        {order.status === 'pending' ? (
          <>
            <SectionLabel>{isPickup ? 'PICKUP TIME' : 'DELIVERY TIME'}</SectionLabel>
            <View style={styles.card}>
              <View style={[styles.timingRow, styles.rowBorderBottom]}>
                <Text style={styles.timingLabel}>Customer asked</Text>
                <Text style={styles.timingValue}>
                  {order.timing.requestedFulfillmentAt
                    ? formatDateTime(order.timing.requestedFulfillmentAt)
                    : 'As soon as ready'}
                </Text>
              </View>
              <View style={styles.proposeBlock}>
                <Text style={styles.proposeTitle}>Confirm or propose a new time</Text>
                <Text style={styles.proposeHint}>
                  Accept confirms the requested time. Kitchen busy? Propose a later one — the customer is notified.
                </Text>

                {/* Split by meaning rather than wrapping five chips raggedly:
                    "As asked" confirms what the customer requested, the offsets
                    propose a later slot. Five equal chips never fit one row, so
                    flexWrap left a lone orphan on row two. */}
                <Chip
                  label="As asked"
                  selected={proposeOffsetMin === null}
                  onPress={() => setProposeOffsetMin(null)}
                />
                <View style={styles.proposeRow}>
                  {([
                    [15, '+15m'],
                    [30, '+30m'],
                    [45, '+45m'],
                    [60, '+1h'],
                  ] as [number, string][]).map(([off, label]) => (
                    <Chip
                      key={label}
                      label={label}
                      selected={proposeOffsetMin === off}
                      onPress={() => setProposeOffsetMin(off)}
                      grow
                    />
                  ))}
                </View>
              </View>
            </View>
          </>
        ) : order.timing.confirmedFulfillmentAt ? (
          <>
            <SectionLabel>{isPickup ? 'PICKUP TIME' : 'DELIVERY TIME'}</SectionLabel>
            <View style={styles.card}>
              <View style={styles.timingRow}>
                <Text style={styles.timingLabel}>
                  {order.timing.fulfillmentTimeStatus === 'proposed' ? 'You proposed' : 'Confirmed'}
                </Text>
                <Text style={styles.timingValue}>
                  {formatDateTime(order.timing.confirmedFulfillmentAt)}
                </Text>
              </View>
            </View>
          </>
        ) : null}

        {/* DELIVERY FEE (#703): the chef sets what they'll charge for delivery at
            accept. The customer was charged the recommended approx-max upfront; the
            chef can lower it (or ₹0 within their free zone) and the difference is
            refunded. Only for a pending self-delivery order that has a fee. */}
        {order.status === 'pending' &&
        order.fulfillmentType !== 'pickup' &&
        order.pricing.deliveryFee > 0 ? (
          <>
            <SectionLabel>DELIVERY FEE</SectionLabel>
            <View style={styles.card}>
              <View style={styles.deliveryFeeRow}>
                <Text style={styles.timingLabel}>You'll charge</Text>
                {/* Bordered field with a ₹ prefix. The bare right-aligned input
                    read as static text, so the chef had no cue it was editable
                    — the one control in this block that must invite a tap. */}
                <View style={styles.deliveryFeeField}>
                  <Text style={styles.deliveryFeePrefix}>₹</Text>
                  <TextInput
                    value={deliveryFeeInput}
                    onChangeText={setDeliveryFeeInput}
                    keyboardType="decimal-pad"
                    placeholder={order.pricing.deliveryFee.toFixed(0)}
                    placeholderTextColor={theme.colors.ink.muted}
                    style={styles.deliveryFeeInput}
                    accessibilityLabel="Delivery fee you'll charge"
                  />
                </View>
              </View>
            </View>
            <Text style={styles.deliveryHint}>
              Customer was charged ₹{order.pricing.deliveryFee.toFixed(0)} (recommended by
              distance). Lower it — or ₹0 if they're inside your free zone — and the difference
              is refunded to them. You can't charge more than ₹
              {order.pricing.deliveryFee.toFixed(0)}.
            </Text>
          </>
        ) : order.pricing.deliveryFee > 0 && typeof order.pricing.deliveryFeeFinal === 'number' ? (
          <>
            <SectionLabel>DELIVERY FEE</SectionLabel>
            <View style={styles.card}>
              <View style={styles.timingRow}>
                <Text style={styles.timingLabel}>You charged</Text>
                <Text style={styles.timingValue}>₹{order.pricing.deliveryFeeFinal.toFixed(0)}</Text>
              </View>
            </View>
          </>
        ) : null}

        {/* TIMING section — read-only status timeline (Task 9). Only steps the
            order data already carries render; "Preparing" has no dedicated
            backend timestamp (only the ready transition is stamped), so it is
            naturally absent rather than fabricated. */}
        <SectionLabel>TIMING</SectionLabel>
        <View style={styles.card}>
          <OrderStatusTimeline
            steps={[
              { label: 'Ordered', timestamp: timing.orderedAt },
              { label: 'Accepted', timestamp: timing.acceptedAt },
              { label: 'Preparing', timestamp: null },
              { label: 'Ready', timestamp: timing.preparedAt },
              {
                label: 'Out for delivery',
                timestamp: isPickup ? null : timing.pickedUpAt,
              },
              {
                label: isPickup ? 'Collected' : 'Delivered',
                timestamp: timing.deliveredAt,
              },
            ]}
          />
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
      </KeyboardAwareScrollView>

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
        canSelfDeliver={canSelfDeliver}
        riderAvailable={order.riderDispatchAvailable}
        overRange={overReadyRange}
        selfDeliveryDistanceKm={order.selfDeliveryDistanceKm}
        selfDeliveryMaxDistanceKm={order.selfDeliveryMaxDistanceKm}
        onAccept={() => {
          // null offset = confirm the request as-is (backend confirms/leaves ASAP).
          // An offset proposes a later time = requested (or ~45m from now for ASAP)
          // plus the offset (#709).
          let confirmedAt: string | undefined;
          if (proposeOffsetMin != null) {
            const base = order.timing.requestedFulfillmentAt
              ? new Date(order.timing.requestedFulfillmentAt)
              : new Date(Date.now() + 45 * 60 * 1000);
            confirmedAt = new Date(base.getTime() + proposeOffsetMin * 60 * 1000).toISOString();
          }
          // Chef's delivery-fee choice (#703): only send when they entered a value;
          // clamp to [0, charged] so they can never charge above the approx-max.
          let deliveryFee: number | undefined;
          const parsed = parseFloat(deliveryFeeInput);
          if (deliveryFeeInput.trim() !== '' && !Number.isNaN(parsed)) {
            deliveryFee = Math.max(0, Math.min(parsed, order.pricing.deliveryFee));
          }
          triggerAction(order.id, 'accepted', undefined, confirmedAt, deliveryFee);
        }}
        onReject={() => triggerAction(order.id, 'rejected')}
        onMarkPreparing={() => {
          fireHaptic();
          updateStatus.mutate({ orderId: order.id, status: 'preparing' });
        }}
        onMarkReady={() => captureAndAdvance('ready', 'ready')}
        onReadyCarrier={(carrier) => captureAndAdvanceWithCarrier(carrier)}
        onSwitchCarrier={(carrier) => switchReadyCarrier(carrier)}
        onMarkHandedOver={() => captureAndAdvance('handover', 'delivered')}
        onMarkOutForDelivery={() => {
          fireHaptic();
          updateStatus.mutate({ orderId: order.id, status: 'picked_up' });
        }}
        onMarkDelivered={() => {
          fireHaptic();
          updateStatus.mutate({ orderId: order.id, status: 'delivered' });
        }}
        onCancel={openCancelSheet}
        onReportDeliveryFailure={openDeliveryFailureSheet}
        deliveryFailureReported={order.deliveryFailureReported}
      />
    </SafeAreaView>
  );
}

// pickReadyPhoto opens the camera on a real device (live proof the food is
// genuinely ready) and falls back to the photo library on a simulator/emulator
// (no camera there — Device.isDevice is false only on a simulator, so
// production behaviour is unchanged). Returns the picked asset, or undefined if
// the chef cancelled. Shared by the plain Mark-Ready flow and the carrier flow.
async function pickReadyPhoto(): Promise<
  ImagePicker.ImagePickerAsset | undefined
> {
  const canUseCamera =
    Device.isDevice &&
    (await ImagePicker.requestCameraPermissionsAsync()).granted;
  if (canUseCamera) {
    const shot = await ImagePicker.launchCameraAsync({
      mediaTypes: ['images'],
      quality: 0.6,
    });
    if (shot.canceled) return undefined;
    return shot.assets[0];
  }
  const lib = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ['images'],
    quality: 0.6,
  });
  if (lib.canceled) return undefined;
  return lib.assets[0];
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

// promptDeliveryFailureReasonAndroid emulates an action sheet with a chained
// Alert for the delivery-failure reason picker (Android caps reliably at 3
// buttons). Mirrors promptCancelReasonAndroid.
function promptDeliveryFailureReasonAndroid(
  submit: (r: DeliveryFailureReason) => void,
): void {
  Alert.alert('What went wrong?', '', [
    {
      text: DELIVERY_FAILURE_REASON_LABEL.customer_unavailable,
      onPress: () => submit('customer_unavailable'),
    },
    {
      text: DELIVERY_FAILURE_REASON_LABEL.customer_refused,
      onPress: () => submit('customer_refused'),
    },
    {
      text: 'More…',
      onPress: () =>
        Alert.alert('What went wrong?', '', [
          {
            text: DELIVERY_FAILURE_REASON_LABEL.wrong_address,
            onPress: () => submit('wrong_address'),
          },
          {
            text: DELIVERY_FAILURE_REASON_LABEL.food_damaged,
            onPress: () => submit('food_damaged'),
          },
          {
            text: DELIVERY_FAILURE_REASON_LABEL.other,
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

  // Soft self-delivery distance warning — amber (functional warning), sits
  // just under the DELIVER TO card. Informational, not a blocking gate.
  rangeWarning: {
    marginHorizontal: theme.spacing[4],
    marginTop: theme.spacing[2],
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
  },
  rangeWarningTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 20,
  },
  rangeWarningBody: {
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

  // Read-only status timeline (Task 9) — dot + connector rail on the left,
  // label + timestamp on the right. Only rendered rows are ones the order
  // data actually has a timestamp for.
  timelineRow: {
    flexDirection: 'row',
    paddingHorizontal: theme.spacing[4],
    gap: theme.spacing[3],
  },
  timelineRail: {
    width: 20,
    alignItems: 'center',
  },
  timelineDot: {
    width: 20,
    height: 20,
    borderRadius: theme.radius.full,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: theme.spacing[3],
  },
  timelineConnector: {
    flex: 1,
    width: 2,
    minHeight: theme.spacing[3],
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: 2,
  },
  timelineContent: {
    flex: 1,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: theme.spacing[3],
    minHeight: 40,
  },
  timelineLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  timelineTimestamp: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    fontVariant: ['tabular-nums'],
  },

  // Delivery-fee input (#703)
  deliveryFeeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
    gap: theme.spacing[3],
  },
  // Bordered container so the field reads as editable; matches the chip radius
  // and hairline so the two control blocks look like one system.
  deliveryFeeField: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    minHeight: 44,
    paddingHorizontal: theme.spacing[3],
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.bone,
  },
  deliveryFeePrefix: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.soft,
  },
  deliveryFeeInput: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'right',
    minWidth: 84,
    paddingVertical: 6,
    // Tabular figures — this sits directly above the pricing rows.
    fontVariant: ['tabular-nums'],
  },
  // Explanatory copy that sits BELOW a card — aligned to the card gutter.
  deliveryHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    lineHeight: 16,
    marginHorizontal: theme.spacing[4],
    marginTop: -theme.spacing[2],
    marginBottom: theme.spacing[4],
    fontVariant: ['tabular-nums'],
  },

  // Home-tiffin scheduling (#709) — propose-time controls. The controls sit in
  // their own padded block below the "Customer asked" row (hairline-separated)
  // so they align with the inset row instead of running to the card edge.
  proposeBlock: {
    paddingHorizontal: theme.spacing[4],
    // More room above than the old spacing[3] so the control group reads as its
    // own sub-section under the "Customer asked" summary, not a continuation.
    paddingTop: theme.spacing[4],
    paddingBottom: theme.spacing[4],
    gap: theme.spacing[3],
  },
  proposeTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  proposeHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    lineHeight: 16,
  },
  // The four offset chips share one row and divide it evenly (chipGrow), so
  // there is no wrapping and no ragged trailing chip.
  proposeRow: {
    flexDirection: 'row',
    gap: theme.spacing[2],
  },
  chip: {
    paddingHorizontal: theme.spacing[3],
    // 44px floor — the vendor touch-target minimum.
    minHeight: 44,
    justifyContent: 'center',
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.bone,
    alignItems: 'center',
  },
  chipGrow: {
    flex: 1,
  },
  // Selected reads as filled, not just tinted — a 1px border swap was too quiet
  // to scan at a glance in a kitchen.
  chipSelected: {
    borderColor: theme.colors.brand[500],
    backgroundColor: theme.colors.brand[100],
  },
  chipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  chipTextSelected: {
    color: theme.colors.brand[500],
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
  // Stacks the two Mark-Ready carrier buttons (+ optional hint) vertically
  // inside the footer instead of the default side-by-side row.
  footerColumn: {
    flexDirection: 'column',
    alignItems: 'stretch',
    gap: theme.spacing[2],
  },
  footerCaptionWrap: {
    justifyContent: 'center',
  },
  // Carrier choice buttons — full-width (no flex, so they don't stretch
  // vertically in the column footer). Primary = ink fill, secondary = ink
  // outline. The emphasis swaps when the drop is over the chef's range.
  carrierBtnPrimary: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
  },
  carrierBtnSecondary: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
  },
  carrierLabelSecondary: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: 0.3,
  },
  carrierHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
    paddingHorizontal: theme.spacing[1],
  },
  footerCaption: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    flex: 1,
  },
  // Photo-required Ready step affordance (Task 9) — sits under the action
  // that leads into the camera so the chef isn't surprised mid-tap.
  photoCaption: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
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
  // Cancel-order link — ink text link, no underline (UI-V2-SPEC §3, vendor
  // reconciliation: every accent text link in the canonical spec resolves to
  // ink here — the app carries zero brand-accent color).
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
  // Quiet carrier-switch link (deliver↔rider) in the `ready` footer — same
  // ink-link treatment as the cancel link (Task 9: link text stays ink, no
  // muted variant; weight comes from position/order, not a dimmer color).
  switchLinkWrap: {
    alignSelf: 'center',
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
  },
  switchLinkLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  // Per-line cancel — smaller link inside the item row. The whole-order
  // cancel is the loud one; this is for the chef mid-prep who only needs to
  // drop one line. Task 9: ink link per §3 (was ink.muted — that read as
  // disabled, not as a quiet tappable action; SemiBold ink keeps it calm
  // without the alarm of red, since this isn't an error/alarm state).
  itemCancelLinkWrap: {
    marginTop: 4,
    paddingVertical: 4,
    alignSelf: 'flex-start',
  },
  itemCancelLinkLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.DEFAULT,
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
    fontVariant: ['tabular-nums'],
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
