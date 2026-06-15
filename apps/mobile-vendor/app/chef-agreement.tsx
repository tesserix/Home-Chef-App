// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).
// Chef & Vendor Agreement — reachable from the More tab Legal group.
// Mirrors the landing /vendor-terms content (the URL vendor onboarding links to).

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '11 June 2026';

const INTRO =
  'This Chef & Vendor Agreement is between you, an independent home chef, and Tesserix Pty Ltd. Home Chef (Fe3dr) is a product of Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia ("Fe3dr", "we", "us"). It governs your use of the vendor app and your sale of home-cooked food through the platform. By submitting a kitchen application and listing a menu, you accept this agreement.';

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
      'We collect customer payments through our payment partner (Razorpay) and hold order proceeds until the order is delivered. After delivery, your share — order value less commission, platform fee, and applicable taxes — is settled to your registered bank account or UPI ID on a weekly payout cycle, subject to the Reserve Bank of India Payment Aggregator framework. Keep your payout details accurate. Payouts may be held where an order is under dispute, a refund or chargeback is pending, or we are required to withhold by law.',
    ],
  },
  {
    heading: '5. FSSAI registration and food safety',
    paragraphs: [
      'You must hold and maintain the food-business registration or licence required by the Food Safety and Standards Authority of India (FSSAI) for your operation, and keep it current. We verify your FSSAI registration before you can list a menu and remind you before it expires. Food safety is your responsibility: you must prepare food hygienically, in line with FSSAI standards and applicable local food-safety law. Fe3dr does not cook, test, or inspect your food and does not warrant its preparation, but we take food-safety reports seriously and may suspend a listing pending investigation.',
    ],
  },
  {
    heading: '6. Allergen and ingredient disclosure',
    paragraphs: [
      'You must accurately disclose the ingredients of each dish and any common allergens you are aware of. Customers rely on this information, and inaccurate or missing allergen disclosure that causes harm is your responsibility. Where a home kitchen handles many ingredients in a shared space, note the risk of cross-contact.',
    ],
  },
  {
    heading: '7. Order acceptance and cancellation duties',
    paragraphs: [
      'Accept or decline orders promptly and prepare accepted orders within your stated prep time. You may decline an order when you are at capacity or an item is unavailable. If you cancel an order after accepting it, the customer receives a full refund. Repeated late cancellations, no-shows, or cancellations after a customer has paid may lead to suspension. Genuine force-majeure events are handled fairly and are not treated as fault, but mark your kitchen closed as early as you can.',
    ],
  },
  {
    heading: '8. Taxes',
    paragraphs: [
      'You are responsible for your own tax affairs, including registering for and paying GST where applicable, and reporting your income. Fe3dr provides earnings statements and any statutory tax documents (such as TDS certificates) to help you, but does not act as your tax adviser.',
    ],
  },
  {
    heading: '9. Suspension and termination',
    paragraphs: [
      'You may stop using the platform and close your account at any time. We may suspend or terminate your listing or account for breach of this agreement, repeated food-safety or hygiene complaints, fraud, an expired or revoked FSSAI registration, or where required by law. Where practical we will tell you why and give you an opportunity to respond. Pending settled payouts for delivered orders remain payable to you.',
    ],
  },
  {
    heading: '10. Liability, governing law, and contact',
    paragraphs: [
      'As an independent business, you are responsible for the food you prepare and for your compliance with the law. To the extent permitted by law, Fe3dr’s liability to you in connection with an order is limited to the platform fees and commission attributable to that order.',
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
