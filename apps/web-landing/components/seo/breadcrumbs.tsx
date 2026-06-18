import Link from 'next/link';

/** Visual breadcrumb. Pair with breadcrumbSchema() for the JSON-LD twin. */
export function Breadcrumbs({ items }: { items: { name: string; path: string }[] }) {
  return (
    <nav aria-label="Breadcrumb" className="text-sm text-neutral-500">
      <ol className="flex flex-wrap items-center gap-1.5">
        {items.map((it, i) => {
          const last = i === items.length - 1;
          return (
            <li key={it.path} className="flex items-center gap-1.5">
              {last ? (
                <span aria-current="page" className="text-neutral-700">
                  {it.name}
                </span>
              ) : (
                <Link href={it.path} className="hover:text-neutral-900">
                  {it.name}
                </Link>
              )}
              {!last ? <span aria-hidden="true">›</span> : null}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
