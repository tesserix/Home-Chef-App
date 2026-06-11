import type { NextConfig } from 'next';

/**
 * fe3dr.com marketing landing — pure static export (SSG).
 * `output: 'export'` emits `out/` with plain HTML/CSS/JS served by nginx.
 * `trailingSlash` makes every route a directory index (`/privacy/index.html`)
 * so nginx `try_files $uri $uri/` resolves without rewrite rules.
 */
const nextConfig: NextConfig = {
  output: 'export',
  trailingSlash: true,
  transpilePackages: ['@tesserix/web'],
  images: {
    // Static export has no image-optimization server; we serve plain <img>.
    unoptimized: true,
  },
};

export default nextConfig;
