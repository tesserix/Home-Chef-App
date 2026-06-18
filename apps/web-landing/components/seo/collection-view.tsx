import { SiteNav } from '@/components/site-nav';
import { SiteFooter } from '@/components/site-footer';
import { JsonLd } from '@/components/seo/json-ld';
import { ChefCard } from '@/components/seo/chef-card';
import { Breadcrumbs } from '@/components/seo/breadcrumbs';
import { AppCta } from '@/components/seo/app-cta';
import { collectionPageSchema, itemListSchema, breadcrumbSchema } from '@/lib/jsonld';
import type { SeoChef } from '@/lib/seo-data';

/** Shared page shell for cuisine + area collection pages (DRY). */
export function CollectionView({
  title,
  intro,
  path,
  chefs,
  crumbs,
}: {
  title: string;
  intro: string;
  path: string;
  chefs: SeoChef[];
  crumbs: { name: string; path: string }[];
}) {
  return (
    <>
      <SiteNav />
      <main id="main" className="mx-auto max-w-5xl px-4 py-10">
        <JsonLd data={collectionPageSchema(title, path, intro)} />
        <JsonLd data={itemListSchema(chefs)} />
        <JsonLd data={breadcrumbSchema(crumbs)} />
        <Breadcrumbs items={crumbs} />
        <h1 className="mt-4 text-3xl font-semibold text-neutral-900 sm:text-4xl">{title}</h1>
        <p className="mt-2 max-w-2xl text-pretty text-neutral-600">{intro}</p>
        <div className="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {chefs.map((c) => (
            <ChefCard key={c.id} chef={c} />
          ))}
        </div>
        <div className="mt-14">
          <AppCta
            title="Hungry? Get the Fe3dr app"
            subtitle="Browse these home chefs, see live menus, and order in a few taps."
          />
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
