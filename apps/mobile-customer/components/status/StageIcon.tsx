// StageIcon — one small animated icon per order stage, so the tracker reads at
// a glance and feels alive (#717). Reuses CookingIndicator for "preparing" and
// adds light reanimated motions for the other stages. Honors reduced-motion
// (renders the static icon). Shared by the detail hero, the tracker, and the
// home active-order card so the stage→icon mapping lives in one place.

import { useEffect, type ReactNode } from 'react';
import { View } from 'react-native';
import Animated, {
  Easing,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withRepeat,
  withSequence,
  withTiming,
} from 'react-native-reanimated';
import { Bike, CheckCircle2, ChefHat, Clock } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { CookingIndicator } from './CookingIndicator';
import type { Order } from '../../types/customer';

interface StageIconProps {
  status: Order['status'];
  size?: number;
  color?: string;
}

function Bob({ children }: { children: ReactNode }) {
  const t = useSharedValue(0);
  const reduced = useReducedMotion();
  useEffect(() => {
    if (!reduced) {
      t.value = withRepeat(withTiming(1, { duration: 900, easing: Easing.inOut(Easing.quad) }), -1, true);
    }
  }, [reduced, t]);
  const style = useAnimatedStyle(() => ({ transform: [{ translateY: -t.value * 3 }] }));
  return <Animated.View style={reduced ? undefined : style}>{children}</Animated.View>;
}

function Slide({ children }: { children: ReactNode }) {
  const t = useSharedValue(0);
  const reduced = useReducedMotion();
  useEffect(() => {
    if (!reduced) {
      t.value = withRepeat(withTiming(1, { duration: 1100, easing: Easing.inOut(Easing.quad) }), -1, true);
    }
  }, [reduced, t]);
  const style = useAnimatedStyle(() => ({ transform: [{ translateX: t.value * 4 - 2 }] }));
  return <Animated.View style={reduced ? undefined : style}>{children}</Animated.View>;
}

function Pop({ children }: { children: ReactNode }) {
  const t = useSharedValue(0);
  const reduced = useReducedMotion();
  useEffect(() => {
    if (!reduced) {
      t.value = withRepeat(
        withSequence(withTiming(1, { duration: 550 }), withTiming(0, { duration: 550 })),
        -1,
        false,
      );
    }
  }, [reduced, t]);
  const style = useAnimatedStyle(() => ({ transform: [{ scale: 1 + t.value * 0.12 }] }));
  return <Animated.View style={reduced ? undefined : style}>{children}</Animated.View>;
}

export function StageIcon({ status, size = 20, color }: StageIconProps) {
  const icon = renderStageIcon(status, size, color);
  if (!icon) return null;
  // Purely decorative: every place this renders already states the status in
  // adjacent text (chip, timeline label, card status line), so announcing the
  // glyph would just duplicate it. Matches the ChevronRight treatment on
  // ActiveOrderCard.
  return (
    <View accessibilityElementsHidden importantForAccessibility="no-hide-descendants">
      {icon}
    </View>
  );
}

function renderStageIcon(
  status: Order['status'],
  size: number,
  color?: string,
): ReactNode {
  const tint = color ?? customerColors.coral.DEFAULT;
  switch (status) {
    case 'pending':
      return <Clock size={size} color={customerColors.charcoal.soft} strokeWidth={2} />;
    case 'accepted':
      return (
        <Pop>
          <CheckCircle2 size={size} color={tint} strokeWidth={2} />
        </Pop>
      );
    case 'preparing':
      return <CookingIndicator size={size} color={tint} />;
    case 'ready':
      return (
        <Bob>
          <ChefHat size={size} color={tint} strokeWidth={2} />
        </Bob>
      );
    case 'picked_up':
    case 'delivering':
      return (
        <Slide>
          <Bike size={size} color={tint} strokeWidth={2} />
        </Slide>
      );
    case 'delivered':
      return (
        <Pop>
          <CheckCircle2 size={size} color={tint} strokeWidth={2} />
        </Pop>
      );
    default:
      return null;
  }
}
