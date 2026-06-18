/** Fe3dr wordmark — bowl+house+steam glyph (gradient tile) + Geist logotype. */
export function Wordmark({ inverted = false }: { inverted?: boolean }) {
  return (
    <span className="flex items-center gap-2.5">
      <svg
        width="28"
        height="28"
        viewBox="0 0 64 64"
        aria-hidden="true"
        focusable="false"
      >
        <defs>
          <linearGradient id="hc-wm-grad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#F34055" />
            <stop offset=".5" stopColor="#FA6F5C" />
            <stop offset="1" stopColor="#FAA15D" />
          </linearGradient>
        </defs>
        <rect width="64" height="64" rx="14" fill="url(#hc-wm-grad)" />
        <g
          fill="none"
          stroke="#fff"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M24.5 10c-2 2.6 2 5.2 0 7.8-2 2.6 2 5.2 0 7.8" />
          <path d="M32 8c-2 2.6 2 5.2 0 7.8-2 2.6 2 5.2 0 7.8" />
          <path d="M39.5 10c-2 2.6 2 5.2 0 7.8-2 2.6 2 5.2 0 7.8" />
          <path d="M13 33h38" />
          <path d="M16.5 36a15.5 15.5 0 0 0 8.5 13.8" />
          <path d="M47.5 36a15.5 15.5 0 0 1-8.5 13.8" />
          <path d="M23 41.8 32 35.2l9 6.6" />
          <path d="M25.5 41.3v10h13v-10" />
          <path d="M37.8 36.4v3" />
        </g>
      </svg>
      <span
        className={`font-display text-lg font-bold tracking-tight ${
          inverted ? 'text-white' : 'text-charcoal'
        }`}
      >
        Fe3dr
      </span>
    </span>
  );
}
