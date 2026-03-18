/**
 * Fe3dr Brand Constants
 * Food & Chef themed design system
 */

// Brand Information
export const BRAND = {
  name: 'Fe3dr',
  tagline: 'Homemade Food Delivered',
  description: 'Connect with home chefs for authentic homemade food delivered to your doorstep',
  copyright: `© ${new Date().getFullYear()} Fe3dr. All rights reserved.`,
  trademark: '™',
};

// Color Palette - Food inspired
export const COLORS = {
  // Primary - Warm Orange (appetite stimulating)
  primary: {
    50: '#fff7ed',
    100: '#ffedd5',
    200: '#fed7aa',
    300: '#fdba74',
    400: '#fb923c',
    500: '#f97316', // Main brand color
    600: '#ea580c',
    700: '#c2410c',
    800: '#9a3412',
    900: '#7c2d12',
    950: '#431407',
  },
  // Secondary - Warm Red (tomato/spice)
  secondary: {
    50: '#fef2f2',
    100: '#fee2e2',
    200: '#fecaca',
    300: '#fca5a5',
    400: '#f87171',
    500: '#ef4444',
    600: '#dc2626',
    700: '#b91c1c',
    800: '#991b1b',
    900: '#7f1d1d',
    950: '#450a0a',
  },
  // Accent - Fresh Green (herbs/freshness)
  accent: {
    50: '#f0fdf4',
    100: '#dcfce7',
    200: '#bbf7d0',
    300: '#86efac',
    400: '#4ade80',
    500: '#22c55e',
    600: '#16a34a',
    700: '#15803d',
    800: '#166534',
    900: '#14532d',
    950: '#052e16',
  },
  // Warm - Golden Yellow (honey/turmeric)
  warm: {
    50: '#fffbeb',
    100: '#fef3c7',
    200: '#fde68a',
    300: '#fcd34d',
    400: '#fbbf24',
    500: '#f59e0b',
    600: '#d97706',
    700: '#b45309',
    800: '#92400e',
    900: '#78350f',
    950: '#451a03',
  },
  // Neutral - Warm Grays
  neutral: {
    50: '#fafaf9',
    100: '#f5f5f4',
    200: '#e7e5e4',
    300: '#d6d3d1',
    400: '#a8a29e',
    500: '#78716c',
    600: '#57534e',
    700: '#44403c',
    800: '#292524',
    900: '#1c1917',
    950: '#0c0a09',
  },
};

// Typography
export const TYPOGRAPHY = {
  fontFamily: {
    heading: '"Inter", system-ui, sans-serif',
    body: '"Inter", system-ui, sans-serif',
    accent: '"Playfair Display", Georgia, serif', // For special headings
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
};

// Spacing
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
};

// Border Radius - Soft, friendly curves
export const RADIUS = {
  sm: '0.375rem',
  md: '0.5rem',
  lg: '0.75rem',
  xl: '1rem',
  '2xl': '1.5rem',
  full: '9999px',
};

// Shadows - Warm tinted
export const SHADOWS = {
  sm: '0 1px 2px 0 rgba(251, 146, 60, 0.05)',
  md: '0 4px 6px -1px rgba(251, 146, 60, 0.1), 0 2px 4px -2px rgba(251, 146, 60, 0.1)',
  lg: '0 10px 15px -3px rgba(251, 146, 60, 0.1), 0 4px 6px -4px rgba(251, 146, 60, 0.1)',
  xl: '0 20px 25px -5px rgba(251, 146, 60, 0.1), 0 8px 10px -6px rgba(251, 146, 60, 0.1)',
};

// Gradients - Food inspired
export const GRADIENTS = {
  primary: 'linear-gradient(135deg, #f97316 0%, #ea580c 100%)',
  warm: 'linear-gradient(135deg, #f59e0b 0%, #f97316 100%)',
  sunset: 'linear-gradient(135deg, #f97316 0%, #ef4444 100%)',
  fresh: 'linear-gradient(135deg, #22c55e 0%, #16a34a 100%)',
  hero: 'linear-gradient(135deg, rgba(249, 115, 22, 0.9) 0%, rgba(234, 88, 12, 0.9) 100%)',
  overlay: 'linear-gradient(180deg, rgba(0, 0, 0, 0) 0%, rgba(0, 0, 0, 0.6) 100%)',
};

// Animation durations
export const ANIMATIONS = {
  fast: '150ms',
  normal: '200ms',
  slow: '300ms',
  slower: '500ms',
};

// Z-Index scale
export const Z_INDEX = {
  dropdown: 1000,
  sticky: 1020,
  fixed: 1030,
  modalBackdrop: 1040,
  modal: 1050,
  popover: 1060,
  tooltip: 1070,
  toast: 1080,
};

// Food-related emojis for fun UI elements
export const FOOD_EMOJIS = {
  chef: '👨‍🍳',
  cooking: '🍳',
  fire: '🔥',
  star: '⭐',
  heart: '❤️',
  delivery: '🛵',
  clock: '⏰',
  location: '📍',
  phone: '📱',
  money: '💰',
  celebration: '🎉',
  thumbsUp: '👍',
};

// Rating system
export const RATINGS = {
  min: 1,
  max: 5,
  excellent: 4.5,
  good: 4.0,
  average: 3.0,
};

// Order status colors
export const ORDER_STATUS_COLORS = {
  pending: COLORS.warm[500],
  confirmed: COLORS.primary[500],
  preparing: COLORS.primary[600],
  ready: COLORS.accent[500],
  out_for_delivery: COLORS.accent[600],
  delivered: COLORS.accent[700],
  cancelled: COLORS.secondary[500],
};
