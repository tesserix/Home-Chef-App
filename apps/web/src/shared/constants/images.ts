/**
 * Placeholder images from Unsplash for Fe3dr app
 * Replace these with actual CDN URLs in production
 */

// Logo placeholder - Replace with actual logo
export const LOGO = {
  light: '/logo-light.svg',
  dark: '/logo-dark.svg',
  icon: '/logo-icon.svg',
};

// Hero banner images
export const HERO_IMAGES = {
  home: 'https://images.unsplash.com/photo-1556909114-f6e7ad7d3136?w=1920&q=80', // Chef cooking
  chefs: 'https://images.unsplash.com/photo-1577219491135-ce391730fb2c?w=1920&q=80', // Professional chef
  catering: 'https://images.unsplash.com/photo-1555244162-803834f70033?w=1920&q=80', // Catering spread
  feed: 'https://images.unsplash.com/photo-1504674900247-0877df9cc836?w=1920&q=80', // Beautiful food
};

// Chef profile placeholder images
export const CHEF_PLACEHOLDERS = [
  'https://images.unsplash.com/photo-1583394293214-28ez9e5a2b8d?w=400&q=80', // Chef portrait 1
  'https://images.unsplash.com/photo-1581349485608-9469926a8e5e?w=400&q=80', // Chef portrait 2
  'https://images.unsplash.com/photo-1595273670150-bd0c3c392e46?w=400&q=80', // Chef portrait 3
  'https://images.unsplash.com/photo-1607631568010-a87245c0dbd8?w=400&q=80', // Chef portrait 4
];

// Food category images
export const FOOD_CATEGORIES = {
  indian: 'https://images.unsplash.com/photo-1585937421612-70a008356fbe?w=600&q=80',
  italian: 'https://images.unsplash.com/photo-1498579150354-977475b7ea0b?w=600&q=80',
  chinese: 'https://images.unsplash.com/photo-1563245372-f21724e3856d?w=600&q=80',
  mexican: 'https://images.unsplash.com/photo-1565299585323-38d6b0865b47?w=600&q=80',
  thai: 'https://images.unsplash.com/photo-1559314809-0d155014e29e?w=600&q=80',
  japanese: 'https://images.unsplash.com/photo-1580822184713-fc5400e7fe10?w=600&q=80',
  mediterranean: 'https://images.unsplash.com/photo-1544025162-d76694265947?w=600&q=80',
  american: 'https://images.unsplash.com/photo-1550547660-d9450f859349?w=600&q=80',
  desserts: 'https://images.unsplash.com/photo-1563729784474-d77dbb933a9e?w=600&q=80',
  healthy: 'https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=600&q=80',
  vegan: 'https://images.unsplash.com/photo-1540914124281-342587941389?w=600&q=80',
  comfort: 'https://images.unsplash.com/photo-1476224203421-9ac39bcb3327?w=600&q=80',
};

// Menu item placeholder images
export const FOOD_PLACEHOLDERS = [
  'https://images.unsplash.com/photo-1546069901-ba9599a7e63c?w=400&q=80', // Healthy bowl
  'https://images.unsplash.com/photo-1567620905732-2d1ec7ab7445?w=400&q=80', // Pancakes
  'https://images.unsplash.com/photo-1565299624946-b28f40a0ae38?w=400&q=80', // Pizza
  'https://images.unsplash.com/photo-1540189549336-e6e99c3679fe?w=400&q=80', // Salad
  'https://images.unsplash.com/photo-1482049016gy584d96f05?w=400&q=80', // Curry
  'https://images.unsplash.com/photo-1555939594-58d7cb561ad1?w=400&q=80', // BBQ
  'https://images.unsplash.com/photo-1606787366850-de6330128bfc?w=400&q=80', // Mixed dishes
  'https://images.unsplash.com/photo-1547592180-85f173990554?w=400&q=80', // Cooking
];

// Background patterns and textures
export const BACKGROUNDS = {
  woodTexture: 'https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=1920&q=80',
  marbleTexture: 'https://images.unsplash.com/photo-1544966503-7cc5ac882d5a?w=1920&q=80',
  kitchenAmbient: 'https://images.unsplash.com/photo-1556909114-f6e7ad7d3136?w=1920&q=80',
  tableSetting: 'https://images.unsplash.com/photo-1414235077428-338989a2e8c0?w=1920&q=80',
};

// Review/testimonial avatars
export const AVATAR_PLACEHOLDERS = [
  'https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=100&q=80',
  'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=100&q=80',
  'https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=100&q=80',
  'https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=100&q=80',
  'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=100&q=80',
];

// Icons and decorative elements
export const DECORATIVE = {
  spices: 'https://images.unsplash.com/photo-1596040033229-a9821ebd058d?w=400&q=80',
  herbs: 'https://images.unsplash.com/photo-1466637574441-749b8f19452f?w=400&q=80',
  ingredients: 'https://images.unsplash.com/photo-1490818387583-1baba5e638af?w=400&q=80',
  utensils: 'https://images.unsplash.com/photo-1556909172-8c2f041fca1e?w=400&q=80',
};

// App Store badges (placeholders)
export const STORE_BADGES = {
  appStore: '/badges/app-store.svg',
  playStore: '/badges/play-store.svg',
};

// Social media icons (use actual brand assets in production)
export const SOCIAL = {
  instagram: '/social/instagram.svg',
  facebook: '/social/facebook.svg',
  twitter: '/social/twitter.svg',
  tiktok: '/social/tiktok.svg',
};

// Helper function to get random food image
export function getRandomFoodImage(): string {
  const index = Math.floor(Math.random() * FOOD_PLACEHOLDERS.length);
  return FOOD_PLACEHOLDERS[index]!;
}

// Helper function to get random chef image
export function getRandomChefImage(): string {
  const index = Math.floor(Math.random() * CHEF_PLACEHOLDERS.length);
  return CHEF_PLACEHOLDERS[index]!;
}

// Helper function to get random avatar
export function getRandomAvatar(): string {
  const index = Math.floor(Math.random() * AVATAR_PLACEHOLDERS.length);
  return AVATAR_PLACEHOLDERS[index]!;
}
