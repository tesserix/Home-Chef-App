/**
 * ThemeProvider — Home Chef's light/dark/auto theme controller.
 *
 * Adds a `dark` class to <html> when dark mode is active. Tailwind's
 * `darkMode: 'class'` then applies all `dark:` variants.
 *
 * - User preference is persisted to localStorage under `hc-theme`.
 * - When set to `auto` (default), follows `prefers-color-scheme` and
 *   updates automatically when the system theme changes.
 * - A small inline script in `index.html` reads the saved preference
 *   before React hydrates so there's no flash of incorrect theme.
 *
 * Usage:
 *   <ThemeProvider>
 *     <App />
 *   </ThemeProvider>
 *
 *   const { theme, resolvedTheme, setTheme } = useTheme();
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

export type Theme = 'light' | 'dark' | 'auto';
export type ResolvedTheme = 'light' | 'dark';

interface ThemeContextValue {
  /** The user's stated preference: `light`, `dark`, or `auto`. */
  theme: Theme;
  /** The actual rendered theme after resolving `auto`. */
  resolvedTheme: ResolvedTheme;
  /** Whether the OS reports a dark-mode preference. */
  systemTheme: ResolvedTheme;
  setTheme: (next: Theme) => void;
}

const STORAGE_KEY = 'hc-theme';
const ThemeContext = createContext<ThemeContextValue | null>(null);

function readStoredTheme(): Theme {
  if (typeof window === 'undefined') return 'auto';
  const stored = window.localStorage.getItem(STORAGE_KEY);
  return stored === 'light' || stored === 'dark' || stored === 'auto' ? stored : 'auto';
}

function readSystemTheme(): ResolvedTheme {
  if (typeof window === 'undefined' || !window.matchMedia) return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyResolvedTheme(resolved: ResolvedTheme): void {
  const root = document.documentElement;
  root.classList.toggle('dark', resolved === 'dark');
  root.classList.toggle('light', resolved === 'light');
  root.dataset.theme = resolved;
}

interface ThemeProviderProps {
  children: ReactNode;
  /** Override the initial theme — useful for SSR / tests. */
  defaultTheme?: Theme;
}

export function ThemeProvider({ children, defaultTheme = 'auto' }: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(() =>
    typeof window === 'undefined' ? defaultTheme : readStoredTheme()
  );
  const [systemTheme, setSystemTheme] = useState<ResolvedTheme>(() => readSystemTheme());

  const resolvedTheme: ResolvedTheme = theme === 'auto' ? systemTheme : theme;

  // Listen for system-theme changes (user toggles dark mode in OS).
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const mql = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => setSystemTheme(e.matches ? 'dark' : 'light');
    mql.addEventListener('change', handler);
    return () => mql.removeEventListener('change', handler);
  }, []);

  // Apply class + dataset whenever the resolved theme changes.
  useEffect(() => {
    applyResolvedTheme(resolvedTheme);
  }, [resolvedTheme]);

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next);
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(STORAGE_KEY, next);
    }
  }, []);

  const value = useMemo<ThemeContextValue>(
    () => ({ theme, resolvedTheme, systemTheme, setTheme }),
    [theme, resolvedTheme, systemTheme, setTheme]
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error('useTheme must be used inside <ThemeProvider>.');
  }
  return ctx;
}

/**
 * The inline script that runs *before* React mounts to set the right theme.
 * Inject this in `index.html` <head> to prevent flash of incorrect theme (FOIT).
 *
 * @example
 *   <script>{NO_FLASH_SCRIPT}</script>
 */
export const NO_FLASH_SCRIPT = `(function(){try{var t=localStorage.getItem('${STORAGE_KEY}');var d=t==='dark'||(t==='auto'||!t)&&window.matchMedia('(prefers-color-scheme: dark)').matches;var r=document.documentElement;r.classList.toggle('dark',d);r.classList.toggle('light',!d);r.dataset.theme=d?'dark':'light';}catch(e){}})();`;
