import type { Metadata, Viewport } from 'next';
import { GeistSans } from 'geist/font/sans';
import { Inter } from 'next/font/google';
import {
  APP_STORE_URL,
  CONTACT_EMAIL,
  IMAGES,
  INSTAGRAM_URL,
  PLAY_STORE_URL,
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

const TITLE = 'Home Chef — Real home-cooked food, delivered';
const DESCRIPTION =
  'Order real home-cooked meals from FSSAI-verified home chefs near you. ' +
  'Browse local kitchens, order in a few taps, and track delivery live. ' +
  'Get the Home Chef app for iOS and Android.';

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
    images: [
      {
        url: IMAGES.og,
        width: 1200,
        height: 630,
        alt: 'A generous Indian home-cooked spread, ready to be delivered',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: TITLE,
    description: DESCRIPTION,
    images: [IMAGES.og],
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
      name: 'Home Chef',
      operatingSystem: 'iOS, Android',
      applicationCategory: 'FoodApplication',
      description: DESCRIPTION,
      author: { '@id': `${SITE_URL}/#organization` },
      installUrl: [APP_STORE_URL, PLAY_STORE_URL],
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
      </body>
    </html>
  );
}
