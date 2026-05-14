import { Link } from 'react-router-dom';
import {
  AlertTriangle,
  Clock,
  CreditCard,
  HelpCircle,
  Info,
  ScrollText,
} from 'lucide-react';
import { Card } from '@/shared/components/ui';

/**
 * Refund and Cancellation Policy — customer-facing legal page.
 *
 * Plain-language refund and cancellation policy for the Fe3dr platform.
 * Aligns with:
 * - RBI Payment Aggregator Master Direction (17 Mar 2020), §8 — refund timeline
 * - Consumer Protection (E-Commerce) Rules 2020, Rule 5(3)(g) — disclosure
 * - Consumer Protection Act 2019 — grievance redressal
 *
 * Style: STYLE-GUIDE.md §5 (legal-page tone) — short sentences, active voice,
 * "we"/"you" voice, callouts for refund clauses, bold first-use defined terms.
 */
export default function RefundPolicyPage() {
  return (
    <div className="min-h-screen bg-paper">
      {/* DRAFT banner */}
      <div
        role="status"
        className="border-b border-amber-300 bg-amber-50 px-4 py-2 text-center text-sm font-medium text-amber-900"
      >
        <AlertTriangle
          aria-hidden="true"
          className="mr-2 inline-block h-4 w-4 align-text-bottom"
        />
        Draft — this policy is under review. Final wording may change before
        launch.
      </div>

      <article className="container-app mx-auto max-w-3xl px-4 py-12 lg:py-16">
        {/* Header */}
        <header className="mb-10 border-b border-mist pb-8">
          <p className="font-mono text-xs uppercase tracking-wider text-ink-soft">
            Legal
          </p>
          <h1 className="mt-2 font-display text-display-lg text-ink">
            Refund and Cancellation Policy
          </h1>
          <p className="mt-3 text-sm text-ink-soft">
            Last updated: 13 May 2026 · Effective: 13 May 2026
          </p>
          <p className="mt-4 text-sm text-ink-soft">
            Fe3dr is a product of <strong>Tesserix Pty Ltd</strong> (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia. Day-to-day operations are conducted from Mumbai, India and Sydney, Australia.
          </p>
        </header>

        {/* Plain-language summary callout */}
        <Card
          variant="filled"
          padding="md"
          className="mb-10 border-l-4 border-herb bg-herb-tint/40"
        >
          <div className="flex gap-3">
            <Info
              aria-hidden="true"
              className="mt-1 h-5 w-5 flex-shrink-0 text-herb"
            />
            <div>
              <p className="font-display text-lg font-semibold text-ink">
                The short version
              </p>
              <p className="mt-2 text-ink-soft">
                Here's how refunds work. You can cancel for free before the chef
                starts cooking. After that, refund eligibility depends on the
                order stage. All refunds go back to your original payment
                method within 7 working days.
              </p>
            </div>
          </div>
        </Card>

        {/* Cancellation eligibility table — high-prominence callout */}
        <section aria-labelledby="cancellation-matrix" className="mb-12">
          <Card
            variant="outlined"
            padding="md"
            className="border-2 border-ink/10"
          >
            <div className="mb-4 flex items-center gap-2">
              <ScrollText
                aria-hidden="true"
                className="h-5 w-5 text-herb"
              />
              <h2
                id="cancellation-matrix"
                className="font-display text-xl font-semibold text-ink"
              >
                Cancellation eligibility at a glance
              </h2>
            </div>
            <p className="mb-4 text-sm text-ink-soft">
              Your refund depends on the <strong>order stage</strong> when you
              cancel. Each order moves through these stages from placement to
              delivery.
            </p>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b-2 border-ink/15">
                    <th
                      scope="col"
                      className="py-2 pr-4 font-semibold text-ink"
                    >
                      Order stage
                    </th>
                    <th
                      scope="col"
                      className="py-2 pr-4 font-semibold text-ink"
                    >
                      Can you cancel?
                    </th>
                    <th
                      scope="col"
                      className="py-2 pr-4 font-semibold text-ink"
                    >
                      Refund amount
                    </th>
                    <th scope="col" className="py-2 font-semibold text-ink">
                      Timeline
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-mist [&_td]:py-3 [&_td]:pr-4 [&_td]:align-top">
                  <tr>
                    <td className="font-medium text-ink">
                      Placed — chef hasn't accepted yet
                    </td>
                    <td className="text-ink-soft">Yes, instantly</td>
                    <td className="font-medium tabular-nums text-herb">
                      100%
                    </td>
                    <td className="text-ink-soft tabular-nums">
                      Within 7 working days
                    </td>
                  </tr>
                  <tr>
                    <td className="font-medium text-ink">
                      Accepted — chef hasn't started cooking
                    </td>
                    <td className="text-ink-soft">Yes</td>
                    <td className="font-medium tabular-nums text-herb">
                      100% (rare: minus ingredient cost — see below)
                    </td>
                    <td className="text-ink-soft tabular-nums">
                      Within 7 working days
                    </td>
                  </tr>
                  <tr>
                    <td className="font-medium text-ink">
                      Preparing — chef is cooking
                    </td>
                    <td className="text-ink-soft">
                      Not for the food. Delivery fee only if order isn't
                      delivered.
                    </td>
                    <td className="font-medium tabular-nums text-ink">
                      Delivery fee only
                    </td>
                    <td className="text-ink-soft tabular-nums">
                      Within 7 working days
                    </td>
                  </tr>
                  <tr>
                    <td className="font-medium text-ink">
                      Out for delivery
                    </td>
                    <td className="text-ink-soft">
                      No — but contact support if something's wrong
                    </td>
                    <td className="font-medium tabular-nums text-ink">
                      Case-by-case
                    </td>
                    <td className="text-ink-soft tabular-nums">
                      Within 7 working days if approved
                    </td>
                  </tr>
                  <tr>
                    <td className="font-medium text-ink">
                      Delivered but unsatisfactory
                    </td>
                    <td className="text-ink-soft">
                      Raise a complaint within 24 hours
                    </td>
                    <td className="font-medium tabular-nums text-ink">
                      Case-by-case (partial or full)
                    </td>
                    <td className="text-ink-soft tabular-nums">
                      Within 7 working days of approval
                    </td>
                  </tr>
                  <tr>
                    <td className="font-medium text-ink">
                      Never delivered
                    </td>
                    <td className="text-ink-soft">
                      Auto-refund after confirmation
                    </td>
                    <td className="font-medium tabular-nums text-herb">
                      100%
                    </td>
                    <td className="text-ink-soft tabular-nums">
                      Within 7 working days
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p className="mt-4 text-xs text-ink-soft">
              "Working days" excludes weekends and bank holidays in India.
              Times start when your refund is approved by our team or the chef.
            </p>
          </Card>
        </section>

        {/* Section: When you can cancel */}
        <section aria-labelledby="when-you-cancel" className="mb-10">
          <h2
            id="when-you-cancel"
            className="font-display text-2xl font-semibold text-ink"
          >
            When you can cancel
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              We tie refunds to the <strong>order stage</strong> because home
              chefs cook to order. Once a chef puts ingredients into a pan,
              they can't unbuy them.
            </p>
            <p>
              <strong>Before the chef accepts.</strong> Cancel from the order
              detail screen. You get 100% back. Most orders sit at this stage
              for under 5 minutes, so act fast.
            </p>
            <p>
              <strong>After acceptance, before cooking starts.</strong> You can
              still cancel for a full refund. In rare cases — for example, a
              custom or pre-shopped catering-style dish — the chef may have
              already bought ingredients. If so, we'll show you the ingredient
              cost before you confirm the cancellation, and refund the rest.
            </p>
            <p>
              <strong>Once "Preparing" begins.</strong> We can't refund the
              food cost. The chef is cooking your food and can't sell it to
              someone else. If the order isn't delivered, we still refund the
              delivery fee.
            </p>
            <p>
              <strong>Once "Out for delivery".</strong> You can't cancel from
              the app. If something's wrong — chef sent the wrong dish, food
              was unsafe to eat, driver lost the order — open a support ticket
              from the order screen.
            </p>
            <p>
              <strong>What "stage" means in practice.</strong> Every order
              moves through the same fixed stages: Placed, Accepted, Preparing,
              Ready, Out for delivery, Delivered. You can see your current
              stage on the order detail screen. We pin that stage to your
              refund eligibility, so there's never any guesswork about what
              you'll get back. The cancel button hides itself once you cross
              the refund threshold — it's not playing tricks; it's just being
              honest that the chef has already started.
            </p>
            <p>
              If you tap a cancel option after the cutoff, the app shows a
              short message: "Order cannot be cancelled at this stage." That
              isn't us being rigid. It's us telling you the chef has started
              cooking. From there, the path forward is a support ticket — not
              a cancellation.
            </p>
          </div>
        </section>

        {/* Section: When chefs cancel */}
        <section aria-labelledby="chef-cancel" className="mb-10">
          <h2
            id="chef-cancel"
            className="font-display text-2xl font-semibold text-ink"
          >
            When chefs cancel
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              Chefs can decline or cancel an order in a few situations: they're
              out of ingredients, they're closing earlier than expected, or a
              kitchen emergency makes the order unsafe to complete.
            </p>
            <p>
              When a chef cancels — at any stage — we issue a 100% automatic
              refund to your <strong>original payment method</strong>. You
              don't need to ask. We send an email, an in-app message, and a
              push notification with the refund amount and timeline.
            </p>
            <p>
              <strong>Force majeure.</strong> Sometimes a chef can't deliver
              because of events outside anyone's control — severe weather,
              power cuts, civic disruptions, sudden illness, traffic
              shutdowns, or a delivery partner going offline mid-route. We
              treat these the same as a chef cancellation: full automatic
              refund within 7 working days, plus an apology message in the
              order timeline. We don't hide behind "force majeure" to avoid
              refunding you — it's a label we use to explain why your food
              didn't arrive, not a reason to keep your money.
            </p>
            <p>
              <strong>If the chef cancels after you've already paid extra.</strong>{' '}
              Sometimes you pay a tip, a priority fee, or a special-instructions
              upcharge before the chef accepts. If the chef then cancels, the
              refund covers everything — food, taxes, delivery, tip, every
              add-on. You should never see a partial refund just because a
              chef called it off.
            </p>
          </div>
        </section>

        {/* Section: How to request a refund — callout */}
        <section aria-labelledby="request-refund" className="mb-10">
          <Card
            variant="filled"
            padding="md"
            className="border-l-4 border-herb bg-herb-tint/30"
          >
            <div className="flex gap-3">
              <HelpCircle
                aria-hidden="true"
                className="mt-1 h-5 w-5 flex-shrink-0 text-herb"
              />
              <div className="flex-1">
                <h2
                  id="request-refund"
                  className="font-display text-xl font-semibold text-ink"
                >
                  How to request a refund
                </h2>
                <ol className="mt-3 list-decimal space-y-2 pl-5 text-ink-soft">
                  <li>
                    Go to <strong>My Orders</strong> in the menu.
                  </li>
                  <li>Open the order you want to dispute.</li>
                  <li>
                    Tap <strong>Need help?</strong> at the bottom of the order
                    screen.
                  </li>
                  <li>
                    Choose <strong>Refund request</strong> and tell us what
                    went wrong. Add photos if you have them.
                  </li>
                  <li>
                    We respond within 48 hours and resolve refund disputes
                    within 7 working days, in line with Consumer Protection
                    (E-Commerce) Rules 2020.
                  </li>
                </ol>
                <p className="mt-3 text-sm text-ink-soft">
                  Raise a complaint within 24 hours of delivery so the chef and
                  driver can recall the order details accurately.
                </p>
              </div>
            </div>
          </Card>
        </section>

        {/* Section: How long refunds take — callout */}
        <section aria-labelledby="refund-timeline" className="mb-10">
          <Card
            variant="outlined"
            padding="md"
            className="border-l-4 border-ink/30"
          >
            <div className="flex gap-3">
              <Clock
                aria-hidden="true"
                className="mt-1 h-5 w-5 flex-shrink-0 text-ink-soft"
              />
              <div>
                <h2
                  id="refund-timeline"
                  className="font-display text-xl font-semibold text-ink"
                >
                  How long refunds take
                </h2>
                <p className="mt-2 text-ink-soft">
                  We process every refund within{' '}
                  <strong>7 working days</strong> from the moment we approve
                  it. This is the maximum window set by the Reserve Bank of
                  India's Payment Aggregator Master Direction (17 March 2020),
                  §8.
                </p>
                <p className="mt-2 text-ink-soft">
                  Most refunds land in 3 to 5 working days. The exact day
                  depends on your bank or wallet, not on us. We send the money
                  to your payment gateway (Razorpay or Stripe) on day 1; from
                  there, it follows your bank's processing cycle. We can't
                  speed up your bank, but we can show you the gateway's
                  reference number if you ask.
                </p>
              </div>
            </div>
          </Card>
        </section>

        {/* Section: Refund channel */}
        <section aria-labelledby="refund-channel" className="mb-10">
          <h2
            id="refund-channel"
            className="font-display text-2xl font-semibold text-ink"
          >
            Where the refund goes
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              Every refund goes back to your{' '}
              <strong>original payment method</strong>. This is a Reserve Bank
              of India rule for payment aggregators, and it protects you from
              fraud.
            </p>
            <ul className="list-disc space-y-2 pl-5">
              <li>
                Paid by UPI? Refund returns to the same UPI ID and bank
                account.
              </li>
              <li>
                Paid by credit or debit card? Refund returns to the same card.
                If the card is lost or replaced, the issuer redirects the
                refund to your new card.
              </li>
              <li>
                Paid by net banking? Refund returns to the same bank account.
              </li>
              <li>
                Paid by a wallet (Paytm, PhonePe, Amazon Pay)? Refund returns
                to that wallet balance.
              </li>
            </ul>
            <p>
              We can't redirect a refund to a different account. If your bank
              account is closed, contact your bank — they'll route the refund
              to your replacement account. We have no way to bypass this; it
              prevents money laundering and protects the platform against
              card-replay fraud.
            </p>
            <p>
              <strong>Currency.</strong> All refunds are issued in Indian
              rupees (INR). If you paid using a card issued outside India, the
              card network handles the currency conversion back to your home
              currency, often at a different rate from the one used at
              checkout. We can't compensate for foreign-exchange differences —
              that's between you and your card issuer.
            </p>
          </div>
        </section>

        {/* Section: Partial refunds */}
        <section aria-labelledby="partial-refunds" className="mb-10">
          <h2
            id="partial-refunds"
            className="font-display text-2xl font-semibold text-ink"
          >
            Partial refunds
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              Sometimes only part of your order is wrong — one dish out of
              three is missing, or the rice is cold but the curry is fine. We
              refund per item, not per order.
            </p>
            <p>
              When you raise a partial-refund request, tell us which items are
              affected. We refund the item's full cost plus its share of taxes
              and the delivery fee.
            </p>
            <p>
              <strong>Worked example.</strong> Your order is ₹500: chicken
              biryani ₹300, raita ₹50, gulab jamun ₹100, delivery ₹50. The
              gulab jamun is missing. We refund ₹100 for the dish plus its
              proportional share of taxes — typically ₹105 to ₹110 total. The
              rest of the order stays paid.
            </p>
            <p>
              If half or more of your order is unusable, raise the issue and
              we'll usually refund the full order — including delivery — rather
              than nickel-and-dime the calculation.
            </p>
            <p>
              <strong>What we look at before approving.</strong> For partial
              refunds we ask for a short description and, where it makes
              sense, a photo. Photos help us close the loop with the chef and
              improve their menu — they're not a hoop you have to jump
              through. If photographing your food feels weird (we get it), a
              clear written description is fine for most situations.
            </p>
            <p>
              We never refund a dish because you didn't like the taste. Food
              is subjective, and chefs work hard on their recipes. Our refund
              path covers things that went measurably wrong — missing items,
              wrong items, spoiled food, food-safety issues, temperature
              issues at delivery, or quantity well below what was advertised.
            </p>
          </div>
        </section>

        {/* Section: Catering orders */}
        <section aria-labelledby="catering-orders" className="mb-10">
          <h2
            id="catering-orders"
            className="font-display text-2xl font-semibold text-ink"
          >
            Catering orders
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              Catering orders need more notice than everyday meals because
              chefs shop and prep ahead. The cancellation window is set by the
              chef when they accept your quote, and shown on the catering
              order screen.
            </p>
            <p>Common chef policies:</p>
            <ul className="list-disc space-y-2 pl-5">
              <li>
                <strong>More than 72 hours before the event:</strong> 100%
                refund.
              </li>
              <li>
                <strong>48 to 72 hours before:</strong> 50% refund (chef has
                shopped, can resell some ingredients).
              </li>
              <li>
                <strong>Less than 48 hours before:</strong> No refund (chef
                has prepped, lost the booking date).
              </li>
            </ul>
            <p>
              The chef's exact policy appears on every catering quote you
              accept. If a chef cancels a catering booking, we refund 100%
              within 7 working days and help you find another chef.
            </p>
            <p>
              <strong>Headcount changes.</strong> For catering, the chef cooks
              for the number of guests you confirmed. If your guest count
              drops after the chef has shopped, the original quote stands —
              we can't refund the food the chef already bought. If your guest
              count goes up, contact the chef as early as possible. They may
              or may not be able to scale up depending on their kitchen
              capacity.
            </p>
            <p>
              <strong>Date changes.</strong> One date change is usually
              possible if you request it more than 72 hours before the event,
              and the chef has availability. After 72 hours, a date change is
              treated as a cancellation of the original booking, with refund
              eligibility based on the timing rule above. There is no
              automatic rebooking — you'll need to accept a fresh quote for
              the new date.
            </p>
          </div>
        </section>

        {/* Section: Chargebacks */}
        <section aria-labelledby="chargebacks" className="mb-10">
          <h2
            id="chargebacks"
            className="font-display text-2xl font-semibold text-ink"
          >
            Chargebacks
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              A chargeback is when you ask your bank to reverse a payment.
              It's your right under your card network's rules (Visa,
              Mastercard, RuPay) and the Reserve Bank of India's customer
              protection framework.
            </p>
            <p>
              We'd rather you talk to us first. Most refund disputes resolve
              faster through our support flow than through a chargeback, which
              can take 30 to 90 days while your bank investigates.
            </p>
            <p>
              If you do file a chargeback, here's what happens:
            </p>
            <ul className="list-disc space-y-2 pl-5">
              <li>
                Our payment partner (Razorpay or Stripe) notifies us.
              </li>
              <li>
                We share order records, delivery confirmation, and chat logs
                with your bank.
              </li>
              <li>
                Your bank decides — usually within 45 days — whether the
                chargeback is valid.
              </li>
              <li>
                If valid, the money returns to your card. If not, the original
                charge stands.
              </li>
            </ul>
            <p>
              We never penalise customers for filing genuine chargebacks. But
              repeated chargebacks on orders that were genuinely delivered may
              lead us to close your account, in line with our Terms of Service.
            </p>
          </div>
        </section>

        {/* Section: Payment failures and auto-reversals */}
        <section aria-labelledby="payment-failures" className="mb-10">
          <h2
            id="payment-failures"
            className="font-display text-2xl font-semibold text-ink"
          >
            Payment failures and auto-reversals
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              Sometimes a payment fails after the money has already left your
              bank — usually a network timeout between your bank, the payment
              gateway, and our servers. Your statement shows a debit, but the
              order didn't go through.
            </p>
            <p>
              When this happens, the payment gateway auto-reverses the
              transaction without us doing anything. The money returns to your
              original payment method within 5 to 7 working days, per the
              Reserve Bank of India's framework for failed transactions. You
              don't need to file a refund request — the reversal is automatic.
            </p>
            <p>
              If you see a debit on your statement but no order in{' '}
              <strong>My Orders</strong>, give it 7 working days. If the
              reversal hasn't appeared after that, contact support with your
              bank statement reference and we'll trace it.
            </p>
          </div>
        </section>

        {/* Section: Non-refundable items */}
        <section aria-labelledby="not-refundable" className="mb-10">
          <h2
            id="not-refundable"
            className="font-display text-2xl font-semibold text-ink"
          >
            What's not refundable
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              We try not to hide the no-refund list at the bottom of a
              policy. Here it is up front, so you know where the limits are
              before you order:
            </p>
            <ul className="list-disc space-y-2 pl-5">
              <li>
                <strong>Taste preference.</strong> "Too spicy", "not enough
                salt", "I didn't enjoy it" — these are subjective and aren't
                grounds for a refund. Use the rating system to give feedback.
              </li>
              <li>
                <strong>Delays under 30 minutes.</strong> Home cooking and
                last-mile delivery aren't perfectly predictable. Delays under
                30 minutes from the original ETA aren't refundable. Beyond
                30 minutes, contact support.
              </li>
              <li>
                <strong>Tips you've already paid.</strong> Once a driver
                accepts your tip, it's theirs. We can't claw it back unless
                the delivery never happened.
              </li>
              <li>
                <strong>Promotional discounts and credits.</strong> If you
                used a coupon or platform credit, the refund covers the
                amount you actually paid in cash, not the discount value. The
                coupon may or may not return to your account depending on its
                terms.
              </li>
              <li>
                <strong>Orders delivered to the wrong address you entered.</strong>{' '}
                If the chef and driver delivered to the address you typed,
                that's a successful delivery. We can't refund a delivery that
                worked correctly.
              </li>
              <li>
                <strong>Self-pickup orders you didn't pick up.</strong> If you
                chose self-pickup and didn't show up within the pickup window,
                the food is wasted and the order is not refundable.
              </li>
            </ul>
            <p>
              These limits aren't about being strict. They're about treating
              chefs and drivers fairly — they did their part of the deal, and
              they deserve to be paid.
            </p>
          </div>
        </section>

        {/* Section: Cash on delivery */}
        <section aria-labelledby="cod" className="mb-10">
          <h2
            id="cod"
            className="font-display text-2xl font-semibold text-ink"
          >
            Cash on delivery
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              Cash orders are simpler. If you cancel before the chef accepts,
              there's nothing to refund — no money has changed hands. If the
              chef cancels at any stage of a cash order, again, nothing to
              refund.
            </p>
            <p>
              If something goes wrong after a cash delivery — wrong dish,
              missing item, food-safety issue — we credit your platform
              wallet for the refund amount instead of the original cash
              payment, because we don't ship cash through the app. You can
              use the wallet credit on your next order or request a bank
              transfer through support.
            </p>
          </div>
        </section>

        {/* Section: Disputes / Grievance Officer */}
        <section aria-labelledby="disputes" className="mb-10">
          <h2
            id="disputes"
            className="font-display text-2xl font-semibold text-ink"
          >
            If a refund doesn't arrive
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              Refunds that don't appear within 7 working days are rare but
              possible — usually a bank routing issue. If your refund is late:
            </p>
            <ol className="list-decimal space-y-2 pl-5">
              <li>
                Check your bank statement carefully. Refunds sometimes appear
                with the original transaction's date, not the refund date.
              </li>
              <li>
                Open the original order in <strong>My Orders</strong>, tap{' '}
                <strong>Need help?</strong> and ask for the refund reference
                number.
              </li>
              <li>
                Share the reference number with your bank. They can trace the
                refund in their system.
              </li>
            </ol>
            <p>
              If we still can't resolve it, escalate to our{' '}
              <strong>Grievance Officer</strong> — required for every Indian
              e-commerce platform under Consumer Protection (E-Commerce)
              Rules 2020, Rule 5(3):
            </p>
            <Card
              variant="outlined"
              padding="sm"
              className="not-prose border border-mist bg-bone"
            >
              <dl className="text-sm">
                <div className="grid grid-cols-[110px_1fr] gap-y-1">
                  <dt className="font-medium text-ink">Name:</dt>
                  <dd className="text-ink-soft">
                    Grievance Officer, Fe3dr
                  </dd>
                  <dt className="font-medium text-ink">Email:</dt>
                  <dd className="text-ink-soft">grievance@homechef.in</dd>
                  <dt className="font-medium text-ink">Response:</dt>
                  <dd className="text-ink-soft">
                    Within 48 hours of your complaint
                  </dd>
                  <dt className="font-medium text-ink">Resolution:</dt>
                  <dd className="text-ink-soft">
                    Within 30 days, per Rule 5(3)
                  </dd>
                </div>
              </dl>
            </Card>
            <p>
              You can also file a complaint with the National Consumer
              Helpline (1915) or the e-Daakhil portal if our resolution
              doesn't satisfy you. We won't take that personally — that's
              what those channels exist for.
            </p>
            <p>
              <strong>Documents to keep handy</strong> when you contact the
              Grievance Officer: your order ID (in the format{' '}
              <span className="font-mono tabular-nums">#HC-2026-XXXXXXXX</span>),
              the payment reference shown on the order screen, your bank
              statement showing the debit, and any chat or email
              correspondence with the chef. Sharing these up front shortens
              the resolution by days.
            </p>
          </div>
        </section>

        {/* Section: Changes to this policy */}
        <section aria-labelledby="changes" className="mb-10">
          <h2
            id="changes"
            className="font-display text-2xl font-semibold text-ink"
          >
            Changes to this policy
          </h2>
          <div className="mt-4 space-y-4 text-ink-soft">
            <p>
              We update this policy when our refund flows change, when
              regulations change, or when we spot a confusing line. The
              "Last updated" date at the top of this page always reflects the
              current version.
            </p>
            <p>
              For material changes — anything that affects how much you get
              back, how long it takes, or how you ask — we'll tell you in the
              app and by email at least 14 days before the change takes effect.
              Orders placed before the change keep the older policy.
            </p>
            <p>
              Smaller changes — fixing a typo, clarifying a sentence,
              tightening a list — go in without a notice. We won't sneak
              substantive changes through under cover of "minor edits". If
              you're ever unsure whether a change affects you, ask support
              and we'll explain plainly.
            </p>
            <p>
              <strong>Older versions.</strong> We keep an archive of past
              versions of this policy for at least three years. If you're
              looking at an old order and want to know which rules applied at
              the time, email{' '}
              <span className="font-mono">legal@homechef.in</span> and we'll
              send the relevant version.
            </p>
          </div>
        </section>

        {/* Cross-links footer */}
        <footer className="mt-12 border-t border-mist pt-8">
          <div className="flex flex-wrap items-center gap-2 text-sm text-ink-soft">
            <CreditCard aria-hidden="true" className="h-4 w-4" />
            <span>Related:</span>
            <Link
              to="/terms"
              className="font-medium text-herb underline-offset-4 hover:underline"
            >
              Terms of Service
            </Link>
            <span aria-hidden="true">·</span>
            <Link
              to="/privacy"
              className="font-medium text-herb underline-offset-4 hover:underline"
            >
              Privacy Policy
            </Link>
            <span aria-hidden="true">·</span>
            <Link
              to="/help"
              className="font-medium text-herb underline-offset-4 hover:underline"
            >
              Help Centre
            </Link>
          </div>
        </footer>
      </article>
    </div>
  );
}
