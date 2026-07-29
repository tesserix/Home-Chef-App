import { ImageResponse } from 'next/og';

/**
 * OpenGraph image — Next 16 file convention. Generated at build into a
 * 1200x630 PNG served on-domain as /opengraph-image, and automatically
 * wired into <meta property="og:image"> + <meta name="twitter:image"> by
 * the metadata API.
 *
 * Replaces the hotlinked Unsplash WebP (IMAGES.og) that link unfurlers —
 * LinkedIn in particular — render unreliably (WebP + third-party host).
 * This resolves the `site.ts` TODO to ship an owned, branded 1200x630 image.
 *
 * Brand: Paper background, Ink wordmark, single Persimmon accent — the
 * paper/ink/persimmon system from .impeccable.md.
 */

export const alt = 'Fe3dr — Real home-cooked food, delivered';
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

export default async function OpenGraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          width: '100%',
          height: '100%',
          background: '#FBFAF7',
          padding: '80px 96px',
          fontFamily: 'system-ui, sans-serif',
        }}
      >
        {/* Top eyebrow */}
        <div
          style={{
            display: 'flex',
            fontSize: 22,
            letterSpacing: '0.16em',
            textTransform: 'uppercase',
            color: '#C2410C',
            fontWeight: 600,
          }}
        >
          Fe3dr
        </div>

        {/* Main typographic moment */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            fontSize: 104,
            fontWeight: 700,
            color: '#1C1917',
            letterSpacing: '-0.035em',
            lineHeight: 1.0,
          }}
        >
          <span>Real home-cooked</span>
          <span>food, delivered.</span>
        </div>

        {/* Bottom: persimmon rule + tagline */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          <div style={{ width: '80px', height: '4px', background: '#C2410C' }} />
          <div style={{ display: 'flex', fontSize: 30, color: '#57534E' }}>
            From FSSAI-verified home chefs near you.
          </div>
        </div>
      </div>
    ),
    { ...size },
  );
}
