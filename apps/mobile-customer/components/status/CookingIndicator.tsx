// CookingIndicator (#50) — a small animated "being cooked" icon: a pulsing
// cooking-pot glow with two rising steam wisps. Reused on the order detail and
// meal-plan day rows to make the live "preparing" state feel alive. Honors
// reduced-motion (renders the static icon).

import { useEffect } from 'react';
import { View, StyleSheet, type ViewStyle } from 'react-native';
import Animated, {
  Easing,
  useReducedMotion,
  useAnimatedStyle,
  useSharedValue,
  withDelay,
  withRepeat,
  withTiming,
} from 'react-native-reanimated';
import { CookingPot } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

interface CookingIndicatorProps {
  size?: number;
  color?: string;
  style?: ViewStyle;
}

// One rising-and-fading steam wisp, offset by `delay` so the two interleave.
function Steam({ delay, color, left }: { delay: number; color: string; left: number }) {
  const t = useSharedValue(0);
  const reduced = useReducedMotion();

  useEffect(() => {
    if (reduced) return;
    t.value = withDelay(
      delay,
      withRepeat(withTiming(1, { duration: 1400, easing: Easing.out(Easing.quad) }), -1, false),
    );
  }, [delay, reduced, t]);

  const style = useAnimatedStyle(() => ({
    opacity: (1 - t.value) * 0.7,
    transform: [{ translateY: -t.value * 7 }, { scaleX: 0.6 + t.value * 0.5 }],
  }));

  if (reduced) return null;
  return <Animated.View style={[styles.steam, { backgroundColor: color, left }, style]} />;
}

export function CookingIndicator({ size = 18, color, style }: CookingIndicatorProps) {
  const tint = color ?? customerColors.coral.DEFAULT;
  const pulse = useSharedValue(0);
  const reduced = useReducedMotion();

  useEffect(() => {
    if (reduced) return;
    pulse.value = withRepeat(withTiming(1, { duration: 1100, easing: Easing.inOut(Easing.quad) }), -1, true);
  }, [reduced, pulse]);

  const potStyle = useAnimatedStyle(() => ({
    transform: [{ scale: 1 + pulse.value * 0.08 }],
  }));

  return (
    <View style={[styles.root, { width: size + 6, height: size + 8 }, style]}>
      <View style={styles.steamRow}>
        <Steam delay={0} color={tint} left={size * 0.28} />
        <Steam delay={700} color={tint} left={size * 0.62} />
      </View>
      <Animated.View style={potStyle}>
        <CookingPot size={size} color={tint} strokeWidth={2} />
      </Animated.View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: 'center', justifyContent: 'flex-end' },
  steamRow: { position: 'absolute', top: 0, left: 0, right: 0, height: 8 },
  steam: {
    position: 'absolute',
    top: 0,
    width: 2.5,
    height: 6,
    borderRadius: 2,
  },
});
