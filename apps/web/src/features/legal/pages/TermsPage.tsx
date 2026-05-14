import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { AlertTriangle, Info, ShieldAlert, Clock } from 'lucide-react';

/**
 * Terms of Service — Fe3dr customer web app.
 *
 * DRAFT. This page is plain-language working copy intended for legal review.
 * It is not legal advice and is not binding until reviewed and signed off by
 * a licensed advocate qualified to practise in India.
 */
export default function TermsPage() {
  useEffect(() => {
    document.title = 'Terms of Service — Fe3dr';
  }, []);

  return (
    <div className="min-h-screen bg-paper">
      <article className="mx-auto max-w-3xl px-6 py-12 text-ink">
        {/* DRAFT banner */}
        <div
          role="note"
          aria-label="Draft notice"
          className="mb-8 flex items-start gap-3 rounded-md border border-amber-300 bg-amber-tint/40 p-4 text-sm text-ink"
        >
          <AlertTriangle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-amber" />
          <div>
            <p className="font-semibold">DRAFT — under legal review.</p>
            <p className="mt-1 text-ink-soft">
              This document has not been reviewed by a licensed advocate and is not binding. We
              are publishing it for transparency while review is in progress. Last updated:
              13 May 2026.
            </p>
          </div>
        </div>

        {/* Title block */}
        <header className="mb-8">
          <h1 className="font-display text-display-md text-ink">Terms of service</h1>
          <p className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-ink-muted">
            <span>Last updated: 13 May 2026</span>
            <span aria-hidden="true">·</span>
            <span className="inline-flex items-center gap-1">
              <Clock aria-hidden="true" className="h-4 w-4" />8 minute read
            </span>
            <span aria-hidden="true">·</span>
            <span>Applies to customers of Fe3dr in India</span>
          </p>
          <p className="mt-4 text-sm text-ink-soft">
            Fe3dr is a product of <strong>Tesserix Pty Ltd</strong> (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia. Day-to-day operations are conducted from Mumbai, India and Sydney, Australia.
          </p>
        </header>

        {/* Plain-language summary */}
        <section
          aria-labelledby="summary-heading"
          className="mb-10 rounded-md border border-border bg-bone p-5"
        >
          <div className="flex items-start gap-3">
            <Info aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-herb" />
            <div>
              <h2 id="summary-heading" className="text-base font-semibold text-ink">
                Here's what this page covers, in one paragraph
              </h2>
              <p className="mt-2 text-sm text-ink-soft">
                Fe3dr is a marketplace that connects you with home cooks in your city. We
                do not cook your food. When you place an order, your contract for the meal is
                with the chef, and your contract for the ride is with the driver. We hold your
                payment safely until your order is delivered, and we refund you within seven
                working days when a refund is due. You read the allergen and ingredient
                information the chef provides; you do not order what you cannot eat. If
                something goes wrong, our grievance officer is your first port of call.
              </p>
            </div>
          </div>
        </section>

        {/* Body content */}
        <div className="prose-legal space-y-10 text-base leading-relaxed text-ink-soft">
          {/* 1. Acceptance */}
          <section aria-labelledby="acceptance">
            <h2 id="acceptance" className="text-xl font-semibold text-ink">
              1. Acceptance of these terms
            </h2>
            <p className="mt-3">
              These terms form a contract between you and us. They apply every time you visit
              our website, create an account, or place an order. By using Fe3dr, you agree
              to these terms. If you do not agree, please do not use the service.
            </p>
            <p className="mt-3">
              Some sections affect important rights. The clauses on payments, refunds, food
              safety, liability, and dispute resolution change what you can recover if
              something goes wrong. Please read them with care.
            </p>
            <p className="mt-3">
              We may update these terms from time to time. We explain how, and how much notice
              you get, in section 15.
            </p>
          </section>

          {/* 2. What Fe3dr does */}
          <section aria-labelledby="what-we-do">
            <h2 id="what-we-do" className="text-xl font-semibold text-ink">
              2. What Fe3dr does
            </h2>
            <p className="mt-3">
              <strong className="font-semibold text-ink">Fe3dr</strong> is a technology
              platform. We connect customers who want home-cooked food with independent home
              cooks (each a <strong className="font-semibold text-ink">Chef</strong>) and
              independent delivery partners (each a <strong className="font-semibold text-ink">Driver</strong>).
              We list menus, take payments, and pass orders to the right chef and driver.
            </p>
            <p className="mt-3">
              We are a marketplace, not a restaurant. We do not cook your food, store it,
              package it, or carry it. The chef cooks the food in their own kitchen. The
              driver picks it up and brings it to you. We do our best to verify chefs and
              drivers before they start using the platform, and we set service standards. We
              do not control the kitchen, the cooking, or the route.
            </p>
            <p className="mt-3">
              We operate as an intermediary under the Information Technology Act, 2000. Our
              role and the protection that goes with that role is explained in section 11.
            </p>
          </section>

          {/* 3. Your account */}
          <section aria-labelledby="your-account">
            <h2 id="your-account" className="text-xl font-semibold text-ink">
              3. Your account
            </h2>
            <p className="mt-3">
              You must be at least 18 years old to use Fe3dr. If you are between 13 and 18,
              a parent or guardian must open the account and place orders for you. We do not
              knowingly serve children under 13.
            </p>
            <p className="mt-3">
              When you sign up, please give us accurate information. That includes your name,
              phone number, email, and delivery address. We rely on this information to deliver
              your food. If your details change, please update them.
            </p>
            <p className="mt-3">
              Keep your password safe. You are responsible for everything that happens through
              your account, unless you tell us someone else has used it. If you think your
              account has been compromised, change your password and email our grievance
              officer right away (see section 16).
            </p>
            <p className="mt-3">
              You may close your account at any time from your settings. We may suspend or
              close your account if you break these terms, if you place fraudulent orders, or
              if a court or regulator tells us to. We will tell you why, and we will give you a
              way to respond, unless we are barred by law from doing so.
            </p>
          </section>

          {/* 4. Placing an order */}
          <section aria-labelledby="placing-order">
            <h2 id="placing-order" className="text-xl font-semibold text-ink">
              4. Placing an order
            </h2>
            <p className="mt-3">
              When you place an{' '}
              <strong className="font-semibold text-ink">Order</strong>, you are making an
              offer to buy a meal from the chef on the terms shown at checkout. The chef may
              accept or decline. If the chef declines, we refund you in full.
            </p>
            <p className="mt-3">
              Your contract for the food is with the chef. Your contract for the delivery is
              with the driver, where the driver is independent. We act as the chef's and
              driver's collection agent for the money, and as their notice agent for messages
              about the order. The chef sets the menu, the price, and the cooking style. The
              driver chooses the route.
            </p>
            <p className="mt-3">
              Orders open when the chef opens their kitchen and close when the chef closes it.
              The chef shows a{' '}
              <strong className="font-semibold text-ink">Pickup Window</strong> on each item,
              which is the time the food will be ready for the driver to collect. Pickup
              windows are estimates and can shift if a chef is busy or if a kitchen runs late.
            </p>
            <p className="mt-3">
              We do not guarantee a specific delivery time. Where we display an expected time,
              we mean it as a typical estimate for that chef in your area, not a promise. Heavy
              traffic, weather, festivals, and chef capacity all affect actual delivery times.
            </p>
            <p className="mt-3">
              Once the order is paid for and accepted, we send you a confirmation by email and
              push notification. That confirmation, together with the order page, sets out what
              you ordered, what you paid, and the chef's allergen notes.
            </p>
          </section>

          {/* 5. Payment */}
          <section aria-labelledby="payment">
            <h2 id="payment" className="text-xl font-semibold text-ink">
              5. Payment
            </h2>
            <p className="mt-3">
              We accept payment in Indian rupees through our payment partners, currently
              Razorpay and Stripe. They are regulated payment service providers. We never store
              your full card number. The partner you use stores the card details on their
              systems.
            </p>
            <p className="mt-3">
              We collect the money for the chef and driver and hold it in escrow until the
              order is delivered. After delivery, we release the chef's share to the chef on
              their settlement cycle, and we release the driver's share to the driver.
              Settlement timings are governed by the Reserve Bank of India's Payment Aggregator
              framework.
            </p>
            <p className="mt-3">
              Prices on the menu are inclusive of the chef's cooking charges. We also charge a
              delivery fee and a small platform fee, both shown at checkout before you pay.
              Taxes such as GST are shown as a separate line, calculated as required by Indian
              tax law.
            </p>

            <Callout
              tone="info"
              title="Refund timeline"
              icon={<Info aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-herb" />}
            >
              When a refund is due, we start it within 24 hours of approval. The money usually
              reaches your account within seven working days, in line with the Reserve Bank of
              India's Payment Aggregator rules. Your bank or card network may take a little
              longer to display the credit. If a refund has not reached you after ten working
              days, contact our grievance officer.
            </Callout>

            <p className="mt-3">
              If a payment fails, no order is placed. If a payment looks suspicious, we may
              hold it for review or ask you for more information before we accept the order. We
              do this to protect you and the chef from fraud.
            </p>
          </section>

          {/* 6. Cancellations and refunds */}
          <section aria-labelledby="cancellations">
            <h2 id="cancellations" className="text-xl font-semibold text-ink">
              6. Cancellations and refunds
            </h2>
            <p className="mt-3">
              You can cancel an order any time before the chef starts cooking, and you get a
              full refund. Once the chef has begun preparing the food, the chef has the choice
              whether to cancel and how much to refund, because the ingredients have already
              been used.
            </p>
            <p className="mt-3">
              Each chef sets a clear cancellation policy that you see at checkout. By placing
              the order, you accept that policy for that order. The window where you can cancel
              for free is called the{' '}
              <strong className="font-semibold text-ink">Refund Period</strong>.
            </p>
            <p className="mt-3">
              If the chef cancels after accepting, you get a full refund. If the food is not
              delivered, you get a full refund. If the food arrives in poor condition, contact
              us within four hours of delivery with a photo and a short description, and we
              will work with the chef to put it right. The fix may be a partial refund, a full
              refund, or a credit for a future order, depending on what happened.
            </p>
            <p className="mt-3">
              For full detail on how cancellations are handled, see our{' '}
              <Link
                to="/refund"
                className="text-herb underline decoration-herb/40 underline-offset-4 hover:decoration-herb"
              >
                Refund Policy
              </Link>
              .
            </p>
          </section>

          {/* 7. Food safety and allergens */}
          <section aria-labelledby="food-safety">
            <h2 id="food-safety" className="text-xl font-semibold text-ink">
              7. Food safety and allergens
            </h2>
            <p className="mt-3">
              The chef cooks the food and is responsible for what is in it. The chef labels
              each dish with its ingredients and any common allergens (milk, eggs, peanuts,
              tree nuts, wheat, soy, fish, shellfish, sesame, sulphites, and any others the
              chef knows about).
            </p>
            <p className="mt-3">
              We facilitate that disclosure. We display what the chef tells us on the menu and
              order screen. We do not test the food. We do not inspect every kitchen for every
              order.
            </p>

            <Callout
              tone="warning"
              title="Allergens — please read before ordering"
              icon={
                <ShieldAlert aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-amber" />
              }
            >
              If you have a food allergy, an intolerance, or a religious or dietary
              restriction, please read the ingredient list on every dish before you order.
              Where you are not sure, message the chef from the menu page and ask.
              Home kitchens often handle many ingredients in the same space, so cross-contact
              is possible even when an ingredient is not listed. We are not responsible for an
              allergic reaction unless we hid information the chef gave us from you, or unless
              applicable law makes us responsible anyway.
            </Callout>

            <p className="mt-3">
              Every chef on Fe3dr must hold the food-business registration or licence
              required by the Food Safety and Standards Authority of India (FSSAI). We check
              that registration before a chef can list a menu. If you suspect a chef is
              breaching FSSAI rules, please tell us via the grievance route in section 16.
            </p>
          </section>

          {/* 8. Reviews and content */}
          <section aria-labelledby="reviews">
            <h2 id="reviews" className="text-xl font-semibold text-ink">
              8. Reviews and content
            </h2>
            <p className="mt-3">
              You can rate and review a chef after an order. Reviews help other customers and
              help chefs improve. Please keep reviews honest, based on your own experience, and
              relevant to the order.
            </p>
            <p className="mt-3">
              When you post a review, photo, or comment on Fe3dr, you keep ownership of
              what you wrote and what you uploaded. You give us a worldwide, non-exclusive,
              royalty-free licence to host, display, and share that content on Fe3dr and in
              our marketing of Fe3dr. You can ask us to take down a review you wrote by
              contacting our grievance officer.
            </p>
            <p className="mt-3">
              Do not post anything that is false, that names third parties without their
              consent, that targets a person rather than the food or service, that is unlawful,
              or that infringes someone else's rights. We may take down reviews that break
              these rules, and we may remove reviews that a court tells us to remove.
            </p>
            <p className="mt-3">
              Defamation rules apply to reviews. A false statement of fact that damages a
              chef's business may expose you to a civil claim under Indian law. Honest opinion,
              honest reporting of a poor experience, and fair criticism are protected.
            </p>
          </section>

          {/* 9. Prohibited use */}
          <section aria-labelledby="prohibited">
            <h2 id="prohibited" className="text-xl font-semibold text-ink">
              9. Prohibited use
            </h2>
            <p className="mt-3">
              When you use Fe3dr, please do not:
            </p>
            <ul className="ml-5 mt-3 list-disc space-y-2 marker:text-ink-muted">
              <li>place an order you do not intend to pay for or receive;</li>
              <li>
                pretend to be someone else, use a false name, or use a payment method that is
                not yours;
              </li>
              <li>
                resell the food, including through dark stores, cloud kitchens, or other
                marketplaces;
              </li>
              <li>
                scrape, copy, or republish menu data, prices, or photos for commercial purposes
                without our written consent;
              </li>
              <li>
                interfere with the platform, including by uploading malware, probing for
                weaknesses, or overloading our systems;
              </li>
              <li>
                harass a chef, a driver, or one of our staff. We will close accounts that send
                threatening or abusive messages;
              </li>
              <li>
                use Fe3dr to break any law, including tax law, food-safety law, and
                consumer-protection law.
              </li>
            </ul>
            <p className="mt-3">
              If we believe you have used Fe3dr for any of the above, we may pause your
              account, refuse future orders, and report you to the relevant authorities.
            </p>
          </section>

          {/* 10. Liability */}
          <section aria-labelledby="liability">
            <h2 id="liability" className="text-xl font-semibold text-ink">
              10. Liability
            </h2>
            <p className="mt-3">
              Nothing in these terms removes any right you have under Indian consumer-protection
              law, including the Consumer Protection Act, 2019 and the Consumer Protection
              (E-Commerce) Rules, 2020, that we cannot remove by contract.
            </p>

            <Callout
              tone="warning"
              title="What we can be held to"
              icon={
                <ShieldAlert aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-amber" />
              }
            >
              For everything else, our total liability to you for any single order is limited
              to the platform fees we collected from you for that order. We do not guarantee
              food quality, taste, freshness beyond what a reasonable home cook can deliver, or
              any specific delivery time. We are not liable for indirect or consequential loss,
              including missed meetings, lost income, or emotional distress, except where
              Indian law requires otherwise.
            </Callout>

            <p className="mt-3">
              We are not responsible for the chef's cooking choices, the driver's route choices,
              or any third-party service we rely on (such as payment partners, mapping
              providers, or telecom networks). Where one of these third parties is at fault, we
              will do our best to help you recover from them.
            </p>
            <p className="mt-3">
              We do our best to keep Fe3dr available. We may have to take the service down
              for maintenance, security work, or to respond to an incident. We do not promise
              the service will be uninterrupted or error-free.
            </p>
          </section>

          {/* 11. Intermediary status */}
          <section aria-labelledby="intermediary">
            <h2 id="intermediary" className="text-xl font-semibold text-ink">
              11. Intermediary status
            </h2>
            <p className="mt-3">
              We operate Fe3dr as an intermediary under section 79 of the Information
              Technology Act, 2000 and the Information Technology (Intermediary Guidelines and
              Digital Media Ethics Code) Rules, 2021. We host content that chefs, drivers, and
              customers post. We do not author menus, reviews, or driver messages, and we are
              not responsible for them in the way the chef, driver, or reviewer is.
            </p>
            <p className="mt-3">
              Chefs and drivers are independent. They are not our employees. We do not direct
              how a chef cooks or how a driver rides. They use Fe3dr as a tool. They are
              responsible for their own tax, their own insurance, and their own compliance with
              the law.
            </p>
            <p className="mt-3">
              We follow the takedown process the Intermediary Rules require. If you believe
              content on Fe3dr is unlawful, you can report it to our grievance officer (see
              section 16). We respond within the timelines those rules require.
            </p>
          </section>

          {/* 12. Governing law */}
          <section aria-labelledby="governing-law">
            <h2 id="governing-law" className="text-xl font-semibold text-ink">
              12. Governing law and jurisdiction
            </h2>
            <p className="mt-3">
              These terms are governed by the laws of New South Wales, Australia for matters relating to the company, and by the laws of India for matters relating to the platform's operations in India, food safety (FSSAI), payments (RBI), and tax (GST). Disputes are heard by the Courts of New South Wales, Australia (for matters governed by Australian law) and the courts of Mumbai, India (for matters governed by Indian law), subject to the dispute-resolution process in section 13.
            </p>
            <p className="mt-3">
              If you are a consumer, this clause does not stop you from bringing a complaint to
              a District Consumer Disputes Redressal Commission with jurisdiction over your
              place of residence, as the Consumer Protection Act, 2019 allows.
            </p>
          </section>

          {/* 13. Dispute resolution */}
          <section aria-labelledby="disputes">
            <h2 id="disputes" className="text-xl font-semibold text-ink">
              13. Dispute resolution
            </h2>
            <p className="mt-3">
              If you have a problem, please tell us first. Most problems can be sorted out
              quickly through our grievance officer (see section 16). The officer acknowledges
              your complaint within 48 hours and aims to resolve it within 30 days, in line
              with the Consumer Protection (E-Commerce) Rules, 2020.
            </p>
            <p className="mt-3">
              If the grievance officer cannot resolve your complaint, you may escalate to:
            </p>
            <ul className="ml-5 mt-3 list-disc space-y-2 marker:text-ink-muted">
              <li>
                the National Consumer Helpline (1915, or consumerhelpline.gov.in) for
                consumer-protection matters;
              </li>
              <li>
                a District Consumer Disputes Redressal Commission with jurisdiction over your
                residence;
              </li>
              <li>
                the Data Protection Board of India for matters under the Digital Personal Data
                Protection Act, 2023;
              </li>
              <li>
                the Reserve Bank of India's Integrated Ombudsman Scheme for unresolved payment
                disputes.
              </li>
            </ul>
            <p className="mt-3">
              For commercial disputes that are not consumer matters, we agree to arbitration in
              India under the Arbitration and Conciliation Act, 1996. The seat is Mumbai.
              There is a single arbitrator, appointed by mutual agreement. The language is
              English. Each side pays its own costs unless the arbitrator decides otherwise.
              This clause does not apply to a consumer dispute that you choose to take to a
              consumer forum.
            </p>
          </section>

          {/* 14. Privacy reference */}
          <section aria-labelledby="privacy-ref">
            <h2 id="privacy-ref" className="text-xl font-semibold text-ink">
              14. Privacy and cookies
            </h2>
            <p className="mt-3">
              We handle your personal data under the Digital Personal Data Protection Act,
              2023. Our{' '}
              <Link
                to="/privacy"
                className="text-herb underline decoration-herb/40 underline-offset-4 hover:decoration-herb"
              >
                Privacy Policy
              </Link>{' '}
              explains what we collect, why, how long we keep it, and how to exercise your
              rights. Our{' '}
              <Link
                to="/cookies"
                className="text-herb underline decoration-herb/40 underline-offset-4 hover:decoration-herb"
              >
                Cookie Policy
              </Link>{' '}
              explains the cookies and similar technologies we use, and how you can manage
              them.
            </p>
            <p className="mt-3">
              By using Fe3dr, you confirm that you have read the Privacy Policy. You can
              withdraw your consent for optional uses (such as marketing emails) from your
              account settings at any time. Withdrawing consent does not affect the legality of
              what we did before you withdrew it.
            </p>
          </section>

          {/* 15. Changes */}
          <section aria-labelledby="changes">
            <h2 id="changes" className="text-xl font-semibold text-ink">
              15. Changes to these terms
            </h2>
            <p className="mt-3">
              We may update these terms. When we do, we update the "last updated" date at the
              top of this page. We also keep the previous version available on request.
            </p>
            <p className="mt-3">
              For material changes — for example, a change to how we handle refunds, how
              disputes are resolved, or what fees you pay — we give you 30 days' notice by
              email and through a banner in the app. If you continue to use Fe3dr after the
              notice period, you accept the new terms. If you do not accept them, you can close
              your account.
            </p>
            <p className="mt-3">
              We do not change the terms that apply to an order you have already placed. The
              terms in force when you confirmed the order govern that order.
            </p>
          </section>

          {/* 16. Contact */}
          <section aria-labelledby="contact">
            <h2 id="contact" className="text-xl font-semibold text-ink">
              16. Contact us
            </h2>
            <p className="mt-3">
              For routine questions, please email{' '}
              <a
                href="mailto:support@homechef.in"
                className="text-herb underline decoration-herb/40 underline-offset-4 hover:decoration-herb"
              >
                support@homechef.in
              </a>
              .
            </p>
            <p className="mt-3">
              For complaints, data-protection requests, and notices that need a formal response,
              please contact our grievance officer:
            </p>
            {/* TODO: replace [Grievance Officer Name] and phone with the real appointed person before public launch (DPDP §13 requirement) */}
            <div className="mt-4 rounded-md border border-border bg-bone p-4 text-sm text-ink">
              <p className="font-semibold">Grievance officer, Fe3dr</p>
              <p className="mt-1">Email: grievance@homechef.in</p>
              <p>Postal: Tesserix Pty Ltd, New South Wales, Australia (operations: Mumbai, India and Sydney, Australia)</p>
              <p className="mt-2 text-ink-soft">
                We acknowledge complaints within 48 hours and aim to resolve them within
                30 days, in line with the Consumer Protection (E-Commerce) Rules, 2020 and the
                Digital Personal Data Protection Act, 2023.
              </p>
            </div>
          </section>
        </div>

        {/* Footer meta */}
        <footer className="mt-12 border-t border-border pt-6 text-sm text-ink-muted">
          <p>Last updated: 13 May 2026.</p>
          <p className="mt-2">
            See also our{' '}
            <Link
              to="/privacy"
              className="text-herb underline decoration-herb/40 underline-offset-4 hover:decoration-herb"
            >
              Privacy Policy
            </Link>
            ,{' '}
            <Link
              to="/refund"
              className="text-herb underline decoration-herb/40 underline-offset-4 hover:decoration-herb"
            >
              Refund Policy
            </Link>
            , and{' '}
            <Link
              to="/cookies"
              className="text-herb underline decoration-herb/40 underline-offset-4 hover:decoration-herb"
            >
              Cookie Policy
            </Link>
            .
          </p>
        </footer>
      </article>
    </div>
  );
}

interface CalloutProps {
  tone: 'info' | 'warning';
  title: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}

/**
 * High-stakes-clause callout. Used for refund timing, allergen warning, and
 * liability cap so the reader cannot miss them in a wall of body text.
 */
function Callout({ tone, title, icon, children }: CalloutProps): React.JSX.Element {
  const toneClass =
    tone === 'warning'
      ? 'border-amber-300 bg-amber-tint/30'
      : 'border-herb/30 bg-herb-tint/30';

  return (
    <aside
      role="note"
      aria-label={title}
      className={`mt-4 flex items-start gap-3 rounded-md border ${toneClass} p-4 text-sm text-ink`}
    >
      {icon}
      <div>
        <p className="font-semibold text-ink">{title}</p>
        <p className="mt-1 text-ink-soft">{children}</p>
      </div>
    </aside>
  );
}
