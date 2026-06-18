/**
 * Build-time SEO data layer (#58). web-landing is a static export, so this runs
 * at `next build`: it fetches the PUBLIC chef API once, derives slug/cuisine/area
 * groupings, and feeds generateStaticParams + the SEO pages + the sitemap.
 *
 * The build needs reachable chef data — `output: export` cannot emit a dynamic
 * route with zero params. Each request retries once to ride out a transient blip;
 * a sustained outage fails the build, which simply leaves the last good image
 * serving (nginx keeps the previous deploy). Refresh cadence is a periodic rebuild.
 */

const API_BASE = process.env.SEO_API_URL ?? 'https://api.fe3dr.com/api/v1';

export interface SeoChef {
  id: string;
  slug: string;
  name: string;
  description: string;
  cuisines: string[];
  city: string;
  state: string;
  lat?: number;
  lng?: number;
  rating: number;
  reviews: number;
  image?: string;
  priceRange?: string;
  verified: boolean;
  foodSafety: boolean;
}

export interface SeoGroup {
  slug: string;
  name: string;
  chefs: SeoChef[];
}

export interface SeoIndex {
  chefs: SeoChef[];
  bySlug: Map<string, SeoChef>;
  cuisines: SeoGroup[];
  areas: SeoGroup[];
}

interface ApiChef {
  id: string;
  slug?: string;
  businessName?: string;
  description?: string;
  cuisines?: string[];
  city?: string;
  state?: string;
  latitude?: number;
  longitude?: number;
  rating?: number;
  totalReviews?: number;
  profileImage?: string;
  bannerImage?: string;
  priceRange?: string;
  verified?: boolean;
  foodSafetyBadge?: boolean;
}

export function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/['’]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

function mapChef(c: ApiChef): SeoChef {
  const name = c.businessName ?? '';
  return {
    id: c.id,
    slug: c.slug || slugify(name),
    name,
    description: c.description ?? '',
    cuisines: (c.cuisines ?? []).filter(Boolean),
    city: c.city ?? '',
    state: c.state ?? '',
    lat: c.latitude,
    lng: c.longitude,
    rating: c.rating ?? 0,
    reviews: c.totalReviews ?? 0,
    image: c.bannerImage || c.profileImage || undefined,
    priceRange: c.priceRange,
    verified: Boolean(c.verified),
    foodSafety: Boolean(c.foodSafetyBadge),
  };
}

function indexChefs(chefs: SeoChef[]): SeoIndex {
  const bySlug = new Map<string, SeoChef>();
  for (const c of chefs) if (c.slug && !bySlug.has(c.slug)) bySlug.set(c.slug, c);

  const group = (key: (c: SeoChef) => { slug: string; name: string } | null) => {
    const m = new Map<string, SeoGroup>();
    for (const c of chefs) {
      const k = key(c);
      if (!k?.slug) continue;
      if (!m.has(k.slug)) m.set(k.slug, { slug: k.slug, name: k.name, chefs: [] });
      m.get(k.slug)!.chefs.push(c);
    }
    return [...m.values()].sort((a, b) => b.chefs.length - a.chefs.length);
  };

  const cuisines: SeoGroup[] = [];
  const cuisineMap = new Map<string, SeoGroup>();
  for (const c of chefs) {
    for (const cu of c.cuisines) {
      const slug = slugify(cu);
      if (!slug) continue;
      if (!cuisineMap.has(slug)) cuisineMap.set(slug, { slug, name: cu, chefs: [] });
      cuisineMap.get(slug)!.chefs.push(c);
    }
  }
  cuisines.push(...[...cuisineMap.values()].sort((a, b) => b.chefs.length - a.chefs.length));

  const areas = group((c) => (c.city ? { slug: slugify(c.city), name: c.city } : null));

  return { chefs, bySlug, cuisines, areas };
}

async function fetchChefsPage(page: number): Promise<Response> {
  const url = `${API_BASE}/chefs?limit=100&page=${page}&sort=rating`;
  try {
    return await fetch(url, { cache: 'force-cache' });
  } catch {
    // Retry once to ride out a transient build-time network blip.
    return fetch(url, { cache: 'no-store' });
  }
}

async function buildIndex(): Promise<SeoIndex> {
  try {
    const chefs: SeoChef[] = [];
    for (let page = 1; page <= 50; page++) {
      const res = await fetchChefsPage(page);
      if (!res.ok) break;
      const body = (await res.json()) as { data?: ApiChef[]; pagination?: { hasNext?: boolean } };
      for (const c of body.data ?? []) {
        if (c.businessName && c.id) chefs.push(mapChef(c));
      }
      if (!body.pagination?.hasNext) break;
    }
    return indexChefs(chefs);
  } catch {
    // API down at build → emit only the static pages.
    return indexChefs([]);
  }
}

let cached: Promise<SeoIndex> | null = null;

/** Memoized so every page + the sitemap share ONE fetch pass per build. */
export function getChefIndex(): Promise<SeoIndex> {
  if (!cached) cached = buildIndex();
  return cached;
}

/** Short, unique-ish meta description for a chef. */
export function chefMetaDescription(c: SeoChef): string {
  const where = c.city ? ` in ${c.city}` : '';
  const food = c.cuisines.length ? `${c.cuisines.slice(0, 3).join(', ')} ` : '';
  return `Order ${food}home-cooked meals from ${c.name}${where} on Fe3dr. ${
    c.reviews > 0 ? `Rated ${c.rating.toFixed(1)}★ by ${c.reviews} diners. ` : ''
  }Get the app to browse the menu and order.`;
}
