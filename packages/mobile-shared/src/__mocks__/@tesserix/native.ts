// Mock for @tesserix/native used in vitest.
// Mirrors the Paper · Ink · Herb design system. Source of truth: /.impeccable.md.

export const colors = {
  paper: '#fafaf7',
  bone: '#f3f2ee',
  mist: '#e6e5e0',
  mistStrong: '#d4d3ce',
  ink: '#1a1a18',
  inkSoft: '#4a4a47',
  inkMuted: '#7a7a76',
  herb: '#3e6b3c',
  herbSoft: '#558257',
  herbTint: '#dde9d8',
  paprika: '#c95b3e',
  amber: '#d1a64a',
  info: '#4a73a3',
  // Semantic aliases for legacy @tesserix/native consumers
  primary: '#1a1a18',          // ink (Cash App / Stripe pattern)
  secondary: '#3e6b3c',        // herb
  background: {
    primary: '#fafaf7',         // paper
    secondary: '#f3f2ee',       // bone
  },
  text: {
    primary: '#1a1a18',         // ink
    secondary: '#4a4a47',       // ink-soft
  },
};

export const spacing = {
  0: 0,
  1: 4,
  2: 8,
  3: 12,
  4: 16,
  5: 20,
  6: 24,
  8: 32,
  10: 40,
  12: 48,
  16: 64,
};

export const typography = {
  fonts: {
    body: 'Inter',
    heading: 'Inter',
    display: 'Geist',
  },
  fontSizes: {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 18,
    xl: 20,
    '2xl': 24,
    '3xl': 30,
  },
};

// Stub components for tests
export const Button = () => null;
export const Input = () => null;
export const Text = () => null;
export const H1 = () => null;
export const H2 = () => null;
