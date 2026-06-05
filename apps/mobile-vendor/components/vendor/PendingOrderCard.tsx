import { Pressable, StyleSheet, Text, View } from 'react-native';
import { theme } from '@homechef/mobile-shared/theme';
import type { Order } from '../../hooks/useVendorOrders';

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
 * The triage card. The single most important block in the vendor app —
 * this is the moment the chef decides to commit or decline an order
 * within seconds, with wet hands. Designed for that moment:
 *  - name + total above (decision variables)
 *  - items summary + age in one line (context)
 *  - Reject as a naked underlined affordance (low gravity)
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
  const topRowContent = (
    <View style={styles.topRow}>
      <View style={styles.nameBlock}>
        <Text style={styles.customerName} numberOfLines={1}>
          {order.customerName}
        </Text>
        <Text style={styles.meta} numberOfLines={1}>
          {formatItemsSummary(order.items)}
          {`  ·  ${formatMinutesAgo(order.createdAt)}`}
        </Text>
      </View>
      <Text style={styles.total}>₹{order.total.toFixed(0)}</Text>
    </View>
  );

  return (
    <View style={styles.root}>
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
        <Pressable
          onPress={onReject}
          disabled={disabled}
          hitSlop={8}
          style={styles.rejectBtn}
          accessibilityRole="button"
          accessibilityLabel={`Reject order from ${order.customerName}`}
        >
          <Text style={styles.rejectLabel}>Reject</Text>
        </Pressable>
        {/* Outer Pressable carries only the pressed/disabled opacity AND
            the flex:1 needed to fill the row. iOS strips flex AND
            backgroundColor when style is returned as an array from the
            function-style prop, which renders the Accept button as
            invisible white-on-white shrink-wrapped to its text width.
            See feedback_ios_pressable_array_style.md. */}
        <Pressable
          onPress={onAccept}
          disabled={disabled}
          style={({ pressed }) => ({
            flex: 1,
            opacity: disabled ? 0.4 : pressed ? 0.85 : 1,
          })}
          accessibilityRole="button"
          accessibilityLabel={`Accept ₹${order.total.toFixed(0)} order from ${order.customerName}`}
        >
          <View style={styles.acceptBtn}>
            <Text style={styles.acceptLabel}>Accept</Text>
          </View>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
    gap: theme.spacing[3],
  },
  topRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: theme.spacing[3],
  },
  nameBlock: { flex: 1 },
  customerName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  meta: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    marginTop: 2,
  },
  total: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    letterSpacing: -0.2,
  },
  instructionsPill: {
    backgroundColor: theme.colors.herb.tint,
    borderRadius: theme.radius.sm,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  instructionsText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 19,
    color: theme.colors.herb.soft,
  },
  buttonRow: {
    flexDirection: 'row',
    gap: theme.spacing[2],
    alignItems: 'center',
  },
  rejectBtn: {
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    minHeight: 44,
    justifyContent: 'center',
  },
  rejectLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    textDecorationLine: 'underline',
  },
  acceptBtn: {
    flex: 1,
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  acceptLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
    letterSpacing: 0.3,
  },
});
