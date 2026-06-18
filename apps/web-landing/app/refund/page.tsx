// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import {
  LEGAL_LAST_UPDATED,
  LEGAL_OPERATOR_FULL,
  LEGAL_SUPPORT_EMAIL,
} from '@/lib/site';

export const metadata: Metadata = {
  title: 'Refund & Cancellation Policy',
  description: 'How cancellations and refunds work for Fe3dr orders.',
  alternates: { canonical: '/refund/' },
};

const SUMMARY =
  'Here is how refunds work. You can cancel for free before the chef starts cooking. After that, refund eligibility depends on the order stage. All refunds go back to your original payment method within 7 working days of approval, in line with the Reserve Bank of India Payment Aggregator Master Direction.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Cancellation eligibility by order stage',
    paragraphs: [
      `Fe3dr is a product of ${LEGAL_OPERATOR_FULL}, and ties refunds to the order stage because home chefs cook to order. Every order moves through fixed stages — Placed, Accepted, Preparing, Ready, Out for delivery, Delivered — and you can see the current stage on the order detail screen.`,
      '• Placed (chef has not accepted yet): cancel instantly, 100% refund within 7 working days.',
      '• Accepted (chef has not started cooking): full refund. In rare cases — a custom or pre-shopped catering dish — the chef may have already bought ingredients; if so, we show you the ingredient cost before you confirm and refund the rest.',
      '• Preparing (chef is cooking): we cannot refund the food cost, because the chef cannot sell the food to someone else. If the order is not delivered, we still refund the delivery fee.',
      '• Out for delivery: you cannot cancel from the app, but contact support if something is wrong — refunds are case-by-case.',
      '• Delivered but unsatisfactory: raise a complaint within 24 hours; refunds are case-by-case, partial or full.',
      '• Never delivered: an automatic 100% refund after confirmation.',
      '"Working days" excludes weekends and bank holidays in India. The timeline starts when your refund is approved by our team or the chef.',
    ],
  },
  {
    heading: '2. When chefs cancel',
    paragraphs: [
      'Chefs can decline or cancel when they are out of ingredients, closing earlier than expected, or a kitchen emergency makes the order unsafe to complete. When a chef cancels at any stage, we issue a 100% automatic refund to your original payment method — you do not need to ask — and send an email, an in-app message, and a push notification with the amount and timeline.',
      'Force majeure — severe weather, power cuts, civic disruptions, sudden illness, or a delivery partner going offline mid-route — is treated the same as a chef cancellation: a full automatic refund within 7 working days. If you paid a tip, priority fee, or add-on before the chef accepted, a chef cancellation refunds everything, including those amounts.',
    ],
  },
  {
    heading: '3. How to request a refund',
    paragraphs: [
      'Open the order in My Orders, tap "Need help?" at the bottom of the order screen, choose "Refund request", and tell us what went wrong — add photos if you have them. We respond within 48 hours and resolve refund disputes within 7 working days, in line with the Consumer Protection (E-Commerce) Rules, 2020.',
      'Raise a complaint within 24 hours of delivery so the chef and driver can recall the order details accurately.',
    ],
  },
  {
    heading: '4. How long refunds take and where they go',
    paragraphs: [
      'We process every refund within 7 working days from the moment we approve it — the maximum window set by the Reserve Bank of India Payment Aggregator Master Direction (17 March 2020). Most refunds land in 3 to 5 working days; the exact day depends on your bank or wallet, not on us.',
      'Every refund goes back to your original payment method — a Reserve Bank of India rule for payment aggregators that protects you from fraud. UPI refunds return to the same UPI ID and bank account; card refunds return to the same card; net-banking refunds return to the same account; wallet refunds return to the wallet balance. We cannot redirect a refund to a different account. All refunds are issued in Indian rupees; if you paid with a card issued outside India, the card network handles currency conversion, often at a different rate from checkout.',
    ],
  },
  {
    heading: '5. Partial refunds',
    paragraphs: [
      'When only part of your order is wrong — one dish missing, or one item cold — we refund per item, not per order. Tell us which items are affected and we refund the item full cost plus its proportional share of taxes and the delivery fee. If half or more of your order is unusable, we usually refund the full order including delivery.',
      'We never refund a dish because you did not like the taste — food is subjective. Our refund path covers things that went measurably wrong: missing or wrong items, spoiled food, food-safety issues, temperature issues at delivery, or quantity well below what was advertised.',
    ],
  },
  {
    heading: '6. Catering orders',
    paragraphs: [
      'Catering orders need more notice than everyday meals because chefs shop and prep ahead. The cancellation window is set by the chef when they accept your quote and shown on the catering order screen. Common chef policies are: more than 72 hours before the event, 100% refund; 48 to 72 hours before, 50% refund; less than 48 hours before, no refund. If a chef cancels a catering booking, we refund 100% within 7 working days and help you find another chef.',
      'For catering, the chef cooks for the number of guests you confirmed. If your guest count drops after the chef has shopped, the original quote stands. One date change is usually possible if you request it more than 72 hours before the event and the chef has availability; after 72 hours a date change is treated as a cancellation.',
    ],
  },
  {
    heading: '7. Payment failures and chargebacks',
    paragraphs: [
      'Sometimes a payment fails after the money has left your bank — usually a network timeout. The payment gateway auto-reverses the transaction without us doing anything, and the money returns to your original payment method within 5 to 7 working days. You do not need to file a refund request; the reversal is automatic. If you see a debit but no order in My Orders, give it 7 working days, then contact support with your bank statement reference.',
      'A chargeback — asking your bank to reverse a payment — is your right, but we would rather you talk to us first, as most disputes resolve faster through our support flow than the 30-to-90-day bank process. If you file a chargeback, our payment partner notifies us, we share order records and delivery confirmation with your bank, and your bank decides. We never penalise customers for genuine chargebacks, but repeated chargebacks on orders that were genuinely delivered may lead us to close your account.',
    ],
  },
  {
    heading: '8. What is not refundable',
    paragraphs: [
      'So you know the limits before you order:',
      '• Taste preference where the food was prepared as described — use the rating system instead.',
      '• Delays under 30 minutes from the original ETA. Beyond 30 minutes, contact support.',
      '• Tips a driver has already accepted, unless the delivery never happened.',
      '• The discount value of a coupon or platform credit — the refund covers the amount you actually paid.',
      '• Orders delivered correctly to the address you entered.',
      '• Self-pickup orders you did not collect within the pickup window.',
      'For cash-on-delivery orders, there is nothing to refund if you cancel before the chef accepts. If something goes wrong after a cash delivery, we credit your platform wallet for the refund amount, which you can use on your next order or withdraw through support.',
    ],
  },
  {
    heading: '9. If a refund does not arrive, and changes to this policy',
    paragraphs: [
      'Refunds that do not appear within 7 working days are rare and usually a bank routing issue. Check your bank statement carefully, ask for the refund reference number in the order screen, and share it with your bank so they can trace it. If we still cannot resolve it, escalate to our Grievance Officer — required for every Indian e-commerce platform under the Consumer Protection (E-Commerce) Rules, 2020 — who responds within 48 hours and resolves within 30 days. You can also file a complaint with the National Consumer Helpline (1915) or the e-Daakhil portal.',
      'We update this policy when our refund flows or regulations change. For material changes we tell you in the app and by email at least 14 days before the change takes effect; orders placed before the change keep the older policy.',
    ],
  },
  {
    heading: '10. Contact us',
    paragraphs: [
      `For any question about a refund or cancellation, contact Fe3dr support and our Grievance Officer at ${LEGAL_SUPPORT_EMAIL}.`,
    ],
  },
];

export default function RefundPage() {
  return (
    <LegalPage
      title="Refund & Cancellation Policy"
      summary={SUMMARY}
      lastUpdated={LEGAL_LAST_UPDATED}
      sections={SECTIONS}
    />
  );
}
