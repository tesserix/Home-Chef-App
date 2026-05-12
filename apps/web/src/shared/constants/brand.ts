/**
 * Fe3dr Brand Constants — Paper · Ink · Herb design system
 * Source of truth for tokens lives in /.impeccable.md and globals.css.
 * This file mirrors CSS tokens for use in TS code (animations, emails,
 * chart libs, etc.) where CSS variables aren't accessible.
 */

export const BRAND = {
  name: 'Fe3dr',
  tagline: 'Homemade Food Delivered',
  description:
    'Connect with home chefs for authentic homemade food delivered to your doorstep',
  copyright: `© ${new Date().getFullYear()} Fe3dr. All rights reserved.`,
  trademark: '™',
} as const;

/**
 * Color palette — Paper / Ink / Herb / functional.
 * Resolved hex values mirror the OKLCH tokens in globals.css for
 * use in non-CSS contexts (chart libs, email templates, OG images).
 * For UI styling, prefer Tailwind classes (`bg-herb`, `text-ink`) over these.
 */
export const COLORS = {
  // Page + elevated surfaces
  paper: '#fafaf7',         // page background (light)
  bone: '#f3f2ee',          // elevated card surface
  mist: '#e6e5e0',          // hairline / muted bg
  mistStrong: '#d4d3ce',    // divider on bone

  // Ink scale
  ink: '#1a1a18',           // primary text + primary CTA
  inkSoft: '#4a4a47',       // secondary text
  inkMuted: '#7a7a76',      // tertiary / placeholders

  // Brand accent — Herb
  herb: '#3e6b3c',          // primary accent (single)
  herbSoft: '#558257',      // hover state
  herbTint: '#dde9d8',      // selected backgrounds

  // Functional only
  paprika: '#c95b3e',       // destructive
  paprikaTint: '#f3dcd2',
  amber: '#d1a64a',         // warning
  amberTint: '#f0e3c0',
  info: '#4a73a3',          // informational only
  infoTint: '#dde5ee',

  // Dark mode mirrors (for chart libs that don't auto-react)
  dark: {
    paper: '#1a1a18',
    bone: '#22221f',
    mist: '#2e2e2a',
    ink: '#f3f2ee',
    inkSoft: '#bdbcb8',
    inkMuted: '#83827e',
    herb: '#7aa274',
    paprika: '#dd7e62',
    amber: '#dfb96a',
    info: '#7196c7',
  },
} as const;

export const TYPOGRAPHY = {
  fontFamily: {
    body: '"Inter Variable", "Inter", system-ui, sans-serif',
    heading: '"Inter Variable", "Inter", system-ui, sans-serif',
    display: '"Geist", "Geist Variable", "Inter", system-ui, sans-serif',
  },
  fontSizes: {
    xs: '0.75rem',
    sm: '0.875rem',
    base: '1rem',
    lg: '1.125rem',
    xl: '1.25rem',
    '2xl': '1.5rem',
    '3xl': '1.875rem',
    '4xl': '2.25rem',
    '5xl': '3rem',
  },
  numerals: {
    tabular: 'tabular-nums',
  },
} as const;

export const SPACING = {
  container: {
    sm: '640px',
    md: '768px',
    lg: '1024px',
    xl: '1280px',
    '2xl': '1400px',
  },
  section: {
    sm: '2rem',
    md: '4rem',
    lg: '6rem',
  },
} as const;

export const RADIUS = {
  sm: '0.25rem',
  md: '0.375rem',
  lg: '0.5rem',
  xl: '1rem',
  '2xl': '1.5rem',
  full: '9999px',
} as const;

/** Three-step elevation — neutral, never tinted. */
export const SHADOWS = {
  1: '0 1px 2px 0 rgba(15, 15, 14, 0.04)',
  2: '0 4px 12px -4px rgba(15, 15, 14, 0.08), 0 2px 4px -2px rgba(15, 15, 14, 0.04)',
  3: '0 20px 40px -8px rgba(15, 15, 14, 0.15), 0 8px 16px -4px rgba(15, 15, 14, 0.06)',
} as const;

/**
 * Photo legibility scrims — the only allowed gradients.
 * Use these on image overlays for text contrast, never decoratively.
 */
export const SCRIMS = {
  bottom: 'linear-gradient(180deg, transparent 0%, rgba(26, 26, 24, 0.7) 100%)',
  top: 'linear-gradient(180deg, rgba(26, 26, 24, 0.4) 0%, transparent 100%)',
  full: 'linear-gradient(180deg, rgba(26, 26, 24, 0) 0%, rgba(26, 26, 24, 0.6) 100%)',
} as const;

export const ANIMATIONS = {
  fast: '150ms',
  normal: '250ms',
  slow: '400ms',
  easing: {
    smooth: 'cubic-bezier(0.22, 1, 0.36, 1)',
    state: 'cubic-bezier(0.4, 0, 0.2, 1)',
  },
} as const;

export const Z_INDEX = {
  dropdown: 1000,
  sticky: 1020,
  fixed: 1030,
  modalBackdrop: 1040,
  modal: 1050,
  popover: 1060,
  tooltip: 1070,
  toast: 1080,
} as const;

/**
 * Order status — maps to functional color tokens.
 * Used in chart legends, email templates, and any non-CSS context.
 */
export const ORDER_STATUS_COLORS = {
  pending: COLORS.amber,
  confirmed: COLORS.ink,
  preparing: COLORS.herbSoft,
  ready: COLORS.herb,
  out_for_delivery: COLORS.herb,
  delivered: COLORS.inkSoft,
  cancelled: COLORS.paprika,
} as const;

export const RATINGS = {
  min: 1,
  max: 5,
  excellent: 4.5,
  good: 4.0,
  average: 3.0,
} as const;
