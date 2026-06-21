import { format, formatDistanceToNow, parseISO } from 'date-fns';

/** ₹1,23,456.78 — Indian-grouped rupees. HomeChef bills in INR (Razorpay). */
export function formatINR(amount: number | null | undefined): string {
  const n = typeof amount === 'number' && Number.isFinite(amount) ? amount : 0;
  const sign = n < 0 ? '-' : '';
  const abs = Math.abs(n);
  const fixed = abs.toFixed(2);
  const [intPart, decPart] = fixed.split('.');
  // Indian grouping: last 3 digits, then groups of 2.
  const last3 = intPart.slice(-3);
  const rest = intPart.slice(0, -3);
  const grouped = rest
    ? rest.replace(/\B(?=(\d{2})+(?!\d))/g, ',') + ',' + last3
    : last3;
  return `${sign}₹${grouped}.${decPart}`;
}

/** Compact integer with thousands separators (1,234). */
export function formatCount(n: number | null | undefined): string {
  const v = typeof n === 'number' && Number.isFinite(n) ? n : 0;
  return v.toLocaleString('en-IN');
}

function toDate(value: string | number | Date | null | undefined): Date | null {
  if (!value) return null;
  try {
    if (value instanceof Date) return value;
    if (typeof value === 'number') return new Date(value);
    return parseISO(value);
  } catch {
    return null;
  }
}

/** "21 Jun 2026, 4:32 PM" */
export function formatDateTime(value: string | number | Date | null | undefined): string {
  const d = toDate(value);
  if (!d || Number.isNaN(d.getTime())) return '—';
  return format(d, 'd MMM yyyy, h:mm a');
}

/** "21 Jun 2026" */
export function formatDate(value: string | number | Date | null | undefined): string {
  const d = toDate(value);
  if (!d || Number.isNaN(d.getTime())) return '—';
  return format(d, 'd MMM yyyy');
}

/** "3 hours ago" */
export function formatRelative(value: string | number | Date | null | undefined): string {
  const d = toDate(value);
  if (!d || Number.isNaN(d.getTime())) return '—';
  return formatDistanceToNow(d, { addSuffix: true });
}

/** snake_case / kebab-case → "Title Case" */
export function titleCase(value: string | null | undefined): string {
  if (!value) return '';
  return value
    .replace(/[_-]+/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

/** Resolve a readable error message from an axios/JS error. */
export function errorMessage(err: unknown): string {
  if (err && typeof err === 'object') {
    const anyErr = err as {
      response?: { data?: { error?: { message?: string } | string; message?: string } };
      message?: string;
    };
    const data = anyErr.response?.data;
    if (data) {
      if (typeof data.error === 'string') return data.error;
      if (data.error?.message) return data.error.message;
      if (typeof data.message === 'string') return data.message;
    }
    if (anyErr.message) return anyErr.message;
  }
  return 'Something went wrong. Please try again.';
}
