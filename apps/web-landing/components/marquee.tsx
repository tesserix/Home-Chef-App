import { MARQUEE_DISHES } from '@/lib/site';

/**
 * Full-bleed dish marquee — the food itself becomes the typography.
 * Alternating solid / outline Geist gives the strip editorial rhythm
 * without a second color. Continuous translateX loop (CSS), pauses on
 * hover, static under reduced motion. The duplicated track is
 * aria-hidden; screen readers get a single clean list.
 */
export function Marquee() {
  return (
    <section
      aria-label="Dishes our home chefs cook"
      className="border-y border-hairline py-7 lg:py-9"
    >
      <div className="marquee" role="presentation">
        <div className="marquee-track">
          <MarqueeRow />
          {/* Seamless loop copy — decorative */}
          <MarqueeRow ariaHidden />
        </div>
      </div>
      {/* Accessible text alternative for the animated strip */}
      <p className="sr-only">{MARQUEE_DISHES.join(', ')}</p>
    </section>
  );
}

function MarqueeRow({ ariaHidden = false }: { ariaHidden?: boolean }) {
  return (
    <div aria-hidden={ariaHidden || undefined} className="flex shrink-0">
      {MARQUEE_DISHES.map((dish, index) => (
        <span
          key={dish}
          className="flex items-center whitespace-nowrap pr-10 font-display text-[clamp(1.75rem,4.5vw,3rem)] font-bold tracking-tight lg:pr-14"
        >
          <span
            className={
              index % 2 === 0 ? 'text-charcoal' : 'text-outline select-none'
            }
          >
            {dish}
          </span>
          <span
            aria-hidden="true"
            className="ml-10 inline-block h-1.5 w-1.5 rounded-full bg-hairline lg:ml-14"
          />
        </span>
      ))}
    </div>
  );
}
