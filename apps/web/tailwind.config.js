/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
    './node_modules/@tesserix/web/dist/**/*.{js,mjs}',
  ],
  theme: {
    extend: {
      colors: {
        // Paper · Ink · Herb tokens
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

        // shadcn / @tesserix/web semantic aliases
        background: 'var(--background)',
        foreground: 'var(--foreground)',
        card: { DEFAULT: 'var(--card)', foreground: 'var(--card-foreground)' },
        popover: { DEFAULT: 'var(--popover)', foreground: 'var(--popover-foreground)' },
        primary: { DEFAULT: 'var(--primary)', foreground: 'var(--primary-foreground)' },
        secondary: { DEFAULT: 'var(--secondary)', foreground: 'var(--secondary-foreground)' },
        muted: { DEFAULT: 'var(--muted)', foreground: 'var(--muted-foreground)' },
        accent: { DEFAULT: 'var(--accent)', foreground: 'var(--accent-foreground)' },
        destructive: { DEFAULT: 'var(--destructive)', foreground: 'var(--destructive-foreground)' },
        success: { DEFAULT: 'var(--success)', foreground: 'var(--success-foreground)' },
        warning: { DEFAULT: 'var(--warning)', foreground: 'var(--warning-foreground)' },
        error: { DEFAULT: 'var(--error)', foreground: 'var(--error-foreground)' },
        info: { DEFAULT: 'var(--info)', foreground: 'var(--info-foreground)' },
        border: 'var(--border)',
        input: 'var(--input)',
        ring: 'var(--ring)',
        sidebar: {
          DEFAULT: 'var(--sidebar)',
          foreground: 'var(--sidebar-foreground)',
          primary: 'var(--sidebar-primary)',
          accent: 'var(--sidebar-accent)',
          border: 'var(--sidebar-border)',
        },

        // Legacy compat: `brand` now resolves to a herb-green scale
        // so existing bg-brand-500 / text-brand-600 keep working but render the new accent.
        brand: {
          50: 'oklch(0.97 0.02 145)',
          100: 'oklch(0.93 0.04 145)',
          200: 'oklch(0.86 0.07 145)',
          300: 'oklch(0.76 0.10 145)',
          400: 'oklch(0.64 0.12 145)',
          500: 'var(--herb)',
          600: 'oklch(0.42 0.13 145)',
          700: 'oklch(0.36 0.12 145)',
          800: 'oklch(0.30 0.10 145)',
          900: 'oklch(0.24 0.08 145)',
          950: 'oklch(0.18 0.06 145)',
        },
        // Legacy: any orange-* / spice-* / golden-* refs (auth/marketing decoration)
        // now alias to ink/herb scale so old screens render coherently.
        spice: {
          50: 'oklch(0.97 0.02 145)',
          500: 'var(--herb)',
          600: 'oklch(0.42 0.13 145)',
        },
        golden: {
          50: 'oklch(0.95 0.02 80)',
          100: 'oklch(0.92 0.03 80)',
        },
      },
      fontFamily: {
        sans: [
          'Inter Variable',
          'Inter',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'sans-serif',
        ],
        display: [
          'Geist',
          'Geist Variable',
          'Inter',
          'system-ui',
          'sans-serif',
        ],
      },
      fontSize: {
        // Tighter, more editorial scale — no 4.5rem chest-thumpers
        'display-2xl': ['clamp(2.5rem, 4.5vw + 0.5rem, 3.5rem)', { lineHeight: '1.05', letterSpacing: '-0.025em', fontWeight: '600' }],
        'display-xl': ['clamp(2.25rem, 3.5vw + 0.5rem, 3rem)', { lineHeight: '1.08', letterSpacing: '-0.02em', fontWeight: '600' }],
        'display-lg': ['clamp(1.875rem, 2.5vw + 0.5rem, 2.5rem)', { lineHeight: '1.1', letterSpacing: '-0.02em', fontWeight: '600' }],
        'display-md': ['clamp(1.5rem, 1.8vw + 0.5rem, 2rem)', { lineHeight: '1.15', letterSpacing: '-0.015em', fontWeight: '600' }],
        'display-sm': ['clamp(1.25rem, 1.2vw + 0.5rem, 1.625rem)', { lineHeight: '1.2', letterSpacing: '-0.01em', fontWeight: '600' }],
        'display-xs': ['1.25rem', { lineHeight: '1.25', letterSpacing: '-0.005em', fontWeight: '600' }],
      },
      spacing: {
        18: '4.5rem',
        22: '5.5rem',
        26: '6.5rem',
        30: '7.5rem',
      },
      borderRadius: {
        sm: 'var(--radius-sm)',
        md: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
        xl: 'var(--radius-xl)',
        '2xl': 'var(--radius-2xl)',
      },
      boxShadow: {
        // Three-step elevation — used through CSS vars so dark mode mirrors
        1: 'var(--shadow-1)',
        2: 'var(--shadow-2)',
        3: 'var(--shadow-3)',
        // Aliases for migrated component code
        card: 'var(--shadow-1)',
        elevated: 'var(--shadow-2)',
        modal: 'var(--shadow-3)',
        'card-hover': 'var(--shadow-2)',
        'soft-sm': 'var(--shadow-1)',
        'soft-md': 'var(--shadow-2)',
        'soft-lg': 'var(--shadow-3)',
      },
      transitionTimingFunction: {
        // ease-out-quart — the canonical Home Chef curve
        smooth: 'cubic-bezier(0.22, 1, 0.36, 1)',
        // ease-in-out — for state changes
        ease: 'cubic-bezier(0.4, 0, 0.2, 1)',
        // Legacy aliases (no bounce, no overshoot — these now resolve to smooth)
        premium: 'cubic-bezier(0.22, 1, 0.36, 1)',
        'bounce-out': 'cubic-bezier(0.22, 1, 0.36, 1)',
      },
      transitionDuration: {
        150: '150ms',
        250: '250ms',
        400: '400ms',
      },
      animation: {
        'fade-in': 'fade-in 250ms cubic-bezier(0.22, 1, 0.36, 1)',
        'fade-in-up': 'fade-in-up 400ms cubic-bezier(0.22, 1, 0.36, 1)',
        'fade-in-down': 'fade-in-down 400ms cubic-bezier(0.22, 1, 0.36, 1)',
        'slide-up': 'slide-up 250ms cubic-bezier(0.22, 1, 0.36, 1)',
        'slide-down': 'slide-down 250ms cubic-bezier(0.22, 1, 0.36, 1)',
        'slide-in-right': 'slide-in-right 250ms cubic-bezier(0.22, 1, 0.36, 1)',
        'slide-in-left': 'slide-in-left 250ms cubic-bezier(0.22, 1, 0.36, 1)',
        'scale-in': 'scale-in 150ms cubic-bezier(0.22, 1, 0.36, 1)',
        shimmer: 'shimmer 2s linear infinite',
        // Legacy aliases — bounce-in is now a plain fade-in-up (no overshoot)
        'bounce-in': 'fade-in-up 400ms cubic-bezier(0.22, 1, 0.36, 1)',
        'pulse-soft': 'pulse-soft 2s ease-in-out infinite',
      },
      keyframes: {
        'pulse-soft': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.7' },
        },
      },
    },
  },
  plugins: [],
};
