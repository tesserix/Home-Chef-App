import { Pressable, StyleSheet, Text, View } from 'react-native';
import Animated, {
  Easing,
  FadeInDown,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withTiming,
} from 'react-native-reanimated';
import * as Haptics from 'expo-haptics';
import { theme } from '@homechef/mobile-shared/theme';
import type { Order } from '../../hooks/useVendorOrders';

const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

// Lifecycle step definitions — the full order journey the chef sees.
// Each step maps to one or more order statuses that place the order at
// that step. Steps 4 and 5 are driver-controlled; the chef cannot advance
// them, only observe.
interface LifecycleStep {
  label: string;
  // Short label for the compact stepper pills.
  shortLabel: string;
}

// Delivery journey: the last two steps are driver-controlled (the chef can
// only observe them). Pickup journey is shorter — the customer collects, so
// the chef completes the order at "Collected" (no driver leg).
const DELIVERY_STEPS: readonly LifecycleStep[] = [
  { label: 'Accepted', shortLabel: 'Accepted' },
  { label: 'Preparing', shortLabel: 'Preparing' },
  { label: 'Ready', shortLabel: 'Ready' },
  { label: 'Out for delivery', shortLabel: 'Delivering' },
  { label: 'Delivered', shortLabel: 'Delivered' },
] as const;

const PICKUP_STEPS: readonly LifecycleStep[] = [
  { label: 'Accepted', shortLabel: 'Accepted' },
  { label: 'Preparing', shortLabel: 'Preparing' },
  { label: 'Ready for pickup', shortLabel: 'Ready' },
  { label: 'Collected', shortLabel: 'Collected' },
] as const;

// Map an order status to the 1-based step index it corresponds to.
const DELIVERY_STATUS_TO_STEP: Record<string, number> = {
  accepted: 1,
  preparing: 2,
  ready: 3,
  picked_up: 4,
  delivering: 4,
  delivered: 5,
};

const PICKUP_STATUS_TO_STEP: Record<string, number> = {
  accepted: 1,
  preparing: 2,
  ready: 3,
  delivered: 4,
};

// The one-tap advance label and next status for chef-controllable transitions.
// Pickup adds a chef-driven terminal step: ready → delivered ("Mark handed over").
const DELIVERY_ADVANCE_LABEL: Record<string, string> = {
  accepted: 'Start Preparing',
  preparing: 'Mark Ready',
};

const PICKUP_ADVANCE_LABEL: Record<string, string> = {
  accepted: 'Start Preparing',
  preparing: 'Mark Ready',
  ready: 'Mark Handed Over',
};

const DELIVERY_NEXT_STATUS: Record<string, Order['status']> = {
  accepted: 'preparing',
  preparing: 'ready',
};

const PICKUP_NEXT_STATUS: Record<string, Order['status']> = {
  accepted: 'preparing',
  preparing: 'ready',
  ready: 'delivered',
};

// Captions shown on driver-controlled states where the chef has no action.
// Pickup has none — the chef always has an action until the order is collected.
const AWAITING_CAPTION: Record<string, string> = {
  ready: 'Ready · awaiting pickup',
  picked_up: 'Out for delivery',
  delivering: 'Out for delivery',
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

function formatItemsSummary(items: Order['items']): string {
  const count = items.reduce((sum, i) => sum + (i.quantity ?? 1), 0);
  if (count === 1 && items[0]) return items[0].name;
  if (items.length === 1 && items[0]) return `${count} × ${items[0].name}`;
  return `${count} item${count === 1 ? '' : 's'}`;
}

// ----- LifecycleStepper -------------------------------------------------------

interface LifecycleStepperProps {
  /** The 1-based index of the current step (1 = Accepted). */
  currentStep: number;
  /** The lifecycle steps for this order's fulfillment type. */
  steps: readonly LifecycleStep[];
}

/**
 * Compact horizontal stepper showing the five lifecycle stages.
 *
 * Visual language (vendor monochrome + per-role density rule):
 *  - Completed steps (< currentStep): filled ink connector + ink dot.
 *  - Current step: emphasized label + ink dot.
 *  - Future steps: muted dot + muted connector.
 *  - Ready step (step 3) uses success green when it is the CURRENT step —
 *    this is the one chef-completed milestone that deserves a positive signal.
 *  - Persimmon reserved for ready as the single accent per .impeccable.md.
 *    We use success green here (vendor palette uses green for positive status,
 *    persimmon retired for vendor). The step connector fills ink for completed.
 */
function LifecycleStepper({ currentStep, steps }: LifecycleStepperProps) {
  return (
    <View style={stepperStyles.root} accessibilityRole="none">
      {steps.map((step, idx) => {
        const stepNum = idx + 1;
        const isCompleted = stepNum < currentStep;
        const isCurrent = stepNum === currentStep;
        const isLast = idx === steps.length - 1;

        // Color logic for the dot:
        // - Completed: ink (filled, done)
        // - Current at step 3 (ready): success green (positive milestone)
        // - Current at other steps: ink (the active step)
        // - Future: mist (not yet reached)
        const dotColor = isCompleted
          ? theme.colors.ink.DEFAULT
          : isCurrent && stepNum === 3
            ? theme.colors.success.DEFAULT
            : isCurrent
              ? theme.colors.ink.DEFAULT
              : theme.colors.mist.DEFAULT;

        const labelColor = isCompleted
          ? theme.colors.ink.muted
          : isCurrent && stepNum === 3
            ? theme.colors.success.soft
            : isCurrent
              ? theme.colors.ink.DEFAULT
              : theme.colors.ink.muted;

        return (
          <View key={step.shortLabel} style={stepperStyles.stepGroup}>
            <View style={stepperStyles.stepColumn}>
              <View
                style={[stepperStyles.dot, { backgroundColor: dotColor }]}
              />
              <Text
                style={[
                  stepperStyles.stepLabel,
                  { color: labelColor },
                  isCurrent && stepperStyles.stepLabelCurrent,
                ]}
                numberOfLines={1}
              >
                {step.shortLabel}
              </Text>
            </View>
            {/* Connector between steps */}
            {!isLast && (
              <View
                style={[
                  stepperStyles.connector,
                  {
                    backgroundColor: isCompleted
                      ? theme.colors.ink.muted
                      : theme.colors.mist.DEFAULT,
                  },
                ]}
              />
            )}
          </View>
        );
      })}
    </View>
  );
}

// ----- ActiveOrderCard -------------------------------------------------------

/**
 * Minimal order shape required by the card. Intentionally narrower than the
 * full `Order` type so the Dashboard can pass `RecentOrder` (which omits
 * `items` and `deliveryAddress`) without a cast. When `items` is provided,
 * the card shows the items summary; otherwise it falls back to a count-only
 * placeholder derived from the total.
 */
export interface ActiveOrderCardOrder {
  id: string;
  customerName: string;
  total: number;
  status: Order['status'];
  createdAt: string;
  items?: Order['items'];
  fulfillmentType?: 'delivery' | 'chef_delivery' | 'pickup';
}

interface ActiveOrderCardProps {
  order: ActiveOrderCardOrder;
  /** True while the status-advance mutation is in flight for this order. */
  isPending?: boolean;
  /** Called when the chef taps the advance-status action button. */
  onAdvance: (orderId: string, nextStatus: Order['status']) => void;
  /** Called when the chef taps the card body — navigates to the detail screen. */
  onOpenDetail: (orderId: string) => void;
}

/**
 * Active-order card for the Dashboard "In Progress" section.
 *
 * Shows:
 *  - Order summary: customer name, item count, total (tabular figures).
 *  - Lifecycle stepper: Accepted → Preparing → Ready → Out for delivery →
 *    Delivered. Completed steps filled, current highlighted, future muted.
 *  - One-tap advance button for chef-controllable transitions (accepted →
 *    Start Preparing, preparing → Mark Ready). When the driver controls the
 *    transition (ready, picked_up/delivering), shows a calm caption instead.
 *
 * Design intent (vendor monochrome, .impeccable.md §Per-role density):
 *  - Ink-filled advance button with paper text — matches PendingOrderCard's
 *    Accept button. NOT the customer coral.
 *  - Inner-View pattern on the Pressable to avoid the iOS function-style
 *    style-drop bug (flex/bg/padding silently lost without it).
 *  - Touch target: minHeight 44 on the advance button (Apple HIG).
 *  - No bounce; 150 ms scale on button press only.
 */
export function ActiveOrderCard({
  order,
  isPending = false,
  onAdvance,
  onOpenDetail,
}: ActiveOrderCardProps) {
  const reduceMotion = useReducedMotion();

  // Subtle scale on the advance button — 150 ms, no bounce.
  const btnScale = useSharedValue(1);
  const btnPressStyle = useAnimatedStyle(() => ({
    transform: [{ scale: btnScale.value }],
  }));

  function onBtnPressIn(): void {
    if (!reduceMotion) {
      btnScale.value = withTiming(0.97, { duration: 150 });
    }
  }
  function onBtnPressOut(): void {
    if (!reduceMotion) {
      btnScale.value = withTiming(1, { duration: 150 });
    }
  }

  const isPickup = order.fulfillmentType === 'pickup';
  const steps = isPickup ? PICKUP_STEPS : DELIVERY_STEPS;
  const statusToStep = isPickup ? PICKUP_STATUS_TO_STEP : DELIVERY_STATUS_TO_STEP;
  const advanceLabelMap = isPickup ? PICKUP_ADVANCE_LABEL : DELIVERY_ADVANCE_LABEL;
  const nextStatusMap = isPickup ? PICKUP_NEXT_STATUS : DELIVERY_NEXT_STATUS;

  function handleAdvance(): void {
    const next = nextStatusMap[order.status];
    if (!next || isPending) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    onAdvance(order.id, next);
  }

  const advanceLabel = advanceLabelMap[order.status];
  const hasAction = !!advanceLabel;
  // Pickup has no driver-controlled waiting state — the chef always has an
  // action (incl. "Mark Handed Over") until the order is collected.
  const awaitingCaption = isPickup ? undefined : AWAITING_CAPTION[order.status];
  const currentStep = statusToStep[order.status] ?? 1;

  return (
    <Animated.View
      style={styles.root}
      entering={
        reduceMotion
          ? undefined
          : FadeInDown.duration(250).easing(ENTRANCE_EASING)
      }
    >
      {/* Card body — tapping anywhere except the button navigates to detail */}
      <Pressable
        onPress={() => onOpenDetail(order.id)}
        accessibilityRole="button"
        accessibilityLabel={`Open order details for ${order.customerName}`}
      >
        {({ pressed }) => (
          // Inner-View pattern — visual styles (bg, radius) on View,
          // not on Pressable, to avoid the iOS function-style style drop.
          <View style={[styles.body, pressed && { opacity: 0.85 }]}>
            {/* Header row: customer name left, total right */}
            <View style={styles.topRow}>
              <Text style={styles.customerName} numberOfLines={1}>
                {order.customerName}
              </Text>
              <Text style={styles.total}>₹{order.total.toFixed(0)}</Text>
            </View>
            {/* Items summary + age — items may be absent on dashboard RecentOrder */}
            <Text style={styles.meta} numberOfLines={1}>
              {order.items && order.items.length > 0
                ? `${formatItemsSummary(order.items)} · `
                : ''}
              {formatMinutesAgo(order.createdAt)}
            </Text>
            {/* Lifecycle stepper */}
            <LifecycleStepper currentStep={currentStep} steps={steps} />
          </View>
        )}
      </Pressable>

      {/* Hairline between card body and action area */}
      <View style={styles.divider} />

      {/* Action zone: advance button OR driver-controlled caption */}
      {hasAction ? (
        // Three-layer pattern: wrapper View → Pressable (interaction) →
        // inner Animated.View (visual). Matches PendingOrderCard's acceptWrap.
        <View style={styles.actionRow}>
          <Pressable
            onPress={handleAdvance}
            onPressIn={onBtnPressIn}
            onPressOut={onBtnPressOut}
            disabled={isPending}
            accessibilityRole="button"
            accessibilityLabel={`${advanceLabel} for order from ${order.customerName}`}
            style={styles.advancePressable}
          >
            <Animated.View
              style={[
                styles.advanceBtn,
                btnPressStyle,
                isPending && styles.advanceBtnPending,
              ]}
            >
              <Text style={styles.advanceBtnLabel}>{advanceLabel}</Text>
            </Animated.View>
          </Pressable>
        </View>
      ) : awaitingCaption ? (
        // Driver-controlled / delivered — calm caption row, no action.
        <View style={styles.awaitingCaptionRow}>
          <View
            style={[
              styles.awaitingDot,
              {
                backgroundColor:
                  order.status === 'ready'
                    ? theme.colors.success.DEFAULT
                    : theme.colors.ink.muted,
              },
            ]}
          />
          <Text
            style={[
              styles.awaitingCaption,
              {
                color:
                  order.status === 'ready'
                    ? theme.colors.success.soft
                    : theme.colors.ink.soft,
              },
            ]}
          >
            {awaitingCaption}
          </Text>
        </View>
      ) : null}
    </Animated.View>
  );
}

// ----- Styles -----------------------------------------------------------------

const stepperStyles = StyleSheet.create({
  root: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginTop: theme.spacing[3],
    // Prevent stepper from consuming a hit target above the divider when
    // the card body Pressable is active.
    pointerEvents: 'none',
  },
  // Each step + its trailing connector
  stepGroup: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
  },
  // Dot + label stacked
  stepColumn: {
    alignItems: 'center',
    gap: 3,
  },
  dot: {
    width: 7,
    height: 7,
    borderRadius: 4,
  },
  stepLabel: {
    fontFamily: 'Inter',
    fontSize: 9,
    letterSpacing: 0.1,
    textAlign: 'center',
    // Tighten so 5 labels fit without ellipsis on 375px.
    maxWidth: 52,
  },
  stepLabelCurrent: {
    fontFamily: 'Inter-SemiBold',
  },
  // Hairline connector between steps
  connector: {
    flex: 1,
    height: StyleSheet.hairlineWidth,
    marginBottom: 10, // align with dot center (dot 7px + label ~12px)
    marginHorizontal: 2,
  },
});

const styles = StyleSheet.create({
  root: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[2],
    overflow: 'hidden',
  },

  // Tappable card body — padding + flex layout.
  body: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[4],
    paddingBottom: theme.spacing[3],
  },
  topRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: theme.spacing[3],
    marginBottom: theme.spacing[1],
  },
  customerName: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  total: {
    fontFamily: 'Geist-Bold',
    fontSize: 22,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    letterSpacing: -0.3,
  },
  meta: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  // Hairline between card body and action area.
  divider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
  },

  // Action zone — advance button row.
  actionRow: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  // Pressable fills the action row width without using flex on the Pressable
  // itself (iOS bug: flex on a function-style Pressable is silently dropped).
  advancePressable: {
    alignSelf: 'stretch',
  },
  advanceBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[4],
  },
  advanceBtnPending: {
    opacity: 0.5,
  },
  advanceBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
    letterSpacing: 0.3,
  },

  // Driver-controlled / awaiting caption row — no action; just status feedback.
  awaitingCaptionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  awaitingDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
  },
  awaitingCaption: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
  },
});
