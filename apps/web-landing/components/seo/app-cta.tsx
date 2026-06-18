import { StoreBadges } from '@/components/store-badges';

/**
 * "Get the app" CTA reused on every SEO page. Optionally takes a chefId to deep
 * link straight into that chef in the app (custom scheme; the app also handles
 * the matching https://fe3dr.com/chef/<slug> universal link once configured).
 */
export function AppCta({ title, subtitle, chefId }: { title: string; subtitle?: string; chefId?: string }) {
  return (
    <section className="rounded-3xl bg-neutral-900 px-6 py-10 text-center text-white sm:px-10">
      <h2 className="text-balance text-2xl font-semibold sm:text-3xl">{title}</h2>
      {subtitle ? <p className="mx-auto mt-2 max-w-xl text-neutral-300">{subtitle}</p> : null}
      <div className="mt-6 flex flex-col items-center gap-4">
        {chefId ? (
          <a
            href={`homechef-customer://chef/${chefId}`}
            className="inline-flex items-center justify-center rounded-full bg-white px-6 py-3 font-medium text-neutral-900 transition-opacity hover:opacity-90"
          >
            Open in the Fe3dr app
          </a>
        ) : null}
        <StoreBadges />
      </div>
    </section>
  );
}
