/**
 * Locale-aware date/time formatters wrapping Intl.DateTimeFormat.
 *
 * All helpers default to the browser locale (undefined = navigator.language).
 * Pass a value that may be undefined / NaN safely — they return an em-dash
 * rather than the literal "Invalid Date" string.
 *
 * Why a helper instead of inlined toLocaleString? Pages had `'en-US'`
 * hardcoded everywhere; localization was effectively impossible. Centralizing
 * here means switching languages later is a single-file change.
 */

const UNFORMATTABLE = '—';

function toDate(value: Date | string | number | null | undefined): Date | null {
  if (value === null || value === undefined || value === '') return null;
  const d = value instanceof Date ? value : new Date(value);
  return Number.isNaN(d.getTime()) ? null : d;
}

/** Date only — "Jan 15, 2026" / "15. Jan. 2026" / "15/1/2026" depending on locale. */
export function formatDate(
  value: Date | string | number | null | undefined,
  opts?: Intl.DateTimeFormatOptions
): string {
  const d = toDate(value);
  if (!d) return UNFORMATTABLE;
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    ...opts,
  }).format(d);
}

/** Date + time — "Jan 15, 2026, 7:30 PM" or locale-specific equivalent. */
export function formatDateTime(
  value: Date | string | number | null | undefined,
  opts?: Intl.DateTimeFormatOptions
): string {
  const d = toDate(value);
  if (!d) return UNFORMATTABLE;
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    ...opts,
  }).format(d);
}

/** Time only — "7:30 PM" / "19:30" depending on locale. */
export function formatTime(
  value: Date | string | number | null | undefined,
  opts?: Intl.DateTimeFormatOptions
): string {
  const d = toDate(value);
  if (!d) return UNFORMATTABLE;
  return new Intl.DateTimeFormat(undefined, {
    hour: 'numeric',
    minute: '2-digit',
    ...opts,
  }).format(d);
}

/** Short weekday — "Mon" / "Lun" / "月". */
export function formatWeekday(value: Date | string | number | null | undefined): string {
  const d = toDate(value);
  if (!d) return UNFORMATTABLE;
  return new Intl.DateTimeFormat(undefined, { weekday: 'short' }).format(d);
}

/**
 * Relative time — "2 hours ago", "yesterday", "in 3 days". Falls back to
 * formatDate for anything older than a week so we don't say "47 days ago".
 */
export function formatRelative(value: Date | string | number | null | undefined): string {
  const d = toDate(value);
  if (!d) return UNFORMATTABLE;
  const now = Date.now();
  const diffMs = d.getTime() - now;
  const absSec = Math.abs(diffMs) / 1000;

  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
  if (absSec < 60) return rtf.format(Math.round(diffMs / 1000), 'second');
  if (absSec < 3600) return rtf.format(Math.round(diffMs / 60000), 'minute');
  if (absSec < 86400) return rtf.format(Math.round(diffMs / 3600000), 'hour');
  if (absSec < 86400 * 7) return rtf.format(Math.round(diffMs / 86400000), 'day');
  return formatDate(d);
}
