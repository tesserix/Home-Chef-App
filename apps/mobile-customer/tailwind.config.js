/** @type {import('tailwindcss').Config} */
module.exports = {
  // Auto-follows OS color scheme on mobile. User-override toggle is in
  // settings → switches via NativeWind's `useColorScheme()` setter.
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
        // Paper · Ink · Herb tokens — mirror web/src/styles/globals.css and
        // mobile-customer/global.css. Keep in sync.
        paper: 'var(--paper)',
        bone: 'var(--bone)',
        mist: {
          DEFAULT: 'var(--mist)',
          strong: 'var(--mist-strong)',
        },
        ink: {
          DEFAULT: 'var(--ink)',
          soft: 'var(--ink-soft)',
          muted: 'var(--ink-muted)',
        },
        herb: {
          DEFAULT: 'var(--herb)',
          soft: 'var(--herb-soft)',
          tint: 'var(--herb-tint)',
        },
        paprika: {
          DEFAULT: 'var(--paprika)',
          tint: 'var(--paprika-tint)',
        },
        amber: {
          DEFAULT: 'var(--amber)',
          tint: 'var(--amber-tint)',
        },
        info: {
          DEFAULT: 'var(--info)',
          tint: 'var(--info-tint)',
        },
        background: 'var(--background)',
        foreground: 'var(--foreground)',

        // Legacy `brand` alias → herb scale (mirror web tailwind.config.js)
        brand: {
          50: '#dde9d8',
          100: '#c8dcc1',
          200: '#abc8a3',
          300: '#8db285',
          400: '#6f9c67',
          500: '#3e6b3c',
          600: '#365e34',
          700: '#2d4f2c',
          800: '#244023',
          900: '#1a301a',
          950: '#0c1f0c',
        },
      },
      fontFamily: {
        sans: ['Inter', 'System'],
        display: ['Geist', 'Inter', 'System'],
      },
    },
  },
  plugins: [],
};
