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
        // ===== Airbnb-style customer palette (owner decision 2026-06-10) =====
        // The customer app gets a premium consumer-marketplace look: white-first
        // canvas, charcoal text, RAUSCH CORAL as the single accent. Mirrors
        // `customerColors` in packages/mobile-shared/src/theme/tokens.ts.
        //
        // GUARDRAILS: `coral` must never appear outside apps/mobile-customer.
        // Customer screens must never use `herb` (persimmon) — it is repointed
        // to coral below only as a migration safety-net and is grep-banned.

        // Brand accent — the ONE colour on screen.
        coral: {
          DEFAULT: '#FF385C', // primary CTA, links, selected, heart-saved, focus
          pressed: '#E00B41',
          tint: '#FFE8EC', // chip / badge tint
        },
        // Text + dark buttons — Airbnb near-black (never pure #000).
        charcoal: {
          DEFAULT: '#222222',
          soft: '#717171', // secondary text, captions, placeholders
        },
        // Surfaces — white-first.
        canvas: '#FFFFFF', // page background
        hairline: '#EBEBEB', // dividers, card borders
        surface: {
          DEFAULT: '#FFFFFF',
          soft: '#F7F7F7', // input fills, soft sections, image placeholders
        },
        success: {
          DEFAULT: '#008A05', // delivered / confirmation only
          tint: '#E6F4E6',
        },

        // ----- Legacy token names, repointed to the Airbnb system so shared
        // components (packages/mobile-shared) and not-yet-migrated screens stay
        // coherent during the migration. New customer code uses the explicit
        // names above. `herb` is intentionally repointed to coral but BANNED in
        // customer screens (grep gate) — migrate any straggler to `coral`.
        paper: '#FFFFFF',
        bone: '#F7F7F7',
        mist: {
          DEFAULT: '#EBEBEB',
          strong: '#DDDDDD',
        },
        ink: {
          DEFAULT: '#222222',
          soft: '#717171',
          muted: '#717171',
        },
        herb: {
          DEFAULT: '#FF385C',
          soft: '#E00B41',
          tint: '#FFE8EC',
        },
        amber: {
          DEFAULT: '#d1a64a',
          tint: '#f0e3c0',
        },
        info: {
          DEFAULT: '#4a73a3',
          tint: '#dde5ee',
        },
        background: '#FFFFFF',
        foreground: '#222222',
      },
      fontFamily: {
        sans: ['Inter', 'System'],
        display: ['Geist', 'Inter', 'System'],
      },
    },
  },
  plugins: [],
};
