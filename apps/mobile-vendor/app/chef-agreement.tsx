// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).
// Chef & Vendor Agreement — reachable from the More tab Legal group.
// Mirrors the landing /vendor-terms content (the URL vendor onboarding links to).

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '26 July 2026';

const INTRO =
  'This Chef & Vendor Agreement is between you, an independent home chef, and Tesserix Pty Ltd. Fe3dr is a product of Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia ("Fe3dr", "we", "us"). It governs your use of the vendor app and your sale of home-cooked food through the platform. By submitting a kitchen application and listing a menu, you accept this agreement.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Independent-contractor status',
    paragraphs: [
      'You are an independent business, not an employee, agent, partner, or franchisee of Fe3dr. You decide when to open your kitchen, what to cook, how to price it, and how to run your operation, subject to the standards in this agreement and applicable law. You are responsible for your own insurance, equipment, staff (if any), and compliance with food-safety, labour, and tax law.',
    ],
  },
  {
    heading: '2. Your menu and pricing',
    paragraphs: [
      'You control your menu, dish descriptions, photographs, prices, and operating hours, and you set the price customers pay for each dish. Keep your menu accurate and up to date, including marking items unavailable when you cannot prepare them. You grant Fe3dr a non-exclusive, royalty-free licence to display your menu content, kitchen name, and photographs on the platform and in marketing of the platform.',
    ],
  },
  {
    heading: '3. Commission and platform fee',
    paragraphs: [
      'Fe3dr charges a commission and a platform fee on each completed order, deducted from the order proceeds before payout. The current rates are shown in the vendor app and your onboarding terms. We will give you reasonable advance notice in the app before any change takes effect. Applicable taxes (such as GST) and statutory deductions (such as TDS under Section 194-O) are handled as required by Indian law and itemised in your earnings statements.',
    ],
  },
  {
    heading: '4. Payouts',
    paragraphs: [
      'Your payouts are made by Zivana Innovations LLP, the Fe3dr operating entity in India, which is part of Tesserix Pty Ltd. Zivana is the name that will appear on your bank statement.',
      'We collect customer payments through our payment partner (Cashfree Payments) and hold order proceeds until the order is delivered. After delivery, your share — order value less commission, platform fee, and applicable taxes — is settled to your registered bank account or UPI ID on a weekly payout cycle, subject to the Reserve Bank of India Payment Aggregator framework. Keep your payout details accurate. Payouts may be held where an order is under dispute, a refund or chargeback is pending, or we are required to withhold by law.',
    ],
  },
  {
    heading: '5. FSSAI registration and food safety',
    paragraphs: [
      'You must hold and maintain the food-business registration or licence required by the Food Safety and Standards Authority of India (FSSAI) for your operation, and keep it current. We verify your FSSAI registration before you can list a menu and remind you before it expires. Food safety is your responsibility: you must prepare food hygienically, in line with FSSAI standards and applicable local food-safety law. Fe3dr does not cook, test, or inspect your food and does not warrant its preparation, but we take food-safety reports seriously and may suspend a listing pending investigation.',
      'Selling without a valid registration is an offence under the Food Safety and Standards Act, 2006. If your licence lapses, is suspended or is cancelled, you must stop selling on Fe3dr immediately and tell us the same day.',
    ],
  },
  {
    heading: '6. What you may sell, and what you may not',
    paragraphs: [
      'This section matters more than any other in this agreement. Please read it fully.',
      'You may sell only:',
      '• Food you cooked yourself, in the kitchen you registered with us.',
      '• Dishes that appear on your approved Fe3dr menu.',
      '• Food made under your own valid, current FSSAI registration.',
      'You must never:',
      '• Sell food when your FSSAI registration has lapsed, been suspended or been cancelled.',
      '• Cook from an address other than your registered kitchen without telling us and having it approved first.',
      '• Buy food from a restaurant, caterer or another cook and sell it as your own home cooking.',
      '• Sell food that is spoiled, stale, reheated beyond safe limits, or made from ingredients past their use-by date.',
      '• Sell meat, fish, eggs or dairy that has not been stored at a safe temperature throughout.',
      '• Sell anything containing a banned or restricted substance, or any adulterant.',
      '• Describe a dish as vegetarian, vegan, jain, halal or free of an allergen when it is not.',
      '• Cook or pack food while you or anyone in your kitchen has a communicable illness.',
      'We call these serious breaches. They are not the same as an ordinary service problem such as a late order.',
    ],
  },
  {
    heading: '7. What happens after a serious breach',
    paragraphs: [
      'If we have reasonable grounds to believe a serious breach has happened, we may do any of the following, without notice:',
      '• Remove your kitchen and menu from the marketplace straight away.',
      '• Hold your pending payouts while we investigate. We will not hold them longer than the investigation needs.',
      '• Cancel affected orders and refund the customers.',
      '• End this agreement and close your account permanently.',
      'We will tell you what we believe happened and give you a fair chance to respond, unless the law or an immediate risk to health prevents it.',
      'Beyond the platform, you should understand clearly:',
      '• We are required to report suspected food-safety offences to FSSAI or the local food-safety authority, and we will. We will give them your registration details, your kitchen address and the records that relate to the matter.',
      '• Selling unsafe or misdescribed food can lead to prosecution, fines and imprisonment under the Food Safety and Standards Act, 2006. Those consequences fall on you, not on Fe3dr.',
      '• If a customer is harmed, the claim is against you as the food business operator. You must have your own arrangements in place for this.',
      '• If your breach costs us money — refunds we had to pay, penalties, investigation costs, or reasonable legal costs — you agree to repay it, and we may recover it from payouts we hold.',
      'None of this applies to an honest mistake you tell us about promptly. If something goes wrong and you report it yourself, we will take that into account.',
    ],
  },
  {
    heading: '8. Allergen and ingredient disclosure',
    paragraphs: [
      'You must accurately disclose the ingredients of each dish and any common allergens you are aware of. Customers rely on this information, and inaccurate or missing allergen disclosure that causes harm is your responsibility. Where a home kitchen handles many ingredients in a shared space, note the risk of cross-contact.',
    ],
  },
  {
    heading: '9. Order acceptance and cancellation duties',
    paragraphs: [
      'Accept or decline orders promptly and prepare accepted orders within your stated prep time. You may decline an order when you are at capacity or an item is unavailable. If you cancel an order after accepting it, the customer receives a full refund. Repeated late cancellations, no-shows, or cancellations after a customer has paid may lead to suspension. Genuine force-majeure events are handled fairly and are not treated as fault, but mark your kitchen closed as early as you can.',
    ],
  },
  {
    heading: '10. Taxes',
    paragraphs: [
      'You are responsible for your own tax affairs, including registering for and paying GST where applicable, and reporting your income. Fe3dr provides earnings statements and any statutory tax documents (such as TDS certificates) to help you, but does not act as your tax adviser.',
    ],
  },
  {
    heading: '11. Suspension and termination',
    paragraphs: [
      'You may stop using the platform and close your account at any time. We may suspend or terminate your listing or account for breach of this agreement, repeated food-safety or hygiene complaints, fraud, an expired or revoked FSSAI registration, or where required by law. Where practical we will tell you why and give you an opportunity to respond. Pending settled payouts for delivered orders remain payable to you.',
    ],
  },
  {
    heading: '12. Honest dealing, ratings and reputation',
    paragraphs: [
      'You are free to disagree with us publicly, to say what you think of the commission, and to raise a dispute. Honest criticism is not a breach of this agreement.',
      'What is a breach is dishonesty. You must not:',
      '• Create fake orders, or ask friends or family to place orders you refund privately, to inflate your ratings or earnings.',
      '• Write or arrange reviews of your own kitchen, or reviews of another chef’s kitchen.',
      '• Offer a customer money, free food or a discount in exchange for changing or removing a review.',
      '• Pressure, threaten or repeatedly contact a customer over a review they left.',
      '• Take a customer’s order off the platform to avoid commission after they found you here.',
      '• Publish a statement of fact about Fe3dr, a customer or another chef that you know to be untrue and that damages them.',
      'If you do any of these we may remove the affected ratings, suspend or close your account, withhold earnings obtained through the conduct, and pursue any legal remedy available to us.',
    ],
  },
  {
    heading: '13. Liability, governing law, and contact',
    paragraphs: [
      'As an independent business, you are responsible for the food you prepare and for your compliance with the law. To the extent permitted by law, Fe3dr’s liability to you in connection with an order is limited to the platform fees and commission attributable to that order.',
      'If a customer, a regulator or anyone else brings a claim against Fe3dr because of food you prepared, how you described it, or your failure to hold a valid registration, you agree to cover what that costs us, including reasonable legal costs. This does not apply to anything caused by our own breach or negligence, and it does not apply where the law does not allow it.',
      'Nothing in this agreement makes Fe3dr the food business operator for your kitchen, and nothing in it transfers your duties under food-safety law to us. Those duties stay with you and cannot be passed on by contract.',
      'This agreement is governed by the laws of New South Wales, Australia, without regard to conflict-of-laws principles. The courts of New South Wales have exclusive jurisdiction, subject to any non-excludable consumer-protection forum rules in your jurisdiction. Before commencing proceedings, the parties will try to resolve any dispute in good faith for at least 30 days, then attempt mediation under the Rules of the Resolution Institute (Australia); nothing prevents either party from seeking urgent interlocutory relief. The agreement is read alongside Indian food-safety (FSSAI), payments (RBI), and tax law that applies to your operation. For any question about this agreement, your payouts, or your account, contact support@fe3dr.com.',
    ],
  },
];

export default function ChefAgreementScreen() {
  return (
    <LegalScreen
      title="Chef & Vendor Agreement"
      lastUpdated={LAST_UPDATED}
      intro={INTRO}
      sections={SECTIONS}
    />
  );
}
