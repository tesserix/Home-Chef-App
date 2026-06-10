/** @type {import('tailwindcss').Config} */
//
// Source of truth: .impeccable.md at repo root.
// TS bridge:        packages/mobile-shared/src/theme/tokens.ts
//
// Keep the three in lockstep. Token values below MUST equal those in
// tokens.ts so a screen can switch between `className="bg-herb"` and
// `style={{ backgroundColor: colors.herb.DEFAULT }}` with no surprise.
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
      // --- COLORS -----------------------------------------------------------
      // Paper · Ink · Persimmon. See .impeccable.md.
      // Names match packages/mobile-shared/src/theme/tokens.ts.colors keys.
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
        // Uber-style vendor app: persimmon retired as the brand accent — Ink
        // now carries every primary action (CTA, toggle, tab, link, focus).
        // `herb` is kept only so any stray legacy class resolves to Ink rather
        // than reintroducing orange; new code should use `ink` directly.
        herb: {
          DEFAULT: '#0E0E0C',
          soft: '#2A2A28',
          tint: '#F5F5F4',
        },
        // Functional only — never decorative.
        destructive: {
          DEFAULT: '#B22B0E',
          tint: '#FBE8E1',
        },
        // Functional success — "go / ready / verified" green. Carries positive
        // operational status now that persimmon is gone. Mirrors tokens.ts.
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
        // Diet indicator colors — FSSAI-style veg/non-veg icons only.
        diet: {
          veg: '#2A9D3E',
          'non-veg': '#B22B0E',
        },
        background: '#FFFFFF',
        foreground: '#0E0E0C',
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

      // --- TYPOGRAPHY ------------------------------------------------------
      // Geist for display, Inter for UI. Fonts must be loaded via
      // expo-font at app boot (apps/mobile-vendor/app/_layout.tsx) — until
      // then the platform falls back to the System sans.
      fontFamily: {
        // RN selects fonts by family name only (fontWeight does not pick
        // a different family for non-system fonts). So each weight is a
        // separate family registered via expo-font in app/_layout.tsx.
        //
        //   font-sans            → body / UI               (Inter 400)
        //   font-sans-medium     → slight emphasis         (Inter 500)
        //   font-sans-semibold   → buttons, strong labels  (Inter 600)
        //   font-display         → headlines, brand        (Geist 600)
        //   font-display-bold    → hero, large numerals    (Geist 700)
        //   font-mono            → IDs / receipts / addrs  (system mono)
        sans: ['Inter', 'System'],
        'sans-medium': ['Inter-Medium', 'Inter', 'System'],
        'sans-semibold': ['Inter-SemiBold', 'Inter', 'System'],
        display: ['Geist', 'Inter', 'System'],
        'display-bold': ['Geist-Bold', 'Geist', 'System'],
        mono: ['Menlo', 'Courier New'],
      },

      // Type ramp. Numbers are absolute pt (RN). Line-height multiplier.
      // Letter-spacing is in em-equivalent (RN treats as pixels).
      fontSize: {
        // Display — Geist
        display: ['32px', { lineHeight: '35px', letterSpacing: '-0.5px' }],
        h1: ['26px', { lineHeight: '30px', letterSpacing: '-0.3px' }],
        h2: ['20px', { lineHeight: '24px', letterSpacing: '-0.2px' }],
        // Body — Inter
        body: ['16px', { lineHeight: '24px', letterSpacing: '0px' }],
        'body-sm': ['14px', { lineHeight: '20px', letterSpacing: '0px' }],
        label: ['13px', { lineHeight: '18px', letterSpacing: '0.1px' }],
        caption: ['11px', { lineHeight: '14px', letterSpacing: '0.2px' }],
      },

      // --- RADII -----------------------------------------------------------
      // 4 / 8 / 12 / 16 / full. No super-round 24px friendly pills.
      borderRadius: {
        none: '0px',
        sm: '4px',
        DEFAULT: '8px',
        md: '12px',
        lg: '16px', // sheets / modals
        full: '9999px', // dots, avatars, FABs only
      },

      // --- SHADOWS --------------------------------------------------------
      // iOS-only — NativeWind translates these to shadowColor/Offset etc.
      // On Android, `elevation` must be set per-component via tokens.ts.
      boxShadow: {
        'elev-1': '0 1px 2px rgba(26, 26, 24, 0.04)',
        'elev-2': '0 2px 6px rgba(26, 26, 24, 0.06)',
        'elev-3': '0 8px 24px rgba(26, 26, 24, 0.12)',
      },

      // --- TOUCH TARGETS --------------------------------------------------
      // 44pt minimum for customer/vendor. Driver app overrides to 48 in
      // its own config. Use as `min-h-touch` on every Pressable.
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
