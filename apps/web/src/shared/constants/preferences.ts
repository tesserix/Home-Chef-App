/**
 * Fallback preference options used while the API options are loading.
 * The real data lives in the `preference_options` database table and is
 * fetched via GET /api/v1/preferences.
 */

export interface PreferenceOption {
  value: string;
  label: string;
  description?: string;
}

export const DIETARY_OPTIONS: PreferenceOption[] = [
  { value: 'vegetarian', label: 'Vegetarian' },
  { value: 'vegan', label: 'Vegan' },
  { value: 'eggetarian', label: 'Eggetarian' },
  { value: 'non_vegetarian', label: 'Non-Vegetarian' },
  { value: 'pescatarian', label: 'Pescatarian' },
  { value: 'jain', label: 'Jain' },
  { value: 'gluten_free', label: 'Gluten Free' },
  { value: 'keto', label: 'Keto' },
  { value: 'low_carb', label: 'Low Carb' },
  { value: 'halal', label: 'Halal' },
];

export const ALLERGY_OPTIONS: PreferenceOption[] = [
  { value: 'peanuts', label: 'Peanuts' },
  { value: 'tree_nuts', label: 'Tree Nuts' },
  { value: 'milk', label: 'Milk / Dairy' },
  { value: 'eggs', label: 'Eggs' },
  { value: 'wheat', label: 'Wheat / Gluten' },
  { value: 'soy', label: 'Soy' },
  { value: 'fish', label: 'Fish' },
  { value: 'shellfish', label: 'Shellfish' },
  { value: 'sesame', label: 'Sesame' },
  { value: 'mustard', label: 'Mustard' },
];

export const CUISINE_OPTIONS: PreferenceOption[] = [
  { value: 'north_indian', label: 'North Indian' },
  { value: 'south_indian', label: 'South Indian' },
  { value: 'bengali', label: 'Bengali' },
  { value: 'gujarati', label: 'Gujarati' },
  { value: 'maharashtrian', label: 'Maharashtrian' },
  { value: 'rajasthani', label: 'Rajasthani' },
  { value: 'punjabi', label: 'Punjabi' },
  { value: 'kerala', label: 'Kerala' },
  { value: 'hyderabadi', label: 'Hyderabadi' },
  { value: 'mughlai', label: 'Mughlai' },
  { value: 'chinese', label: 'Indo-Chinese' },
  { value: 'continental', label: 'Continental' },
  { value: 'street_food', label: 'Street Food' },
  { value: 'desserts', label: 'Desserts & Sweets' },
];

export const SPICE_LEVELS: PreferenceOption[] = [
  { value: 'mild', label: 'Mild', description: 'No or very little spice' },
  { value: 'medium', label: 'Medium', description: 'Moderate spice' },
  { value: 'hot', label: 'Hot', description: 'Spicy food lover' },
  { value: 'extra_hot', label: 'Extra Hot', description: 'Bring the heat!' },
];

export const HOUSEHOLD_SIZES: PreferenceOption[] = [
  { value: '1', label: 'Just me' },
  { value: '2', label: '2 people' },
  { value: '3-4', label: '3-4 people' },
  { value: '5-6', label: '5-6 people' },
  { value: '7+', label: '7+ people' },
];

/** Map API category name to fallback array */
export const FALLBACK_OPTIONS: Record<string, PreferenceOption[]> = {
  dietary: DIETARY_OPTIONS,
  allergy: ALLERGY_OPTIONS,
  cuisine: CUISINE_OPTIONS,
  spice_level: SPICE_LEVELS,
  household_size: HOUSEHOLD_SIZES,
};
