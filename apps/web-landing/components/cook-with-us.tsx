import { CHEFS_EMAIL, IMAGES } from '@/lib/site';
import { Parallax } from '@/components/parallax';
import { Reveal } from '@/components/reveal';

/**
 * Chef-recruitment band — the page's single dark beat. The photo
 * breaks the section edge and carries a floating vendor-app order
 * card (illustrative product UI, hidden from AT). The CTA stays
 * white-on-charcoal so coral remains the customer journey's accent.
 */
export function CookWithUs() {
  return (
    <section
      id="for-chefs"
      aria-labelledby="chefs-heading"
      className="scroll-mt-20 overflow-hidden bg-charcoal text-white"
    >
      <div className="mx-auto grid max-w-6xl items-center gap-14 px-5 py-20 sm:px-8 lg:grid-cols-12 lg:gap-10 lg:py-32">
        <div className="lg:col-span-7 lg:pr-10">
          <Reveal>
            <p className="text-sm font-semibold uppercase tracking-[0.14em] text-white/60">
              Cook with us
            </p>
            <h2
              id="chefs-heading"
              className="mt-5 max-w-xl font-display text-[clamp(2.25rem,5.5vw,3.5rem)] font-bold leading-[1.02] tracking-tight"
            >
              Your kitchen.
              <span className="block text-white/55">Your menu. Your price.</span>
            </h2>
            <p className="mt-6 max-w-lg text-lg leading-relaxed text-white/75">
              If you cook well, you already have everything you need. Cook when
              it suits you — Fe3dr brings the orders, the payments, and the
              delivery.
            </p>
          </Reveal>
          <Reveal delay={120}>
            <div className="mt-9 flex flex-wrap items-center gap-5">
              {/* TODO(owner): point at the chef onboarding flow / vendor app
                  listing once it's live; mailto is the interim channel. */}
              <a
                href={`mailto:${CHEFS_EMAIL}?subject=I want to cook on Fe3dr`}
                className="inline-flex h-12 items-center rounded-full bg-white px-7 text-[15px] font-semibold text-charcoal transition-opacity duration-micro ease-state hover:opacity-90"
              >
                Become a home chef
              </a>
              <p className="text-sm text-white/60">
                The Fe3dr vendor app arrives with our launch.
              </p>
            </div>
          </Reveal>
        </div>

        <div className="lg:col-span-5">
          <Reveal variant="right" delay={80}>
            <div className="relative">
              {/* PLACEHOLDER PHOTOGRAPHY: swap for owned photos (lib/site.ts). */}
              <img
                src={IMAGES.cookWithUs.src}
                alt={IMAGES.cookWithUs.alt}
                width={987}
                height={1234}
                loading="lazy"
                className="aspect-[4/5] w-full rounded-2xl object-cover"
              />
              {/* Vendor app fragment — illustrative, not real data */}
              <Parallax
                speed={0.09}
                className="absolute -bottom-6 -left-3 sm:-left-8"
              >
                <div
                  aria-hidden="true"
                  className="rounded-xl bg-canvas p-4 text-charcoal shadow-3"
                >
                  <div className="flex items-center justify-between gap-10">
                    <p className="text-[12px] font-semibold uppercase tracking-[0.1em] text-charcoal-soft">
                      New order
                    </p>
                    <p className="tnum text-[12px] font-medium text-charcoal-soft">
                      7:12 pm
                    </p>
                  </div>
                  <p className="mt-1.5 text-sm font-semibold">
                    Thalipeeth ×2 &middot; Solkadhi ×1
                  </p>
                  <div className="mt-3 flex gap-2">
                    <span className="inline-flex h-8 items-center rounded-full bg-charcoal px-4 text-[12px] font-semibold text-white">
                      Accept
                    </span>
                    <span className="inline-flex h-8 items-center rounded-full border border-hairline px-4 text-[12px] font-medium text-charcoal-soft">
                      Decline
                    </span>
                  </div>
                </div>
              </Parallax>
            </div>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
