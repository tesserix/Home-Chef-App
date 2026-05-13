import { Link } from 'react-router-dom';
import {
  ShieldAlert,
  Mail,
  Phone,
  Scale,
  UserCheck,
  Baby,
  AlertTriangle,
  Clock,
  Globe2,
  FileText,
} from 'lucide-react';

/**
 * PrivacyPolicyPage
 *
 * Customer-facing Privacy Policy aligned to India's Digital Personal Data
 * Protection Act, 2023 (DPDP Act). Plain language, active voice, headings
 * every ~200 words, callouts for high-stakes clauses (Grievance Officer,
 * Your rights, Children's data).
 *
 * This is a DRAFT. Legal counsel must review before production rollout.
 * Placeholders (in square brackets) require sign-off by the legal entity
 * before launch.
 */
export default function PrivacyPolicyPage() {
  const lastUpdated = '13 May 2026';
  const readingTime = '14 min read';

  return (
    <div className="min-h-screen bg-paper">
      {/* DRAFT banner */}
      <div
        role="status"
        aria-live="polite"
        className="bg-amber-50 border-b border-amber-200"
      >
        <div className="container-app py-3">
          <div className="flex items-start gap-3">
            <AlertTriangle
              aria-hidden="true"
              className="h-5 w-5 flex-shrink-0 text-amber-700 mt-0.5"
            />
            <div className="text-sm text-amber-900">
              <strong className="font-semibold">DRAFT — not yet in force.</strong>{' '}
              This Privacy Policy is under legal review and will become
              effective only after counsel approval and a public notice. Do
              not rely on it as a current legal document. Placeholders in
              square brackets are pending finalisation.
            </div>
          </div>
        </div>
      </div>

      <article className="container-app py-12 lg:py-16">
        <div className="mx-auto max-w-3xl">
          {/* Header */}
          <header className="mb-10 border-b border-mist pb-8">
            <p className="text-sm text-ink-muted font-medium tracking-wide uppercase">
              Legal
            </p>
            <h1 className="mt-2 font-display text-display-lg text-ink">
              Privacy Policy
            </h1>
            <div className="mt-4 flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-ink-muted">
              <span className="inline-flex items-center gap-1.5">
                <Clock aria-hidden="true" className="h-4 w-4" />
                Last updated: {lastUpdated}
              </span>
              <span className="inline-flex items-center gap-1.5">
                <FileText aria-hidden="true" className="h-4 w-4" />
                {readingTime}
              </span>
            </div>
          </header>

          {/* Plain-language summary */}
          <section
            aria-labelledby="summary"
            className="mb-12 rounded-lg border border-herb/30 bg-herb-tint/30 p-6"
          >
            <h2
              id="summary"
              className="text-lg font-semibold text-ink mb-2"
            >
              In one paragraph
            </h2>
            <p className="text-ink leading-relaxed">
              Here&apos;s what this page covers, in one paragraph. We collect the
              minimum personal data we need to deliver food: your name, contact
              details, delivery address, and payment information. We don&apos;t
              sell your data. We share it only with the partners who help us
              run the service — payment processors, cloud hosts, communication
              providers — and only for the purpose we collected it. You can
              ask us for a copy, fix mistakes, or delete your account at any
              time. If something goes wrong, our Grievance Officer responds
              within 15 days.
            </p>
          </section>

          {/* Table of contents */}
          <nav
            aria-label="On this page"
            className="mb-12 rounded-lg border border-mist bg-bone p-5"
          >
            <p className="text-sm font-semibold text-ink mb-3">On this page</p>
            <ol className="grid gap-1.5 sm:grid-cols-2 text-sm text-ink list-decimal list-inside">
              <li><a href="#who-we-are" className="hover:text-herb underline-offset-2 hover:underline">Who we are</a></li>
              <li><a href="#what-we-collect" className="hover:text-herb underline-offset-2 hover:underline">What we collect</a></li>
              <li><a href="#why-we-collect" className="hover:text-herb underline-offset-2 hover:underline">Why we collect it</a></li>
              <li><a href="#who-we-share-with" className="hover:text-herb underline-offset-2 hover:underline">Who we share it with</a></li>
              <li><a href="#where-stored" className="hover:text-herb underline-offset-2 hover:underline">Where it&apos;s stored</a></li>
              <li><a href="#retention" className="hover:text-herb underline-offset-2 hover:underline">How long we keep it</a></li>
              <li><a href="#your-rights" className="hover:text-herb underline-offset-2 hover:underline">Your rights</a></li>
              <li><a href="#exercise-rights" className="hover:text-herb underline-offset-2 hover:underline">How to exercise your rights</a></li>
              <li><a href="#cookies" className="hover:text-herb underline-offset-2 hover:underline">Cookies and tracking</a></li>
              <li><a href="#security" className="hover:text-herb underline-offset-2 hover:underline">Security</a></li>
              <li><a href="#breach" className="hover:text-herb underline-offset-2 hover:underline">Breach notification</a></li>
              <li><a href="#children" className="hover:text-herb underline-offset-2 hover:underline">Children&apos;s data</a></li>
              <li><a href="#changes" className="hover:text-herb underline-offset-2 hover:underline">Changes to this policy</a></li>
              <li><a href="#grievance" className="hover:text-herb underline-offset-2 hover:underline">Grievance Officer</a></li>
            </ol>
          </nav>

          {/* SECTION 1 — Who we are */}
          <section id="who-we-are" aria-labelledby="who-we-are-h" className="mb-12">
            <h2 id="who-we-are-h" className="font-display text-display-sm text-ink mb-4">
              1. Who we are
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              <strong>[Legal Entity Name]</strong> (a company registered under
              the Companies Act 2013, with registered office at{' '}
              <strong>[Address]</strong>, CIN <strong>[CIN]</strong>) operates
              Home Chef. In this policy we call ourselves &quot;we,&quot;
              &quot;us,&quot; or Home Chef. We call you &quot;you.&quot;
            </p>
            <p className="text-ink leading-relaxed mb-4">
              Under the Digital Personal Data Protection Act, 2023 (the{' '}
              <strong>DPDP Act</strong>), we are the{' '}
              <strong>Data Fiduciary</strong>. That means we decide why and
              how your personal data is processed. You are the{' '}
              <strong>Data Principal</strong> — the person the data is about.
            </p>
            <p className="text-ink leading-relaxed">
              When we say <strong>Personal Data</strong>, we mean any
              information that identifies you or could identify you — your
              name, your phone number, your address, your face in a profile
              photo. When we say <strong>Processing</strong>, we mean anything
              we do with that data: collecting it, storing it, using it,
              sharing it, or deleting it.
            </p>
          </section>

          {/* SECTION 2 — What we collect */}
          <section id="what-we-collect" aria-labelledby="what-we-collect-h" className="mb-12">
            <h2 id="what-we-collect-h" className="font-display text-display-sm text-ink mb-4">
              2. What we collect
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              We collect different categories of personal data depending on
              how you use Home Chef. We collect only what we need.
            </p>

            <h3 className="text-lg font-semibold text-ink mt-6 mb-2">
              When you create an account
            </h3>
            <ul className="list-disc pl-6 space-y-1.5 text-ink leading-relaxed">
              <li>Your full name</li>
              <li>Your email address</li>
              <li>Your mobile phone number</li>
              <li>A password (we store only a one-way hash, never the password itself)</li>
              <li>Date of birth, so we can confirm you are 18 or older</li>
            </ul>

            <h3 className="text-lg font-semibold text-ink mt-6 mb-2">
              When you sign in with Google, Facebook, or Apple
            </h3>
            <p className="text-ink leading-relaxed">
              We receive your name, email, profile photo, and a unique account
              identifier from the provider. We do not receive your password.
              The provider&apos;s own privacy policy also applies to that data.
            </p>

            <h3 className="text-lg font-semibold text-ink mt-6 mb-2">
              When you place an order
            </h3>
            <ul className="list-disc pl-6 space-y-1.5 text-ink leading-relaxed">
              <li>Your delivery address and any access instructions you add</li>
              <li>The items you order and your dietary notes to the chef</li>
              <li>Your payment instrument — card, UPI ID, or wallet — in tokenised form (we never store your full card number or CVV)</li>
              <li>The location coordinates of your delivery address, so the driver can find it</li>
            </ul>

            <h3 className="text-lg font-semibold text-ink mt-6 mb-2">
              When you message a chef or driver
            </h3>
            <p className="text-ink leading-relaxed">
              We store the text of in-app messages so chefs can read special
              requests and so we can investigate complaints. We do not record
              voice calls placed through the masking service unless you tell
              us a call needs investigating.
            </p>

            <h3 className="text-lg font-semibold text-ink mt-6 mb-2">
              When you use the app
            </h3>
            <ul className="list-disc pl-6 space-y-1.5 text-ink leading-relaxed">
              <li>Device type, operating system, app version</li>
              <li>IP address and approximate location based on it</li>
              <li>The pages you view and the buttons you tap (event-level usage data)</li>
              <li>Crash reports, if the app crashes</li>
            </ul>

            <p className="text-ink leading-relaxed mt-6">
              <strong>We do not collect</strong> your contacts, photos,
              microphone, or precise GPS location unless you grant a permission
              for a specific feature (for example, location at checkout to find
              chefs near you).
            </p>
          </section>

          {/* SECTION 3 — Why we collect it */}
          <section id="why-we-collect" aria-labelledby="why-we-collect-h" className="mb-12">
            <h2 id="why-we-collect-h" className="font-display text-display-sm text-ink mb-4">
              3. Why we collect it
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              We collect each piece of data for a specific reason. Here are
              the purposes, one per line.
            </p>
            <ul className="list-disc pl-6 space-y-2 text-ink leading-relaxed">
              <li>We use your name and email to create your account and let you sign in.</li>
              <li>We use your phone number to send order updates by SMS — when the chef accepts, when the driver is on the way, when the food arrives.</li>
              <li>We use your delivery address to route your order to the right place.</li>
              <li>We use your payment details to charge you for orders and to refund you when needed.</li>
              <li>We use your order history to show you past orders, suggest dishes you might like, and help our chefs improve their menus.</li>
              <li>We use your device and app-usage data to fix bugs and improve the app.</li>
              <li>We use your IP address and login activity to detect fraud — for example, sudden sign-in from a new country.</li>
              <li>We use your messages with chefs and drivers to investigate complaints if you raise one.</li>
              <li>We use your email to send service announcements you cannot opt out of (security alerts, policy changes, receipts).</li>
              <li>We use your email and phone to send marketing only if you opt in. You can withdraw that consent any time.</li>
            </ul>
            <p className="text-ink leading-relaxed mt-4">
              We do not process your personal data for any purpose other than
              the ones listed above. If we ever need to add a new purpose, we
              will ask for your consent first.
            </p>
          </section>

          {/* SECTION 4 — Who we share with */}
          <section id="who-we-share-with" aria-labelledby="who-we-share-h" className="mb-12">
            <h2 id="who-we-share-h" className="font-display text-display-sm text-ink mb-4">
              4. Who we share it with
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              We do not sell your personal data. We share it only with the
              partners who help us run Home Chef, and only for the purpose
              we collected it.
            </p>

            <div className="overflow-x-auto -mx-4 sm:mx-0">
              <table className="min-w-full text-sm text-left text-ink border-t border-b border-mist">
                <thead className="bg-bone text-xs uppercase text-ink-muted">
                  <tr>
                    <th scope="col" className="px-4 py-3 font-semibold">Recipient</th>
                    <th scope="col" className="px-4 py-3 font-semibold">What they receive</th>
                    <th scope="col" className="px-4 py-3 font-semibold">Why</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-mist">
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Razorpay (India)</td>
                    <td className="px-4 py-3 align-top">Name, email, phone, payment instrument, order total</td>
                    <td className="px-4 py-3 align-top">Process card / UPI / netbanking payments and refunds</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Stripe (United States)</td>
                    <td className="px-4 py-3 align-top">Name, email, payment instrument, order total</td>
                    <td className="px-4 py-3 align-top">Process international cards. Data leaves India. See section 5.</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Home chef</td>
                    <td className="px-4 py-3 align-top">Your first name, delivery address, dietary notes, order items</td>
                    <td className="px-4 py-3 align-top">So the chef can prepare and hand off your order</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Delivery driver</td>
                    <td className="px-4 py-3 align-top">Your first name, delivery address, masked phone number</td>
                    <td className="px-4 py-3 align-top">So the driver can deliver and contact you if needed</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Google Cloud (India region)</td>
                    <td className="px-4 py-3 align-top">All data stored in our databases and file storage</td>
                    <td className="px-4 py-3 align-top">Host our application servers and databases</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">SendGrid / AWS SES</td>
                    <td className="px-4 py-3 align-top">Your email and the content of transactional and marketing emails</td>
                    <td className="px-4 py-3 align-top">Deliver email reliably</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">SMS aggregator (India)</td>
                    <td className="px-4 py-3 align-top">Your phone number and the SMS content</td>
                    <td className="px-4 py-3 align-top">Send OTPs and order updates</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Firebase (Google)</td>
                    <td className="px-4 py-3 align-top">Device token, your name</td>
                    <td className="px-4 py-3 align-top">Send push notifications about your order</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">OpenPanel / PostHog</td>
                    <td className="px-4 py-3 align-top">Anonymised event data (pages, taps, crashes)</td>
                    <td className="px-4 py-3 align-top">Analyse how the app is used. Disabled if you opt out of analytics.</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Government, courts, regulators</td>
                    <td className="px-4 py-3 align-top">Whatever they lawfully require</td>
                    <td className="px-4 py-3 align-top">Comply with valid orders, summons, or tax obligations</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <p className="text-ink leading-relaxed mt-6">
              Every partner above is bound by a written contract that limits
              their use of your data to the purpose we share it for. They are
              <strong> Data Processors</strong> acting on our instructions —
              not separate fiduciaries.
            </p>
          </section>

          {/* SECTION 5 — Where stored / cross-border */}
          <section id="where-stored" aria-labelledby="where-stored-h" className="mb-12">
            <h2 id="where-stored-h" className="font-display text-display-sm text-ink mb-4">
              5. Where it&apos;s stored
            </h2>
            <div className="flex items-start gap-3 mb-4">
              <Globe2 aria-hidden="true" className="h-5 w-5 text-herb mt-1" />
              <p className="text-ink leading-relaxed">
                We store the bulk of your personal data on servers physically
                located in Mumbai, India, operated by Google Cloud.
              </p>
            </div>
            <p className="text-ink leading-relaxed mb-4">
              Some processing takes place outside India:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-ink leading-relaxed">
              <li>If you pay with an international card, Stripe processes that payment in the United States. Stripe is bound by its own privacy commitments and by our processor agreement.</li>
              <li>SendGrid and AWS SES may route email through servers in the United States or Europe.</li>
              <li>Crash reports and anonymised analytics may be processed in the United States.</li>
            </ul>
            <p className="text-ink leading-relaxed mt-4">
              The DPDP Act permits transfer of personal data outside India
              unless the Central Government restricts a specific country
              (Section 16). We monitor those notifications and will stop
              transfers if a country is restricted.
            </p>
          </section>

          {/* SECTION 6 — Retention */}
          <section id="retention" aria-labelledby="retention-h" className="mb-12">
            <h2 id="retention-h" className="font-display text-display-sm text-ink mb-4">
              6. How long we keep it
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              We keep your personal data only as long as we need it for the
              purpose we collected it, plus any period the law requires us to
              hold it.
            </p>
            <div className="overflow-x-auto -mx-4 sm:mx-0">
              <table className="min-w-full text-sm text-left text-ink border-t border-b border-mist">
                <thead className="bg-bone text-xs uppercase text-ink-muted">
                  <tr>
                    <th scope="col" className="px-4 py-3 font-semibold">Category</th>
                    <th scope="col" className="px-4 py-3 font-semibold">Retention period</th>
                    <th scope="col" className="px-4 py-3 font-semibold">Why</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-mist">
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Active account data</td>
                    <td className="px-4 py-3 align-top">While your account is open</td>
                    <td className="px-4 py-3 align-top">To run the service</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Order and payment records</td>
                    <td className="px-4 py-3 align-top">8 years from the order date</td>
                    <td className="px-4 py-3 align-top">Income Tax Act 1961 record-keeping rules</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Food-safety related complaints</td>
                    <td className="px-4 py-3 align-top">3 years from resolution</td>
                    <td className="px-4 py-3 align-top">FSSAI traceability obligations</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Address and delivery history</td>
                    <td className="px-4 py-3 align-top">3 years from last order</td>
                    <td className="px-4 py-3 align-top">Service improvement and fraud detection</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">In-app messages</td>
                    <td className="px-4 py-3 align-top">1 year from the conversation</td>
                    <td className="px-4 py-3 align-top">Complaint investigation window</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Analytics events</td>
                    <td className="px-4 py-3 align-top">25 months</td>
                    <td className="px-4 py-3 align-top">Year-over-year comparison</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 align-top font-medium">Closed-account data</td>
                    <td className="px-4 py-3 align-top">Deleted within 90 days, except records the law requires us to keep</td>
                    <td className="px-4 py-3 align-top">DPDP §8(7) storage limitation</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p className="text-ink leading-relaxed mt-6">
              Once the period ends, we delete the data or anonymise it so it
              can no longer be linked back to you.
            </p>
          </section>

          {/* SECTION 7 — Your rights (CALLOUT) */}
          <section id="your-rights" aria-labelledby="your-rights-h" className="mb-12">
            <h2 id="your-rights-h" className="font-display text-display-sm text-ink mb-4">
              7. Your rights
            </h2>
            <div className="rounded-lg border-l-4 border-herb bg-herb-tint/40 p-6 mb-6">
              <div className="flex items-start gap-3">
                <UserCheck
                  aria-hidden="true"
                  className="h-6 w-6 text-herb mt-1 flex-shrink-0"
                />
                <div>
                  <h3 className="text-lg font-semibold text-ink mb-2">
                    What the DPDP Act gives you
                  </h3>
                  <p className="text-ink leading-relaxed mb-4">
                    As a Data Principal, you have five rights under Sections
                    11 to 14 of the DPDP Act. You can use any of them, free
                    of charge, at any time.
                  </p>
                  <ol className="space-y-3 text-ink list-decimal list-inside leading-relaxed">
                    <li>
                      <strong>Right to access.</strong> You can ask us for a
                      summary of the personal data we hold about you and who
                      we&apos;ve shared it with.
                    </li>
                    <li>
                      <strong>Right to correction and erasure.</strong> You can
                      ask us to fix data that is wrong and to delete data we
                      no longer need.
                    </li>
                    <li>
                      <strong>Right to grievance redressal.</strong> You can
                      complain to our Grievance Officer if we mishandle your
                      data, and we&apos;ll respond within 15 days.
                    </li>
                    <li>
                      <strong>Right to nominate.</strong> You can nominate
                      another person to exercise your rights if you die or
                      become permanently incapacitated.
                    </li>
                    <li>
                      <strong>Right to withdraw consent.</strong> You can
                      withdraw any consent you gave us. It will not affect
                      processing we did before you withdrew.
                    </li>
                  </ol>
                </div>
              </div>
            </div>
            <p className="text-ink leading-relaxed">
              If we cannot honour a request — for example, because the law
              requires us to keep certain records — we will tell you why and
              point you to the law.
            </p>
          </section>

          {/* SECTION 8 — How to exercise rights */}
          <section id="exercise-rights" aria-labelledby="exercise-h" className="mb-12">
            <h2 id="exercise-h" className="font-display text-display-sm text-ink mb-4">
              8. How to exercise your rights
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              You can use most rights from inside the app:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-ink leading-relaxed">
              <li>Go to <strong>Settings &gt; Privacy</strong> to download your data, correct your profile, or delete your account.</li>
              <li>Go to <strong>Settings &gt; Communications</strong> to opt in or out of marketing email and SMS.</li>
              <li>Go to <strong>Settings &gt; Nominate</strong> to add a nominee.</li>
            </ul>
            <p className="text-ink leading-relaxed mt-6 mb-4">
              You can also email our Grievance Officer (see section 14). We
              will verify your identity before acting on a request, to make
              sure we don&apos;t hand your data to the wrong person.
            </p>
            <p className="text-ink leading-relaxed mb-4">
              We respond within <strong>15 days</strong> for grievances and
              within <strong>30 days</strong> for access or erasure requests.
              If we need more time for a complex request, we&apos;ll tell you
              and explain why.
            </p>
            <p className="text-ink leading-relaxed">
              If you withdraw consent for processing we rely on to provide
              the service (for example, to charge your card), we may have to
              suspend or close your account. We will tell you the consequence
              before you withdraw, so you can decide.
            </p>
          </section>

          {/* SECTION 9 — Cookies */}
          <section id="cookies" aria-labelledby="cookies-h" className="mb-12">
            <h2 id="cookies-h" className="font-display text-display-sm text-ink mb-4">
              9. Cookies and tracking
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              We use cookies and similar technologies to keep you signed in,
              remember your preferences, and measure how the app is used. We
              ask for your consent the first time you visit, and you can
              change your choice any time from the cookie banner footer.
            </p>
            <p className="text-ink leading-relaxed">
              For the full list of cookies and how to control them, see our{' '}
              <Link
                to="/cookies"
                className="text-herb underline underline-offset-2 hover:text-herb-dark"
              >
                Cookie Policy
              </Link>
              .
            </p>
          </section>

          {/* SECTION 10 — Security */}
          <section id="security" aria-labelledby="security-h" className="mb-12">
            <h2 id="security-h" className="font-display text-display-sm text-ink mb-4">
              10. Security
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              We put reasonable safeguards in place to protect your personal
              data, in line with Section 8(5) of the DPDP Act. These include:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-ink leading-relaxed">
              <li>Encryption of data in transit (TLS 1.2 or higher)</li>
              <li>Encryption of sensitive data at rest using managed keys</li>
              <li>Access controls — only the staff who need data to do their job can see it</li>
              <li>Audit logs of who accessed what</li>
              <li>Independent security reviews and penetration tests on a regular cadence</li>
              <li>Vendor security reviews before we add a new processor</li>
            </ul>
            <p className="text-ink leading-relaxed mt-6">
              No system is perfect, and we do not promise that data is 100%
              secure. We promise to act quickly, openly, and in good faith if
              something goes wrong.
            </p>
          </section>

          {/* SECTION 11 — Breach notification */}
          <section id="breach" aria-labelledby="breach-h" className="mb-12">
            <h2 id="breach-h" className="font-display text-display-sm text-ink mb-4">
              11. Breach notification
            </h2>
            <div className="flex items-start gap-3 mb-4">
              <ShieldAlert aria-hidden="true" className="h-5 w-5 text-amber-700 mt-1" />
              <p className="text-ink leading-relaxed">
                If we discover a personal data breach that could harm you, we
                will notify you and the Data Protection Board of India without
                undue delay. We will tell you what happened, what data was
                involved, what we&apos;re doing about it, and what you can do
                to protect yourself.
              </p>
            </div>
            <p className="text-ink leading-relaxed">
              We won&apos;t hide breaches or downplay them. That commitment is
              part of our DPDP §8(6) duty as Data Fiduciary.
            </p>
          </section>

          {/* SECTION 12 — Children */}
          <section id="children" aria-labelledby="children-h" className="mb-12">
            <h2 id="children-h" className="font-display text-display-sm text-ink mb-4">
              12. Children&apos;s data
            </h2>
            <div className="rounded-lg border-l-4 border-amber-500 bg-amber-50 p-6">
              <div className="flex items-start gap-3">
                <Baby aria-hidden="true" className="h-6 w-6 text-amber-700 mt-1 flex-shrink-0" />
                <div>
                  <h3 className="text-lg font-semibold text-ink mb-2">
                    We do not knowingly collect data from minors under 18
                  </h3>
                  <p className="text-ink leading-relaxed mb-3">
                    Home Chef is intended for adults. Under Section 9 of the
                    DPDP Act, processing a child&apos;s personal data requires
                    verifiable parental consent, and we do not have a parental
                    consent mechanism today.
                  </p>
                  <p className="text-ink leading-relaxed mb-3">
                    You confirm you are 18 or older when you create an
                    account. If we discover a user is under 18, we delete
                    their account and all their personal data within 14 days,
                    unless the law requires us to keep specific records.
                  </p>
                  <p className="text-ink leading-relaxed">
                    If you are a parent and believe a child has signed up,
                    please contact our Grievance Officer.
                  </p>
                </div>
              </div>
            </div>
          </section>

          {/* SECTION 13 — Changes */}
          <section id="changes" aria-labelledby="changes-h" className="mb-12">
            <h2 id="changes-h" className="font-display text-display-sm text-ink mb-4">
              13. Changes to this policy
            </h2>
            <p className="text-ink leading-relaxed mb-4">
              We update this policy when our processing changes, when the law
              changes, or when a partner changes. We keep an archive of every
              past version.
            </p>
            <p className="text-ink leading-relaxed mb-4">
              For minor changes (typo fixes, clearer wording, new partner of
              the same category), we update the &quot;Last updated&quot; date
              at the top of this page.
            </p>
            <p className="text-ink leading-relaxed">
              For material changes (new purpose, new category of partner, new
              country your data goes to), we notify you by email and in the
              app at least <strong>30 days</strong> before the change takes
              effect. If you do not agree, you can close your account before
              that date.
            </p>
          </section>

          {/* SECTION 14 — Grievance Officer (CALLOUT) */}
          <section id="grievance" aria-labelledby="grievance-h" className="mb-12">
            <h2 id="grievance-h" className="font-display text-display-sm text-ink mb-4">
              14. Grievance Officer
            </h2>
            <div className="rounded-lg border border-herb bg-herb-tint/50 p-6 shadow-1">
              <div className="flex items-start gap-3 mb-4">
                <Scale aria-hidden="true" className="h-6 w-6 text-herb mt-1 flex-shrink-0" />
                <div>
                  <h3 className="text-lg font-semibold text-ink">
                    Contact our Grievance Officer
                  </h3>
                  <p className="text-sm text-ink-muted mt-1">
                    Appointed under Section 10(2) of the DPDP Act and Rule 3
                    of the IT (Intermediary Guidelines) Rules 2021.
                  </p>
                </div>
              </div>
              <dl className="grid gap-y-3 sm:grid-cols-[160px_1fr] text-ink">
                <dt className="font-semibold">Name</dt>
                <dd>[Grievance Officer Name]</dd>

                <dt className="font-semibold">Designation</dt>
                <dd>Grievance Officer, [Legal Entity Name]</dd>

                <dt className="font-semibold inline-flex items-center gap-1.5">
                  <Mail aria-hidden="true" className="h-4 w-4" /> Email
                </dt>
                <dd>
                  <a
                    href="mailto:grievance@homechef.in"
                    className="text-herb underline underline-offset-2 hover:text-herb-dark"
                  >
                    grievance@homechef.in
                  </a>
                </dd>

                <dt className="font-semibold inline-flex items-center gap-1.5">
                  <Phone aria-hidden="true" className="h-4 w-4" /> Phone
                </dt>
                <dd>[+91 XXXXX XXXXX] (Mon–Fri, 10:00–18:00 IST)</dd>

                <dt className="font-semibold">Postal address</dt>
                <dd>[Registered Office Address]</dd>

                <dt className="font-semibold">Response time</dt>
                <dd>Acknowledgement within 24 hours, resolution within 15 days</dd>
              </dl>
              <p className="text-sm text-ink-muted mt-4">
                If we don&apos;t resolve your grievance, you can complain to
                the Data Protection Board of India under Section 28 of the
                DPDP Act.
              </p>
            </div>
          </section>

          {/* Related documents */}
          <section
            aria-labelledby="related"
            className="border-t border-mist pt-8 mt-12"
          >
            <h2 id="related" className="text-sm font-semibold text-ink-muted uppercase tracking-wide mb-4">
              Related documents
            </h2>
            <ul className="grid gap-2 sm:grid-cols-3 text-ink">
              <li>
                <Link
                  to="/terms"
                  className="inline-flex items-center gap-2 text-herb underline underline-offset-2 hover:text-herb-dark"
                >
                  <FileText aria-hidden="true" className="h-4 w-4" />
                  Terms of Service
                </Link>
              </li>
              <li>
                <Link
                  to="/cookies"
                  className="inline-flex items-center gap-2 text-herb underline underline-offset-2 hover:text-herb-dark"
                >
                  <FileText aria-hidden="true" className="h-4 w-4" />
                  Cookie Policy
                </Link>
              </li>
              <li>
                <Link
                  to="/refund"
                  className="inline-flex items-center gap-2 text-herb underline underline-offset-2 hover:text-herb-dark"
                >
                  <FileText aria-hidden="true" className="h-4 w-4" />
                  Refund Policy
                </Link>
              </li>
            </ul>
          </section>
        </div>
      </article>
    </div>
  );
}
