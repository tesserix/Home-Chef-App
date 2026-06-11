import { IMAGES, LAUNCH_CITY } from '@/lib/site';
import { Parallax } from '@/components/parallax';
import { RouteMotif } from '@/components/route-motif';
import { StoreBadges } from '@/components/store-badges';

/**
 * Hero — the moment. Oversized bilingual Geist headline with a
 * line-mask rise entrance, against a layered collage: one large
 * home-spread photograph with floating dish/kitchen cards drifting
 * at different parallax depths, anchored by the signature animated
 * order-route card. Coral appears exactly where it matters: the
 * full stop, the live dot, the travelling order.
 */
export function Hero() {
  return (
    <section aria-labelledby="hero-heading" className="overflow-hidden">
      <div className="mx-auto grid max-w-6xl items-center gap-14 px-5 pb-20 pt-10 sm:px-8 lg:grid-cols-12 lg:gap-6 lg:pb-28 lg:pt-16">
        {/* Copy — left-weighted, editorial */}
        <div className="lg:col-span-6 lg:pr-4">
          <p className="fade-up inline-flex items-center gap-2.5 rounded-full border border-hairline py-2 pl-3 pr-4 text-[13px] font-semibold text-charcoal">
            <span className="relative inline-block h-2 w-2 rounded-full bg-coral pulse-dot" />
            Now launching in {LAUNCH_CITY}
          </p>

          <h1
            id="hero-heading"
            className="mt-7 font-display text-[clamp(2.75rem,8.5vw,4.5rem)] font-bold leading-[0.98] tracking-[-0.03em] text-charcoal"
          >
            <span className="line-mask">
              <span className="line-rise" style={{ animationDelay: '80ms' }}>
                Ghar ka khana,
              </span>
            </span>
            <span className="line-mask">
              <span className="line-rise" style={{ animationDelay: '180ms' }}>
                delivered<span className="text-coral">.</span>
              </span>
            </span>
          </h1>

          <p
            className="fade-up mt-7 max-w-md text-lg leading-relaxed text-charcoal-soft"
            style={{ animationDelay: '320ms' }}
          >
            Real home cooks in your neighbourhood, cooking what they make
            best. Browse their kitchens, order today&rsquo;s menu, and watch
            dinner travel from their stove to your door.
          </p>

          <div
            id="get-the-app"
            className="fade-up mt-9 scroll-mt-28"
            style={{ animationDelay: '440ms' }}
          >
            <StoreBadges />
          </div>

          <p
            className="fade-up mt-6 text-sm text-charcoal-soft"
            style={{ animationDelay: '540ms' }}
          >
            FSSAI-verified kitchens&ensp;&middot;&ensp;Secure payments
          </p>
        </div>

        {/* Collage — layered depth, food is the star */}
        <div className="relative lg:col-span-6">
          <div
            className="fade-up relative mx-auto w-full max-w-[480px] lg:ml-auto"
            style={{ animationDelay: '200ms' }}
          >
            {/* Warm tint field behind the composition (the one soft
                coral-tint moment on the page) */}
            <div
              aria-hidden="true"
              className="absolute -right-10 -top-10 hidden h-72 w-72 rounded-[2.5rem] bg-coral-tint sm:block"
            />

            {/* PLACEHOLDER PHOTOGRAPHY: swap for owned photos (lib/site.ts). */}
            <img
              src={IMAGES.heroMain.src}
              alt={IMAGES.heroMain.alt}
              width={1080}
              height={1350}
              fetchPriority="high"
              className="relative aspect-[4/5] w-full rounded-2xl object-cover shadow-2"
            />

            {/* Floating dish card — drifts slower than the page */}
            <Parallax
              speed={0.16}
              className="absolute -left-4 top-10 w-36 sm:-left-10 sm:w-44"
            >
              <figure className="float-slow overflow-hidden rounded-xl bg-canvas shadow-2">
                <img
                  src={IMAGES.heroDosa.src}
                  alt={IMAGES.heroDosa.alt}
                  width={480}
                  height={360}
                  loading="lazy"
                  className="aspect-[4/3] w-full object-cover"
                />
                <figcaption className="flex items-baseline justify-between px-3.5 py-2.5">
                  <span className="text-[13px] font-semibold text-charcoal">
                    Masala dosa
                  </span>
                  <span className="tnum text-[13px] font-medium text-charcoal-soft">
                    ₹110
                  </span>
                </figcaption>
              </figure>
            </Parallax>

            {/* Floating kitchen card — cooking right now */}
            <Parallax
              speed={-0.1}
              className="absolute -right-3 top-1/2 hidden w-40 sm:block lg:-right-12"
            >
              <figure className="overflow-hidden rounded-xl bg-canvas shadow-2">
                <img
                  src={IMAGES.heroCooking.src}
                  alt={IMAGES.heroCooking.alt}
                  width={480}
                  height={320}
                  loading="lazy"
                  className="aspect-[3/2] w-full object-cover"
                />
                <figcaption className="flex items-center gap-2 px-3.5 py-2.5">
                  <span className="relative inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-coral pulse-dot" />
                  <span className="text-[12px] font-medium text-charcoal">
                    Sunita&rsquo;s kitchen — cooking now
                  </span>
                </figcaption>
              </figure>
            </Parallax>

            {/* The signature: a live order tracking to your door.
                Illustrative — not real data, hidden from AT. */}
            <div
              aria-hidden="true"
              className="absolute -bottom-9 -left-2 rounded-xl bg-canvas p-4 shadow-3 sm:-left-8"
            >
              <div className="flex items-baseline justify-between gap-8">
                <p className="text-[13px] font-medium text-charcoal-soft">
                  Aarti&rsquo;s rajma chawal
                </p>
                <p className="tnum text-[13px] font-semibold text-charcoal">
                  7:05 pm
                </p>
              </div>
              <RouteMotif className="mt-1.5" />
              <p className="mt-1 text-[12px] font-medium text-charcoal-soft">
                Picked up &middot; 2.1 km away
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
