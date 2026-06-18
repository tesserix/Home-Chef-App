import Link from 'next/link';
import type { SeoChef } from '@/lib/seo-data';

/** Chef summary card — reused on cuisine, area, and explore lists. */
export function ChefCard({ chef }: { chef: SeoChef }) {
  return (
    <Link
      href={`/chef/${chef.slug}/`}
      className="group flex flex-col overflow-hidden rounded-2xl border border-black/5 bg-white shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="aspect-[16/10] w-full overflow-hidden bg-neutral-100">
        {chef.image ? (
          <img
            src={chef.image}
            alt=""
            loading="lazy"
            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
          />
        ) : null}
      </div>
      <div className="flex flex-1 flex-col gap-1 p-4">
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-semibold text-neutral-900">{chef.name}</h3>
          {chef.reviews > 0 ? (
            <span className="shrink-0 text-sm tabular-nums text-neutral-700">
              ★ {chef.rating.toFixed(1)}
            </span>
          ) : null}
        </div>
        {chef.cuisines.length ? (
          <p className="text-sm text-neutral-500">{chef.cuisines.slice(0, 3).join(' · ')}</p>
        ) : null}
        <div className="mt-1 flex items-center gap-2 text-xs text-neutral-500">
          {chef.city ? <span>{chef.city}</span> : null}
          {chef.foodSafety ? <span className="text-green-700">✓ Food-safety verified</span> : null}
        </div>
      </div>
    </Link>
  );
}
