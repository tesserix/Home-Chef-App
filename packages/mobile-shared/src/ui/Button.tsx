import { type ReactNode } from 'react';
import {
  ActivityIndicator,
  Pressable,
  type PressableProps,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { theme } from '../theme/tokens';

type Variant = 'primary' | 'secondary' | 'ghost' | 'destructive';
type Size = 'md' | 'lg';

interface ButtonProps extends Omit<PressableProps, 'children' | 'style'> {
  label: string;
  /** Visual treatment. `primary` = persimmon fill; `secondary` = ink-bordered
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
 * Press feedback: a 50ms opacity drop on Pressable's `onPressIn` — no
 * bounce, no scale-up. Per .impeccable.md §Motion: state-change easing,
 * opacity/transform only.
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
    >
      {({ pressed }) => (
        <View
          style={[
            styles.base,
            size === 'lg' ? styles.sizeLg : styles.sizeMd,
            v.container,
            accentContainer,
            fullWidth && styles.fullWidth,
            pressed && !isInert && { opacity: 0.85 },
            isInert && styles.disabled,
          ]}
        >
          {loading ? (
            <ActivityIndicator color={accentColor && variant !== 'primary' ? accentColor : v.spinner} />
          ) : (
            <View style={styles.row}>
              {iconLeft ? <View style={styles.icon}>{iconLeft}</View> : null}
              <Text style={[styles.label, v.label, accentLabel]}>{label}</Text>
              {iconRight ? <View style={styles.icon}>{iconRight}</View> : null}
            </View>
          )}
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
  disabled: { opacity: 0.4 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 8 },
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
  },
  secondary: {
    container: {
      backgroundColor: theme.colors.paper,
      borderWidth: 1,
      borderColor: theme.colors.ink.DEFAULT,
    },
    label: { color: theme.colors.ink.DEFAULT },
    spinner: theme.colors.ink.DEFAULT,
  },
  ghost: {
    container: { backgroundColor: 'transparent' },
    label: { color: theme.colors.ink.DEFAULT },
    spinner: theme.colors.ink.DEFAULT,
  },
  destructive: {
    container: { backgroundColor: theme.colors.destructive.DEFAULT },
    label: { color: theme.colors.paper },
    spinner: theme.colors.paper,
  },
};
