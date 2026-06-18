import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { CollectionView } from '@/components/seo/collection-view';
import { getChefIndex } from '@/lib/seo-data';
import { SITE_NAME } from '@/lib/site';

export const dynamicParams = false;

export async function generateStaticParams() {
  const { areas } = await getChefIndex();
  return areas.map((a) => ({ slug: a.slug }));
}

function intro(name: string, count: number): string {
  return `Order home-cooked food in ${name} from ${count} local home ${count === 1 ? 'chef' : 'chefs'} on Fe3dr. Fresh, made-to-order meals from cooks near you — browse and order in the app.`;
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const group = (await getChefIndex()).areas.find((a) => a.slug === slug);
  if (!group) return {};
  return {
    title: `Home chefs in ${group.name}`,
    description: intro(group.name, group.chefs.length),
    alternates: { canonical: `/area/${slug}/` },
    openGraph: { title: `Home chefs in ${group.name} · ${SITE_NAME}`, url: `/area/${slug}/` },
  };
}

export default async function AreaPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const group = (await getChefIndex()).areas.find((a) => a.slug === slug);
  if (!group) notFound();
  return (
    <CollectionView
      title={`Home chefs in ${group.name}`}
      intro={intro(group.name, group.chefs.length)}
      path={`/area/${slug}/`}
      chefs={group.chefs}
      crumbs={[
        { name: 'Home', path: '/' },
        { name: 'Explore', path: '/explore/' },
        { name: group.name, path: `/area/${slug}/` },
      ]}
    />
  );
}
