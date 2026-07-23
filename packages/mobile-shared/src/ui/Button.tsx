import { type ReactNode } from 'react';
import {
  ActivityIndicator,
  Platform,
  Pressable,
  type PressableProps,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { theme } from '../theme/tokens';

// Appends an 8-digit hex alpha channel to an existing token colour to get a
// translucent ripple tint — RN's colour parser accepts #RRGGBBAA. This never
// introduces a new literal colour; it only dims/lightens a token we already
// ship (Android `android_ripple` needs a colour prop, there's no "auto"
// option that inherits the platform ripple tint like a native Button would).
function withAlpha(hex: string, alphaHex: string): string {
  return `${hex}${alphaHex}`;
}

type Variant = 'primary' | 'secondary' | 'ghost' | 'destructive';
type Size = 'md' | 'lg';

interface ButtonProps extends Omit<PressableProps, 'children' | 'style'> {
  label: string;
  /** Visual treatment. `primary` = ink fill; `secondary` = ink-bordered
   *  outline; `ghost` = text-only; `destructive` = red fill for irreversible
   *  actions (delete, cancel order). Default `primary`. */
  variant?: Variant;
  /** Tap target height. `md` = 44pt (default customer/vendor), `lg` = 48pt
   *  (driver app, or any high-stakes confirm where the user can't miss). */
  size?: Size;
  /** Renders an icon-shaped element to the left of the label. The caller
   *  passes a lucide / phosphor / custom svg element; the button only
   *  reserves space and never restyles it. */
  iconLeft?: ReactNode;
  /** Same as iconLeft, right side. Useful for `Next →` style buttons. */
  iconRight?: ReactNode;
  /** When true, swaps the label for a centred spinner and disables press. */
  loading?: boolean;
  /** Full-width by default; pass `inline` for a hug-content button (e.g.
   *  the "+ New" pill in the menu category picker). */
  fullWidth?: boolean;
  /** Optional accent override for the `primary` variant fill (and the
   *  `ghost`/`secondary` label). Lets the customer app paint its Airbnb
   *  coral CTA without forking this primitive. Undefined → default ink
   *  palette (vendor / driver unchanged). */
  accentColor?: string;
}

/**
 * <Button> — the only Pressable we ever ship as a button.
 *
 * Why this exists: the vendor app currently mixes `TouchableOpacity` +
 * inline `className="bg-herb"` patterns. Every screen reinvents the
 * disabled colour, the loading spinner colour, the touch target. This
 * primitive owns all of that.
 *
 * Press feedback is platform-split so the two native "this is being
 * pressed" languages never fight each other: iOS gets the inner-View
 * opacity (0.85) + scale (0.97) treatment (no native ripple exists there);
 * Android gets a bounded `android_ripple` and skips the extra opacity/scale
 * — layering both would read as a double, janky press. Per .impeccable.md
 * §Motion: opacity/transform only, no bounce/scale-up beyond 1.
 */
export function Button({
  label,
  variant = 'primary',
  size = 'md',
  iconLeft,
  iconRight,
  loading = false,
  disabled,
  fullWidth = true,
  accentColor,
  ...pressable
}: ButtonProps) {
  const v = variantStyles[variant];
  const isInert = disabled || loading;

  // Accent override (customer Airbnb coral). Only repaints the fill of a
  // primary button or the label/border of ghost/secondary — never touches
  // the default ink palette when accentColor is undefined.
  const accentContainer =
    accentColor && variant === 'primary'
      ? { backgroundColor: accentColor }
      : accentColor && variant === 'secondary'
        ? { borderColor: accentColor }
        : undefined;
  const accentLabel =
    accentColor && (variant === 'ghost' || variant === 'secondary')
      ? { color: accentColor }
      : undefined;

  // Pressable wraps a styled View instead of carrying the visual styles
  // itself. iOS occasionally drops `backgroundColor` and `borderWidth` on
  // a Pressable with a function-based `style` prop; pushing the visual
  // layer onto a View renders reliably.
  return (
    <Pressable
      {...pressable}
      disabled={isInert}
      accessibilityRole="button"
      accessibilityState={{ disabled: isInert, busy: loading }}
      style={fullWidth ? styles.fullWidth : undefined}
      android_ripple={isInert ? undefined : { color: v.ripple, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.base,
            size === 'lg' ? styles.sizeLg : styles.sizeMd,
            v.container,
            accentContainer,
            fullWidth && styles.fullWidth,
            pressed && !isInert && Platform.OS === 'ios' && styles.pressedIOS,
            isInert && styles.disabled,
          ]}
        >
          <View style={[styles.row, loading && styles.rowHidden]}>
            {iconLeft ? <View style={styles.icon}>{iconLeft}</View> : null}
            <Text style={[styles.label, v.label, accentLabel]}>{label}</Text>
            {iconRight ? <View style={styles.icon}>{iconRight}</View> : null}
          </View>
          {loading ? (
            <View style={styles.spinnerOverlay}>
              <ActivityIndicator
                color={accentColor && variant !== 'primary' ? accentColor : v.spinner}
              />
            </View>
          ) : null}
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: theme.radius.md,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[4],
  },
  sizeMd: { minHeight: theme.touchTarget.vendor },
  sizeLg: { minHeight: theme.touchTarget.driver },
  fullWidth: { width: '100%' },
  // iOS-only pressed treatment (see the platform note above the component).
  pressedIOS: { opacity: 0.85, transform: [{ scale: 0.97 }] },
  disabled: { opacity: 0.4 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  // Loading swaps the spinner in via an absolute overlay instead of
  // replacing `row` outright, so the label keeps reserving its layout width
  // (invisible, not unmounted) and a hug-content button never narrows.
  rowHidden: { opacity: 0 },
  spinnerOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    alignItems: 'center',
    justifyContent: 'center',
  },
  icon: { alignItems: 'center', justifyContent: 'center' },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    letterSpacing: 0.1,
  },
});

interface VariantSpec {
  container: { backgroundColor?: string; borderWidth?: number; borderColor?: string };
  label: { color: string };
  spinner: string;
  /** Android `android_ripple` colour — a translucent tint derived from an
   *  existing token via `withAlpha`, never a new literal colour. Light
   *  ripple on dark fills, dark ripple on light/transparent fills. */
  ripple: string;
}

// Variant palette — Uber-like for vendor/driver. Primary is ink (black),
// NOT persimmon. Persimmon is reserved for accents (links, brand label,
// peek toggle, focus, status badges) — never a primary CTA colour here.
// The customer app will override these at the app level with an
// Airbnb-style palette when wired.
const variantStyles: Record<Variant, VariantSpec> = {
  primary: {
    container: { backgroundColor: theme.colors.ink.DEFAULT },
    label: { color: theme.colors.paper },
    spinner: theme.colors.paper,
    ripple: withAlpha(theme.colors.paper, '33'),
  },
  secondary: {
    container: {
      backgroundColor: theme.colors.paper,
      borderWidth: 1,
      borderColor: theme.colors.ink.DEFAULT,
    },
    label: { color: theme.colors.ink.DEFAULT },
    spinner: theme.colors.ink.DEFAULT,
    ripple: withAlpha(theme.colors.ink.DEFAULT, '14'),
  },
  ghost: {
    container: { backgroundColor: 'transparent' },
    label: { color: theme.colors.ink.DEFAULT },
    spinner: theme.colors.ink.DEFAULT,
    ripple: withAlpha(theme.colors.ink.DEFAULT, '14'),
  },
  destructive: {
    container: { backgroundColor: theme.colors.destructive.DEFAULT },
    label: { color: theme.colors.paper },
    spinner: theme.colors.paper,
    ripple: withAlpha(theme.colors.paper, '33'),
  },
};
