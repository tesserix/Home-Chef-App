import { IMAGES } from '@/lib/site';
import { Parallax } from '@/components/parallax';
import { Reveal } from '@/components/reveal';

const VALUES = [
  {
    title: 'Real local cooks',
    copy: 'Every dish is made by a person, not a production line. Browse a chef’s kitchen, their story, and what they’re cooking this week.',
  },
  {
    title: 'FSSAI-verified kitchens',
    copy: 'Every chef on Fe3dr holds a valid FSSAI registration. We verify documents before a kitchen goes live — no exceptions.',
  },
  {
    title: 'Live delivery tracking',
    copy: 'Watch your order move: accepted, cooking, picked up, at your door. Minute-by-minute, from the first tap to the handoff.',
  },
  {
    title: 'Fair to chefs',
    copy: 'Home cooks set their own menus and their own prices, and keep what they earn. When you order, your money goes to a neighbour.',
  },
] as const;

export function WhyHomeChef() {
  return (
    <section
      aria-labelledby="why-heading"
      className="border-t border-hairline bg-surface"
    >
      <div className="mx-auto grid max-w-6xl gap-12 px-5 py-16 sm:px-8 lg:grid-cols-12 lg:gap-16 lg:py-28">
        {/* Tall supporting photo — sticky against the scrolling list,
            with a floating caption chip for depth */}
        <div className="order-last lg:order-first lg:col-span-5">
          <Reveal variant="left" className="lg:sticky lg:top-28">
            <div className="relative">
              {/* PLACEHOLDER PHOTOGRAPHY: swap for owned photos (lib/site.ts). */}
              <img
                src={IMAGES.whyKitchen.src}
                alt={IMAGES.whyKitchen.alt}
                width={987}
                height={1234}
                loading="lazy"
                className="aspect-[4/5] w-full rounded-2xl object-cover"
              />
              <Parallax
                speed={0.08}
                className="absolute -right-3 bottom-8 sm:-right-5"
              >
                <p
                  aria-hidden="true"
                  className="rounded-lg bg-canvas px-4 py-3 text-[13px] font-medium text-charcoal shadow-2"
                >
                  Cooked fresh today
                  <span className="tnum mt-0.5 block font-semibold">
                    — never reheated off a line
                  </span>
                </p>
              </Parallax>
            </div>
            <p className="sr-only">
              Cooked fresh in a home kitchen — never reheated off a line.
            </p>
          </Reveal>
        </div>

        <div className="lg:col-span-7">
          <Reveal>
            <h2
              id="why-heading"
              className="max-w-lg font-display text-[clamp(2rem,4.5vw,3rem)] font-bold leading-[1.05] tracking-tight text-charcoal"
            >
              This isn&rsquo;t restaurant food. That&rsquo;s the point.
            </h2>
          </Reveal>

          <dl className="mt-4">
            {VALUES.map((value, index) => (
              <Reveal key={value.title} delay={index * 70}>
                <div
                  className={`py-7 lg:py-8 ${
                    index > 0 ? 'border-t border-hairline' : ''
                  }`}
                >
                  <dt className="font-display text-xl font-semibold tracking-tight text-charcoal">
                    {value.title}
                  </dt>
                  <dd className="mt-2 max-w-xl leading-relaxed text-charcoal-soft">
                    {value.copy}
                  </dd>
                </div>
              </Reveal>
            ))}
          </dl>
        </div>
      </div>
    </section>
  );
}
