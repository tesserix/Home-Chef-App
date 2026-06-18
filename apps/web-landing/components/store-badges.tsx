import { APP_STORE_URL, PLAY_STORE_URL } from '@/lib/site';

/**
 * App Store + Google Play badges, drawn inline so the page has zero
 * third-party asset dependencies.
 *
 * TODO(owner): when the apps publish, (1) swap APP_STORE_URL /
 * PLAY_STORE_URL in lib/site.ts for the real listing URLs and
 * (2) replace these drawings with the official badge artwork required
 * by Apple/Google marketing guidelines.
 */

interface StoreBadgesProps {
  /** Pixel height of each badge (width scales proportionally). */
  height?: number;
  className?: string;
}

export function StoreBadges({ height = 52, className }: StoreBadgesProps) {
  return (
    <div className={`flex flex-wrap items-center gap-3 ${className ?? ''}`}>
      <a
        href={APP_STORE_URL}
        aria-label="Download Fe3dr on the App Store"
        className="inline-flex rounded-lg transition-opacity duration-micro ease-state hover:opacity-80"
      >
        <AppStoreBadge height={height} />
      </a>
      <a
        href={PLAY_STORE_URL}
        aria-label="Get Fe3dr on Google Play"
        className="inline-flex rounded-lg transition-opacity duration-micro ease-state hover:opacity-80"
      >
        <GooglePlayBadge height={height} />
      </a>
    </div>
  );
}

function AppStoreBadge({ height }: { height: number }) {
  return (
    <svg
      width={height * 3}
      height={height}
      viewBox="0 0 120 40"
      role="img"
      aria-hidden="true"
      focusable="false"
    >
      <rect width="120" height="40" rx="8" fill="#000" />
      <rect
        x="0.5"
        y="0.5"
        width="119"
        height="39"
        rx="7.5"
        fill="none"
        stroke="#a6a6a6"
        strokeWidth="1"
      />
      {/* Apple mark */}
      <g transform="translate(11 8) scale(0.047)" fill="#fff">
        <path d="M318.7 268.7c-.2-36.7 16.4-64.4 50-84.8-18.8-26.9-47.2-41.7-84.7-44.6-35.5-2.9-74.3 20.7-88.5 20.7-15 0-49.4-19.7-76.4-19.7C63.3 141.2 4 184.8 4 273.5q0 39.3 14.4 81.2c12.8 36.7 59 126.7 107.2 125.2 25.2-.6 43-17.9 75.8-17.9 31.8 0 48.3 17.9 76.4 17.9 48.6-.7 90.4-82.5 102.6-119.3-65.2-30.7-61.7-90-61.7-91.9zm-56.6-164.2c27.3-32.4 24.8-61.9 24-72.5-24.1 1.4-52 16.4-67.9 34.9-17.5 19.8-27.8 44.3-25.6 71.9 26.1 2 49.9-11.4 69.5-34.3z" />
      </g>
      <text
        x="38"
        y="17"
        fill="#fff"
        fontFamily="-apple-system, 'Helvetica Neue', Arial, sans-serif"
        fontSize="8.5"
      >
        Download on the
      </text>
      <text
        x="38"
        y="31"
        fill="#fff"
        fontFamily="-apple-system, 'Helvetica Neue', Arial, sans-serif"
        fontSize="15"
        fontWeight="600"
      >
        App Store
      </text>
    </svg>
  );
}

function GooglePlayBadge({ height }: { height: number }) {
  return (
    <svg
      width={height * 3.375}
      height={height}
      viewBox="0 0 135 40"
      role="img"
      aria-hidden="true"
      focusable="false"
    >
      <rect width="135" height="40" rx="8" fill="#000" />
      <rect
        x="0.5"
        y="0.5"
        width="134"
        height="39"
        rx="7.5"
        fill="none"
        stroke="#a6a6a6"
        strokeWidth="1"
      />
      {/* Play mark */}
      <g transform="translate(9.5 8) scale(0.6)">
        <path
          d="M2.6 1.3C2 1.9 1.7 2.8 1.7 4v32c0 1.2.3 2.1.9 2.7l.1.1L20.6 20.9v-.4L2.7 1.2l-.1.1z"
          fill="#00c4ff"
        />
        <path
          d="M26.5 26.9 20.6 20.9v-.4l5.9-5.9.1.1 7.1 4c2 1.1 2 3 0 4.2l-7.1 4h-.1z"
          fill="#ffce00"
        />
        <path
          d="M26.6 26.8 20.6 20.7 2.6 38.7c.7.7 1.8.8 3 .1l21-12z"
          fill="#ff3a44"
        />
        <path
          d="M26.6 14.6 5.6 2.6c-1.2-.7-2.3-.6-3 .1l18 18 6-6.1z"
          fill="#00e97a"
        />
      </g>
      <text
        x="42"
        y="16"
        fill="#fff"
        fontFamily="Roboto, 'Helvetica Neue', Arial, sans-serif"
        fontSize="8"
        letterSpacing="0.5"
      >
        GET IT ON
      </text>
      <text
        x="42"
        y="31"
        fill="#fff"
        fontFamily="Roboto, 'Helvetica Neue', Arial, sans-serif"
        fontSize="15"
        fontWeight="600"
      >
        Google Play
      </text>
    </svg>
  );
}
