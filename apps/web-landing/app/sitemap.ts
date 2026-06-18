import type { MetadataRoute } from 'next';
import { SITE_URL } from '@/lib/site';
import { getChefIndex } from '@/lib/seo-data';

export const dynamic = 'force-static';

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const lastModified = new Date();

  const base: MetadataRoute.Sitemap = [
    { url: `${SITE_URL}/`, lastModified, changeFrequency: 'weekly', priority: 1 },
    { url: `${SITE_URL}/explore/`, lastModified, changeFrequency: 'daily', priority: 0.8 },
    { url: `${SITE_URL}/privacy/`, lastModified, changeFrequency: 'monthly', priority: 0.3 },
    { url: `${SITE_URL}/terms/`, lastModified, changeFrequency: 'monthly', priority: 0.3 },
    { url: `${SITE_URL}/refund/`, lastModified, changeFrequency: 'monthly', priority: 0.3 },
  ];

  // SEO chef/cuisine/area pages (#58) — from the same build-time index.
  const { chefs, cuisines, areas } = await getChefIndex();
  const seo: MetadataRoute.Sitemap = [
    ...chefs.map((c) => ({ url: `${SITE_URL}/chef/${c.slug}/`, lastModified, changeFrequency: 'weekly' as const, priority: 0.7 })),
    ...cuisines.map((c) => ({ url: `${SITE_URL}/cuisine/${c.slug}/`, lastModified, changeFrequency: 'weekly' as const, priority: 0.6 })),
    ...areas.map((a) => ({ url: `${SITE_URL}/area/${a.slug}/`, lastModified, changeFrequency: 'weekly' as const, priority: 0.6 })),
  ];

  return [...base, ...seo];
}
