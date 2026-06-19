// Dietary & allergen taxonomy + conflict matcher (#41), shared by the customer
// and vendor apps. Mirrors apps/api/services/dietary.go — keep the two in sync.
// The API is authoritative (GET /dietary/options); these constants drive the
// chips offline and the matcher is reused for badges + checkout warnings.

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

// Full set of major declarable allergens (EU-14 / FDA "big 9" union — the basis
// of India's FSSAI labelling rules). Mirrors AllergenOptions in services/dietary.go.
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

// Diets that an explicitly non-veg dish violates.
const VEG_DIETS = new Set(['vegetarian', 'vegan', 'jain']);

export interface DietaryProfile {
  dietaryPreferences?: string[];
  foodAllergies?: string[];
}

/** The minimal item shape the matcher needs (menu item or weekly-menu cell). */
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

/**
 * Returns the conflicts between a customer's dietary profile and a dish.
 * Conservative — only high-confidence clashes (an avoided allergen is present,
 * or a veg-diet customer + an explicitly non-veg dish) so warnings stay
 * trustworthy. Mirrors services/dietary.go DietaryConflicts.
 */
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

/** True when the profile expresses any preference worth checking against. */
export function hasDietaryProfile(profile: DietaryProfile): boolean {
  return Boolean(profile.dietaryPreferences?.length || profile.foodAllergies?.length);
}
