// Horizontal 4-step progress bar for the order detail screen.
// More compact than the vertical OrderTimeline — better for the detail
// screen's hero section where we want a quick visual status, not a full
// vertical list of steps.

import { StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';
import { getStepIndex, getStepLabels } from '../../lib/orderSteps';

interface OrderProgressBarProps {
  status: Order['status'];
  fulfillmentType?: Order['fulfillmentType'];
}

export function OrderProgressBar({
  status,
  fulfillmentType,
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
      {/* Segment track */}
      <View style={styles.track}>
        {STEPS.map((_, i) => (
          <View
            key={i}
            style={[
              styles.segment,
              i <= activeIndex ? styles.segmentFilled : styles.segmentEmpty,
              i < STEPS.length - 1 && styles.segmentGap,
            ]}
          />
        ))}
      </View>

      {/* Step labels */}
      <View style={styles.labelRow}>
        {STEPS.map((label, i) => (
          <Text
            key={label}
            style={[
              styles.label,
              i === activeIndex && styles.labelActive,
              i < activeIndex && styles.labelDone,
              i > activeIndex && styles.labelFuture,
            ]}
            numberOfLines={1}
          >
            {label}
          </Text>
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: 16,
    paddingBottom: 4,
  },
  track: {
    flexDirection: 'row',
    gap: 4,
    height: 4,
  },
  segment: {
    flex: 1,
    borderRadius: 2,
  },
  segmentFilled: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  segmentEmpty: {
    backgroundColor: customerColors.hairline,
  },
  segmentGap: {
    // gap handles spacing — kept as a descriptor-only style key
  },
  labelRow: {
    flexDirection: 'row',
    marginTop: 6,
  },
  label: {
    flex: 1,
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
});
