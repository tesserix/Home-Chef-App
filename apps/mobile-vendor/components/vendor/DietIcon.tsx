import { StyleSheet, View } from 'react-native';
import { theme } from '@homechef/mobile-shared/theme';

interface DietIconProps {
  isVeg: boolean;
  /** Outer square edge length in pt. Inner glyph scales with this. */
  size?: number;
}

/**
 * FSSAI-style veg / non-veg indicator. Square outline + filled dot (veg)
 * or filled upward triangle (non-veg). Built from plain Views so we
 * don't pull in react-native-svg just for this. The convention is
 * dominant in Indian food UX (Swiggy, Zomato, every restaurant menu) so
 * we honor it instead of substituting with our generic info/destructive
 * dots.
 */
export function DietIcon({ isVeg, size = 14 }: DietIconProps) {
  const color = isVeg ? theme.colors.diet.veg : theme.colors.diet.nonVeg;
  const inner = Math.round(size * 0.43); // ~6 at size 14

  return (
    <View
      accessibilityRole="image"
      accessibilityLabel={isVeg ? 'Vegetarian' : 'Non-vegetarian'}
      style={[
        styles.outer,
        {
          width: size,
          height: size,
          borderColor: color,
        },
      ]}
    >
      {/* Centered filled dot — same shape for both veg and non-veg, the
          color carries the meaning. Simpler than the triangle variant
          and reads correctly at 12pt where the triangle was ambiguous. */}
      <View
        style={{
          width: inner,
          height: inner,
          borderRadius: inner / 2,
          backgroundColor: color,
        }}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  outer: {
    borderWidth: 1.5,
    borderRadius: 2,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
