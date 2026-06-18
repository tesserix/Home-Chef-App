import type { Metadata } from 'next';
import Link from 'next/link';
import { SiteNav } from '@/components/site-nav';
import { SiteFooter } from '@/components/site-footer';
import { JsonLd } from '@/components/seo/json-ld';
import { ChefCard } from '@/components/seo/chef-card';
import { AppCta } from '@/components/seo/app-cta';
import { getChefIndex } from '@/lib/seo-data';
import { collectionPageSchema } from '@/lib/jsonld';

export const dynamic = 'force-static';

const DESCRIPTION =
  'Explore home chefs by cuisine and area on Fe3dr — real home-cooked meals from local cooks, ordered from your phone.';

export const metadata: Metadata = {
  title: 'Explore home chefs',
  description: DESCRIPTION,
  alternates: { canonical: '/explore/' },
  openGraph: { title: 'Explore home chefs · Fe3dr', url: '/explore/' },
};

function Pills({ label, items, base }: { label: string; items: { slug: string; name: string; chefs: unknown[] }[]; base: string }) {
  if (!items.length) return null;
  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold text-neutral-900">{label}</h2>
      <div className="mt-4 flex flex-wrap gap-2">
        {items.map((it) => (
          <Link
            key={it.slug}
            href={`${base}${it.slug}/`}
            className="rounded-full bg-neutral-100 px-4 py-1.5 text-sm text-neutral-700 hover:bg-neutral-200"
          >
            {it.name} <span className="text-neutral-400">({it.chefs.length})</span>
          </Link>
        ))}
      </div>
    </section>
  );
}

export default async function ExplorePage() {
  const { chefs, cuisines, areas } = await getChefIndex();
  const featured = chefs.slice(0, 6);
  return (
    <>
      <SiteNav />
      <main id="main" className="mx-auto max-w-5xl px-4 py-10">
        <JsonLd data={collectionPageSchema('Explore home chefs', '/explore/', DESCRIPTION)} />
        <h1 className="text-3xl font-semibold text-neutral-900 sm:text-4xl">Explore home chefs</h1>
        <p className="mt-2 max-w-2xl text-pretty text-neutral-600">{DESCRIPTION}</p>

        <Pills label="Browse by cuisine" items={cuisines} base="/cuisine/" />
        <Pills label="Browse by area" items={areas} base="/area/" />

        {featured.length ? (
          <section className="mt-12">
            <h2 className="text-xl font-semibold text-neutral-900">Top-rated home chefs</h2>
            <div className="mt-4 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
              {featured.map((c) => (
                <ChefCard key={c.id} chef={c} />
              ))}
            </div>
          </section>
        ) : null}

        <div className="mt-14">
          <AppCta title="Get the Fe3dr app" subtitle="Order home-cooked meals from cooks near you." />
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
