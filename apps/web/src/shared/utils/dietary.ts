// Dietary & allergen taxonomy + conflict matcher (#41) for the web app. Mirrors
// apps/api/services/dietary.go and packages/mobile-shared/src/dietary — keep in
// sync. (Web can't import the mobile-shared package, hence this small twin.)

export interface DietaryOption {
  value: string;
  label: string;
}

export const DIET_OPTIONS: DietaryOption[] = [
  { value: 'vegetarian', label: 'Vegetarian' },
  { value: 'vegan', label: 'Vegan' },
  { value: 'jain', label: 'Jain' },
  { value: 'eggetarian', label: 'Eggetarian' },
  { value: 'halal', label: 'Halal' },
  { value: 'kosher', label: 'Kosher' },
  { value: 'gluten-free', label: 'Gluten-Free' },
  { value: 'dairy-free', label: 'Dairy-Free' },
  { value: 'nut-free', label: 'Nut-Free' },
  { value: 'low-carb', label: 'Low-Carb' },
];

// Full major-allergen set (EU-14 / FDA big-9 union — basis of India's FSSAI rules).
export const ALLERGEN_OPTIONS: DietaryOption[] = [
  { value: 'gluten', label: 'Gluten (wheat, barley, rye)' },
  { value: 'peanuts', label: 'Peanuts' },
  { value: 'tree-nuts', label: 'Tree Nuts' },
  { value: 'dairy', label: 'Dairy (milk)' },
  { value: 'eggs', label: 'Eggs' },
  { value: 'soy', label: 'Soy' },
  { value: 'fish', label: 'Fish' },
  { value: 'shellfish', label: 'Shellfish (crustaceans)' },
  { value: 'molluscs', label: 'Molluscs' },
  { value: 'sesame', label: 'Sesame' },
  { value: 'mustard', label: 'Mustard' },
  { value: 'celery', label: 'Celery' },
  { value: 'lupin', label: 'Lupin' },
  { value: 'sulphites', label: 'Sulphites' },
];

const VEG_DIETS = new Set(['vegetarian', 'vegan', 'jain']);

export interface DietaryProfile {
  dietaryPreferences?: string[];
  foodAllergies?: string[];
}

export interface DietaryItem {
  dietaryTags?: string[];
  allergens?: string[];
  isVeg?: boolean | null;
}

export interface DietConflict {
  type: 'allergen' | 'diet';
  label: string;
  detail: string;
}

const norm = (s: string) => s.trim().toLowerCase();

function labelFor(token: string, options: DietaryOption[]): string {
  const t = norm(token);
  const found = options.find((o) => o.value === t);
  if (found) return found.label;
  if (!t) return token;
  return t.charAt(0).toUpperCase() + t.slice(1);
}

function itemIsNonVeg(item: DietaryItem): boolean {
  if (item.isVeg === true) return false;
  if (item.isVeg === false) return true;
  const tags = (item.dietaryTags ?? []).map(norm);
  return ['non-veg', 'non-vegetarian', 'nonveg', 'non veg'].some((t) => tags.includes(t));
}

/** Conservative conflict matcher — see services/dietary.go for the canonical rules. */
export function findItemConflicts(profile: DietaryProfile, item: DietaryItem): DietConflict[] {
  const conflicts: DietConflict[] = [];

  const avoid = new Set((profile.foodAllergies ?? []).map(norm).filter(Boolean));
  if (avoid.size > 0) {
    for (const a of item.allergens ?? []) {
      if (avoid.has(norm(a))) {
        const lbl = labelFor(a, ALLERGEN_OPTIONS);
        conflicts.push({ type: 'allergen', label: lbl, detail: `Contains ${lbl}, which you avoid` });
      }
    }
  }

  const prefs = (profile.dietaryPreferences ?? []).map(norm);
  const vegPref = prefs.find((d) => VEG_DIETS.has(d));
  if (vegPref && itemIsNonVeg(item)) {
    const lbl = labelFor(vegPref, DIET_OPTIONS);
    conflicts.push({ type: 'diet', label: lbl, detail: `Not ${lbl} — this dish is non-vegetarian` });
  }

  return conflicts;
}
