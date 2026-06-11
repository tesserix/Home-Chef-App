/**
 * fe3dr.com landing — customer coral palette (Airbnb-style).
 * Every value traces to a CSS variable defined in app/globals.css;
 * components never hardcode hex. Source of truth: /.impeccable.md
 * → "Customer palette addendum".
 *
 * @type {import('tailwindcss').Config}
 */
export default {
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './node_modules/@tesserix/web/dist/**/*.{js,mjs}',
    '../../node_modules/@tesserix/web/dist/**/*.{js,mjs}',
  ],
  theme: {
    extend: {
      colors: {
        // Coral customer tokens
        coral: {
          DEFAULT: 'var(--coral)',
          pressed: 'var(--coral-pressed)',
          tint: 'var(--coral-tint)',
        },
        charcoal: {
          DEFAULT: 'var(--charcoal)',
          soft: 'var(--charcoal-soft)',
        },
        canvas: 'var(--canvas)',
        hairline: 'var(--hairline)',
        surface: 'var(--surface)',
        success: 'var(--success)',
        destructive: {
          DEFAULT: 'var(--destructive)',
          foreground: 'var(--destructive-foreground)',
        },

        // shadcn / @tesserix/web semantic aliases
        background: 'var(--background)',
        foreground: 'var(--foreground)',
        card: 'var(--card)',
        'card-foreground': 'var(--card-foreground)',
        popover: 'var(--popover)',
        'popover-foreground': 'var(--popover-foreground)',
        primary: {
          DEFAULT: 'var(--primary)',
          foreground: 'var(--primary-foreground)',
        },
        secondary: {
          DEFAULT: 'var(--secondary)',
          foreground: 'var(--secondary-foreground)',
        },
        muted: {
          DEFAULT: 'var(--muted)',
          foreground: 'var(--muted-foreground)',
        },
        accent: {
          DEFAULT: 'var(--accent)',
          foreground: 'var(--accent-foreground)',
        },
        border: 'var(--border)',
        input: 'var(--input)',
        ring: 'var(--ring)',
      },
      fontFamily: {
        display: ['var(--font-geist)', 'sans-serif'],
        sans: ['var(--font-inter)', 'sans-serif'],
      },
      borderRadius: {
        DEFAULT: '0.5rem',
        lg: '0.75rem',
        xl: '1rem',
      },
      boxShadow: {
        // Three-step elevation scale — never decorative.
        1: 'var(--shadow-1)',
        2: 'var(--shadow-2)',
        3: 'var(--shadow-3)',
      },
      transitionTimingFunction: {
        out: 'cubic-bezier(0.22, 1, 0.36, 1)',
        state: 'cubic-bezier(0.4, 0, 0.2, 1)',
      },
      transitionDuration: {
        micro: '150ms',
        page: '400ms',
      },
    },
  },
};
