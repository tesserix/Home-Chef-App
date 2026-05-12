import { Toaster, type ToasterProps } from 'sonner';
import { useTheme } from './ThemeProvider';

/**
 * Sonner Toaster wired to Home Chef's resolved theme.
 *
 * Sonner's own `theme="system"` reads `prefers-color-scheme` directly,
 * which would diverge from a user override (e.g., user forces light mode
 * while system is dark). This wrapper bridges that gap.
 */
export function ThemedToaster(props: Omit<ToasterProps, 'theme'>) {
  const { resolvedTheme } = useTheme();
  return <Toaster theme={resolvedTheme} {...props} />;
}
