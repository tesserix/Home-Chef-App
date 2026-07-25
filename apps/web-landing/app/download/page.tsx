// The canonical "get the app" destination, linked from the nav and footer of
// every page. It exists so there is one honest answer to "where do I download
// it" — including while the listings are still unpublished, when the right
// answer is "not yet, here's how to hear about it" rather than a dead badge.

import type { Metadata } from 'next';
import { CONTACT_EMAIL, CUSTOMER_APP, isDownloadable, VENDOR_APP } from '@/lib/site';
import { SiteFooter } from '@/components/site-footer';
import { SiteNav } from '@/components/site-nav';
import { StoreBadges } from '@/components/store-badges';

export const metadata: Metadata = {
  title: 'Get the app',
  description:
    'Download Fe3dr — order home-cooked food from kitchens near you, or run your kitchen with the chef app.',
  alternates: { canonical: '/download/' },
};

/** What each app is actually for, in the user's words rather than the system's. */
const CUSTOMER_DOES = [
  'Browse home kitchens near you and see today’s menu',
  'Order a single meal or subscribe to a weekly plan',
  'Follow your order from the chef’s stove to your door',
  'Pay by UPI or card, and spend wallet credit and loyalty points',
];

const CHEF_DOES = [
  'Take orders and set how many you can cook each day',
  'Build your menu, with photos, portions and prices',
  'Plan the week ahead and prep against a daily list',
  'Track earnings and get paid out to your bank',
];

const NOTIFY_SUBJECT = encodeURIComponent('Tell me when the Fe3dr app is out');

export default function DownloadPage() {
  const anythingLive = isDownloadable(CUSTOMER_APP) || isDownloadable(VENDOR_APP);

  // The nav CTA below has to point at a section that actually exists in both
  // states: #notify only renders while nothing is live (below), and #apps —
  // the store badges — always renders. Without this branch, the moment a
  // listing flips to `live` the nav CTA keeps pointing at the now-gone
  // #notify section.
  const ctaHref = anythingLive ? '#apps' : '#notify';
  const ctaLabel = anythingLive ? 'Get the app' : 'Get notified';

  return (
    <>
      <SiteNav ctaHref={ctaHref} ctaLabel={ctaLabel} />
      <main id="main">
        {/* Header — states the real status up front rather than burying it. */}
        <section className="border-b border-hairline">
          <div className="mx-auto max-w-6xl px-5 pb-14 pt-12 sm:px-8 lg:pb-20 lg:pt-16">
            <h1 className="max-w-3xl font-display text-[clamp(2.5rem,7vw,4rem)] font-bold leading-[1.0] tracking-[-0.03em] text-charcoal">
              Get the Fe3dr app<span className="text-coral">.</span>
            </h1>

            <p className="mt-6 max-w-xl text-lg leading-relaxed text-charcoal-soft">
              {anythingLive
                ? 'Two apps: one to eat from, one to cook with. Pick yours below.'
                : 'Two apps — one to eat from, one to cook with. Neither is on the stores yet. Leave your email and we’ll send you the link the day it goes live.'}
            </p>
          </div>
        </section>

        {/* One block per app. The heading says who it's for, because that is
            the only thing a visitor needs in order to choose. */}
        <section id="apps" aria-labelledby="apps-heading" className="scroll-mt-24 border-b border-hairline">
          <h2 id="apps-heading" className="sr-only">
            The Fe3dr apps
          </h2>
          {/* The divider is a border on the second block, not a background
              behind a grid gap — a `bg-hairline` parent paints the horizontal
              padding too, which shows as grey bands down both edges on mobile. */}
          <div className="mx-auto grid max-w-6xl px-5 sm:px-8 lg:grid-cols-2 lg:px-0 [&>*+*]:border-t [&>*+*]:border-hairline lg:[&>*+*]:border-l lg:[&>*+*]:border-t-0">
            <AppBlock
              eyebrow="For eating"
              title="Fe3dr"
              blurb="Home-cooked food from kitchens around you."
              does={CUSTOMER_DOES}
              app={CUSTOMER_APP}
            />
            <AppBlock
              eyebrow="For cooking"
              title="Fe3dr for Chefs"
              blurb="Run your kitchen — orders, menu, capacity and payouts."
              does={CHEF_DOES}
              app={VENDOR_APP}
            />
          </div>
        </section>

        {/* The single primary action on the page. */}
        {!anythingLive ? (
          <section id="notify" aria-labelledby="notify-heading" className="scroll-mt-24">
            <div className="mx-auto max-w-6xl px-5 py-16 sm:px-8 lg:py-24">
              <h2
                id="notify-heading"
                className="max-w-2xl font-display text-[clamp(1.75rem,4vw,2.5rem)] font-bold leading-[1.05] tracking-[-0.02em] text-charcoal"
              >
                Want the link the moment it&rsquo;s live?
              </h2>
              <p className="mt-4 max-w-md text-lg leading-relaxed text-charcoal-soft">
                Email us and we&rsquo;ll write to you once — on launch day, with the
                download link. Nothing else.
              </p>
              <a
                href={`mailto:${CONTACT_EMAIL}?subject=${NOTIFY_SUBJECT}`}
                className="mt-8 inline-flex h-12 items-center rounded-lg bg-coral px-6 text-[15px] font-semibold text-white transition-colors duration-micro ease-state hover:bg-coral-pressed"
              >
                Email {CONTACT_EMAIL}
              </a>
            </div>
          </section>
        ) : null}
      </main>
      <SiteFooter />
    </>
  );
}

function AppBlock({
  eyebrow,
  title,
  blurb,
  does,
  app,
}: {
  eyebrow: string;
  title: string;
  blurb: string;
  does: string[];
  app: typeof CUSTOMER_APP;
}) {
  return (
    <div className="py-12 lg:px-10 lg:py-16">
      <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-charcoal-soft">
        {eyebrow}
      </p>
      <h3 className="mt-3 font-display text-2xl font-bold tracking-[-0.01em] text-charcoal sm:text-3xl">
        {title}
      </h3>
      <p className="mt-3 max-w-sm leading-relaxed text-charcoal-soft">{blurb}</p>

      <ul className="mt-7 space-y-3">
        {does.map((line) => (
          <li key={line} className="flex gap-3 text-[15px] leading-relaxed text-charcoal">
            <span
              aria-hidden="true"
              className="mt-[0.6em] h-1 w-1 shrink-0 rounded-full bg-charcoal-soft"
            />
            {line}
          </li>
        ))}
      </ul>

      <StoreBadges app={app} height={48} className="mt-9" />
    </div>
  );
}
