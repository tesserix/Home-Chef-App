import { LAUNCH_CITY, SHOWCASE_DISHES } from '@/lib/site';
import { Reveal } from '@/components/reveal';

/**
 * "What's cooking" — a horizontally scrolling rail of home-cooked
 * dishes. Native scroll-snap (swipe on mobile, trackpad/keyboard on
 * desktop), photo-forward cards with a quiet hover lift, and the
 * Indian veg/non-veg mark as an authentic functional detail.
 *
 * Menus are illustrative until real kitchens go live — footnoted.
 */
export function DishShowcase() {
  return (
    <section
      id="whats-cooking"
      aria-labelledby="cooking-heading"
      className="scroll-mt-20 overflow-hidden border-t border-hairline"
    >
      <div className="mx-auto max-w-6xl px-5 pt-16 sm:px-8 lg:pt-24">
        <Reveal>
          <div className="flex flex-wrap items-end justify-between gap-6">
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.14em] text-charcoal-soft">
                What&rsquo;s cooking
              </p>
              <h2
                id="cooking-heading"
                className="mt-4 max-w-xl font-display text-[clamp(2rem,4.5vw,3rem)] font-bold leading-[1.05] tracking-tight text-charcoal"
              >
                Tonight, in {LAUNCH_CITY}&rsquo;s home kitchens
              </h2>
            </div>
            <p className="hidden max-w-[16rem] text-sm leading-relaxed text-charcoal-soft sm:block">
              Every dish cooked to order by one person, in one kitchen — never
              off a line.
            </p>
          </div>
        </Reveal>
      </div>

      <Reveal delay={100}>
        {/* Keyboard-focusable scroll region; rail bleeds to the right edge */}
        <div
          role="region"
          aria-label="Example dishes from home chefs"
          tabIndex={0}
          className="rail mt-10 flex snap-x snap-mandatory gap-5 overflow-x-auto px-5 pb-4 sm:px-8 lg:mt-12 lg:pl-[max(2rem,calc((100vw-72rem)/2+2rem))]"
        >
          {SHOWCASE_DISHES.map((dish) => (
            <figure
              key={dish.name}
              className="group w-[70vw] max-w-[300px] shrink-0 snap-start sm:w-64"
            >
              <div className="overflow-hidden rounded-xl">
                {/* PLACEHOLDER PHOTOGRAPHY: swap for owned photos (lib/site.ts). */}
                <img
                  src={dish.img.src}
                  alt={dish.img.alt}
                  width={640}
                  height={800}
                  loading="lazy"
                  className="aspect-[4/5] w-full object-cover transition-transform duration-page ease-out group-hover:scale-[1.04]"
                />
              </div>
              <figcaption className="mt-3.5">
                <div className="flex items-center justify-between gap-3">
                  <span className="flex items-center gap-2 font-display text-lg font-semibold tracking-tight text-charcoal">
                    <VegMark veg={dish.veg} />
                    {dish.name}
                  </span>
                  <span className="tnum text-[15px] font-semibold text-charcoal">
                    {dish.price}
                  </span>
                </div>
                <p className="mt-1 text-sm text-charcoal-soft">
                  {dish.chef}&rsquo;s kitchen &middot; {dish.area}
                </p>
              </figcaption>
            </figure>
          ))}
          {/* Right-edge breathing room at the end of the rail */}
          <div aria-hidden="true" className="w-1 shrink-0 sm:w-4" />
        </div>
      </Reveal>

      <div className="mx-auto max-w-6xl px-5 pb-16 sm:px-8 lg:pb-24">
        <p className="mt-4 text-[13px] text-charcoal-soft">
          Illustrative menus &amp; prices — real kitchens go live at launch.
        </p>
      </div>
    </section>
  );
}

/** The Indian FSSAI veg / non-veg mark — functional, never decorative. */
function VegMark({ veg }: { veg: boolean }) {
  const color = veg ? 'var(--success)' : 'var(--destructive)';
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 14 14"
      role="img"
      aria-label={veg ? 'Vegetarian' : 'Non-vegetarian'}
      className="shrink-0"
    >
      <rect
        x="1"
        y="1"
        width="12"
        height="12"
        rx="2"
        fill="none"
        stroke={color}
        strokeWidth="1.5"
      />
      <circle cx="7" cy="7" r="3" fill={color} />
    </svg>
  );
}
