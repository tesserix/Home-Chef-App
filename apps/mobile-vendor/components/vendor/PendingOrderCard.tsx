import { useEffect, useState } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import Animated, {
  Easing,
  FadeInDown,
  useReducedMotion,
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
  /** Tapping the card opens the detail screen, where the chef reviews the
   *  pickup time + pricing before accepting or rejecting. Accept/Reject are
   *  deliberately NOT on this card — the chef must open the order first. */
  onOpenDetail: () => void;
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
}: PendingOrderCardProps) {
  // Live clock — ticks every 45 s so the age label and urgency colour
  // update without hammering the JS thread. Clears on unmount (T-grk-02).
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 45_000);
    return () => clearInterval(id);
  }, []);

  const reduceMotion = useReducedMotion();

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
          {/* Pickup vs delivery, at a glance on the INCOMING card — a pickup order
              means no packing for a rider and the customer collects, so the chef
              needs to know before accepting, not only after opening the detail. */}
          {order.fulfillmentType === "pickup" ? (
            <View style={styles.pickupBadge}>
              <Text style={styles.pickupBadgeText}>PICKUP</Text>
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
      entering={
        reduceMotion
          ? undefined
          : FadeInDown.duration(250).easing(ENTRANCE_EASING)
      }
    >
      {/* The whole card is the tap target — it opens the detail screen where
          the chef reviews pickup time + pricing and then accepts/rejects.
          No inline accept/reject here by design (they can't decide blind). */}
      <Pressable
        onPress={onOpenDetail}
        disabled={disabled}
        accessibilityRole="button"
        accessibilityLabel={`Review order from ${order.customerName}, ₹${order.total.toFixed(0)}`}
        accessibilityHint="Opens the order to review and respond"
      >
        {({ pressed }) => (
          <View style={[styles.root, pressed && { opacity: 0.9 }]}>
            {topRowContent}

            {showInstructions && order.specialInstructions ? (
              <View style={styles.instructionsPill}>
                <Text style={styles.instructionsText} numberOfLines={2}>
                  {order.specialInstructions}
                </Text>
              </View>
            ) : null}

            <View style={styles.ctaRow}>
              <Text style={styles.ctaLabel}>Review & respond</Text>
              <Text style={styles.ctaChevron}>›</Text>
            </View>
          </View>
        )}
      </Pressable>
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
  pickupBadge: {
    borderRadius: 999,
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 2,
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  pickupBadgeText: {
    fontFamily: "Inter-SemiBold",
    fontSize: 10,
    letterSpacing: 0.4,
    color: theme.colors.paper,
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
  ctaRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginTop: theme.spacing[1],
    paddingTop: theme.spacing[3],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },
  ctaLabel: {
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  ctaChevron: {
    fontFamily: "Inter-SemiBold",
    fontSize: 22,
    lineHeight: 22,
    color: theme.colors.ink.soft,
  },
});
