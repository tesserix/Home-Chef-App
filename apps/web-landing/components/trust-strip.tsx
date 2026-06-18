import { LAUNCH_CITY } from '@/lib/site';
import { Reveal } from '@/components/reveal';

/**
 * Honest, pre-launch trust signals as an editorial stat band —
 * oversized tabular numerals, no invented metrics, no FOMO.
 */
const SIGNALS = [
  {
    figure: '100%',
    title: 'FSSAI-verified kitchens',
    copy: 'Every chef is registered and document-checked before going live.',
  },
  {
    figure: '1',
    title: `City to start — ${LAUNCH_CITY}`,
    copy: 'We’re growing kitchen by kitchen, neighbourhood by neighbourhood.',
  },
  {
    figure: '0',
    title: 'Cloud kitchens',
    copy: 'Every order comes from a real home, cooked by the person named on it.',
  },
] as const;

export function TrustStrip() {
  return (
    <section aria-labelledby="trust-heading" className="border-t border-hairline">
      <div className="mx-auto max-w-6xl px-5 py-16 sm:px-8 lg:py-24">
        <h2 id="trust-heading" className="sr-only">
          Why you can trust Fe3dr
        </h2>
        <ul className="grid gap-10 sm:grid-cols-3 sm:gap-6">
          {SIGNALS.map((signal, index) => (
            <li
              key={signal.title}
              className={`sm:pr-6 ${
                index > 0
                  ? 'border-t border-hairline pt-10 sm:border-l sm:border-t-0 sm:pl-10 sm:pt-0'
                  : ''
              }`}
            >
              <Reveal delay={index * 90}>
                <p
                  aria-hidden="true"
                  className="tnum font-display text-[clamp(3.25rem,7vw,4.75rem)] font-bold leading-none tracking-tight text-charcoal"
                >
                  {signal.figure}
                </p>
                <h3 className="mt-4 text-base font-semibold text-charcoal">
                  <span className="sr-only">{signal.figure} </span>
                  {signal.title}
                </h3>
                <p className="mt-1.5 max-w-[17rem] text-sm leading-relaxed text-charcoal-soft">
                  {signal.copy}
                </p>
              </Reveal>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
