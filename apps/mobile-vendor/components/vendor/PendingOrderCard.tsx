import { useEffect, useState } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import Animated, {
  Easing,
  FadeInDown,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withTiming,
} from "react-native-reanimated";
import { theme } from "@homechef/mobile-shared/theme";
import type { Order } from "../../hooks/useVendorOrders";
import { orderSourceBadge } from "../../lib/orderSource";

const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

function formatMinutesAgo(iso: string, nowMs: number): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const mins = Math.max(0, Math.floor((nowMs - t) / 60_000));
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function formatItemsSummary(items: Order["items"]): string {
  const count = items.reduce((sum, i) => sum + (i.quantity ?? 1), 0);
  if (count === 1 && items[0]) return items[0].name;
  if (items.length === 1 && items[0]) return `${count} × ${items[0].name}`;
  return `${count} item${count === 1 ? "" : "s"}`;
}

interface PendingOrderCardProps {
  order: Order;
  disabled?: boolean;
  /** When true (Orders tab), surface `specialInstructions` as a
   *  persimmon-tint callout. When false (dashboard), keep terse. */
  showInstructions?: boolean;
  /** When provided, the customer/total header becomes tappable and opens
   *  the detail screen. Accept/Reject still work as before. */
  onOpenDetail?: () => void;
  onAccept: () => void;
  onReject: () => void;
}

/**
 * The triage card — v2 "canvas + cards" hero. The single most important
 * block in the vendor app: the moment the chef decides to commit or
 * decline an order within seconds, with wet hands. Designed for that:
 *  - name left, big tabular total right (decision variables)
 *  - items summary + escalating age chip in one line (context)
 *  - Reject as a quiet ghost button (low gravity)
 *  - Accept as the dominant ink-filled bar (commitment)
 *
 * Brand lock: Accept fill is ink (#0E0E0C), NOT persimmon. Persimmon
 * appears only as the special-instructions callout tint when surfaced.
 */
export function PendingOrderCard({
  order,
  disabled,
  showInstructions = false,
  onOpenDetail,
  onAccept,
  onReject,
}: PendingOrderCardProps) {
  // Live clock — ticks every 45 s so the age label and urgency colour
  // update without hammering the JS thread. Clears on unmount (T-grk-02).
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 45_000);
    return () => clearInterval(id);
  }, []);

  // Press feedback — 0.97 scale on the Accept bar (150 ms state-change
  // timing, no bounce). Skipped entirely under reduced motion.
  const reduceMotion = useReducedMotion();
  const acceptScale = useSharedValue(1);
  const acceptPressStyle = useAnimatedStyle(() => ({
    transform: [{ scale: acceptScale.value }],
  }));
  function onAcceptPressIn(): void {
    if (!reduceMotion) {
      acceptScale.value = withTiming(0.97, { duration: 150 });
    }
  }
  function onAcceptPressOut(): void {
    if (!reduceMotion) {
      acceptScale.value = withTiming(1, { duration: 150 });
    }
  }

  const ageMins = Math.max(
    0,
    Math.floor((now - new Date(order.createdAt).getTime()) / 60_000),
  );
  // Age chip escalation — same thresholds as the old inline-text colour,
  // now expressed as a tinted pill per UI-V2 §2.
  const ageChipBg =
    ageMins >= 10
      ? theme.colors.destructive.tint
      : ageMins >= 5
        ? theme.colors.amber.tint
        : theme.colors.mist.DEFAULT;
  const ageChipText =
    ageMins >= 10
      ? theme.colors.destructive.DEFAULT
      : ageMins >= 5
        ? theme.colors.ink.DEFAULT
        : theme.colors.ink.soft;

  const sourceBadge = orderSourceBadge(order.source);

  const topRowContent = (
    <View style={styles.headerBlock}>
      <View style={styles.topRow}>
        <View style={styles.nameWithBadge}>
          <Text style={styles.customerName} numberOfLines={1}>
            {order.customerName}
          </Text>
          {sourceBadge ? (
            <View style={[styles.sourceBadge, { backgroundColor: sourceBadge.bg }]}>
              <Text style={[styles.sourceBadgeText, { color: sourceBadge.color }]}>
                {sourceBadge.label}
              </Text>
            </View>
          ) : null}
        </View>
        <Text style={styles.total}>₹{order.total.toFixed(0)}</Text>
      </View>
      <View style={styles.metaRow}>
        <Text style={styles.meta} numberOfLines={1}>
          {formatItemsSummary(order.items)}
        </Text>
        <View style={[styles.ageChip, { backgroundColor: ageChipBg }]}>
          <Text style={[styles.ageChipLabel, { color: ageChipText }]}>
            {formatMinutesAgo(order.createdAt, now)}
          </Text>
        </View>
      </View>
    </View>
  );

  return (
    <Animated.View
      style={styles.root}
      entering={
        reduceMotion
          ? undefined
          : FadeInDown.duration(250).easing(ENTRANCE_EASING)
      }
    >
      {onOpenDetail ? (
        <Pressable
          onPress={onOpenDetail}
          accessibilityRole="button"
          accessibilityLabel={`Open order details for ${order.customerName}`}
        >
          {({ pressed }) => (
            // Inner-View pattern — iOS strips flex on Pressable function-style
            // sometimes. The topRow already has its own flexDirection so we
            // wrap it in a minimal opacity layer instead of restyling.
            <View style={pressed ? { opacity: 0.85 } : undefined}>
              {topRowContent}
            </View>
          )}
        </Pressable>
      ) : (
        topRowContent
      )}

      {showInstructions && order.specialInstructions ? (
        <View style={styles.instructionsPill}>
          <Text style={styles.instructionsText} numberOfLines={2}>
            {order.specialInstructions}
          </Text>
        </View>
      ) : null}

      <View style={styles.buttonRow}>
        {/* Ghost button. Visual styles (bg, border) live on the inner
            View — same iOS Pressable rule as Accept below. The outer
            Pressable returns a plain OBJECT (never an array) from the
            function-style prop. */}
        <Pressable
          onPress={onReject}
          disabled={disabled}
          hitSlop={8}
          style={({ pressed }) => ({
            opacity: disabled ? 0.4 : pressed ? 0.85 : 1,
          })}
          accessibilityRole="button"
          accessibilityLabel={`Reject order from ${order.customerName}`}
        >
          <View style={styles.rejectBtn}>
            <Text style={styles.rejectLabel}>Reject</Text>
          </View>
        </Pressable>
        {/* flex:1 lives on a PLAIN wrapper View — never on the Pressable's
            function-style prop and never on the visual inner View (iOS
            drops flex from the former and the latter fights the row's
            cross-axis alignment → squashed bar).
            See feedback_ios_pressable_array_style.md. */}
        <View style={styles.acceptWrap}>
          <Pressable
            onPress={onAccept}
            onPressIn={onAcceptPressIn}
            onPressOut={onAcceptPressOut}
            disabled={disabled}
            accessibilityRole="button"
            accessibilityLabel={`Accept ₹${order.total.toFixed(0)} order from ${order.customerName}`}
          >
            <Animated.View
              style={[
                styles.acceptBtn,
                acceptPressStyle,
                disabled && { opacity: 0.4 },
              ]}
            >
              <Text style={styles.acceptLabel}>Accept</Text>
            </Animated.View>
          </Pressable>
        </View>
      </View>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  root: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    gap: theme.spacing[3],
    ...theme.shadow[2],
  },
  headerBlock: {
    gap: theme.spacing[1],
  },
  topRow: {
    flexDirection: "row",
    alignItems: "flex-start",
    gap: theme.spacing[3],
  },
  nameWithBadge: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    gap: theme.spacing[2],
  },
  customerName: {
    flexShrink: 1,
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  sourceBadge: {
    borderRadius: 999,
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 2,
  },
  sourceBadgeText: {
    fontFamily: "Inter-SemiBold",
    fontSize: 11,
  },
  metaRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: theme.spacing[3],
  },
  meta: {
    flex: 1,
    fontFamily: "Inter",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  ageChip: {
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 3,
  },
  ageChipLabel: {
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.caption.size,
  },
  total: {
    fontFamily: "Geist-Bold",
    fontSize: 22,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ["tabular-nums"],
    letterSpacing: -0.3,
  },
  instructionsPill: {
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  instructionsText: {
    fontFamily: "Inter",
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 19,
    color: theme.colors.ink.DEFAULT,
  },
  buttonRow: {
    flexDirection: "row",
    gap: theme.spacing[2],
    alignItems: "stretch",
  },
  acceptWrap: {
    flex: 1,
  },
  rejectBtn: {
    width: 96,
    minHeight: 48,
    backgroundColor: theme.colors.paper,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.md,
    alignItems: "center",
    justifyContent: "center",
  },
  rejectLabel: {
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  acceptBtn: {
    width: "100%",
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    minHeight: 52,
    alignItems: "center",
    justifyContent: "center",
  },
  acceptLabel: {
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
    letterSpacing: 0.3,
  },
});
