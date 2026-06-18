import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { CollectionView } from '@/components/seo/collection-view';
import { getChefIndex } from '@/lib/seo-data';
import { SITE_NAME } from '@/lib/site';

export const dynamicParams = false;

export async function generateStaticParams() {
  const { cuisines } = await getChefIndex();
  return cuisines.map((c) => ({ slug: c.slug }));
}

function intro(name: string, count: number): string {
  return `Discover ${count} home ${count === 1 ? 'chef' : 'chefs'} cooking ${name} on Fe3dr — real home-cooked meals, ordered from your phone. Browse menus and order in the app.`;
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const group = (await getChefIndex()).cuisines.find((c) => c.slug === slug);
  if (!group) return {};
  return {
    title: `${group.name} home chefs`,
    description: intro(group.name, group.chefs.length),
    alternates: { canonical: `/cuisine/${slug}/` },
    openGraph: { title: `${group.name} home chefs · ${SITE_NAME}`, url: `/cuisine/${slug}/` },
  };
}

export default async function CuisinePage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const group = (await getChefIndex()).cuisines.find((c) => c.slug === slug);
  if (!group) notFound();
  return (
    <CollectionView
      title={`${group.name} home chefs`}
      intro={intro(group.name, group.chefs.length)}
      path={`/cuisine/${slug}/`}
      chefs={group.chefs}
      crumbs={[
        { name: 'Home', path: '/' },
        { name: 'Explore', path: '/explore/' },
        { name: group.name, path: `/cuisine/${slug}/` },
      ]}
    />
  );
}
