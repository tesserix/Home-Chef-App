// Design token bridge: @tesserix/native → @homechef/mobile-shared
// Per FOUND-03: all mobile UI must use @tesserix/native brand tokens.
//
// How to use in screens:
//   import { colors, spacing, typography } from '@homechef/mobile-shared/theme';
//   <View style={{ backgroundColor: colors.background.primary }} />
//
// IMPORTANT: Read the @tesserix/native package exports before editing this file.
// The exact import paths depend on the package's public API surface.
// If @tesserix/native exports a `tokens` object, use:
//   import { tokens } from '@tesserix/native';
// If it exports named token groups, use named imports:
//   import { colors, spacing } from '@tesserix/native/tokens';

// Attempt primary import path — adjust if @tesserix/native uses a different export path.
// The executor MUST read node_modules/@tesserix/native/package.json (or its index) to
// confirm the correct import path before writing this file.
import * as NativeTokens from '@tesserix/native';

/**
 * Re-exported color tokens from @tesserix/native.
 * Use these in place of raw hex values in all mobile screens.
 *
 * Example: colors.primary, colors.background.default, colors.text.primary
 */
export const colors =
  (NativeTokens as Record<string, unknown>).colors ??
  (NativeTokens as Record<string, unknown>).theme?.colors ??
  {};

/**
 * Re-exported spacing tokens from @tesserix/native.
 * Use these in place of raw pixel values.
 *
 * Example: spacing[4] === 16, spacing[2] === 8
 */
export const spacing =
  (NativeTokens as Record<string, unknown>).spacing ??
  (NativeTokens as Record<string, unknown>).theme?.spacing ??
  {};

/**
 * Re-exported typography tokens from @tesserix/native.
 * Use these for font families, sizes, and weights.
 */
export const typography =
  (NativeTokens as Record<string, unknown>).typography ??
  (NativeTokens as Record<string, unknown>).theme?.typography ??
  {};

/**
 * Full token namespace — use when you need access to the complete token tree.
 */
export const nativeTokens = NativeTokens;
