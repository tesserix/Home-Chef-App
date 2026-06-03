// packages/mobile-shared/src/screens/_socialIcons.tsx
//
// Shared social-icon primitives used by LoginScreen and RegisterScreen.
// Prefixed with _ to signal this is a co-located implementation detail,
// not a public export from the screens barrel.
//
// Icons are rendered via react-native-svg using paths sourced from the
// Tesserix design system (Apple) and a standard Google brand silhouette
// (Google). Monochrome by default — color is driven by `fill` so the
// caller can theme via context if needed.

import React from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import Svg, { Path } from 'react-native-svg';
import { theme } from '../theme/tokens';

interface SocialIconButtonProps {
  /** Accessibility label — e.g. "Continue with Google". */
  label: string;
  onPress: () => void;
  /** Rendered icon. Caller controls the shape. */
  icon: React.ReactNode;
}

export function SocialIconButton({ label, onPress, icon }: SocialIconButtonProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={label}
    >
      {({ pressed }) => (
        <View style={[styles.btn, pressed && { opacity: 0.7 }]}>
          {icon}
        </View>
      )}
    </Pressable>
  );
}

interface GlyphProps {
  size?: number;
  color?: string;
}

/**
 * GoogleGlyph — clean monochrome "G" silhouette. Matches the brand
 * convention (Sign-in-with-Google ID assets), rendered in ink for
 * cohesion with the rest of the auth screen. Multi-color official
 * Google G is available but visually loud against our minimal palette.
 */
export function GoogleGlyph({ size = 22, color }: GlyphProps) {
  const fill = color ?? theme.colors.ink.DEFAULT;
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M21.35 11.1H12v3.2h5.59c-.56 2.74-2.93 4.6-5.59 4.6-3.36 0-6.1-2.74-6.1-6.1s2.74-6.1 6.1-6.1c1.42 0 2.78.5 3.86 1.36L18.39 5.4C16.6 3.78 14.36 2.9 12 2.9 6.99 2.9 2.9 6.99 2.9 12s4.09 9.1 9.1 9.1c5.45 0 8.5-3.84 8.5-9.34 0-.42-.05-.84-.15-1.16z"
        fill={fill}
      />
    </Svg>
  );
}

/**
 * AppleGlyph — Apple Inc. logo silhouette sourced from the Tesserix
 * design system's `Apple` custom icon. Monochrome ink fill.
 */
export function AppleGlyph({ size = 22, color }: GlyphProps) {
  const fill = color ?? theme.colors.ink.DEFAULT;
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d="M17.05 20.28c-.98.95-2.05.8-3.08.35-1.09-.46-2.09-.48-3.24 0-1.44.62-2.2.44-3.06-.35C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.54 4.09l.01-.01zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.29 2.58-2.34 4.5-3.74 4.25z"
        fill={fill}
      />
    </Svg>
  );
}

const styles = StyleSheet.create({
  btn: {
    width: 64,
    height: 56,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.paper,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
