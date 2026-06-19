import { View, Text, StyleSheet } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';

interface OrderTimelineProps {
  status: Order['status'];
  estimatedDeliveryTime?: string;
}

const STEPS = ['confirmed', 'preparing', 'picked_up', 'delivered'] as const;
type StepKey = (typeof STEPS)[number];

const STEP_LABELS: Record<StepKey, string> = {
  confirmed: 'Confirmed',
  preparing: 'Preparing',
  picked_up: 'On the Way',
  delivered: 'Delivered',
};

// Map order statuses to the closest timeline step index
function getStepIndex(status: Order['status']): number {
  switch (status) {
    case 'pending':
      return -1;
    case 'accepted':
      return 0;
    case 'preparing':
      return 1;
    case 'ready':
      return 1; // "ready" sits between preparing and picked_up
    case 'picked_up':
      return 2;
    case 'delivering':
      return 2; // out for delivery — same step as picked_up
    case 'delivered':
      return 3;
    case 'cancelled':
      return -1;
    default:
      return -1;
  }
}

export function OrderTimeline({
  status,
  estimatedDeliveryTime,
}: OrderTimelineProps) {
  const currentIndex = getStepIndex(status);

  return (
    <View style={styles.container}>
      {estimatedDeliveryTime && (
        <Text style={styles.eta}>
          Est. arrival:{' '}
          {new Date(estimatedDeliveryTime).toLocaleTimeString('en-IN', {
            hour: '2-digit',
            minute: '2-digit',
          })}
        </Text>
      )}
      <View style={styles.steps}>
        {STEPS.map((step, index) => {
          const isCompleted = index < currentIndex;
          const isCurrent = index === currentIndex;
          const isFuture = index > currentIndex;

          return (
            <View key={step} style={styles.stepRow}>
              {/* Vertical connector line above (except first step) */}
              {index > 0 && (
                <View
                  style={[
                    styles.connector,
                    index <= currentIndex
                      ? styles.connectorActive
                      : styles.connectorPending,
                  ]}
                />
              )}

              <View style={styles.stepContent}>
                {/* Step dot/circle */}
                {isCurrent ? (
                  // Current step: coral ring with filled center
                  <View style={styles.dotCurrentOuter}>
                    <View style={styles.dotCurrentInner} />
                  </View>
                ) : (
                  // Completed = coral filled dot; future = hairline dot
                  <View
                    style={[
                      styles.dot,
                      isCompleted && styles.dotCompleted,
                      isFuture && styles.dotFuture,
                    ]}
                  />
                )}

                {/* Step label */}
                <Text
                  style={[
                    styles.label,
                    isCompleted && styles.labelCompleted,
                    isCurrent && styles.labelCurrent,
                    isFuture && styles.labelFuture,
                  ]}
                >
                  {STEP_LABELS[step]}
                </Text>
              </View>
            </View>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginTop: 12,
  },
  // ETA line — tabular numeral for the time portion
  eta: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 16,
    fontVariant: ['tabular-nums'],
  },
  steps: {
    // No gap — spacing is handled by connector heights + stepContent paddingVertical
  },
  stepRow: {
    position: 'relative',
  },

  // Hairline connector between steps — active = coral, pending = hairline grey
  connector: {
    position: 'absolute',
    // Left-center relative to the 12px dot (dot left margin ~0, dot width 12 → center = 5)
    left: 5,
    top: -12,
    width: 1,
    height: 12,
  },
  connectorActive: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  connectorPending: {
    backgroundColor: customerColors.hairline,
  },

  stepContent: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 14,
    paddingVertical: 12,
  },

  // Completed step: coral filled dot (12×12)
  dot: {
    width: 12,
    height: 12,
    borderRadius: 6,
  },
  dotCompleted: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  // Future step: hairline outlined dot
  dotFuture: {
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: 1.5,
    borderColor: customerColors.hairline,
  },

  // Current step: coral outer ring + smaller coral filled center
  dotCurrentOuter: {
    width: 16,
    height: 16,
    borderRadius: 8,
    borderWidth: 2,
    borderColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    // Compensate for the extra 4px so step content stays aligned
    marginLeft: -2,
  },
  dotCurrentInner: {
    width: 7,
    height: 7,
    borderRadius: 4,
    backgroundColor: customerColors.coral.DEFAULT,
  },

  // Labels
  label: {
    fontFamily: 'Inter',
    fontSize: 14,
  },
  labelCompleted: {
    fontFamily: 'Inter-SemiBold',
    color: customerColors.charcoal.DEFAULT,
  },
  labelCurrent: {
    fontFamily: 'Inter-SemiBold',
    color: customerColors.coral.DEFAULT,
  },
  labelFuture: {
    color: customerColors.charcoal.soft,
  },
});
