import type { MetadataRoute } from 'next';
import { SITE_URL } from '@/lib/site';

export const dynamic = 'force-static';

export default function sitemap(): MetadataRoute.Sitemap {
  const lastModified = new Date();

  return [
    { url: `${SITE_URL}/`, lastModified, changeFrequency: 'weekly', priority: 1 },
    { url: `${SITE_URL}/privacy/`, lastModified, changeFrequency: 'monthly', priority: 0.3 },
    { url: `${SITE_URL}/terms/`, lastModified, changeFrequency: 'monthly', priority: 0.3 },
    { url: `${SITE_URL}/refund/`, lastModified, changeFrequency: 'monthly', priority: 0.3 },
  ];
}
