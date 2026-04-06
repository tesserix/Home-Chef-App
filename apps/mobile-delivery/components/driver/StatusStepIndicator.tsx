import React, { useEffect } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import Animated, {
  useAnimatedStyle,
  useSharedValue,
  withRepeat,
  withSequence,
  withTiming,
} from 'react-native-reanimated';

const STEPS = ['Picked Up', 'In Transit', 'At Dropoff', 'Delivered'];

const ORANGE = '#FF6B35';
const GREY = '#D1D5DB';

function statusToStepIndex(status: string): number {
  switch (status) {
    case 'assigned':
    case 'at_pickup':
      return 0;
    case 'picked_up':
      return 1;
    case 'in_transit':
      return 2;
    case 'at_dropoff':
      return 3;
    case 'delivered':
      return 4;
    default:
      return 0;
  }
}

interface PulsingDotProps {
  color: string;
  isPulsing: boolean;
}

function PulsingDot({ color, isPulsing }: PulsingDotProps) {
  const scale = useSharedValue(1);

  useEffect(() => {
    if (isPulsing) {
      scale.value = withRepeat(
        withSequence(
          withTiming(1.35, { duration: 500 }),
          withTiming(1, { duration: 500 }),
        ),
        -1,
        false,
      );
    } else {
      scale.value = 1;
    }
  }, [isPulsing, scale]);

  const animatedStyle = useAnimatedStyle(() => ({
    transform: [{ scale: scale.value }],
  }));

  return (
    <Animated.View
      style={[
        styles.dot,
        { backgroundColor: color },
        animatedStyle,
      ]}
    />
  );
}

interface StatusStepIndicatorProps {
  currentStatus: string;
}

export function StatusStepIndicator({ currentStatus }: StatusStepIndicatorProps) {
  const activeIndex = statusToStepIndex(currentStatus);

  return (
    <View style={styles.container}>
      {STEPS.map((step, index) => {
        const isCompleted = index < activeIndex;
        const isCurrent = index === activeIndex && currentStatus !== 'delivered';
        const dotColor = isCompleted || isCurrent ? ORANGE : GREY;
        const isLast = index === STEPS.length - 1;

        return (
          <React.Fragment key={step}>
            <View style={styles.stepWrapper}>
              <PulsingDot color={dotColor} isPulsing={isCurrent} />
              <Text
                style={[
                  styles.stepLabel,
                  { color: isCompleted || isCurrent ? ORANGE : GREY },
                ]}
                numberOfLines={1}
              >
                {step}
              </Text>
            </View>
            {!isLast && (
              <View
                style={[
                  styles.connector,
                  { backgroundColor: isCompleted ? ORANGE : GREY },
                ]}
              />
            )}
          </React.Fragment>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    paddingVertical: 12,
    paddingHorizontal: 4,
  },
  stepWrapper: {
    alignItems: 'center',
    width: 64,
  },
  dot: {
    width: 14,
    height: 14,
    borderRadius: 7,
    marginBottom: 4,
  },
  stepLabel: {
    fontSize: 10,
    fontWeight: '500',
    textAlign: 'center',
  },
  connector: {
    flex: 1,
    height: 2,
    marginTop: 6,
    borderRadius: 1,
  },
});
