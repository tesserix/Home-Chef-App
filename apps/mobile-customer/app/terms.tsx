// Terms of Service — linked from checkout (consent + payment summary).
// Content is a starting template; have it reviewed by legal counsel before launch.

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '26 July 2026';

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
      'You must not misuse Fe3dr. The following are not allowed:',
      '• Placing orders you do not intend to pay for or receive.',
      '• Abusing, threatening or harassing a chef, a delivery partner, or our staff.',
      '• Using someone else’s payment method, or an account that is not yours.',
      '• Tampering with the app, or trying to get around its security or payment checks.',
      '• Anything unlawful.',
      'If you break these rules we may suspend or close your account. Where a breach causes us or a chef a loss, we may also take the steps set out in sections 9 and 10.',
    ],
  },
  {
    heading: '9. Refund fraud and payment disputes',
    paragraphs: [
      'Most refund claims are honest and we deal with them quickly. This section is about the small number that are not.',
      'You must not:',
      '• Claim an order did not arrive when it did.',
      '• Claim food was unsafe, missing or wrong when it was not.',
      '• Use photographs or messages that are not of your own order to support a claim.',
      '• Repeat the same claim across many orders to obtain free food.',
      '• Ask your bank or card issuer to reverse a payment (a chargeback) for an order you received and were satisfied with, instead of raising it with us first.',
      'How we check: we look at order records, delivery data, chef records and your claim history. We will ask you for your side before we decide.',
      'What happens if we find a claim was dishonest:',
      '• We may refuse the refund and suspend or close your account.',
      '• We may recover the amount already refunded to you, and any bank charges we incurred, as a debt.',
      '• We may report the matter to the police or another authority, and give them the records that relate to it.',
      'A refund we decline is not automatically a dishonest claim. We only treat a claim as dishonest where the evidence shows it was deliberately false. Honest mistakes are not penalised.',
    ],
  },
  {
    heading: '10. False statements that damage a chef or Fe3dr',
    paragraphs: [
      'You are free to leave an honest review, including a negative one. We will not act against you for a truthful account of your experience, however critical it is. That right is not affected by anything in these terms.',
      'What is not allowed is publishing a statement of fact that you know to be untrue and that harms a chef, a delivery partner, or us. For example, claiming a chef’s food made you ill when it did not, or that a kitchen is unlicensed when it is licensed.',
      'If you publish a knowingly false statement of that kind, we may remove it, suspend or close your account, and pursue any legal remedy available to us or to the chef affected. Nothing here limits your rights under consumer law.',
    ],
  },
  {
    heading: '11. Your responsibility for the information you give us',
    paragraphs: [
      'You are responsible for what you tell us, because we and the chef act on it.',
      '• Allergies and dietary needs. If you have an allergy or intolerance, check the dish details and tell the chef before you order. Do not rely on the app alone. We cannot guarantee any dish is free of a given ingredient, because home kitchens prepare many dishes in the same space.',
      '• Delivery details. Give an address and phone number that are correct and reachable. Orders sent to details you supplied incorrectly are not refundable.',
      '• Who receives the order. If someone else collects it for you, that is your responsibility.',
    ],
  },
  {
    heading: '12. Costs you cause us',
    paragraphs: [
      'If you break these terms and that causes a claim, a loss or a cost to us, a chef or a delivery partner, you agree to cover it. That includes reasonable legal costs.',
      'This does not apply to anything caused by our own breach or negligence, and it does not apply where the law does not allow it.',
    ],
  },
  {
    heading: '13. Limitation of liability',
    paragraphs: [
      'To the extent permitted by law, Fe3dr’s liability in connection with an order is limited to the amount you paid for that order. We are not liable for indirect or consequential losses.',
      'Nothing in these terms excludes liability that cannot be excluded under applicable law. If you are an Indian consumer, your rights under the Consumer Protection Act, 2019 are not affected by anything written here.',
    ],
  },
  {
    heading: '14. Changes to these terms',
    paragraphs: [
      'We may update these terms from time to time. Material changes will be notified in the app or by email. Continuing to use Fe3dr after changes take effect means you accept the updated terms.',
    ],
  },
  {
    heading: '15. Governing law and contact',
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
