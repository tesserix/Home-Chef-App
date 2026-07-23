// Reusable save/favorite heart overlay (#237). Used on dish cards; mirrors the
// chef-card heart (coral fill when saved, scale-pop gated by reduced motion,
// dark circular backdrop for legibility over photos). Kept generic so any photo
// card can drop it in without re-implementing the animation + a11y wiring.

import { Pressable, StyleSheet, View } from 'react-native';
import Animated, {
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withSequence,
  withTiming,
} from 'react-native-reanimated';
import { Heart } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

// Android ripple tint — translucent canvas derived from the token (never a
// new literal colour), matching the ChefCard heart's ripple convention.
const HEART_RIPPLE = `${customerColors.canvas}59`;

interface FavoriteHeartProps {
  /** Whether the entity is currently saved (heart filled coral). */
  filled: boolean;
  /** Toggle handler — fired after the pop animation kicks off. */
  onToggle: () => void;
  /** Entity name for the accessibility label, e.g. the dish name. */
  label: string;
  /** Heart glyph size (the circular backdrop scales with it). */
  size?: number;
}

export function FavoriteHeart({ filled, onToggle, label, size = 18 }: FavoriteHeartProps) {
  const reduceMotion = useReducedMotion();
  const scale = useSharedValue(1);
  const animStyle = useAnimatedStyle(() => ({ transform: [{ scale: scale.value }] }));

  function handlePress() {
    if (!reduceMotion) {
      scale.value = withSequence(
        withTiming(1.25, { duration: 75 }),
        withTiming(1, { duration: 75 }),
      );
    }
    onToggle();
  }

  return (
    // Intercept touches so a card-level Pressable underneath doesn't also fire.
    <View style={styles.touchable} onStartShouldSetResponder={() => true}>
      <Pressable
        onPress={handlePress}
        hitSlop={8}
        accessibilityRole="togglebutton"
        accessibilityLabel={filled ? `Remove ${label} from saved` : `Save ${label}`}
        accessibilityState={{ checked: filled }}
        android_ripple={{ color: HEART_RIPPLE, borderless: true, radius: 20 }}
      >
        <Animated.View style={[styles.button, animStyle]}>
          <Heart
            size={size}
            color={filled ? customerColors.coral.DEFAULT : customerColors.canvas}
            fill={filled ? customerColors.coral.DEFAULT : 'transparent'}
          />
        </Animated.View>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  touchable: {
    position: 'absolute',
    top: 0,
    right: 0,
    padding: 8,
  },
  button: {
    width: 30,
    height: 30,
    borderRadius: 15,
    backgroundColor: 'rgba(0,0,0,0.25)',
    alignItems: 'center',
    justifyContent: 'center',
  },
});
