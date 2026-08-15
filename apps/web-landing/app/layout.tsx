import type { Metadata, Viewport } from 'next';
import { Analytics } from './analytics';
import { GeistSans } from 'geist/font/sans';
import { Inter } from 'next/font/google';
import {
  CONTACT_EMAIL,
  CUSTOMER_APP,
  INSTAGRAM_URL,
  liveStoreUrls,
  SITE_NAME,
  SITE_URL,
  X_URL,
} from '@/lib/site';
import './globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const TITLE = 'Fe3dr — Real home-cooked food from kitchens near you';
const DESCRIPTION =
  'Order real home-cooked meals from FSSAI-verified home chefs near you. ' +
  'Browse local kitchens, order in a few taps, and track your order live. ' +
  'Collect it yourself or have your chef bring it. ' +
  'Get the Fe3dr app for iOS and Android.';

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: TITLE,
    template: `%s · ${SITE_NAME}`,
  },
  description: DESCRIPTION,
  alternates: {
    canonical: '/',
  },
  openGraph: {
    type: 'website',
    url: SITE_URL,
    siteName: SITE_NAME,
    title: TITLE,
    description: DESCRIPTION,
    // og:image is supplied by app/opengraph-image.tsx (Next file convention) —
    // an on-domain generated PNG. Do not re-add an explicit `images` here: the
    // previous hotlinked Unsplash WebP unfurled unreliably (WebP + third-party
    // host), which is why LinkedIn rejected the link.
  },
  twitter: {
    card: 'summary_large_image',
    title: TITLE,
    description: DESCRIPTION,
    // twitter:image also comes from app/opengraph-image.tsx.
  },
  applicationName: SITE_NAME,
  keywords: [
    'home-cooked food delivery',
    'home chef',
    'homemade food',
    'local home cooks',
    'FSSAI verified kitchens',
    'fe3dr',
  ],
};

export const viewport: Viewport = {
  themeColor: '#ffffff',
  width: 'device-width',
  initialScale: 1,
};

/** JSON-LD: Organization + the customer mobile app. */
const structuredData = {
  '@context': 'https://schema.org',
  '@graph': [
    {
      '@type': 'Organization',
      '@id': `${SITE_URL}/#organization`,
      name: SITE_NAME,
      url: SITE_URL,
      email: CONTACT_EMAIL,
      logo: `${SITE_URL}/icon.svg`,
      sameAs: [INSTAGRAM_URL, X_URL],
    },
    {
      '@type': 'MobileApplication',
      name: 'Fe3dr',
      operatingSystem: 'iOS, Android',
      applicationCategory: 'FoodApplication',
      description: DESCRIPTION,
      author: { '@id': `${SITE_URL}/#organization` },
      // Only advertise store listings that actually resolve — an indexed
      // `installUrl` pointing at an unpublished listing is a crawlable 404.
      // The key drops out entirely until the first platform goes live.
      ...(liveStoreUrls(CUSTOMER_APP).length > 0
        ? { installUrl: liveStoreUrls(CUSTOMER_APP) }
        : {}),
      offers: {
        '@type': 'Offer',
        price: '0',
        priceCurrency: 'INR',
      },
    },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${GeistSans.variable} ${inter.variable}`}>
      <body>
        <a href="#main" className="skip-link">
          Skip to main content
        </a>
        {children}
        <script
          type="application/ld+json"
          // Static, build-time JSON — no user input flows through here.
          dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
        />
        <Analytics />
      </body>
    </html>
  );
}
