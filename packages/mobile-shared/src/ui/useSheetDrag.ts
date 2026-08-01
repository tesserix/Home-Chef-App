// useSheetDrag — drag-the-grabber-down-to-dismiss for bottom sheets.
//
// Every sheet in the app draws a grabber pill at the top, which reads as "you
// can drag this". Until now that was decorative: there was no gesture handler
// anywhere in SheetBase, so the only way out was tapping the backdrop. This
// makes the affordance honest.
//
// NATIVE-DRIVER CONSTRAINT (the reason this uses setValue rather than
// Animated.event): the `translateY` value passed in is already animated by
// `useNativeDriver: true` timings for the sheet's enter/exit. Attaching a
// JS-driven `Animated.event` to that same node throws at runtime —
// "Attempting to run JS driven animation on animated node that has been moved
// to native". `Animated.Value.setValue()` is legal on a native-driven node
// (it forwards to the native animated module), so the drag tracks by calling
// setValue directly, and the spring-back is itself a native-driven timing.

import { useMemo, useRef } from 'react';
import {
  Animated,
  Easing,
  PanResponder,
  type PanResponderInstance,
} from 'react-native';
import { theme } from '../theme/tokens';

/** Drag further than this and release → dismiss. */
const DISMISS_DISTANCE = 96;
/** Or flick faster than this (px/ms), regardless of distance → dismiss. */
const DISMISS_VELOCITY = 0.5;
/** Ignore sub-pixel jitter so a tap on the grabber isn't read as a drag. */
const DRAG_ACTIVATION_SLOP = 4;

const SPRING_BACK_DURATION = theme.motion.duration.micro;
const SPRING_BACK_EASING = Easing.bezier(...theme.motion.easing.state);

export interface UseSheetDragOptions {
  /** The same Animated.Value the sheet shell uses for its enter/exit transform. */
  translateY: Animated.Value;
  /** Called when the drag passes the dismiss threshold. */
  onDismiss: () => void;
  /** When true, skip the spring-back animation (snap instead). */
  reduceMotion?: boolean;
  /** Set false to disable dragging entirely (keeps hook order stable). */
  enabled?: boolean;
}

export interface UseSheetDragResult {
  panHandlers: PanResponderInstance['panHandlers'];
}

export function useSheetDrag({
  translateY,
  onDismiss,
  reduceMotion = false,
  enabled = true,
}: UseSheetDragOptions): UseSheetDragResult {
  // Keep the latest callback/flags in refs so the PanResponder can stay
  // memoised for the component's lifetime. Rebuilding it mid-gesture would
  // drop the in-flight drag.
  const onDismissRef = useRef(onDismiss);
  onDismissRef.current = onDismiss;
  const reduceMotionRef = useRef(reduceMotion);
  reduceMotionRef.current = reduceMotion;
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  // Returns the panel to its resting position after a drag that didn't reach
  // the dismiss threshold. Reads only refs and the stable Animated.Value, so
  // the copy captured by the memoised responder never goes stale.
  const settleBack = useRef(() => {
    if (reduceMotionRef.current) {
      translateY.setValue(0);
      return;
    }
    Animated.timing(translateY, {
      toValue: 0,
      duration: SPRING_BACK_DURATION,
      easing: SPRING_BACK_EASING,
      useNativeDriver: true,
    }).start();
  }).current;

  const responder = useMemo(
    () =>
      PanResponder.create({
        // Don't claim the gesture on touch-down — that would swallow taps.
        onStartShouldSetPanResponder: () => false,
        onMoveShouldSetPanResponder: (_e, g) =>
          enabledRef.current &&
          g.dy > DRAG_ACTIVATION_SLOP &&
          g.dy > Math.abs(g.dx),
        onPanResponderMove: (_e, g) => {
          // Downward only. Dragging up shouldn't lift the sheet off its
          // resting position — there's no expanded snap point to reach.
          translateY.setValue(Math.max(0, g.dy));
        },
        onPanResponderRelease: (_e, g) => {
          if (g.dy > DISMISS_DISTANCE || g.vy > DISMISS_VELOCITY) {
            // Hand off to the shell's own exit animation, which runs from
            // wherever the finger left the panel.
            onDismissRef.current();
            return;
          }
          settleBack();
        },
        // Interrupted by a system gesture / another responder — treat as cancel.
        onPanResponderTerminate: () => settleBack(),
        onPanResponderTerminationRequest: () => false,
      }),
    // `translateY` is a stable ref-held Animated.Value at every call site.
    [translateY, settleBack],
  );

  return { panHandlers: responder.panHandlers };
}
