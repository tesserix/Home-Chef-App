import { Moon, Sun, Monitor } from 'lucide-react';
import { useTheme, type Theme } from './ThemeProvider';

/**
 * A 3-state segmented toggle: Light · Auto · Dark.
 *
 * Drop into any header / settings page. Persists user choice via `useTheme`.
 *
 * @example
 *   <ThemeToggle />
 *   <ThemeToggle size="sm" />
 */

interface ThemeToggleProps {
  size?: 'sm' | 'md';
  className?: string;
}

const OPTIONS: Array<{ value: Theme; label: string; Icon: typeof Sun }> = [
  { value: 'light', label: 'Light', Icon: Sun },
  { value: 'auto', label: 'System', Icon: Monitor },
  { value: 'dark', label: 'Dark', Icon: Moon },
];

export function ThemeToggle({ size = 'md', className = '' }: ThemeToggleProps) {
  const { theme, setTheme } = useTheme();

  const sizeStyles = size === 'sm' ? 'h-8 p-0.5 text-xs' : 'h-9 p-1 text-sm';
  const buttonSizeStyles = size === 'sm' ? 'h-7 w-7' : 'h-7 w-9';

  return (
    <div
      role="radiogroup"
      aria-label="Theme"
      className={`inline-flex items-center rounded-md border border-mist bg-bone ${sizeStyles} ${className}`}
    >
      {OPTIONS.map(({ value, label, Icon }) => {
        const active = theme === value;
        return (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={label}
            onClick={() => setTheme(value)}
            className={[
              'flex items-center justify-center rounded transition-colors',
              buttonSizeStyles,
              active
                ? 'bg-foreground text-background'
                : 'text-ink-muted hover:text-foreground',
            ].join(' ')}
          >
            <Icon className="h-4 w-4" aria-hidden />
          </button>
        );
      })}
    </div>
  );
}

/**
 * A single-button variant that cycles light → dark → auto → light.
 * Useful in tight nav bars where the segmented toggle is too wide.
 */
export function ThemeToggleCompact({ className = '' }: { className?: string }) {
  const { theme, resolvedTheme, setTheme } = useTheme();
  const Icon = theme === 'auto' ? Monitor : resolvedTheme === 'dark' ? Moon : Sun;

  const cycle = () => {
    const next: Theme = theme === 'light' ? 'dark' : theme === 'dark' ? 'auto' : 'light';
    setTheme(next);
  };

  return (
    <button
      type="button"
      onClick={cycle}
      aria-label={`Theme: ${theme}. Click to change.`}
      title={`Theme: ${theme}`}
      className={`inline-flex h-9 w-9 items-center justify-center rounded-md text-ink-muted transition-colors hover:bg-muted hover:text-foreground touch-target ${className}`}
    >
      <Icon className="h-4 w-4" aria-hidden />
    </button>
  );
}
