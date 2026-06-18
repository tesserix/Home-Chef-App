import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { SiteNav } from '@/components/site-nav';
import { SiteFooter } from '@/components/site-footer';
import { JsonLd } from '@/components/seo/json-ld';
import { AppCta } from '@/components/seo/app-cta';
import { Breadcrumbs } from '@/components/seo/breadcrumbs';
import { getChefIndex, chefMetaDescription } from '@/lib/seo-data';
import { restaurantSchema, breadcrumbSchema, chefUrl } from '@/lib/jsonld';
import { SITE_NAME } from '@/lib/site';

export const dynamicParams = false;

export async function generateStaticParams() {
  const { chefs } = await getChefIndex();
  return chefs.map((c) => ({ slug: c.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const chef = (await getChefIndex()).bySlug.get(slug);
  if (!chef) return {};
  const description = chefMetaDescription(chef);
  return {
    title: `${chef.name}${chef.city ? ` · ${chef.city}` : ''}`,
    description,
    alternates: { canonical: `/chef/${slug}/` },
    openGraph: {
      url: chefUrl(slug),
      title: `${chef.name} · ${SITE_NAME}`,
      description,
      images: chef.image ? [{ url: chef.image }] : undefined,
    },
  };
}

export default async function ChefPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const chef = (await getChefIndex()).bySlug.get(slug);
  if (!chef) notFound();

  const crumbs = [
    { name: 'Home', path: '/' },
    { name: 'Explore', path: '/explore/' },
    { name: chef.name, path: `/chef/${slug}/` },
  ];

  return (
    <>
      <SiteNav />
      <main id="main" className="mx-auto max-w-3xl px-4 py-10">
        <JsonLd data={restaurantSchema(chef)} />
        <JsonLd data={breadcrumbSchema(crumbs)} />
        <Breadcrumbs items={crumbs} />

        {chef.image ? (
          <div className="mt-4 aspect-[16/9] w-full overflow-hidden rounded-3xl bg-neutral-100">
            <img src={chef.image} alt="" className="h-full w-full object-cover" />
          </div>
        ) : null}

        <header className="mt-6">
          <h1 className="text-3xl font-semibold text-neutral-900 sm:text-4xl">{chef.name}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-neutral-600">
            {chef.reviews > 0 ? (
              <span className="tabular-nums">★ {chef.rating.toFixed(1)} ({chef.reviews})</span>
            ) : null}
            {chef.city ? <span>{[chef.city, chef.state].filter(Boolean).join(', ')}</span> : null}
            {chef.foodSafety ? <span className="text-green-700">✓ Food-safety verified</span> : null}
            {chef.priceRange ? <span>{chef.priceRange}</span> : null}
          </div>
        </header>

        {chef.cuisines.length ? (
          <div className="mt-4 flex flex-wrap gap-2">
            {chef.cuisines.map((cu) => (
              <a
                key={cu}
                href={`/cuisine/${cu.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')}/`}
                className="rounded-full bg-neutral-100 px-3 py-1 text-sm text-neutral-700 hover:bg-neutral-200"
              >
                {cu}
              </a>
            ))}
          </div>
        ) : null}

        {chef.description ? (
          <p className="mt-6 text-pretty leading-relaxed text-neutral-700">{chef.description}</p>
        ) : null}

        <div className="mt-10">
          <AppCta
            title={`Order from ${chef.name} on Fe3dr`}
            subtitle="Browse the full menu, see live availability, and order — all in the app."
            chefId={chef.id}
          />
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
