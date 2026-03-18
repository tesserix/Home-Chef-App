/**
 * Design Tokens for Fe3dr
 * Aligned with CSS variable system in globals.css
 */

// Semantic colors (matching CSS variables)
export const colors = {
  background: 'var(--background)',       // #FFFAF5
  foreground: 'var(--foreground)',       // #1C1917
  card: 'var(--card)',                   // #FFFFFF
  cardForeground: 'var(--card-foreground)', // #1C1917
  popover: 'var(--popover)',             // #FFFFFF
  popoverForeground: 'var(--popover-foreground)', // #1C1917
  primary: 'var(--primary)',             // #C2410C
  primaryForeground: 'var(--primary-foreground)', // #FFFFFF
  secondary: 'var(--secondary)',         // #F5F0EB
  secondaryForeground: 'var(--secondary-foreground)', // #44403C
  muted: 'var(--muted)',                 // #FAF5F0
  mutedForeground: 'var(--muted-foreground)', // #78716C
  accent: 'var(--accent)',               // #FEF3C7
  accentForeground: 'var(--accent-foreground)', // #92400E
  destructive: 'var(--destructive)',     // #DC2626
  destructiveForeground: 'var(--destructive-foreground)', // #FFFFFF
  border: 'var(--border)',               // #E7E0D9
  input: 'var(--input)',                 // #E7E0D9
  ring: 'var(--ring)',                   // #C2410C
  success: 'var(--success)',             // #059669
  successForeground: 'var(--success-foreground)', // #FFFFFF
  warning: 'var(--warning)',             // #D97706
  warningForeground: 'var(--warning-foreground)', // #1C1917
  error: 'var(--error)',                 // #DC2626
  errorForeground: 'var(--error-foreground)', // #FFFFFF
  info: 'var(--info)',                   // #2563EB
  infoForeground: 'var(--info-foreground)', // #FFFFFF
  sidebar: 'var(--sidebar)',             // #1C1917
  sidebarForeground: 'var(--sidebar-foreground)', // #F5F0EB
  sidebarPrimary: 'var(--sidebar-primary)', // #C2410C
  sidebarAccent: 'var(--sidebar-accent)', // #292524
  sidebarBorder: 'var(--sidebar-border)', // #44403C

  // Brand scale - terracotta shades for direct access
  brand: {
    50: '#fff7ed',
    100: '#ffedd5',
    200: '#fed7aa',
    300: '#fdba74',
    400: '#fb923c',
    500: '#ea580c',
    600: '#c2410c',
    700: '#9a3412',
    800: '#7c2d12',
    900: '#6c2710',
    950: '#431407',
  },
} as const;

// Typography scale
export const typography = {
  fontFamily: {
    sans: ['Inter', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
    display: ['Playfair Display', 'Georgia', 'serif'],
    accent: ['Caveat', 'cursive'],
  },
  fontSize: {
    xs: '0.75rem',
    sm: '0.875rem',
    base: '1rem',
    lg: '1.125rem',
    xl: '1.25rem',
    '2xl': '1.5rem',
    '3xl': '1.875rem',
    '4xl': '2.25rem',
    '5xl': '3rem',
    '6xl': '3.75rem',
    '7xl': '4.5rem',
  },
  fontWeight: {
    normal: '400',
    medium: '500',
    semibold: '600',
    bold: '700',
  },
} as const;

// Spacing scale (8px base)
export const spacing = {
  0: '0',
  0.5: '0.125rem',
  1: '0.25rem',
  1.5: '0.375rem',
  2: '0.5rem',
  2.5: '0.625rem',
  3: '0.75rem',
  3.5: '0.875rem',
  4: '1rem',
  5: '1.25rem',
  6: '1.5rem',
  7: '1.75rem',
  8: '2rem',
  9: '2.25rem',
  10: '2.5rem',
  11: '2.75rem',
  12: '3rem',
  14: '3.5rem',
  16: '4rem',
  18: '4.5rem',
  20: '5rem',
  24: '6rem',
} as const;

// Border radius scale
export const borderRadius = {
  none: '0',
  sm: '0.375rem',
  md: '0.5rem',
  lg: 'var(--radius)',
  xl: '1rem',
  '2xl': '1.25rem',
  '3xl': '1.5rem',
  '4xl': '2rem',
  full: '9999px',
} as const;

// Shadow scale
export const shadows = {
  none: 'none',
  sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
  md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
  lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
  xl: '0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)',
  'soft-sm': '0 2px 8px -2px rgb(0 0 0 / 0.08)',
  'soft-md': '0 4px 16px -4px rgb(0 0 0 / 0.1)',
  'soft-lg': '0 8px 24px -6px rgb(0 0 0 / 0.12)',
  'soft-xl': '0 16px 40px -8px rgb(0 0 0 / 0.15)',
  'soft-2xl': '0 24px 56px -12px rgb(0 0 0 / 0.18)',
  card: '0 1px 3px 0 rgb(0 0 0 / 0.04), 0 1px 2px -1px rgb(0 0 0 / 0.02)',
  'card-hover': '0 8px 24px -4px rgb(0 0 0 / 0.1), 0 4px 8px -2px rgb(0 0 0 / 0.04)',
  elevated: '0 4px 12px -2px rgb(0 0 0 / 0.08), 0 2px 4px -2px rgb(0 0 0 / 0.04)',
  'elevated-lg': '0 12px 32px -4px rgb(0 0 0 / 0.12), 0 4px 8px -2px rgb(0 0 0 / 0.06)',
  modal: '0 24px 48px -12px rgb(0 0 0 / 0.2), 0 12px 24px -8px rgb(0 0 0 / 0.1)',
} as const;

// Animation timing functions
export const easing = {
  premium: 'cubic-bezier(0.16, 1, 0.3, 1)',
  bounce: 'cubic-bezier(0.34, 1.56, 0.64, 1)',
  smooth: 'cubic-bezier(0.4, 0, 0.2, 1)',
  linear: 'linear',
  in: 'cubic-bezier(0.4, 0, 1, 1)',
  out: 'cubic-bezier(0, 0, 0.2, 1)',
  inOut: 'cubic-bezier(0.4, 0, 0.2, 1)',
} as const;

// Animation durations
export const durations = {
  instant: '0ms',
  fast: '150ms',
  normal: '200ms',
  moderate: '300ms',
  slow: '400ms',
  slower: '500ms',
} as const;

// Z-index scale
export const zIndex = {
  hide: -1,
  auto: 'auto',
  base: 0,
  dropdown: 1000,
  sticky: 1100,
  fixed: 1200,
  modalBackdrop: 1300,
  modal: 1400,
  popover: 1500,
  tooltip: 1600,
  toast: 1700,
} as const;

// Breakpoints (matching Tailwind defaults)
export const breakpoints = {
  sm: '640px',
  md: '768px',
  lg: '1024px',
  xl: '1280px',
  '2xl': '1536px',
} as const;

// Icon sizes
export const iconSizes = {
  xs: 12,
  sm: 16,
  md: 20,
  lg: 24,
  xl: 32,
  '2xl': 40,
} as const;

// Component-specific tokens
export const components = {
  button: {
    height: {
      xs: '1.75rem',
      sm: '2rem',
      md: '2.5rem',
      lg: '2.75rem',
      xl: '3rem',
    },
    padding: {
      xs: '0 0.5rem',
      sm: '0 0.75rem',
      md: '0 1rem',
      lg: '0 1.25rem',
      xl: '0 1.5rem',
    },
    fontSize: {
      xs: '0.75rem',
      sm: '0.875rem',
      md: '0.875rem',
      lg: '1rem',
      xl: '1rem',
    },
    borderRadius: 'var(--radius)',
  },
  card: {
    padding: {
      sm: '1rem',
      md: '1.5rem',
      lg: '2rem',
    },
    borderRadius: '1.25rem',
  },
  input: {
    height: {
      sm: '2.25rem',
      md: '2.5rem',
      lg: '2.75rem',
    },
    padding: {
      sm: '0 0.75rem',
      md: '0 1rem',
      lg: '0 1.25rem',
    },
    borderRadius: '0.5rem',
  },
  avatar: {
    size: {
      xs: '1.5rem',
      sm: '2rem',
      md: '2.5rem',
      lg: '3rem',
      xl: '4rem',
      '2xl': '5rem',
    },
  },
  badge: {
    height: {
      sm: '1.25rem',
      md: '1.5rem',
      lg: '1.75rem',
    },
    padding: {
      sm: '0 0.5rem',
      md: '0 0.625rem',
      lg: '0 0.75rem',
    },
    fontSize: {
      sm: '0.625rem',
      md: '0.75rem',
      lg: '0.875rem',
    },
    borderRadius: '9999px',
  },
} as const;

// Export all tokens
export const tokens = {
  colors,
  typography,
  spacing,
  borderRadius,
  shadows,
  easing,
  durations,
  zIndex,
  breakpoints,
  iconSizes,
  components,
} as const;

export default tokens;
