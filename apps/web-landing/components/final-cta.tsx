import { BROWSE_PATH, LAUNCH_CITY, WEB_APP_LIVE } from '@/lib/site';
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
            {WEB_APP_LIVE
              ? `Dinner from a ${LAUNCH_CITY} home kitchen is a few clicks away — no app needed.`
              : `Get the app, and dinner from a ${LAUNCH_CITY} home kitchen is a few taps away.`}
          </p>
        </Reveal>
        <Reveal delay={160}>
          {/* Closes on the action that works today. This section used to end on
              the store badges alone, so the last thing a convinced visitor saw
              was a button for an app that has not shipped. */}
          {WEB_APP_LIVE ? (
            <div className="mt-9">
              <a
                href={BROWSE_PATH}
                className="inline-flex h-14 items-center rounded-xl bg-coral px-8 text-base font-semibold text-white transition-colors hover:bg-coral-pressed focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-coral/30"
              >
                Browse kitchens near you
              </a>
              <div className="mt-8 border-t border-hairline pt-7">
                <p className="mb-3 text-sm font-medium text-charcoal">
                  Prefer an app? It&rsquo;s on the way.
                </p>
                <StoreBadges />
              </div>
            </div>
          ) : (
            <div className="mt-9">
              <StoreBadges />
            </div>
          )}
        </Reveal>
      </div>
    </section>
  );
}
