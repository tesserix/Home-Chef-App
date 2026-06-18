/**
 * schema.org structured-data builders for the SEO pages (#58). Pure functions
 * returning plain objects; rendered by <JsonLd> (components/seo/json-ld.tsx).
 */
import { SITE_URL, SITE_NAME } from '@/lib/site';
import type { SeoChef } from '@/lib/seo-data';

export function chefUrl(slug: string): string {
  return `${SITE_URL}/chef/${slug}/`;
}

/** Restaurant + aggregateRating for a per-chef page. */
export function restaurantSchema(c: SeoChef) {
  const schema: Record<string, unknown> = {
    '@context': 'https://schema.org',
    '@type': 'Restaurant',
    '@id': `${chefUrl(c.slug)}#restaurant`,
    name: c.name,
    url: chefUrl(c.slug),
    servesCuisine: c.cuisines,
    priceRange: c.priceRange || '$$',
    image: c.image ? [c.image] : undefined,
    description: c.description || undefined,
  };
  if (c.city || c.state) {
    schema.address = {
      '@type': 'PostalAddress',
      addressLocality: c.city || undefined,
      addressRegion: c.state || undefined,
      addressCountry: 'IN',
    };
  }
  if (typeof c.lat === 'number' && typeof c.lng === 'number' && (c.lat !== 0 || c.lng !== 0)) {
    schema.geo = { '@type': 'GeoCoordinates', latitude: c.lat, longitude: c.lng };
  }
  if (c.reviews > 0 && c.rating > 0) {
    schema.aggregateRating = {
      '@type': 'AggregateRating',
      ratingValue: c.rating.toFixed(1),
      reviewCount: c.reviews,
      bestRating: '5',
    };
  }
  return schema;
}

/** ItemList of restaurants for a cuisine/area collection page. */
export function itemListSchema(chefs: SeoChef[]) {
  return {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    itemListElement: chefs.map((c, i) => ({
      '@type': 'ListItem',
      position: i + 1,
      url: chefUrl(c.slug),
      name: c.name,
    })),
  };
}

/** BreadcrumbList for any page. items: [{ name, path }] (path relative to SITE_URL). */
export function breadcrumbSchema(items: { name: string; path: string }[]) {
  return {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: items.map((it, i) => ({
      '@type': 'ListItem',
      position: i + 1,
      name: it.name,
      item: `${SITE_URL}${it.path}`,
    })),
  };
}

/** CollectionPage wrapper for cuisine/area/explore pages. */
export function collectionPageSchema(name: string, path: string, description: string) {
  return {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: `${name} · ${SITE_NAME}`,
    url: `${SITE_URL}${path}`,
    description,
  };
}
