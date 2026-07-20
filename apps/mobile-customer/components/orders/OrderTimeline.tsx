import { View, Text, StyleSheet } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';
import { getStepIndex, getStepLabels, getStepStatuses } from '../../lib/orderSteps';
import { StageIcon } from '../status/StageIcon';

interface OrderTimelineProps {
  status: Order['status'];
  fulfillmentType?: Order['fulfillmentType'];
  estimatedDeliveryTime?: string;
}

export function OrderTimeline({
  status,
  fulfillmentType,
  estimatedDeliveryTime,
}: OrderTimelineProps) {
  const STEPS = getStepLabels(fulfillmentType);
  const currentIndex = getStepIndex(status, fulfillmentType);
  // Index-aligned with STEPS — gives the current step its own animated icon.
  const STEP_STATUSES = getStepStatuses(fulfillmentType);

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
        {STEPS.map((label, index) => {
          const isCompleted = index < currentIndex;
          const isCurrent = index === currentIndex;
          const isFuture = index > currentIndex;

          return (
            <View key={label} style={styles.stepRow}>
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
                  // Current step carries the animated stage icon — the one
                  // moving element in the timeline. The container keeps the
                  // 16px footprint (and -2 offset) of the old coral ring so the
                  // connector line and labels stay aligned.
                  <View style={styles.dotCurrentOuter}>
                    <StageIcon
                      status={STEP_STATUSES[index] ?? status}
                      size={14}
                      color={customerColors.coral.DEFAULT}
                    />
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
                  {label}
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

  // Current step: holds the animated StageIcon. Same 16px footprint and -2
  // offset as the coral ring it replaced, so the connector (left: 5) and the
  // label column stay aligned. No border — the icon is the affordance.
  dotCurrentOuter: {
    width: 16,
    height: 16,
    alignItems: 'center',
    justifyContent: 'center',
    // Compensate for the extra 4px so step content stays aligned
    marginLeft: -2,
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
