import { Moon, Sun } from 'lucide-react';
import { useTheme, type Theme } from './ThemeProvider';

/**
 * A 2-state segmented toggle: Light · Dark.
 *
 * "System" was removed deliberately: a chef picks the mode that suits their
 * kitchen's lighting, and having the app change underneath them when their
 * laptop crossed into night mode mid-service was noise, not a feature.
 *
 * The provider still understands 'auto' (older saved preferences, and the
 * pre-choice default), so an account already on it is shown whichever mode it
 * currently resolves to — the control is never left with nothing selected — and
 * the first click pins a concrete choice.
 *
 * @example
 *   <ThemeToggle />
 *   <ThemeToggle size="sm" />
 */

interface ThemeToggleProps {
  size?: 'sm' | 'md';
  className?: string;
}

const OPTIONS: Array<{ value: Exclude<Theme, 'auto'>; label: string; Icon: typeof Sun }> = [
  { value: 'light', label: 'Light', Icon: Sun },
  { value: 'dark', label: 'Dark', Icon: Moon },
];

export function ThemeToggle({ size = 'md', className = '' }: ThemeToggleProps) {
  const { theme, resolvedTheme, setTheme } = useTheme();
  // An account still on 'auto' highlights whatever it resolves to right now,
  // so the control always shows a selection instead of appearing broken.
  const effective = theme === 'auto' ? resolvedTheme : theme;

  const sizeStyles = size === 'sm' ? 'h-8 p-0.5 text-xs' : 'h-9 p-1 text-sm';
  const buttonSizeStyles = size === 'sm' ? 'h-7 w-7' : 'h-7 w-9';

  return (
    <div
      role="radiogroup"
      aria-label="Theme"
      className={`inline-flex items-center rounded-md border border-mist bg-bone ${sizeStyles} ${className}`}
    >
      {OPTIONS.map(({ value, label, Icon }) => {
        const active = effective === value;
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
 * A single-button variant that toggles light ↔ dark.
 * Useful in tight nav bars where the segmented toggle is too wide.
 */
export function ThemeToggleCompact({ className = '' }: { className?: string }) {
  const { theme, resolvedTheme, setTheme } = useTheme();
  const effective = theme === 'auto' ? resolvedTheme : theme;
  const Icon = effective === 'dark' ? Moon : Sun;

  const cycle = () => setTheme(effective === 'dark' ? 'light' : 'dark');

  return (
    <button
      type="button"
      onClick={cycle}
      aria-label={`Theme: ${effective}. Switch to ${effective === 'dark' ? 'light' : 'dark'}.`}
      title={`Theme: ${effective}`}
      className={`inline-flex h-9 w-9 items-center justify-center rounded-md text-ink-muted transition-colors hover:bg-muted hover:text-foreground touch-target ${className}`}
    >
      <Icon className="h-4 w-4" aria-hidden />
    </button>
  );
}
