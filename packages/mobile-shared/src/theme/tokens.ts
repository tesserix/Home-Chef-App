// Design tokens — single source of truth for non-NativeWind contexts
// (StyleSheet, Reanimated, programmatic color/animation lookups).
//
// NativeWind class names (e.g. `bg-herb`, `text-ink`) read from the
// vendor app's `tailwind.config.js`. Keep these two files in sync — the
// vendor tailwind config imports nothing from here so values must match
// by convention. The token names below mirror the tailwind extend.colors
// keys exactly so a screen can fall back from `className` to inline
// `style={{ color: colors.ink.DEFAULT }}` without a rename.
//
// Source: .impeccable.md at repo root (the Home Chef Design Context).
// When the brand brief changes, update that file FIRST then propagate
// here and to the per-app tailwind configs.

// ----- COLORS ----------------------------------------------------------------

export const colors = {
  // Surface — Uber-like: stark white base, cool greys for elevation and
  // hairlines. Stripped of warmth so persimmon stays the only colour on
  // screen.
  paper: '#FFFFFF', // pure white page background
  bone: '#F5F5F4', // elevated surface (cards, sheets, popovers)
  mist: {
    DEFAULT: '#E5E5E5', // hairline borders, dividers, muted bg
    strong: '#D6D6D6', // stronger hairline (focus, hover)
  },

  // Text & primary ink — deeper black for Uber-like contrast.
  ink: {
    DEFAULT: '#0E0E0C', // primary text + primary CTA
    soft: '#525252', // secondary text, labels (cool grey, not warm)
    muted: '#888888', // tertiary text, placeholders
  },

  // Brand accent — Persimmon. Token name kept as `herb` for backwards
  // compatibility with the 49 files that already class against `bg-herb`.
  herb: {
    DEFAULT: '#C2410C',
    soft: '#9A3412', // hover / pressed
    tint: '#FFEDD5', // background tint for chips, badges
  },

  // Functional signal — never decorative.
  destructive: {
    DEFAULT: '#B22B0E', // deep paprika red. NOT brand orange.
    tint: '#FBE8E1',
  },
  amber: {
    DEFAULT: '#d1a64a',
    tint: '#f0e3c0',
  },
  info: {
    DEFAULT: '#4a73a3',
    tint: '#dde5ee',
  },

  // Legacy paprika alias for old code paths — same value as destructive.
  paprika: {
    DEFAULT: '#c95b3e',
    tint: '#f3dcd2',
  },

  // Diet indicator colors — used ONLY for FSSAI-style veg/non-veg icons
  // in the menu UI. The FSSAI convention (square outline + dot for veg /
  // triangle for non-veg) is too dominant in Indian food UX to override
  // with our neutral tokens. These colors are functional, not decorative,
  // and should not appear anywhere outside DietIcon.
  diet: {
    veg: '#2A9D3E', // slightly muted FSSAI green
    nonVeg: '#B22B0E', // same as destructive — FSSAI red-brown
  },

  // Brand scale — used sparingly. Most surfaces should stick to the named
  // tokens above. Scale exists for gradients-from-nowhere or focus rings
  // where a single named value isn't enough.
  brand: {
    50: '#FFF7ED',
    100: '#FFEDD5',
    200: '#FED7AA',
    300: '#FDBA74',
    400: '#FB923C',
    500: '#C2410C', // same as herb.DEFAULT
    600: '#9A3412', // same as herb.soft
    700: '#7C2D12',
    800: '#5C1F0A',
    900: '#3F1409',
    950: '#2A0C06',
  },
} as const;

// ----- SPACING ---------------------------------------------------------------
// 4 / 8 / 12 / 16 / 24 / 32 / 48 / 64 / 96. Matches .impeccable.md.
// Use the numeric keys directly: spacing[4] === 16.

export const spacing = {
  0: 0,
  px: 1,
  0.5: 2,
  1: 4,
  2: 8,
  3: 12,
  4: 16,
  5: 20,
  6: 24,
  7: 28,
  8: 32,
  10: 40,
  12: 48,
  14: 56,
  16: 64,
  20: 80,
  24: 96,
} as const;

// ----- RADIUS ----------------------------------------------------------------
// 4 steps. Default 8 ("rounded-md"). Sheets/modals 16. No super-round pills.

export const radius = {
  none: 0,
  sm: 4,
  DEFAULT: 8,
  md: 12,
  lg: 16, // sheets, modals
  full: 9999, // circular: dots, avatars, fab — NOT cards or buttons
} as const;

// ----- TYPOGRAPHY ------------------------------------------------------------
// Fluid clamp() isn't available in React Native, so we ship two scales:
// one for "compact" devices (<= 380pt width — iPhone SE / mini) and one
// for everything larger. A helper hook in screens/_layout chooses.
//
// Line heights account for italic descender clearance per skill §4.1
// (italic display words with y/g/j/p/q need leading-[1.1] minimum).

export const typography = {
  // Font families. The names must match the keys passed to expo-font's
  // useFonts() at app boot — see apps/mobile-vendor/app/_layout.tsx.
  family: {
    display: 'Geist', // 600 / 700 — headlines, brand numerals
    body: 'Inter', // 400 / 500 / 600 — UI, body, labels
  },
  weight: {
    regular: '400',
    medium: '500',
    semibold: '600',
    bold: '700',
  },

  // Size ramp. Numbers are points (RN). Line-height is a multiplier of size.
  size: {
    // Display — Geist
    display: { size: 32, lineHeight: 1.1, letterSpacing: -0.5 },
    h1: { size: 26, lineHeight: 1.15, letterSpacing: -0.3 },
    h2: { size: 20, lineHeight: 1.2, letterSpacing: -0.2 },
    // Body — Inter
    body: { size: 16, lineHeight: 1.5, letterSpacing: 0 },
    bodySm: { size: 14, lineHeight: 1.45, letterSpacing: 0 },
    label: { size: 13, lineHeight: 1.35, letterSpacing: 0.1 },
    caption: { size: 11, lineHeight: 1.3, letterSpacing: 0.2 },
  },

  // Tabular figures for prices, IDs, ETAs, stats.
  tabular: { fontVariant: ['tabular-nums'] as const },
} as const;

// ----- SHADOWS ---------------------------------------------------------------
// Three steps. iOS-only — Android uses elevation (set separately on View).
// Values match .impeccable.md's "elevation only, never decorative" rule.

export const shadow = {
  // Hairline lift (cards on bone surface)
  1: {
    shadowColor: colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.04,
    shadowRadius: 2,
    elevation: 1,
  },
  // Card lift (interactive cards, list rows)
  2: {
    shadowColor: colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2,
  },
  // Modal / sheet
  3: {
    shadowColor: colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.12,
    shadowRadius: 24,
    elevation: 8,
  },
} as const;

// ----- MOTION ----------------------------------------------------------------
// ease-out-quart for entrances, ease-in-out for state changes. NO bounce,
// NO elastic, NO overshoot. .impeccable.md §Motion is unambiguous on this.

export const motion = {
  easing: {
    // Entrance (modals, sheets, toasts) — `cubic-bezier(0.22, 1, 0.36, 1)`
    entrance: [0.22, 1, 0.36, 1] as const,
    // State change (toggle, switch, color) — `cubic-bezier(0.4, 0, 0.2, 1)`
    state: [0.4, 0, 0.2, 1] as const,
  },
  duration: {
    micro: 150,
    default: 250,
    page: 400,
  },
} as const;

// ----- TOUCH TARGETS --------------------------------------------------------
// Skill + .impeccable.md: 44pt minimum for customer/vendor (Apple HIG),
// 48pt for driver (gloves + motion). Use these as the minHeight on any
// pressable.

export const touchTarget = {
  customer: 44,
  vendor: 44,
  driver: 48,
} as const;

// ----- ROLE-CODED CONVENIENCE -------------------------------------------------
// Some screens need the brand palette but with role-specific density.
// .impeccable.md §Design Principles #4: "per-role density, shared language."
// Vendor packs tighter than customer; driver strips down to glanceable.

export const roleScale = {
  customer: { textBase: 16, padding: spacing[4], gap: spacing[3] },
  vendor: { textBase: 15, padding: spacing[3], gap: spacing[2] },
  driver: { textBase: 18, padding: spacing[5], gap: spacing[4] },
} as const;

// ----- AGGREGATE THEME -------------------------------------------------------
// Single import surface. Most screens import `theme` rather than each
// individual token: `import { theme } from '@homechef/mobile-shared/theme'`.

export const theme = {
  colors,
  spacing,
  radius,
  typography,
  shadow,
  motion,
  touchTarget,
  roleScale,
} as const;

export type Theme = typeof theme;
