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
        // Paper · Ink · Herb tokens — inlined as hex (was var(--*) refs into
        // global.css, but nativewind 5 + lightningcss can't process the CSS
        // file. Light-mode values only; dark-mode handling deferred.
        paper: '#fafaf7',
        bone: '#f3f2ee',
        mist: {
          DEFAULT: '#e6e5e0',
          strong: '#d4d3ce',
        },
        ink: {
          DEFAULT: '#1a1a18',
          soft: '#4a4a47',
          muted: '#7a7a76',
        },
        herb: {
          DEFAULT: '#3e6b3c',
          soft: '#558257',
          tint: '#dde9d8',
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
        background: '#fafaf7',
        foreground: '#1a1a18',

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
