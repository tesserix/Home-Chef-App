import { View, Text, StyleSheet } from 'react-native';
import type { Order } from '../../types/customer';

interface OrderTimelineProps {
  status: Order['status'];
  estimatedDeliveryTime?: string;
}

const STEPS = ['confirmed', 'preparing', 'picked_up', 'delivered'] as const;
type StepKey = typeof STEPS[number];

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
    case 'confirmed':
      return 0;
    case 'preparing':
      return 1;
    case 'ready':
      return 1; // "ready" sits between preparing and picked_up
    case 'picked_up':
      return 2;
    case 'delivered':
      return 3;
    case 'cancelled':
      return -1;
    default:
      return -1;
  }
}

export function OrderTimeline({ status, estimatedDeliveryTime }: OrderTimelineProps) {
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
          const isCompleted = index <= currentIndex;
          const isCurrent = index === currentIndex;

          return (
            <View key={step} style={styles.stepRow}>
              {/* Connector line above (except first step) */}
              {index > 0 && (
                <View
                  style={[
                    styles.connector,
                    index <= currentIndex ? styles.connectorActive : styles.connectorPending,
                  ]}
                />
              )}
              <View style={styles.stepContent}>
                {/* Step circle */}
                <View
                  style={[
                    styles.circle,
                    isCompleted && styles.circleCompleted,
                    isCurrent && styles.circleCurrent,
                  ]}
                >
                  <Text
                    style={[
                      styles.circleText,
                      isCompleted && styles.circleTextCompleted,
                    ]}
                  >
                    {index + 1}
                  </Text>
                </View>
                {/* Step label */}
                <Text
                  style={[
                    styles.label,
                    isCompleted && styles.labelCompleted,
                    isCurrent && styles.labelCurrent,
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
  eta: {
    fontSize: 13,
    color: '#7a7a76',
    marginBottom: 12,
  },
  steps: {
    gap: 0,
  },
  stepRow: {
    position: 'relative',
  },
  connector: {
    position: 'absolute',
    left: 11,
    top: -10,
    width: 2,
    height: 10,
  },
  connectorActive: {
    backgroundColor: '#3e6b3c',
  },
  connectorPending: {
    backgroundColor: '#d4d3ce',
  },
  stepContent: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 8,
  },
  circle: {
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: '#d4d3ce',
    alignItems: 'center',
    justifyContent: 'center',
  },
  circleCompleted: {
    backgroundColor: '#3e6b3c',
  },
  circleCurrent: {
    backgroundColor: '#3e6b3c',
  },
  circleText: {
    fontSize: 11,
    fontWeight: '700',
    color: '#7a7a76',
  },
  circleTextCompleted: {
    color: '#fafaf7',
  },
  label: {
    fontSize: 14,
    color: '#7a7a76',
  },
  labelCompleted: {
    color: '#1a1a18',
    fontWeight: '500',
  },
  labelCurrent: {
    color: '#3e6b3c',
    fontWeight: '600',
  },
});
