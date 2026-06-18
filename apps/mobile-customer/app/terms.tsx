// Terms of Service — linked from checkout (consent + payment summary).
// Content is a starting template; have it reviewed by legal counsel before launch.

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '11 June 2026';

const INTRO =
  'These Terms of Service govern your use of the Fe3dr customer app. Fe3dr is a product of Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia ("Fe3dr", "we", "us"). By creating an account or placing an order you agree to these terms. Please read them carefully.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. About Fe3dr',
    paragraphs: [
      'Fe3dr is an online marketplace that connects customers with independent home chefs who prepare food, and with delivery partners who deliver it. Fe3dr facilitates these transactions and handles payments, but the food is prepared by independent chefs who are solely responsible for its quality, safety, and description.',
    ],
  },
  {
    heading: '2. Eligibility and your account',
    paragraphs: [
      'You must be at least 18 years old and able to enter into a binding contract to use Fe3dr. You are responsible for keeping your account credentials secure and for all activity under your account. Provide accurate delivery and contact details — orders sent to incorrect details that you supplied are not refundable.',
    ],
  },
  {
    heading: '3. Orders and pricing',
    paragraphs: [
      'When you place an order you make an offer to purchase the selected items from the chef. The order is confirmed once the chef accepts it. Prices shown include the item price; applicable taxes, delivery fees, and platform service fees are shown before you pay. A minimum order value may apply per chef.',
      'Chefs may decline orders — for example when they are at capacity or an item is unavailable. If a chef declines, you are not charged, or any amount taken is refunded.',
    ],
  },
  {
    heading: '4. Payments',
    paragraphs: [
      'Payments are processed by Razorpay, an RBI-licensed payment aggregator. Order proceeds are settled to your chef, less the platform service fee and applicable taxes. We do not store your full card details — they are handled by the payment gateway.',
    ],
  },
  {
    heading: '5. Food safety and chef responsibility',
    paragraphs: [
      'Home chefs are independent providers responsible for preparing food hygienically and accurately describing their dishes, including ingredients and allergens. If you have allergies or dietary restrictions, review the dish details and contact the chef where needed. Fe3dr is not the maker of the food and does not warrant its preparation, but we take food-safety reports seriously and act on them.',
    ],
  },
  {
    heading: '6. Cancellations and refunds',
    paragraphs: [
      'Cancellations and refunds are governed by our Refund & Cancellation Policy, which forms part of these terms. In short, refunds depend on the stage your order has reached, and approved refunds are returned within 7 working days.',
    ],
  },
  {
    heading: '7. Delivery',
    paragraphs: [
      'Delivery is carried out by independent delivery partners. Estimated delivery times are indicative and depend on the chef’s preparation time and the route. You must be available to receive the order at the address provided.',
    ],
  },
  {
    heading: '8. Acceptable use',
    paragraphs: [
      'You agree not to misuse the platform — including submitting false orders, abusive behavior toward chefs or delivery partners, fraudulent payment or refund claims, or any unlawful activity. We may suspend or close accounts that breach these terms.',
    ],
  },
  {
    heading: '9. Limitation of liability',
    paragraphs: [
      'To the extent permitted by law, Fe3dr’s liability in connection with an order is limited to the amount you paid for that order. We are not liable for indirect or consequential losses. Nothing in these terms excludes liability that cannot be excluded under applicable law.',
    ],
  },
  {
    heading: '10. Changes to these terms',
    paragraphs: [
      'We may update these terms from time to time. Material changes will be notified in the app or by email. Continuing to use Fe3dr after changes take effect means you accept the updated terms.',
    ],
  },
  {
    heading: '11. Governing law and contact',
    paragraphs: [
      'These terms are governed by the laws of New South Wales, Australia, without regard to conflict-of-laws principles. The courts of New South Wales have exclusive jurisdiction, subject to any non-excludable consumer-protection forum rules in your jurisdiction — including, if you are an Indian consumer, your right to bring a complaint to a District Consumer Disputes Redressal Commission with jurisdiction over your place of residence under the Consumer Protection Act, 2019.',
      'Before commencing proceedings, the parties will try to resolve any dispute in good faith for at least 30 days, then attempt mediation under the Rules of the Resolution Institute (Australia); nothing in this clause prevents either party from seeking urgent interlocutory relief. For any question about these terms, contact support@fe3dr.com.',
    ],
  },
];

export default function TermsScreen() {
  return (
    <LegalScreen
      title="Terms of Service"
      lastUpdated={LAST_UPDATED}
      intro={INTRO}
      sections={SECTIONS}
    />
  );
}
