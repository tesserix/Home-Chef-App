/** @type {import('tailwindcss').Config} */
//
// Source of truth: .impeccable.md at repo root.
// TS bridge:        packages/mobile-shared/src/theme/tokens.ts
//
// Admin shares the Uber-style ink-centric palette used by the vendor app:
// Ink carries every primary action; functional colours (green/red/amber)
// are never decorative. Keep values in lockstep with tokens.ts.
module.exports = {
  darkMode: 'media',
  content: [
    './app/**/*.{js,ts,jsx,tsx}',
    './components/**/*.{js,ts,jsx,tsx}',
    './hooks/**/*.{js,ts,jsx,tsx}',
    './lib/**/*.{js,ts,jsx,tsx}',
    './store/**/*.{js,ts,jsx,tsx}',
    '../../packages/mobile-shared/src/**/*.{js,ts,jsx,tsx}',
  ],
  presets: [require('nativewind/preset')],
  theme: {
    extend: {
      colors: {
        paper: '#FFFFFF',
        bone: '#F5F5F4',
        mist: {
          DEFAULT: '#E5E5E5',
          strong: '#D6D6D6',
        },
        ink: {
          DEFAULT: '#0E0E0C',
          soft: '#525252',
          muted: '#888888',
        },
        // Persimmon retired as the brand accent — Ink carries primary actions.
        herb: {
          DEFAULT: '#0E0E0C',
          soft: '#2A2A28',
          tint: '#F5F5F4',
        },
        destructive: {
          DEFAULT: '#B22B0E',
          tint: '#FBE8E1',
        },
        success: {
          DEFAULT: '#008A05',
          soft: '#046A06',
          tint: '#E6F4E6',
        },
        paprika: {
          DEFAULT: '#c95b3e',
          tint: '#f3dcd2',
        },
        amber: {
          DEFAULT: '#d1a64a',
          tint: '#f0e3c0',
        },
        info: {
          DEFAULT: '#4a73a3',
          tint: '#dde5ee',
        },
        diet: {
          veg: '#2A9D3E',
          'non-veg': '#B22B0E',
        },
        background: '#FFFFFF',
        foreground: '#0E0E0C',
      },
      fontFamily: {
        sans: ['Inter', 'System'],
        'sans-medium': ['Inter-Medium', 'Inter', 'System'],
        'sans-semibold': ['Inter-SemiBold', 'Inter', 'System'],
        display: ['Geist', 'Inter', 'System'],
        'display-bold': ['Geist-Bold', 'Geist', 'System'],
        mono: ['Menlo', 'Courier New'],
      },
      fontSize: {
        display: ['32px', { lineHeight: '35px', letterSpacing: '-0.5px' }],
        h1: ['26px', { lineHeight: '30px', letterSpacing: '-0.3px' }],
        h2: ['20px', { lineHeight: '24px', letterSpacing: '-0.2px' }],
        body: ['16px', { lineHeight: '24px', letterSpacing: '0px' }],
        'body-sm': ['14px', { lineHeight: '20px', letterSpacing: '0px' }],
        label: ['13px', { lineHeight: '18px', letterSpacing: '0.1px' }],
        caption: ['11px', { lineHeight: '14px', letterSpacing: '0.2px' }],
      },
      borderRadius: {
        none: '0px',
        sm: '4px',
        DEFAULT: '8px',
        md: '12px',
        lg: '16px',
        full: '9999px',
      },
      boxShadow: {
        'elev-1': '0 1px 2px rgba(26, 26, 24, 0.04)',
        'elev-2': '0 2px 6px rgba(26, 26, 24, 0.06)',
        'elev-3': '0 8px 24px rgba(26, 26, 24, 0.12)',
      },
      minHeight: {
        touch: '44px',
        'touch-lg': '48px',
      },
      minWidth: {
        touch: '44px',
        'touch-lg': '48px',
      },
    },
  },
  plugins: [],
};
