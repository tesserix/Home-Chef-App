import { Reveal } from '@/components/reveal';

const STEPS = [
  {
    number: '01',
    title: 'Browse home chefs',
    copy: 'See who’s cooking near you — real menus from real kitchens, updated every day.',
  },
  {
    number: '02',
    title: 'Order in a few taps',
    copy: 'Pick your meal and pay securely. The chef starts cooking just for you.',
  },
  {
    number: '03',
    title: 'Track it to your door',
    copy: 'Follow your order live — from the chef’s stove to your doorstep.',
  },
] as const;

/**
 * Editorial steps: a sticky asymmetric title column against oversized
 * ghost numerals. A hairline on the left of the list fills with ink as
 * you scroll past (CSS scroll-driven animation — progressive
 * enhancement, static hairline everywhere else).
 */
export function HowItWorks() {
  return (
    <section
      id="how-it-works"
      aria-labelledby="how-heading"
      className="scroll-mt-20 border-t border-hairline"
    >
      <div className="mx-auto grid max-w-6xl gap-10 px-5 py-16 sm:px-8 lg:grid-cols-12 lg:gap-8 lg:py-28">
        {/* Sticky section title — asymmetric editorial column */}
        <div className="lg:col-span-5">
          <div className="lg:sticky lg:top-28">
            <Reveal>
              <p className="text-sm font-semibold uppercase tracking-[0.14em] text-charcoal-soft">
                How it works
              </p>
              <h2
                id="how-heading"
                className="mt-4 max-w-sm font-display text-[clamp(2rem,4.5vw,3rem)] font-bold leading-[1.05] tracking-tight text-charcoal"
              >
                From their stove to your table
              </h2>
              <p className="mt-5 max-w-xs leading-relaxed text-charcoal-soft">
                No dark kitchens, no warming trays. One cook, one order, one
                doorstep.
              </p>
            </Reveal>
          </div>
        </div>

        {/* Steps with oversized ghost numerals + scroll progress line */}
        <div className="relative lg:col-span-7">
          <div
            aria-hidden="true"
            className="progress-line absolute bottom-4 left-0 top-4 hidden w-px sm:block"
          />
          <ol className="sm:pl-10 lg:pl-14">
            {STEPS.map((step, index) => (
              <li
                key={step.number}
                className={index > 0 ? 'border-t border-hairline' : undefined}
              >
                <Reveal delay={index * 90}>
                  <div className="flex items-start gap-6 py-10 sm:gap-10 lg:py-12">
                    <span
                      aria-hidden="true"
                      className="text-outline tnum -mt-3 select-none font-display text-[clamp(3.25rem,8vw,5rem)] font-bold leading-none"
                    >
                      {step.number}
                    </span>
                    <div className="pt-1">
                      <h3 className="font-display text-xl font-semibold tracking-tight text-charcoal lg:text-2xl">
                        {step.title}
                      </h3>
                      <p className="mt-2.5 max-w-md leading-relaxed text-charcoal-soft">
                        {step.copy}
                      </p>
                    </div>
                  </div>
                </Reveal>
              </li>
            ))}
          </ol>
        </div>
      </div>
    </section>
  );
}
