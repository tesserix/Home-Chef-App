/**
 * Single source of truth for site-wide constants.
 * Anything marked TODO needs a real value before the public cutover.
 */

export const SITE_URL = 'https://fe3dr.com';
export const SITE_NAME = 'Fe3dr';

/**
 * Paths owned by the customer SPA (apps/web), which is served from this same
 * origin — see the `web` entry in apps/auth-bff/homechef-products.yaml, whose
 * hosts are fe3dr.com/www.fe3dr.com and whose BFF is reached same-origin at
 * /bff. Kept as constants so the landing never hardcodes SPA routes inline.
 */
export const APP_LOGIN_PATH = '/login';
export const APP_REGISTER_PATH = '/register';

/* ── App store listings ─────────────────────────────────────────────────
 *
 * Neither app is published yet. Every download surface therefore renders a
 * "coming soon" state instead of a link, because a badge that 404s is worse
 * than one that tells the truth — and these URLs are also emitted as JSON-LD
 * `installUrl`, where a dead link gets indexed.
 *
 * The package ids below are the real, final ones. Going live is a two-field
 * edit per platform: set `status: 'live'` and fill in `url`.
 */

/** Final Play Store application ids — already set on the Expo builds. */
export const ANDROID_PACKAGES = {
  customer: 'com.tesserix.homechef.customer',
  vendor: 'com.tesserix.homechef.vendor',
} as const;

/** Play listing URL. Only resolves once the listing is public. */
export function playStoreUrl(packageName: string): string {
  return `https://play.google.com/store/apps/details?id=${packageName}`;
}

/** App Store listing URL, built from Apple's numeric app id. */
export function appStoreUrl(appleAppId: string): string {
  return `https://apps.apple.com/in/app/id${appleAppId}`;
}

export type StoreStatus = 'live' | 'coming-soon';

export interface StoreListing {
  status: StoreStatus;
  /** Public listing URL — `null` until `status` is `'live'`. */
  url: string | null;
}

export interface AppListing {
  /** Product name as it reads in badge and download copy. */
  name: string;
  ios: StoreListing;
  android: StoreListing;
}

export const CUSTOMER_APP: AppListing = {
  name: 'Fe3dr',
  // TODO(owner): on publish → { status: 'live', url: appStoreUrl('<apple numeric id>') }
  ios: { status: 'coming-soon', url: null },
  // TODO(owner): on publish → { status: 'live', url: playStoreUrl(ANDROID_PACKAGES.customer) }
  android: { status: 'coming-soon', url: null },
};

export const VENDOR_APP: AppListing = {
  name: 'Fe3dr for Chefs',
  // TODO(owner): on publish → { status: 'live', url: appStoreUrl('<apple numeric id>') }
  ios: { status: 'coming-soon', url: null },
  // TODO(owner): on publish → { status: 'live', url: playStoreUrl(ANDROID_PACKAGES.vendor) }
  android: { status: 'coming-soon', url: null },
};

/** Every live listing URL for an app — empty while it is still unpublished. */
export function liveStoreUrls(app: AppListing): string[] {
  return [app.ios, app.android]
    .filter((listing): listing is StoreListing & { url: string } => listing.url !== null)
    .map((listing) => listing.url);
}

/** Whether the app can be downloaded on at least one platform. */
export function isDownloadable(app: AppListing): boolean {
  return liveStoreUrls(app).length > 0;
}

// TODO(owner): confirm the launch city shown across the page.
export const LAUNCH_CITY = 'Pune';

// TODO(owner): confirm contact + chef-recruitment addresses.
export const CONTACT_EMAIL = 'hello@fe3dr.com';
export const CHEFS_EMAIL = 'chefs@fe3dr.com';

// Canonical legal-document values — kept identical across landing + both apps.
// LEGAL_SUPPORT_EMAIL is the general legal/support contact (distinct from the
// general CONTACT_EMAIL above). Do NOT repurpose CONTACT_EMAIL for legal copy.
export const LEGAL_SUPPORT_EMAIL = 'support@fe3dr.com';
export const LEGAL_OPERATOR = 'Tesserix Pty Ltd';
// Full operator identity — confirmed against sibling product mark8ly (same
// parent). Used on every operator mention across landing + both apps. Mirrors
// the mark8ly precedent of putting the ACN/ABN in user-facing copy.
export const LEGAL_OPERATOR_FULL =
  'Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia';
// India grievance-officer contact (DPDP §13). Distinct from the general
// LEGAL_SUPPORT_EMAIL; mark8ly exposes a dedicated dpo@ alongside general support.
// TODO(ops): provision dpo@fe3dr.com mailbox + name a resident Grievance Officer (DPDP §13)
export const LEGAL_GRIEVANCE_EMAIL = 'dpo@fe3dr.com';
export const LEGAL_LAST_UPDATED = '11 June 2026';

// TODO(owner): real social profiles (placeholders until accounts exist).
export const INSTAGRAM_URL = 'https://instagram.com/fe3dr';
export const X_URL = 'https://x.com/fe3dr';

/**
 * PLACEHOLDER PHOTOGRAPHY — curated Unsplash shots of authentic Indian
 * home cooking (every URL verified to return HTTP 200 and visually
 * checked for content). Swap each entry for owned photography before
 * launch; keep the same warm, candid, home-kitchen register.
 */
export const IMAGES = {
  /** Hero — a homestyle Indian spread, kadhais and warm hands. */
  heroMain: {
    src: 'https://images.unsplash.com/photo-1728910156510-77488f19b152?auto=format&fit=crop&w=1080&h=1350&q=80',
    alt: 'A generous Indian home-cooked spread — kadhais of curry, fresh naan and biryani being served by hand',
  },
  /** Hero floating card — masala dosa with chutneys. */
  heroDosa: {
    src: 'https://images.unsplash.com/photo-1668236543090-82eba5ee5976?auto=format&fit=crop&w=480&q=75',
    alt: 'A crisp masala dosa served with sambar and chutneys',
  },
  /** Hero floating card — puris frying on a home gas stove. */
  heroCooking: {
    src: 'https://images.unsplash.com/photo-1596450514659-4ff64fdde903?auto=format&fit=crop&w=480&q=75',
    alt: 'Fresh puris being fried in a kadhai on a home gas stove',
  },
  /** Why section — a curry simmering in a pan on a home stove. */
  whyKitchen: {
    src: 'https://images.unsplash.com/photo-1596797038530-2c107229654b?auto=format&fit=crop&w=987&q=80',
    alt: 'A coriander-topped curry simmering in a pan on a home stove',
  },
  /** Cook-with-us — a home cook at her own kitchen window, steam rising. */
  cookWithUs: {
    src: 'https://images.unsplash.com/photo-1528712306091-ed0763094c98?auto=format&fit=crop&w=987&q=80',
    alt: 'A home cook stirring a steaming pan by the window of her own kitchen',
  },
  /** OG image — wide crop of the hero spread. TODO(owner): replace with
      an owned, branded 1200×630 image before launch. */
  og: 'https://images.unsplash.com/photo-1728910156510-77488f19b152?auto=format&fit=crop&w=1200&h=630&q=80',
} as const;

/**
 * "What's cooking" showcase — illustrative dishes (not live data).
 * Photography verified Indian home-style; prices are placeholders.
 */
export const SHOWCASE_DISHES = [
  {
    name: 'Pav bhaji',
    chef: 'Asha',
    area: 'Kothrud',
    price: '₹140',
    veg: true,
    img: {
      src: 'https://images.unsplash.com/photo-1606491956689-2ea866880c84?auto=format&fit=crop&w=640&q=75',
      alt: 'Buttery pav bhaji with a basket of soft laadi pav',
    },
  },
  {
    name: 'Thalipeeth',
    chef: 'Manda',
    area: 'Sadashiv Peth',
    price: '₹90',
    veg: true,
    img: {
      src: 'https://images.unsplash.com/photo-1725483990188-41d4fb0d1e5a?auto=format&fit=crop&w=640&q=75',
      alt: 'Maharashtrian thalipeeth roasting on a tawa beside spice bowls',
    },
  },
  {
    name: 'Chicken biryani',
    chef: 'Rizwana',
    area: 'Camp',
    price: '₹220',
    veg: false,
    img: {
      src: 'https://images.unsplash.com/photo-1589302168068-964664d93dc0?auto=format&fit=crop&w=640&q=75',
      alt: 'Dum chicken biryani with mint, served with raita and salan',
    },
  },
  {
    name: 'Paneer butter masala',
    chef: 'Gurpreet',
    area: 'Aundh',
    price: '₹180',
    veg: true,
    img: {
      src: 'https://images.unsplash.com/photo-1631452180519-c014fe946bc7?auto=format&fit=crop&w=640&q=75',
      alt: 'Paneer butter masala in a copper kadhai with jeera rice and roti',
    },
  },
  {
    name: 'Mysore masala dosa',
    chef: 'Lakshmi',
    area: 'Baner',
    price: '₹110',
    veg: true,
    img: {
      src: 'https://images.unsplash.com/photo-1694849789325-914b71ab4075?auto=format&fit=crop&w=640&q=75',
      alt: 'Mysore masala dosa on a banana leaf with chutney and sambar',
    },
  },
  {
    name: 'Ghar ki thali',
    chef: 'Vandana',
    area: 'Hadapsar',
    price: '₹160',
    veg: true,
    img: {
      src: 'https://images.unsplash.com/photo-1680993032090-1ef7ea9b51e5?auto=format&fit=crop&w=640&q=75',
      alt: 'A steel thali with dal, sabzi, puri, rice and gulab jamun',
    },
  },
] as const;

/** Marquee strip — the food itself, Pune-first. */
export const MARQUEE_DISHES = [
  'Misal pav',
  'Sunday biryani',
  'Puran poli',
  'Ghar ki dal',
  'Thalipeeth',
  'Masala dosa',
  'Pav bhaji',
  'Sabudana khichdi',
  'Rajma chawal',
  'Modak',
] as const;
