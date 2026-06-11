// Refund & Cancellation Policy — linked from checkout (consent + payment summary).
// Content is a starting template; have it reviewed by legal counsel before launch.

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '11 June 2026';

const INTRO =
  'Fe3dr is a marketplace operated by Tesserix Pty Ltd that connects you with independent home chefs and delivery partners. This policy explains when you can cancel an order and how refunds are handled. It applies to all orders placed through the Fe3dr customer app.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Cancelling an order',
    paragraphs: [
      'You can cancel an order from the order screen. Whether a refund applies depends on the stage your order has reached when you cancel:',
      '• Before the chef accepts — free cancellation, full refund.',
      '• After the chef accepts but before they begin cooking — full refund, less any payment-gateway charges that are non-refundable.',
      '• After the chef has begun preparing your food — a partial refund may apply, since ingredients and effort have already been committed. The exact amount depends on how far preparation has progressed.',
      '• After the order is handed to a delivery partner — the order generally cannot be cancelled and is non-refundable, except where the food is not delivered or arrives in an unacceptable condition.',
    ],
  },
  {
    heading: '2. Refunds for quality or delivery issues',
    paragraphs: [
      'If your order is not delivered, is materially incorrect, arrives spoiled, or is otherwise unsafe to eat, you are entitled to a refund. Report the issue through the app within 24 hours of the delivery time and, where possible, include photos so we can assess it quickly.',
      'Depending on the issue we may issue a full refund, a partial refund, or a credit toward a future order. We assess each report fairly and in good faith.',
    ],
  },
  {
    heading: '3. How and when refunds are paid',
    paragraphs: [
      'Payments are processed by Razorpay, an RBI-licensed payment aggregator. Approved refunds are returned to your original payment method.',
      'Refunds are completed within 7 working days of approval, in line with the Reserve Bank of India Payment Aggregator Master Direction. The time it takes for the amount to appear in your account afterward depends on your bank or card issuer.',
    ],
  },
  {
    heading: '4. Situations that are not refundable',
    paragraphs: [
      '• Orders cancelled after a delivery partner has collected the food, except for non-delivery or quality failures.',
      '• Issues reported more than 24 hours after delivery.',
      '• Dissatisfaction based on personal taste preference where the food was prepared as described.',
      '• Incorrect delivery details (address, contact number) provided by you.',
    ],
  },
  {
    heading: '5. How to request a refund',
    paragraphs: [
      'Open the order in the app and use the help or report option, or contact our support team. Tell us your order number and what went wrong. We aim to respond within 24 hours.',
    ],
  },
  {
    heading: '6. Contact us',
    paragraphs: [
      'For any question about a refund or cancellation, contact Fe3dr support at support@fe3dr.com.',
    ],
  },
];

export default function RefundPolicyScreen() {
  return (
    <LegalScreen
      title="Refund & Cancellation Policy"
      lastUpdated={LAST_UPDATED}
      intro={INTRO}
      sections={SECTIONS}
    />
  );
}
