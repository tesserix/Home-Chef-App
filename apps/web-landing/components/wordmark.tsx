/** Home Chef wordmark — steaming-bowl glyph + Geist logotype. */
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
        <rect width="64" height="64" rx="14" fill="#FF385C" />
        <path
          d="M14 34h36a2 2 0 0 1 2 2c0 7.5-6 13.5-14 15.4V53a2 2 0 0 1-2 2H28a2 2 0 0 1-2-2v-1.6C18 49.5 12 43.5 12 36a2 2 0 0 1 2-2Z"
          fill="#fff"
        />
        <path
          d="M26 12c0 3-3 4.5-3 8 0 2.4 1.3 4 2.4 5.2.5.6 1.4.2 1.4-.6 0-2.6 2.7-3.9 2.7-7.3 0-2.4-1.2-4-2.3-5.3-.4-.6-1.2-.7-1.2 0Z"
          fill="#fff"
          opacity=".85"
        />
        <path
          d="M38 12c0 3-3 4.5-3 8 0 2.4 1.3 4 2.4 5.2.5.6 1.4.2 1.4-.6 0-2.6 2.7-3.9 2.7-7.3 0-2.4-1.2-4-2.3-5.3-.4-.6-1.2-.7-1.2 0Z"
          fill="#fff"
          opacity=".85"
        />
      </svg>
      <span
        className={`font-display text-lg font-bold tracking-tight ${
          inverted ? 'text-white' : 'text-charcoal'
        }`}
      >
        Home Chef
      </span>
    </span>
  );
}
