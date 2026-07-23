import { useState, type ReactNode } from 'react';
import {
  Platform,
  Pressable,
  StyleSheet,
  type StyleProp,
  Text,
  TextInput,
  type TextInputProps,
  type TextStyle,
  View,
} from 'react-native';
import { theme } from '../theme/tokens';

// See Button.tsx for the full rationale — an 8-digit hex alpha channel
// appended to an existing token colour, never a new literal colour.
function withAlpha(hex: string, alphaHex: string): string {
  return `${hex}${alphaHex}`;
}

interface InputProps extends Omit<TextInputProps, 'style'> {
  /** Label rendered above the field. Pass an empty string to hide. */
  label: string;
  /** Helper text under the field. Shown unless `error` is set. */
  helper?: string;
  /** Error message — when present, takes precedence over `helper`, recolours
   *  the border, and is announced to assistive tech via `accessibilityHint`
   *  on the field plus a `role="alert"` live region on the footer text. */
  error?: string;
  /** Show a character counter in the footer (right-aligned). Useful for
   *  bio/description fields with a maxLength. */
  showCounter?: boolean;
  /** Replace the default footer with custom content (status text, formatter
   *  hint, etc). Leave undefined for the helper/error/counter trio. */
  footer?: ReactNode;
  /** When true and `secureTextEntry` is set, renders a peek toggle. */
  passwordPeek?: boolean;
  /** Optional leading element (icon, currency symbol, country code). */
  leading?: ReactNode;
  /** Optional trailing element rendered before the password peek. */
  trailing?: ReactNode;
  /** Per-field style override for the inner TextInput. Use sparingly —
   *  most fields should stay on the default 44pt single-line height.
   *  Pass `{ minHeight: N, textAlignVertical: 'top' }` to grow a textarea
   *  in `multiline` fields. */
  inputStyle?: StyleProp<TextStyle>;
  /** Optional accent override for the focused-field 2px border. Lets the
   *  customer app paint its Airbnb coral focus ring without forking this
   *  primitive. Undefined → default ink focus border (vendor/driver
   *  unchanged). */
  accentColor?: string;
}

/**
 * <Input> — the only TextInput wrapper we ship.
 *
 * Owns: label, helper/error/counter triad, focus styling, password peek.
 * Field height matches the 44pt touch target. A visible 2px focus border
 * (ink by default, `accentColor` to repaint) per .impeccable.md
 * §Accessibility. This RN version's `AccessibilityState`/`aria-*` surface
 * has no `invalid` field, so the error state is wired through the two
 * channels RN actually ships: `accessibilityHint` on the field (read on
 * focus) and `accessibilityRole="alert"` + `accessibilityLiveRegion="polite"`
 * on the footer error text (announced the moment it appears).
 */
export function Input({
  label,
  helper,
  error,
  showCounter = false,
  footer,
  passwordPeek = false,
  leading,
  trailing,
  inputStyle,
  accentColor,
  secureTextEntry,
  maxLength,
  value,
  ...input
}: InputProps) {
  const [focused, setFocused] = useState(false);
  const [peek, setPeek] = useState(false);

  const obscured = secureTextEntry && !peek;
  const hasError = Boolean(error);
  const showLabel = label.length > 0;
  // editable defaults to true if unset, so only the explicit false counts
  // as the disabled state.
  const isDisabled = input.editable === false;

  const showFooter = footer !== undefined || error || helper || showCounter;

  return (
    <View style={styles.root}>
      {showLabel ? (
        <Text
          style={[
            styles.label,
            hasError && { color: theme.colors.destructive.DEFAULT },
          ]}
        >
          {label}
        </Text>
      ) : null}

      <View
        style={[
          styles.field,
          focused && styles.fieldFocused,
          focused && accentColor ? { borderColor: accentColor } : null,
          hasError && styles.fieldError,
          isDisabled && styles.fieldDisabled,
        ]}
      >
        {leading ? <View style={styles.leading}>{leading}</View> : null}
        <TextInput
          {...input}
          value={value}
          maxLength={maxLength}
          secureTextEntry={obscured}
          onFocus={(e) => {
            setFocused(true);
            input.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            input.onBlur?.(e);
          }}
          placeholderTextColor={theme.colors.ink.muted}
          style={[styles.input, isDisabled && styles.inputDisabled, inputStyle]}
          accessibilityState={{ disabled: isDisabled }}
          accessibilityHint={error || undefined}
        />
        {trailing ? <View style={styles.trailing}>{trailing}</View> : null}
        {passwordPeek && secureTextEntry ? (
          <Pressable
            onPress={() => setPeek((p) => !p)}
            accessibilityRole="button"
            accessibilityLabel={peek ? 'Hide password' : 'Show password'}
            hitSlop={8}
            style={styles.peekToggle}
            android_ripple={{ color: withAlpha(theme.colors.ink.DEFAULT, '14'), borderless: true }}
          >
            {({ pressed }) => (
              <Text
                style={[
                  styles.peekText,
                  pressed && Platform.OS === 'ios' && styles.peekPressedIOS,
                ]}
              >
                {peek ? 'Hide' : 'Show'}
              </Text>
            )}
          </Pressable>
        ) : null}
      </View>

      {showFooter ? (
        <View style={styles.footer}>
          <View style={styles.footerLeft}>
            {footer ? (
              footer
            ) : error ? (
              <Text
                style={styles.errorText}
                accessibilityLiveRegion="polite"
                accessibilityRole="alert"
              >
                {error}
              </Text>
            ) : helper ? (
              <Text style={styles.helperText}>{helper}</Text>
            ) : null}
          </View>
          {showCounter && maxLength ? (
            <Text style={styles.counter}>
              {value?.length ?? 0} / {maxLength}
            </Text>
          ) : null}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { marginBottom: theme.spacing[3] },
  label: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[1],
  },
  field: {
    flexDirection: 'row',
    alignItems: 'center',
    minHeight: theme.touchTarget.vendor,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.paper,
    paddingHorizontal: theme.spacing[3],
  },
  fieldFocused: {
    borderColor: theme.colors.ink.DEFAULT,
    borderWidth: 2,
    paddingHorizontal: theme.spacing[3] - 1, // compensate for 2px border
  },
  fieldError: { borderColor: theme.colors.destructive.DEFAULT },
  fieldDisabled: {
    backgroundColor: theme.colors.bone,
    borderColor: theme.colors.mist.DEFAULT,
  },
  leading: { marginRight: theme.spacing[2] },
  trailing: { marginLeft: theme.spacing[2] },
  input: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    paddingVertical: 0, // RN adds default vertical padding on Android
  },
  inputDisabled: { color: theme.colors.ink.muted },
  peekToggle: { marginLeft: theme.spacing[2], padding: theme.spacing[1] },
  peekText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },
  // iOS-only pressed treatment (Android relies on android_ripple instead).
  peekPressedIOS: { opacity: 0.6 },
  footer: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: theme.spacing[1],
    minHeight: theme.spacing[4],
  },
  footerLeft: { flex: 1 },
  helperText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  errorText: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
  },
  counter: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    fontVariant: ['tabular-nums'],
  },
});
