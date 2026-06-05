import { StyleSheet, View } from 'react-native';
import { theme } from '../theme/tokens';

interface DietIconProps {
  /** 'veg' | 'non-veg' | 'unknown' */
  kind: 'veg' | 'non-veg' | 'unknown';
  /** Outer square edge length in pt. Scales all inner geometry. Default 12. */
  size?: number;
}

/**
 * FSSAI-style veg / non-veg indicator.
 *
 * Veg:     green square outline + filled green dot inside.
 * Non-veg: brown/red square outline + filled brown/red dot inside.
 * Unknown: muted hairline square outline, no fill.
 *
 * Built from plain Views — no react-native-svg dependency. The indicator
 * colours are `theme.colors.diet.veg` and `theme.colors.diet.nonVeg` which
 * are purposely separate from the main brand palette per .impeccable.md.
 *
 * Exported from `@homechef/mobile-shared/ui` so all three apps can use it.
 * The vendor app's local `DietIcon` (components/vendor/DietIcon.tsx) accepts
 * a boolean `isVeg` prop and is kept for backwards compat — it delegates to
 * this canonical component internally.
 */
export function DietIcon({ kind, size = 12 }: DietIconProps) {
  const inner = Math.round(size * 0.43);

  if (kind === 'unknown') {
    return (
      <View
        accessibilityRole="image"
        accessibilityLabel="Diet type unknown"
        style={[
          styles.outer,
          {
            width: size,
            height: size,
            borderColor: theme.colors.mist.strong,
          },
        ]}
      />
    );
  }

  const color =
    kind === 'veg' ? theme.colors.diet.veg : theme.colors.diet.nonVeg;

  return (
    <View
      accessibilityRole="image"
      accessibilityLabel={kind === 'veg' ? 'Vegetarian' : 'Non-vegetarian'}
      style={[
        styles.outer,
        {
          width: size,
          height: size,
          borderColor: color,
        },
      ]}
    >
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
