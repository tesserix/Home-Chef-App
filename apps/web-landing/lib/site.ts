/**
 * Single source of truth for site-wide constants.
 * Anything marked TODO needs a real value before the public cutover.
 */

export const SITE_URL = 'https://fe3dr.com';
export const SITE_NAME = 'Home Chef';

// TODO(owner): replace with the real App Store listing URL once the
// customer app is published (current value is a placeholder).
export const APP_STORE_URL = 'https://apps.apple.com/app/idTODO';

// TODO(owner): replace with the real Play Store listing URL once the
// customer app is published (current value is a placeholder).
export const PLAY_STORE_URL =
  'https://play.google.com/store/apps/details?id=com.homechef.customerTODO';

// TODO(owner): confirm the launch city shown across the page.
export const LAUNCH_CITY = 'Pune';

// TODO(owner): confirm contact + chef-recruitment addresses.
export const CONTACT_EMAIL = 'hello@fe3dr.com';
export const CHEFS_EMAIL = 'chefs@fe3dr.com';

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
