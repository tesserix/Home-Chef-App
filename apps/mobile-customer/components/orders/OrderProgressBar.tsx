// Horizontal 4-step progress stepper for the order detail screen.
// More compact than the vertical OrderTimeline — better for the detail
// screen's hero section where we want a quick visual status, not a full
// vertical list of steps.
//
// Spec §5 (order tracking/detail): "status stepper as a shared-quality
// component (coral progress, check icons, timestamps)". Each step is a node
// (coral-filled + white check once completed, coral ring for the current
// step, hairline outline for future steps) joined by a coral/hairline
// connector line — timestamps render per-step when the order data carries
// one (none of today's order fields expose per-step timestamps yet, so the
// row silently omits them rather than fabricating a value).

import { StyleSheet, Text, View } from 'react-native';
import { Check } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';
import { getStepIndex, getStepLabels } from '../../lib/orderSteps';

interface OrderProgressBarProps {
  status: Order['status'];
  fulfillmentType?: Order['fulfillmentType'];
  /** Optional per-step timestamp, index-aligned with the step labels. Only
   *  rendered when a caller actually has the data — omitted entirely today. */
  timestamps?: (string | undefined)[];
}

function formatStepTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-IN', { hour: 'numeric', minute: '2-digit' });
}

export function OrderProgressBar({
  status,
  fulfillmentType,
  timestamps,
}: OrderProgressBarProps) {
  const STEPS = getStepLabels(fulfillmentType);
  // getStepIndex returns -1 for a `pending` order (placed, but the chef hasn't
  // confirmed — and it may not even be paid yet). Do NOT clamp that up to 0: that
  // lit the first "Confirmed" step for an unconfirmed order, contradicting the
  // "Payment pending / awaiting confirmation" state shown elsewhere on the screen
  // and in the orders list. A pending order shows an un-started bar; step 0 fills
  // only once the chef accepts (status 'accepted').
  const activeIndex = getStepIndex(status, fulfillmentType);

  return (
    <View style={styles.container}>
      <View style={styles.stepRow}>
        {STEPS.map((label, i) => {
          const isCompleted = i < activeIndex;
          const isCurrent = i === activeIndex;
          const isFuture = i > activeIndex;
          // Connector segments: the line between node i-1/i (left) is "passed"
          // once step i has been reached; the line between i/i+1 (right) is
          // "passed" once step i is fully completed.
          const leftActive = i <= activeIndex && i > 0;
          const rightActive = i < activeIndex;

          return (
            <View key={label} style={styles.stepItem}>
              <View style={styles.nodeRow}>
                {i === 0 ? (
                  <View style={styles.connectorSpacer} />
                ) : (
                  <View
                    style={[styles.connector, leftActive ? styles.connectorActive : styles.connectorInactive]}
                  />
                )}
                <View
                  style={[
                    styles.node,
                    isCompleted && styles.nodeCompleted,
                    isCurrent && styles.nodeCurrent,
                  ]}
                >
                  {isCompleted ? (
                    <Check size={10} color={customerColors.canvas} strokeWidth={3} />
                  ) : isCurrent ? (
                    <View style={styles.nodeCurrentDot} />
                  ) : null}
                </View>
                {i === STEPS.length - 1 ? (
                  <View style={styles.connectorSpacer} />
                ) : (
                  <View
                    style={[styles.connector, rightActive ? styles.connectorActive : styles.connectorInactive]}
                  />
                )}
              </View>

              {/* Step label */}
              <Text
                style={[
                  styles.label,
                  isCurrent && styles.labelActive,
                  isCompleted && styles.labelDone,
                  isFuture && styles.labelFuture,
                ]}
                numberOfLines={1}
              >
                {label}
              </Text>
              {/* Per-step timestamp — only when the caller supplies one. */}
              {timestamps?.[i] ? (
                <Text style={styles.timeLabel} numberOfLines={1}>
                  {formatStepTime(timestamps[i]!)}
                </Text>
              ) : null}
            </View>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: 16,
    paddingBottom: 4,
  },
  stepRow: {
    flexDirection: 'row',
  },
  stepItem: {
    flex: 1,
    alignItems: 'center',
  },
  nodeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    width: '100%',
  },
  connector: {
    flex: 1,
    height: 2,
    borderRadius: 1,
  },
  connectorSpacer: {
    flex: 1,
  },
  connectorActive: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  connectorInactive: {
    backgroundColor: customerColors.hairline,
  },
  // Step node — hairline ring by default; coral fill + check once completed;
  // coral ring + coral dot while current (the one moving element).
  node: {
    width: 18,
    height: 18,
    borderRadius: 9,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1.5,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
  },
  nodeCompleted: {
    backgroundColor: customerColors.coral.DEFAULT,
    borderColor: customerColors.coral.DEFAULT,
  },
  nodeCurrent: {
    borderColor: customerColors.coral.DEFAULT,
    backgroundColor: customerColors.canvas,
  },
  nodeCurrentDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: customerColors.coral.DEFAULT,
  },
  label: {
    marginTop: 6,
    fontFamily: 'Inter',
    fontSize: 10,
    textAlign: 'center',
  },
  labelActive: {
    color: customerColors.coral.DEFAULT,
    fontFamily: 'Inter-SemiBold',
  },
  labelDone: {
    color: customerColors.charcoal.DEFAULT,
    fontFamily: 'Inter-SemiBold',
  },
  labelFuture: {
    color: customerColors.charcoal.soft,
  },
  timeLabel: {
    marginTop: 1,
    fontFamily: 'Inter',
    fontSize: 9,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
});
