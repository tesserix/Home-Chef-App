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

// Status chip palette for active orders — mirrors index.tsx STATUS_CHIP.
// accepted = info tint (blue-ish), preparing = mist (neutral working),
// ready = success tint (green, no chef action needed).
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

const STATUS_CHIP_FALLBACK = {
  bg: theme.colors.mist.DEFAULT,
  text: theme.colors.ink.soft,
};

// Label for each active status shown on the chip.
const STATUS_LABEL: Record<string, string> = {
  accepted: 'Accepted',
  preparing: 'Preparing',
  ready: 'Ready',
};

// The one-tap label on the advance button.
const ADVANCE_LABEL: Record<string, string> = {
  accepted: 'Start Preparing',
  preparing: 'Mark Ready',
};

// The next status to transition to.
const NEXT_STATUS: Record<string, Order['status']> = {
  accepted: 'preparing',
  preparing: 'ready',
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

interface ActiveOrderCardProps {
  order: Order;
  /** True while the status-advance mutation is in flight for this order. */
  isPending?: boolean;
  /** Called when the chef taps the advance-status action button. */
  onAdvance: (orderId: string, nextStatus: Order['status']) => void;
  /** Called when the chef taps the card body — navigates to the detail screen. */
  onOpenDetail: (orderId: string) => void;
}

/**
 * One-tap active-order card for the Active/Cooking tab.
 *
 * Design intent (vendor monochrome, .impeccable.md §Per-role density):
 *  - Card body tappable → opens detail (full info).
 *  - Action button: ink-filled "Start Preparing" or "Mark Ready" depending
 *    on current status. Rendered with the inner-View pattern to avoid the
 *    iOS Pressable function-style style-drop bug.
 *  - `ready` state: no action button; calm success-chip + "Ready · awaiting
 *    pickup" copy instead. Green success chip is consistent with the detail
 *    screen's status chip and communicates "all done from chef's side".
 *  - Mutation pending: button opacity 0.5 + disabled — prevents double-tap.
 *  - Touch target: `minHeight: 44` on the action button satisfies Apple HIG.
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

  function handleAdvance(): void {
    const next = NEXT_STATUS[order.status];
    if (!next || isPending) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    onAdvance(order.id, next);
  }

  const chip = STATUS_CHIP[order.status] ?? STATUS_CHIP_FALLBACK;
  const advanceLabel = ADVANCE_LABEL[order.status];
  const hasAction = !!advanceLabel;

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
            {/* Meta row: items summary + age + status chip */}
            <View style={styles.metaRow}>
              <Text style={styles.meta} numberOfLines={1}>
                {formatItemsSummary(order.items)}
              </Text>
              <Text style={styles.age}>{formatMinutesAgo(order.createdAt)}</Text>
              <View style={[styles.chip, { backgroundColor: chip.bg }]}>
                <Text style={[styles.chipLabel, { color: chip.text }]}>
                  {STATUS_LABEL[order.status] ?? order.status}
                </Text>
              </View>
            </View>
          </View>
        )}
      </Pressable>

      {/* Separator between body and action area */}
      <View style={styles.divider} />

      {/* Action zone: advance button OR ready-state caption */}
      {hasAction ? (
        // Outer wrapper supplies full width; Pressable wraps only the button
        // visual. This is the three-layer pattern from PendingOrderCard's
        // acceptWrap: wrapper View → Pressable (interaction) → inner View
        // (visual).
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
      ) : (
        // `ready` state — no action. Success-tinted caption row.
        <View style={styles.readyCaptionRow}>
          <View style={styles.readyDot} />
          <Text style={styles.readyCaption}>Ready · awaiting pickup</Text>
        </View>
      )}
    </Animated.View>
  );
}

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
    gap: theme.spacing[1],
  },
  topRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: theme.spacing[3],
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
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
  },
  meta: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  age: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
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

  // Ready-state caption row — no action; just status feedback.
  readyCaptionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  readyDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: theme.colors.success.DEFAULT,
  },
  readyCaption: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.success.soft,
  },
});
