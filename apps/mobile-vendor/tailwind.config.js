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
        // Brand accent: Persimmon (warm editorial orange).
        // Token name kept as `herb` to avoid a 49-file rename across the apps;
        // class names (`bg-herb`, `text-herb`) now render orange.
        herb: {
          DEFAULT: '#C2410C',
          soft: '#9A3412',
          tint: '#FFEDD5',
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

        // Legacy `brand` alias → persimmon scale (mirror web tailwind.config.js)
        brand: {
          50: '#FFF7ED',
          100: '#FFEDD5',
          200: '#FED7AA',
          300: '#FDBA74',
          400: '#FB923C',
          500: '#C2410C',
          600: '#9A3412',
          700: '#7C2D12',
          800: '#5C1F0A',
          900: '#3F1409',
          950: '#2A0C06',
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
