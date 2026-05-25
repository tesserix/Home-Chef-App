import * as Haptics from 'expo-haptics';
import { useCallback, useRef, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { PanGestureHandler, PanGestureHandlerGestureEvent } from 'react-native-gesture-handler';
import Animated, {
  runOnJS,
  useAnimatedGestureHandler,
  useAnimatedStyle,
  useSharedValue,
  withSpring,
} from 'react-native-reanimated';

const THUMB_SIZE = 56;
const TRACK_HEIGHT = 64;

interface SlideToConfirmProps {
  label: string;
  onConfirm: () => void;
  disabled?: boolean;
  color?: string;
}

type GestureContext = {
  startX: number;
};

export function SlideToConfirm({
  label,
  onConfirm,
  disabled = false,
  color = '#C2410C',
}: SlideToConfirmProps) {
  const translateX = useSharedValue(0);
  const [trackWidth, setTrackWidth] = useState(0);
  const confirmedRef = useRef(false);

  const fireConfirm = useCallback(() => {
    if (confirmedRef.current) return;
    confirmedRef.current = true;
    Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    onConfirm();
  }, [onConfirm]);

  const gestureHandler = useAnimatedGestureHandler<
    PanGestureHandlerGestureEvent,
    GestureContext
  >({
    onStart: (_, ctx) => {
      ctx.startX = translateX.value;
    },
    onActive: (event, ctx) => {
      const maxTranslate = trackWidth - THUMB_SIZE;
      translateX.value = Math.max(
        0,
        Math.min(ctx.startX + event.translationX, maxTranslate),
      );
    },
    onEnd: () => {
      const maxTranslate = trackWidth - THUMB_SIZE;
      if (maxTranslate > 0 && translateX.value > maxTranslate * 0.75) {
        translateX.value = withSpring(maxTranslate);
        runOnJS(fireConfirm)();
      } else {
        translateX.value = withSpring(0, { damping: 15 });
      }
    },
  });

  const thumbStyle = useAnimatedStyle(() => ({
    transform: [{ translateX: translateX.value }],
  }));

  return (
    <View style={[styles.container, disabled && styles.disabled]}>
      <View
        style={[styles.track, { backgroundColor: color + '26' }]}
        onLayout={(e) => setTrackWidth(e.nativeEvent.layout.width)}
      >
        <Text style={[styles.label, { color }]} numberOfLines={1}>
          Slide to {label}
        </Text>
        <PanGestureHandler
          onGestureEvent={gestureHandler}
          enabled={!disabled && trackWidth > 0}
        >
          <Animated.View
            style={[
              styles.thumb,
              { backgroundColor: color },
              thumbStyle,
            ]}
          />
        </PanGestureHandler>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginHorizontal: 0,
  },
  disabled: {
    opacity: 0.5,
  },
  track: {
    height: TRACK_HEIGHT,
    borderRadius: TRACK_HEIGHT / 2,
    justifyContent: 'center',
    alignItems: 'center',
    overflow: 'hidden',
    position: 'relative',
  },
  label: {
    fontSize: 16,
    fontWeight: '600',
    position: 'absolute',
    alignSelf: 'center',
  },
  thumb: {
    width: THUMB_SIZE,
    height: THUMB_SIZE,
    borderRadius: THUMB_SIZE / 2,
    position: 'absolute',
    left: 0,
    top: (TRACK_HEIGHT - THUMB_SIZE) / 2,
  },
});
