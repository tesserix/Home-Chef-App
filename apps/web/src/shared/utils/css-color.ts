/**
 * Resolve a CSS custom property to a hex/rgb string at runtime.
 *
 * Why: external SDKs (Razorpay Checkout, Stripe Elements, embedded iframes)
 * accept hex/rgb colors only — they can't read CSS variables or oklch. This
 * helper lets us keep brand colors in `--herb` (oklch) and feed the same
 * resolved value to those SDKs, so a theme change ripples there too.
 *
 * Uses canvas fillStyle normalization: setting `ctx.fillStyle` to any valid
 * CSS color (including oklch) and reading it back yields the browser's
 * normalized form ("#rrggbb" or "rgb(r, g, b)"), which downstream consumers
 * universally accept.
 */
export function resolveCssVarColor(varName: string, fallback: string): string {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return fallback;
  }
  try {
    const raw = getComputedStyle(document.documentElement)
      .getPropertyValue(varName)
      .trim();
    if (!raw) return fallback;
    const ctx = document.createElement('canvas').getContext('2d');
    if (!ctx) return fallback;
    ctx.fillStyle = raw;
    // Browser normalizes oklch → "#rrggbb" or "rgba(...)". Both forms are
    // accepted by Razorpay / Stripe / most embedded payment SDKs.
    return ctx.fillStyle;
  } catch {
    return fallback;
  }
}
