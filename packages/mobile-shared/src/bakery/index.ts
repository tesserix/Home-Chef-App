// Bakery vertical (#1065) — shared types plus the pricing rules the cake
// configurator runs locally. Mirrors apps/api/models/bakery.go and
// apps/api/services/bakery.go; keep the three in sync, because a mismatch shows
// the customer one price and charges another.

export type BakeryProductType =
  | 'cake'
  | 'cupcake'
  | 'pastry'
  | 'bread'
  | 'cookie'
  | 'dessert_box'
  | 'hamper'
  | 'savoury';

export const BAKERY_PRODUCT_TYPES: { value: BakeryProductType; label: string }[] = [
  { value: 'cake', label: 'Cakes' },
  { value: 'cupcake', label: 'Cupcakes' },
  { value: 'pastry', label: 'Pastries' },
  { value: 'bread', label: 'Breads' },
  { value: 'cookie', label: 'Cookies' },
  { value: 'dessert_box', label: 'Dessert boxes' },
  { value: 'hamper', label: 'Hampers' },
  { value: 'savoury', label: 'Savouries' },
];

export type BakeryOptionKind =
  | 'shape'
  | 'tier'
  | 'flavour'
  | 'sponge'
  | 'frosting'
  | 'egg'
  | 'sweetness';

// Canonical render order — the configurator and the vendor editor both follow it.
export const BAKERY_OPTION_KINDS: BakeryOptionKind[] = [
  'shape',
  'tier',
  'flavour',
  'sponge',
  'frosting',
  'egg',
  'sweetness',
];

export const BAKERY_OPTION_KIND_LABELS: Record<BakeryOptionKind, string> = {
  shape: 'Shape',
  tier: 'Tiers',
  flavour: 'Flavour',
  sponge: 'Sponge',
  frosting: 'Frosting',
  egg: 'Egg preference',
  sweetness: 'Sweetness',
};

export type BakeryPriceMode = 'flat' | 'per_kg';

export const BAKERY_OCCASIONS: { value: string; label: string }[] = [
  { value: 'birthday', label: 'Birthday' },
  { value: 'anniversary', label: 'Anniversary' },
  { value: 'wedding', label: 'Wedding' },
  { value: 'baby-shower', label: 'Baby Shower' },
  { value: 'engagement', label: 'Engagement' },
  { value: 'house-party', label: 'House Party' },
  { value: 'corporate', label: 'Corporate' },
  { value: 'festival', label: 'Festival' },
  { value: 'farewell', label: 'Farewell' },
  { value: 'graduation', label: 'Graduation' },
];

export interface BakeryOption {
  id: string;
  kind: BakeryOptionKind;
  name: string;
  priceDelta: number;
  priceMode: BakeryPriceMode;
  dietaryTags?: string[];
  allergens?: string[];
  imageUrl?: string;
  isAvailable: boolean;
  isDefault: boolean;
  sortOrder: number;
}

export interface BakerySpec {
  id: string;
  menuItemId: string;
  productType: BakeryProductType;
  pricePerKg: number;
  minWeightKg: number;
  maxWeightKg: number;
  weightStepKg: number;
  servesPerKg: number;
  allowMessage: boolean;
  maxMessageChars: number;
  allowReferencePhoto: boolean;
  leadTimeHours: number;
  occasions?: string[];
  options: BakeryOption[];
  weightChoices?: number[];
}

export interface BakeryLineInput {
  weightKg?: number;
  optionIds?: string[];
  messageOnCake?: string;
  referencePhotoUrl?: string;
  occasion?: string;
}

export interface BakerySelection {
  kind: BakeryOptionKind;
  label: string;
  name: string;
  priceDelta: number;
}

export interface OrderItemBakery {
  productType: BakeryProductType;
  weightKg: number;
  serves: number;
  basePrice: number;
  selections: BakerySelection[];
  messageOnCake?: string;
  referencePhotoUrl?: string;
  occasion?: string;
  dietaryTags?: string[];
  allergens?: string[];
}

// Absorbs float noise — 1.5 arrives as 1.5000000000000002 often enough to matter.
const TOLERANCE = 0.001;

const round2 = (n: number): number => Math.round(n * 100) / 100;

// 0.50 renders as "0.5", 5.00 as "5".
const trimNum = (n: number): string => String(round2(n));

export function weightChoices(spec: Pick<BakerySpec, 'pricePerKg' | 'minWeightKg' | 'maxWeightKg' | 'weightStepKg'>): number[] {
  if (spec.pricePerKg <= 0) return [];
  const min = spec.minWeightKg > 0 ? spec.minWeightKg : 0.5;
  const step = spec.weightStepKg > 0 ? spec.weightStepKg : 0.5;
  const max = spec.maxWeightKg > 0 && spec.maxWeightKg >= min ? spec.maxWeightKg : min;
  const out: number[] = [];
  for (let w = min; w <= max + TOLERANCE && out.length < 40; w += step) out.push(round2(w));
  return out;
}

export function bakeryServes(spec: Pick<BakerySpec, 'servesPerKg'>, weightKg: number): number {
  if (weightKg <= 0) return 0;
  const perKg = spec.servesPerKg > 0 ? spec.servesPerKg : 8;
  return Math.round(weightKg * perKg);
}

// The soonest slot a product needing this much notice can be fulfilled.
export function earliestBakeryFulfillment(now: Date, leadTimeHours: number): Date {
  if (leadTimeHours <= 0) return now;
  return new Date(now.getTime() + leadTimeHours * 3600_000);
}

export interface PricedBakeryLine {
  price: number;
  snapshot: OrderItemBakery;
}

// Validates the configuration and returns the per-unit price plus the snapshot
// the server will store. Throws with a customer-facing message.
export function priceBakeryLine(spec: BakerySpec, basePrice: number, input: BakeryLineInput): PricedBakeryLine {
  const { weight, base } = resolveBasePrice(spec, basePrice, input.weightKg ?? 0);
  const { selections, dietaryTags, allergens, delta } = resolveOptions(spec, weight, input.optionIds ?? []);

  const message = (input.messageOnCake ?? '').trim();
  if (message) {
    if (!spec.allowMessage) throw new Error("This baker doesn't offer a message on this item");
    const max = spec.maxMessageChars > 0 ? spec.maxMessageChars : 40;
    if ([...message].length > max) throw new Error(`Keep the message to ${max} characters or fewer`);
  }

  const photo = (input.referencePhotoUrl ?? '').trim();
  if (photo && !spec.allowReferencePhoto) {
    throw new Error("This baker doesn't accept a reference photo on this item");
  }

  return {
    price: round2(base + delta),
    snapshot: {
      productType: spec.productType,
      weightKg: weight,
      serves: bakeryServes(spec, weight),
      basePrice: round2(base),
      selections,
      messageOnCake: message || undefined,
      referencePhotoUrl: photo || undefined,
      occasion: (input.occasion ?? '').trim() || undefined,
      dietaryTags,
      allergens,
    },
  };
}

function resolveBasePrice(spec: BakerySpec, flatPrice: number, weightKg: number): { weight: number; base: number } {
  if (spec.pricePerKg <= 0) return { weight: 0, base: flatPrice };
  if (weightKg <= 0) throw new Error('Choose a size for this item');

  const min = spec.minWeightKg > 0 ? spec.minWeightKg : 0.5;
  const max = spec.maxWeightKg > 0 ? spec.maxWeightKg : 10;
  if (weightKg < min - TOLERANCE || weightKg > max + TOLERANCE) {
    throw new Error(`Size must be between ${trimNum(min)} kg and ${trimNum(max)} kg`);
  }
  if (spec.weightStepKg > 0) {
    const steps = (weightKg - min) / spec.weightStepKg;
    if (Math.abs(steps - Math.round(steps)) > TOLERANCE) {
      throw new Error(`Size is sold in ${trimNum(spec.weightStepKg)} kg steps`);
    }
  }
  return { weight: weightKg, base: spec.pricePerKg * weightKg };
}

// One choice per offered kind, and every kind with something orderable in it is
// required — a cake with no flavour picked is not something a baker can start.
function resolveOptions(
  spec: BakerySpec,
  weight: number,
  selectedIds: string[],
): { selections: BakerySelection[]; dietaryTags: string[]; allergens: string[]; delta: number } {
  const selected = new Set(selectedIds);
  const matched = new Set<string>();
  const selections: BakerySelection[] = [];
  const tags = new Set<string>();
  const allergens = new Set<string>();
  let delta = 0;

  for (const kind of BAKERY_OPTION_KINDS) {
    const options = spec.options.filter((o) => o.kind === kind);
    if (options.length === 0) continue;
    const label = BAKERY_OPTION_KIND_LABELS[kind];

    const chosen = options.filter((o) => selected.has(o.id));
    for (const o of chosen) {
      if (!o.isAvailable) throw new Error(`${o.name} is no longer available`);
      matched.add(o.id);
    }

    if (chosen.length === 0) {
      if (!options.some((o) => o.isAvailable)) continue;
      throw new Error(`Please choose an option for "${label}"`);
    }
    if (chosen.length > 1) throw new Error(`Pick just one ${label}`);

    const o = chosen[0]!;
    const d = o.priceMode === 'per_kg' && weight > 0 ? o.priceDelta * weight : o.priceDelta;
    delta += d;
    selections.push({ kind, label, name: o.name, priceDelta: round2(d) });
    for (const t of o.dietaryTags ?? []) tags.add(t);
    for (const a of o.allergens ?? []) allergens.add(a);
  }

  for (const id of selectedIds) {
    if (!matched.has(id)) throw new Error('Invalid option selection');
  }

  return {
    selections,
    dietaryTags: [...tags].sort(),
    allergens: [...allergens].sort(),
    delta,
  };
}

// The one line every surface prints — cart, kitchen docket, invoice, receipt.
export function bakerySummary(b: OrderItemBakery | null | undefined): string {
  if (!b) return '';
  const parts: string[] = [];
  if (b.weightKg > 0) parts.push(`${trimNum(b.weightKg)} kg`);
  for (const s of b.selections ?? []) parts.push(s.name);
  if (b.messageOnCake) parts.push(`“${b.messageOnCake}”`);
  return parts.join(' · ');
}

export function productTypeLabel(t: BakeryProductType | string): string {
  return BAKERY_PRODUCT_TYPES.find((p) => p.value === t)?.label ?? String(t);
}

export function occasionLabel(o: string): string {
  return BAKERY_OCCASIONS.find((x) => x.value === o)?.label ?? o;
}
