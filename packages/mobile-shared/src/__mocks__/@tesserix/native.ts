// Mock for @tesserix/native used in vitest
// Provides the same shape that tokens.ts expects AND stub React components for tests

export const colors = {
  primary: '#FF6B35',
  secondary: '#2D3748',
  background: {
    primary: '#FFFFFF',
    secondary: '#F7FAFC',
  },
  text: {
    primary: '#1A202C',
    secondary: '#718096',
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
};

export const typography = {
  fonts: {
    body: 'System',
    heading: 'System',
  },
  fontSizes: {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 18,
    xl: 20,
  },
};

// Stub components — only used in tests that import @tesserix/native directly
// Screen rendering tests live in each app's jest-expo test suite
export const Button = () => null;
export const Input = () => null;
export const Text = () => null;
export const H1 = () => null;
export const H2 = () => null;
