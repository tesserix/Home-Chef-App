import { LAUNCH_CITY } from '@/lib/site';
import { Reveal } from '@/components/reveal';
import { RouteMotif } from '@/components/route-motif';
import { StoreBadges } from '@/components/store-badges';

/**
 * Closing moment — one oversized line, the signature route device,
 * and the store badges. Nothing else competes.
 */
export function FinalCta() {
  return (
    <section
      aria-labelledby="cta-heading"
      className="border-t border-hairline"
    >
      <div className="mx-auto max-w-6xl px-5 py-20 sm:px-8 lg:py-32">
        <Reveal>
          <RouteMotif />
        </Reveal>
        <Reveal delay={80}>
          <h2
            id="cta-heading"
            className="mt-8 max-w-3xl font-display text-[clamp(2.75rem,8vw,4.75rem)] font-bold leading-[0.98] tracking-[-0.03em] text-charcoal"
          >
            Hungry already<span className="text-coral">?</span>
          </h2>
          <p className="mt-6 max-w-md text-lg leading-relaxed text-charcoal-soft">
            Get the app, and dinner from a {LAUNCH_CITY} home kitchen is a few
            taps away.
          </p>
        </Reveal>
        <Reveal delay={160}>
          <div className="mt-9">
            <StoreBadges />
          </div>
        </Reveal>
      </div>
    </section>
  );
}
