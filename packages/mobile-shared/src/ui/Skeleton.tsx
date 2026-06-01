import { useEffect, useRef } from 'react';
import { Animated, StyleSheet, View, type ViewStyle } from 'react-native';
import { theme } from '../theme/tokens';

interface SkeletonProps {
  width?: number | `${number}%`;
  height?: number;
  radius?: number;
  style?: ViewStyle;
}

/**
 * <Skeleton> — a single hairline shimmer block.
 *
 * Used to reserve space while remote data is loading. Per .impeccable.md
 * design principle #1 (chrome-light), skeletons are *just* the geometry of
 * the eventual content — no decorative shadows, no icons, no rotating
 * spinners on top.
 *
 * Animates opacity only (1.0 → 0.4 → 1.0 over 1100ms) per the motion
 * tokens — no scale, no slide.
 */
export function Skeleton({
  width = '100%',
  height = 16,
  radius = theme.radius.sm,
  style,
}: SkeletonProps) {
  const opacity = useRef(new Animated.Value(0.6)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, {
          toValue: 1,
          duration: 550,
          useNativeDriver: true,
        }),
        Animated.timing(opacity, {
          toValue: 0.4,
          duration: 550,
          useNativeDriver: true,
        }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [opacity]);

  return (
    <Animated.View
      style={[
        styles.base,
        { width, height, borderRadius: radius, opacity },
        style,
      ]}
    />
  );
}

const styles = StyleSheet.create({
  base: {
    backgroundColor: theme.colors.mist.DEFAULT,
  },
});

/**
 * <SkeletonGroup> — convenience wrapper that stacks N skeleton bars with
 * consistent gaps. Useful for list-row placeholders.
 */
export function SkeletonGroup({
  rows = 3,
  rowHeight = 16,
  gap = theme.spacing[2],
}: {
  rows?: number;
  rowHeight?: number;
  gap?: number;
}) {
  return (
    <View style={{ gap }}>
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} height={rowHeight} width={i === rows - 1 ? '70%' : '100%'} />
      ))}
    </View>
  );
}
