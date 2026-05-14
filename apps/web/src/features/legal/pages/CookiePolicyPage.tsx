import { Link } from 'react-router-dom';
import { AlertTriangle, Cookie, Mail, ExternalLink } from 'lucide-react';
import { Badge, Card } from '@/shared/components/ui';

/**
 * Cookie Policy — customer-facing page for the Fe3dr web app.
 *
 * Covers DPDP Act §5 (Notice) requirements + best-practice cookie disclosures:
 * cookie categories, third-party cookies (Razorpay/Stripe), how to manage
 * consent, retention, withdrawal, and grievance officer contact.
 *
 * Style: plain English, short sentences, "we"/"you", per STYLE-GUIDE.md §5.
 */
export default function CookiePolicyPage() {
  const lastUpdated = '13 May 2026';
  const effectiveDate = '13 May 2026';

  return (
    <div className="min-h-screen bg-paper">
      {/* DRAFT banner */}
      <div
        role="status"
        aria-live="polite"
        className="bg-amber-tint border-b border-amber/30"
      >
        <div className="container-app py-3">
          <div className="flex items-start gap-3">
            <AlertTriangle
              aria-hidden="true"
              className="mt-0.5 h-5 w-5 shrink-0 text-amber"
            />
            <div className="text-sm text-ink">
              <span className="font-semibold">Draft.</span>{' '}
              This Cookie Policy is in review and not yet legally binding.
              We&apos;ll update this banner when it&apos;s finalised.
            </div>
          </div>
        </div>
      </div>

      {/* Header */}
      <section className="bg-bone py-12 lg:py-16">
        <div className="container-app">
          <div className="mx-auto max-w-3xl">
            <Badge variant="brand" className="mb-4">
              <Cookie aria-hidden="true" className="mr-1 h-3 w-3" />
              Legal
            </Badge>
            <h1 className="font-display text-display-lg text-ink">
              Cookie Policy
            </h1>
            <p className="mt-4 text-ink-soft">
              How we use cookies on Fe3dr, what each category does, and how
              to change your preferences.
            </p>
            <dl className="mt-6 flex flex-wrap gap-x-8 gap-y-2 text-sm text-ink-muted">
              <div className="flex gap-2">
                <dt className="font-medium">Last updated:</dt>
                <dd className="tabular-nums">{lastUpdated}</dd>
              </div>
              <div className="flex gap-2">
                <dt className="font-medium">Effective:</dt>
                <dd className="tabular-nums">{effectiveDate}</dd>
              </div>
            </dl>
            <p className="mt-4 text-sm text-ink-soft">
              Fe3dr is a product of <strong>Tesserix Pty Ltd</strong> (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia. Day-to-day operations are conducted from Mumbai, India and Sydney, Australia.
            </p>
          </div>
        </div>
      </section>

      <section className="py-12 lg:py-16">
        <div className="container-app">
          <div className="mx-auto max-w-3xl space-y-12">
            {/* Plain-language summary callout */}
            <Card variant="filled" padding="lg" className="border-l-4 border-herb">
              <h2 className="font-display text-xl font-semibold text-ink">
                In one paragraph
              </h2>
              <p className="mt-3 text-ink-soft">
                Here&apos;s what cookies we use. Strictly necessary cookies
                keep you signed in and your cart working — they can&apos;t
                be turned off without breaking the app. We don&apos;t use
                marketing or advertising cookies. We don&apos;t currently
                use analytics cookies either. Third parties — our payment
                processors, Razorpay and Stripe — set their own cookies
                during checkout, and those follow their own rules. You
                can manage everything from our cookie banner or your
                browser settings, and withdrawing consent is as easy as
                giving it.
              </p>
            </Card>

            {/* 1. What cookies are */}
            <article id="what-cookies-are">
              <h2 className="font-display text-2xl font-semibold text-ink">
                1. What cookies are
              </h2>
              <p className="mt-4 text-ink-soft">
                Cookies are small text files a website saves on your device
                when you visit. They let the site remember things — like
                that you&apos;re signed in, what&apos;s in your cart, or your
                preferred language — between page loads and visits.
                Without them, you&apos;d have to sign in on every page and
                rebuild your cart every time you opened a new tab.
              </p>
              <p className="mt-4 text-ink-soft">
                Some cookies disappear when you close the browser (session
                cookies). Others stay for a set period — anywhere from a
                few hours to a few months — and are called persistent
                cookies. Cookies set by us are{' '}
                <strong>first-party</strong> cookies. Cookies set by
                services embedded on our pages — like a payment processor
                during checkout — are <strong>third-party</strong>
                {' '}cookies, and they follow that service&apos;s own
                rules.
              </p>
              <p className="mt-4 text-ink-soft">
                We also use a few similar technologies — like
                <code className="mx-1 rounded bg-mist px-1.5 py-0.5 font-mono text-sm">
                  localStorage
                </code>{' '}
                in your browser — to remember preferences and to keep your
                cart available if you refresh the page. When we say
                &quot;cookies&quot; in this policy, we mean those too,
                because the rules and your choices apply the same way.
              </p>
            </article>

            {/* 2. Categories we use */}
            <article id="categories">
              <h2 className="font-display text-2xl font-semibold text-ink">
                2. Categories we use
              </h2>
              <p className="mt-4 text-ink-soft">
                Cookies fall into four categories. The table below shows
                exactly which ones we set today, what they do, and how long
                they live.
              </p>

              {/* Categories TABLE — central element */}
              <div className="mt-6 overflow-x-auto rounded-lg border border-mist">
                <table className="w-full text-sm">
                  <caption className="sr-only">
                    Cookie categories used on Fe3dr, including purpose,
                    examples, whether they&apos;re required, and retention.
                  </caption>
                  <thead className="bg-bone">
                    <tr>
                      <th
                        scope="col"
                        className="px-4 py-3 text-left font-semibold text-ink"
                      >
                        Category
                      </th>
                      <th
                        scope="col"
                        className="px-4 py-3 text-left font-semibold text-ink"
                      >
                        Purpose
                      </th>
                      <th
                        scope="col"
                        className="px-4 py-3 text-left font-semibold text-ink"
                      >
                        Examples
                      </th>
                      <th
                        scope="col"
                        className="px-4 py-3 text-left font-semibold text-ink"
                      >
                        Required?
                      </th>
                      <th
                        scope="col"
                        className="px-4 py-3 text-left font-semibold text-ink"
                      >
                        Retention
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-mist">
                    <tr>
                      <th
                        scope="row"
                        className="px-4 py-3 text-left font-medium text-ink"
                      >
                        Strictly necessary
                      </th>
                      <td className="px-4 py-3 text-ink-soft">
                        Auth, CSRF protection, cart
                      </td>
                      <td className="px-4 py-3 text-ink-soft">
                        <code className="font-mono text-xs">session</code>,{' '}
                        <code className="font-mono text-xs">csrf_token</code>
                      </td>
                      <td className="px-4 py-3 text-ink-soft">
                        Yes (can&apos;t be disabled)
                      </td>
                      <td className="px-4 py-3 tabular-nums text-ink-soft">
                        Session / 24h
                      </td>
                    </tr>
                    <tr>
                      <th
                        scope="row"
                        className="px-4 py-3 text-left font-medium text-ink"
                      >
                        Functional
                      </th>
                      <td className="px-4 py-3 text-ink-soft">Preferences</td>
                      <td className="px-4 py-3 text-ink-soft">
                        <code className="font-mono text-xs">currency</code>,{' '}
                        <code className="font-mono text-xs">language</code>
                      </td>
                      <td className="px-4 py-3 text-ink-soft">No</td>
                      <td className="px-4 py-3 tabular-nums text-ink-soft">
                        30 days
                      </td>
                    </tr>
                    <tr>
                      <th
                        scope="row"
                        className="px-4 py-3 text-left font-medium text-ink"
                      >
                        Analytics
                      </th>
                      <td className="px-4 py-3 text-ink-soft">Usage metrics</td>
                      <td className="px-4 py-3 text-ink-soft">
                        (none currently)
                      </td>
                      <td className="px-4 py-3 text-ink-soft">No</td>
                      <td className="px-4 py-3 tabular-nums text-ink-soft">
                        N/A
                      </td>
                    </tr>
                    <tr>
                      <th
                        scope="row"
                        className="px-4 py-3 text-left font-medium text-ink"
                      >
                        Marketing
                      </th>
                      <td className="px-4 py-3 text-ink-soft">
                        Ads, retargeting
                      </td>
                      <td className="px-4 py-3 text-ink-soft">(none)</td>
                      <td className="px-4 py-3 text-ink-soft">No</td>
                      <td className="px-4 py-3 tabular-nums text-ink-soft">
                        N/A
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <h3 className="mt-8 font-display text-lg font-semibold text-ink">
                Strictly necessary
              </h3>
              <p className="mt-3 text-ink-soft">
                These keep the basics working. The{' '}
                <code className="rounded bg-mist px-1.5 py-0.5 font-mono text-sm">
                  session
                </code>{' '}
                cookie remembers you&apos;re signed in so you don&apos;t
                have to log in on every page. It expires when you sign
                out or after 24 hours of inactivity, whichever comes
                first. The{' '}
                <code className="rounded bg-mist px-1.5 py-0.5 font-mono text-sm">
                  csrf_token
                </code>{' '}
                cookie protects forms — checkout, profile edits, order
                actions — from cross-site request forgery, a common type
                of web attack. It&apos;s rotated for every request and
                cleared when the session ends.
              </p>
              <p className="mt-3 text-ink-soft">
                Your cart is stored on your device — in{' '}
                <code className="rounded bg-mist px-1.5 py-0.5 font-mono text-sm">
                  localStorage
                </code>
                {' '}rather than a cookie — so it survives a page refresh
                and persists between visits until you check out or clear
                it. We don&apos;t need consent for strictly necessary
                cookies because the app literally cannot run without
                them, and the law recognises that. If you disable them in
                your browser, you&apos;ll be signed out and the app
                won&apos;t work.
              </p>

              <h3 className="mt-6 font-display text-lg font-semibold text-ink">
                Functional
              </h3>
              <p className="mt-3 text-ink-soft">
                These remember your preferences — currency, language,
                last-used delivery address, and similar non-essential
                choices that improve the experience but aren&apos;t
                required for the app to work. For example, if you set
                the currency to Indian Rupees on one visit, the{' '}
                <code className="rounded bg-mist px-1.5 py-0.5 font-mono text-sm">
                  currency
                </code>
                {' '}cookie keeps it on Rupees the next time you visit
                so you don&apos;t have to set it again.
              </p>
              <p className="mt-3 text-ink-soft">
                You can turn functional cookies off without breaking
                anything; the app will simply forget what you picked and
                use sensible defaults next time you visit. We don&apos;t
                share functional-cookie data with anyone, and we
                don&apos;t use it to build a profile of you.
              </p>

              <h3 className="mt-6 font-display text-lg font-semibold text-ink">
                Analytics
              </h3>
              <p className="mt-3 text-ink-soft">
                We don&apos;t use analytics cookies today. Nothing on our
                site is tracking your page views, click paths, or session
                duration for product analytics. If we add a
                privacy-friendly tool later — Plausible or PostHog with
                cookieless mode, for example — we&apos;ll list it in the
                table above, update this policy, and ask for your consent
                via the banner before it loads. We won&apos;t turn it on
                quietly.
              </p>

              <h3 className="mt-6 font-display text-lg font-semibold text-ink">
                Marketing
              </h3>
              <p className="mt-3 text-ink-soft">
                We don&apos;t use marketing, advertising, or retargeting
                cookies at all. No ad-network pixels, no Facebook Pixel,
                no Google Ads tags, no LinkedIn Insight tag. If you see
                a Fe3dr ad somewhere, it&apos;s a normal display ad —
                we&apos;re not following you around the web. If that ever
                changes, we&apos;ll update this policy and ask for
                explicit, separate consent first.
              </p>
            </article>

            {/* 3. Third-party cookies */}
            <article id="third-party">
              <h2 className="font-display text-2xl font-semibold text-ink">
                3. Third-party cookies
              </h2>
              <p className="mt-4 text-ink-soft">
                When you check out, our payment processors load on the
                page and set their own cookies for fraud prevention and
                to keep your payment session secure. We don&apos;t set
                these cookies, we can&apos;t read them, and we
                don&apos;t control how long they last — they&apos;re
                governed entirely by the payment provider&apos;s own
                policy. We&apos;re telling you about them because they
                set on our domain context, and we&apos;d rather you knew
                what each one is for.
              </p>

              <Card variant="default" padding="md" className="mt-6">
                <ul className="space-y-4">
                  <li>
                    <h3 className="font-medium text-ink">Razorpay</h3>
                    <p className="mt-1 text-sm text-ink-soft">
                      Sets cookies during checkout in India for fraud
                      detection and payment-session continuity.
                    </p>
                    <a
                      href="https://razorpay.com/cookie-policy/"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="mt-2 inline-flex items-center gap-1 text-sm font-medium text-herb hover:underline"
                    >
                      Razorpay Cookie Policy
                      <ExternalLink
                        aria-hidden="true"
                        className="h-3.5 w-3.5"
                      />
                      <span className="sr-only">(opens in a new tab)</span>
                    </a>
                  </li>
                  <li className="border-t border-mist pt-4">
                    <h3 className="font-medium text-ink">Stripe</h3>
                    <p className="mt-1 text-sm text-ink-soft">
                      Sets cookies during checkout for international cards
                      and Stripe-routed payments — used for fraud signals
                      and to maintain the secure payment session.
                    </p>
                    <a
                      href="https://stripe.com/cookies-policy/legal"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="mt-2 inline-flex items-center gap-1 text-sm font-medium text-herb hover:underline"
                    >
                      Stripe Cookie Policy
                      <ExternalLink
                        aria-hidden="true"
                        className="h-3.5 w-3.5"
                      />
                      <span className="sr-only">(opens in a new tab)</span>
                    </a>
                  </li>
                </ul>
              </Card>

              <p className="mt-4 text-sm text-ink-muted">
                Payment-processor cookies are required for checkout to
                work. If you block them in your browser, you won&apos;t
                be able to pay. We use Razorpay for cards and UPI
                originating in India and Stripe for international cards
                — only one set of these cookies loads in any single
                checkout, depending on which provider routes your
                payment. We never share your cookie data, browsing
                history, or any other personal data with these providers
                beyond what the payment itself requires.
              </p>
            </article>

            {/* 4. How to manage cookies */}
            <article id="manage">
              <h2 className="font-display text-2xl font-semibold text-ink">
                4. How to manage cookies
              </h2>
              <p className="mt-4 text-ink-soft">
                You have two ways to control cookies on Fe3dr: our own
                cookie banner (the simplest option for most people) and
                your browser&apos;s built-in settings (the strongest
                option, because it overrides ours). You can use either
                or both.
              </p>

              <h3 className="mt-6 font-display text-lg font-semibold text-ink">
                Our cookie banner
              </h3>
              <p className="mt-3 text-ink-soft">
                The first time you visit, you&apos;ll see a banner at the
                bottom of the page asking which categories you allow.
                There&apos;s no pre-checked &quot;Accept all&quot; — every
                non-essential category starts off, and you have to opt
                in. You can accept everything, reject all non-essential
                cookies, or pick category by category. You can change
                your mind any time from the &quot;Cookie
                preferences&quot; link in our footer; the new choice
                takes effect immediately.
              </p>

              <h3 className="mt-6 font-display text-lg font-semibold text-ink">
                Your browser settings
              </h3>
              <p className="mt-3 text-ink-soft">
                Every browser lets you view, block, or delete cookies for
                specific sites or all sites. Look for &quot;Privacy&quot;
                or &quot;Cookies and site data&quot; in your settings.
                Browser-level controls override anything you set in our
                banner, so they&apos;re the strongest option. Useful
                guides for the major browsers:
              </p>
              <ul className="mt-3 space-y-2 text-ink-soft">
                <li className="flex items-baseline gap-2">
                  <span aria-hidden="true" className="text-herb">
                    •
                  </span>
                  <a
                    href="https://support.google.com/chrome/answer/95647"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="font-medium text-herb hover:underline"
                  >
                    Chrome
                    <span className="sr-only"> (opens in a new tab)</span>
                  </a>
                </li>
                <li className="flex items-baseline gap-2">
                  <span aria-hidden="true" className="text-herb">
                    •
                  </span>
                  <a
                    href="https://support.mozilla.org/en-US/kb/cookies-information-websites-store-on-your-computer"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="font-medium text-herb hover:underline"
                  >
                    Firefox
                    <span className="sr-only"> (opens in a new tab)</span>
                  </a>
                </li>
                <li className="flex items-baseline gap-2">
                  <span aria-hidden="true" className="text-herb">
                    •
                  </span>
                  <a
                    href="https://support.apple.com/en-in/guide/safari/sfri11471/mac"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="font-medium text-herb hover:underline"
                  >
                    Safari
                    <span className="sr-only"> (opens in a new tab)</span>
                  </a>
                </li>
                <li className="flex items-baseline gap-2">
                  <span aria-hidden="true" className="text-herb">
                    •
                  </span>
                  <a
                    href="https://support.microsoft.com/en-us/microsoft-edge/delete-cookies-in-microsoft-edge-63947406-40ac-c3b8-57b9-2a946a29ae09"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="font-medium text-herb hover:underline"
                  >
                    Edge
                    <span className="sr-only"> (opens in a new tab)</span>
                  </a>
                </li>
              </ul>
              <p className="mt-4 text-sm text-ink-muted">
                Blocking strictly necessary cookies will sign you out and
                stop the cart from working. Blocking payment-processor
                cookies will stop checkout. If you&apos;ve set your
                browser to clear cookies whenever it closes, you&apos;ll
                see the cookie banner again on your next visit because
                your previous choice was wiped along with everything
                else — that&apos;s normal, not a bug.
              </p>
            </article>

            {/* 5. Withdrawing consent */}
            <article id="withdraw">
              <h2 className="font-display text-2xl font-semibold text-ink">
                5. Withdrawing consent
              </h2>
              <p className="mt-4 text-ink-soft">
                Under the Digital Personal Data Protection Act 2023,
                withdrawing consent has to be as easy as giving it.
                We&apos;ve built our cookie controls to match that. To
                withdraw consent: open &quot;Cookie preferences&quot; in
                the footer, switch off the categories you no longer want,
                and save. The change applies straight away. We&apos;ll
                delete any non-essential cookies the next time you load a
                page, and we won&apos;t set new ones in those categories.
              </p>
              <p className="mt-4 text-ink-soft">
                Withdrawing consent doesn&apos;t affect anything
                we&apos;ve already done lawfully under your earlier
                consent — it just stops us setting new non-essential
                cookies from that point on. Strictly necessary cookies
                stay, because the app can&apos;t work without them, and
                consent isn&apos;t the legal basis for those (the law
                permits them on a different ground).
              </p>
              <p className="mt-4 text-ink-soft">
                If you want every cookie gone — including strictly
                necessary ones — sign out, then clear cookies for
                fe3dr.com from your browser settings. That will fully
                reset everything tied to your device.
              </p>
            </article>

            {/* 6. Changes to this policy */}
            <article id="changes">
              <h2 className="font-display text-2xl font-semibold text-ink">
                6. Changes to this policy
              </h2>
              <p className="mt-4 text-ink-soft">
                If we change the cookies we use — adding analytics, for
                example, or switching payment providers — we&apos;ll
                update this page and bump the &quot;Last updated&quot;
                date at the top. If the change is material — a new
                category, a new third party getting data, a new purpose —
                we&apos;ll show the banner again and ask for fresh
                consent before any new cookie loads.
              </p>
              <p className="mt-4 text-ink-soft">
                For small clarifications — fixing a typo, rewording a
                paragraph, splitting a long section — we&apos;ll update
                the date but won&apos;t bother you with a fresh banner.
                If you want to know exactly what changed, you can
                compare versions by date; we keep the previous version
                on file for at least 24 months after a change, in case
                someone needs to verify what they consented to.
              </p>
            </article>

            {/* 7. Contact (Grievance Officer) */}
            <article id="contact">
              <h2 className="font-display text-2xl font-semibold text-ink">
                7. Contact
              </h2>
              <p className="mt-4 text-ink-soft">
                Questions about cookies, or want to file a complaint about
                how we handle them? Reach our Grievance Officer.
              </p>
              {/* TODO: replace [Grievance Officer Name] and phone with the real appointed person before public launch (DPDP §13 requirement) */}
              <Card variant="filled" padding="md" className="mt-6">
                <dl className="space-y-3 text-sm">
                  <div className="flex flex-wrap gap-x-3">
                    <dt className="font-medium text-ink">
                      Grievance Officer:
                    </dt>
                    <dd className="text-ink-soft">[Grievance Officer Name]</dd>
                  </div>
                  <div className="flex flex-wrap gap-x-3">
                    <dt className="font-medium text-ink">Email:</dt>
                    <dd>
                      <a
                        href="mailto:grievance@homechef.in"
                        className="inline-flex items-center gap-1 font-medium text-herb hover:underline"
                      >
                        <Mail aria-hidden="true" className="h-3.5 w-3.5" />
                        grievance@homechef.in
                      </a>
                    </dd>
                  </div>
                  <div className="flex flex-wrap gap-x-3">
                    <dt className="font-medium text-ink">Response time:</dt>
                    <dd className="text-ink-soft">
                      Within 7 working days, per DPDP Act §13.
                    </dd>
                  </div>
                </dl>
              </Card>
            </article>

            {/* Cross-links */}
            <nav
              aria-label="Related legal pages"
              className="border-t border-mist pt-8"
            >
              <h2 className="font-display text-lg font-semibold text-ink">
                Related
              </h2>
              <ul className="mt-4 space-y-2 text-ink-soft">
                <li>
                  <Link
                    to="/privacy"
                    className="font-medium text-herb hover:underline"
                  >
                    Privacy Policy
                  </Link>
                  <span className="ml-2 text-sm text-ink-muted">
                    — what data we collect and how we use it
                  </span>
                </li>
                <li>
                  <Link
                    to="/terms"
                    className="font-medium text-herb hover:underline"
                  >
                    Terms of Service
                  </Link>
                  <span className="ml-2 text-sm text-ink-muted">
                    — the rules for using Fe3dr
                  </span>
                </li>
              </ul>
            </nav>
          </div>
        </div>
      </section>
    </div>
  );
}
