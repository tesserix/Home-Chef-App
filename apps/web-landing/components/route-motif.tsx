/**
 * The Fe3dr signature device: a dashed delivery route from a
 * chef's stove to your door, with a coral dot travelling along it.
 * The dot animates via CSS offset-path (transform-only, see
 * globals.css `.route-dot`); reduced-motion shows it parked mid-route.
 *
 * Decorative — always render inside an aria-hidden container.
 */

const ROUTE_PATH = 'M6 44 C 56 6, 120 54, 196 16';

export function RouteMotif({ className }: { className?: string }) {
  return (
    <div
      aria-hidden="true"
      className={`relative h-[56px] w-[208px] ${className ?? ''}`}
    >
      <svg
        width="208"
        height="56"
        viewBox="0 0 208 56"
        fill="none"
        className="absolute inset-0"
        focusable="false"
      >
        {/* The route */}
        <path
          d={ROUTE_PATH}
          stroke="var(--ink-outline)"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeDasharray="1 7"
        />
        {/* Origin — the chef's stove burner */}
        <circle cx="6" cy="44" r="3" fill="var(--charcoal-soft)" />
        <circle
          cx="6"
          cy="44"
          r="6.5"
          stroke="var(--charcoal-soft)"
          strokeOpacity="0.35"
          strokeWidth="1.5"
        />
        {/* Destination — your door */}
        <path
          d="M189 17.5 L196 11.5 L203 17.5 V25 a1.5 1.5 0 0 1 -1.5 1.5 h-11 a1.5 1.5 0 0 1 -1.5 -1.5 Z"
          fill="var(--charcoal)"
        />
        <rect x="194" y="20.5" width="4" height="6" rx="0.8" fill="#fff" />
      </svg>
      {/* Travelling order — coral, the single moving accent */}
      <div className="route-dot" style={{ offsetPath: `path("${ROUTE_PATH}")` }} />
    </div>
  );
}
