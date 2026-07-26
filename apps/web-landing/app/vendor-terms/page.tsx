// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import {
  LEGAL_GRIEVANCE_EMAIL,
  LEGAL_LAST_UPDATED,
  LEGAL_INDIA_OPERATOR,
  LEGAL_OPERATOR,
  LEGAL_OPERATOR_FULL,
  LEGAL_SUPPORT_EMAIL,
} from '@/lib/site';

export const metadata: Metadata = {
  title: 'Chef & Vendor Agreement',
  description:
    'The agreement between home chefs and Fe3dr — commission, payouts, food safety, and cancellation duties.',
  alternates: { canonical: '/vendor-terms/' },
};

const SUMMARY =
  'This agreement sets out the relationship between you, an independent home chef, and Fe3dr. You control your own menu, pricing, and kitchen. We provide the platform, take payments on your behalf, and pay you out on a weekly cycle after deducting our commission and platform fee. You remain responsible for your FSSAI registration, food safety, accurate allergen disclosure, and your own taxes.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Who this agreement is between',
    paragraphs: [
      `This Chef & Vendor Agreement is between you (the "Chef") and ${LEGAL_OPERATOR}. Fe3dr is a product of ${LEGAL_OPERATOR_FULL} ("Fe3dr", "we", "us"). It governs your use of the Fe3dr vendor app and your sale of home-cooked food to customers through the platform. By submitting a kitchen application and listing a menu, you accept this agreement.`,
    ],
  },
  {
    heading: '2. Independent-contractor status',
    paragraphs: [
      'You are an independent business, not an employee, agent, partner, or franchisee of Fe3dr. Nothing in this agreement creates an employment or partnership relationship. You decide when to open your kitchen, what to cook, how to price it, and how to run your operation, subject to the standards in this agreement and applicable law.',
      'You are responsible for your own insurance, your own equipment, your own staff (if any), and your own compliance with food-safety, labour, and tax law.',
    ],
  },
  {
    heading: '3. Your menu and pricing',
    paragraphs: [
      'You control your menu, your dish descriptions, your photographs, your prices, and your operating hours. You set the price customers pay for each dish. You are responsible for keeping your menu accurate and up to date, including marking items unavailable when you cannot prepare them.',
      'You grant Fe3dr a non-exclusive, royalty-free licence to display your menu content, kitchen name, and photographs on the platform and in marketing of the platform.',
    ],
  },
  {
    heading: '4. Commission and platform fee',
    paragraphs: [
      'Fe3dr charges a commission and a platform fee on each completed order, deducted from the order proceeds before payout. The current commission and platform-fee rates are shown in the vendor app and in your onboarding terms. We will give you reasonable advance notice in the app before any change to these rates takes effect.',
      'Applicable taxes (such as GST) and any statutory deductions (such as TDS under Section 194-O) are handled as required by Indian law and itemised in your earnings statements.',
    ],
  },
  {
    heading: '5. Payouts',
    paragraphs: [
      `Your payouts are made by ${LEGAL_INDIA_OPERATOR}, the Fe3dr operating entity in India, which is part of ${LEGAL_OPERATOR}. That is the name that will appear on your bank statement.`,
      'We collect customer payments through our payment partner (Razorpay) and hold order proceeds until the order is delivered. After delivery, your share — order value less commission, platform fee, and applicable taxes — is settled to your registered bank account or UPI ID on a weekly payout cycle, subject to the Reserve Bank of India Payment Aggregator framework.',
      'You are responsible for keeping your payout details accurate. Payouts may be held where an order is under dispute, a refund or chargeback is pending, or we are required to withhold by law.',
    ],
  },
  {
    heading: '6. FSSAI registration and food safety',
    paragraphs: [
      'You must hold and maintain the food-business registration or licence required by the Food Safety and Standards Authority of India (FSSAI) for your operation, and keep it current. We verify your FSSAI registration before you can list a menu and we will remind you before it expires.',
      'Food safety is your responsibility. You must prepare food hygienically, in line with FSSAI standards and all applicable local food-safety law. Fe3dr does not cook, test, or inspect your food and does not warrant its preparation, but we take food-safety reports seriously and may suspend a listing pending investigation.',
    ],
  },
  {
    heading: '7. What you may sell, and what you may not',
    paragraphs: [
      'This is the most important section of this agreement. Please read all of it.',
      'You may sell only food you cooked yourself, in the kitchen you registered with us, from your approved Fe3dr menu, under your own valid and current FSSAI registration.',
      'You must never:',
      '• sell food when your FSSAI registration has lapsed, been suspended, or been cancelled;',
      '• cook from an address other than your registered kitchen without telling us and having it approved first;',
      '• buy food from a restaurant, caterer, or another cook and sell it as your own home cooking;',
      '• sell food that is spoiled, stale, reheated beyond safe limits, or made from ingredients past their use-by date;',
      '• sell meat, fish, eggs, or dairy that has not been kept at a safe temperature throughout;',
      '• sell anything containing a banned or restricted substance, or any adulterant;',
      '• describe a dish as vegetarian, vegan, jain, halal, or free of an allergen when it is not;',
      '• cook or pack food while you or anyone in your kitchen has a communicable illness.',
      'We call these serious breaches. They are different from an ordinary service problem such as a late order.',
    ],
  },
  {
    heading: '8. What happens after a serious breach',
    paragraphs: [
      'If we have reasonable grounds to believe a serious breach has happened, we may, without notice: remove your kitchen and menu from the marketplace; hold your pending payouts while we investigate, for no longer than the investigation needs; cancel affected orders and refund those customers; and end this agreement and close your account permanently.',
      'We will tell you what we believe happened and give you a fair chance to respond, unless the law or an immediate risk to health prevents it.',
      'Beyond the platform, please be clear about the following:',
      '• We are required to report suspected food-safety offences to FSSAI or the local food-safety authority, and we will. We will provide your registration details, your kitchen address, and the records that relate to the matter.',
      '• Selling unsafe or misdescribed food can lead to prosecution, fines, and imprisonment under the Food Safety and Standards Act, 2006. Those consequences fall on you, not on Fe3dr.',
      '• If a customer is harmed, the claim is against you as the food business operator. You should have your own arrangements in place for this.',
      '• If your breach costs us money \u2014 refunds we had to pay, penalties, investigation costs, or reasonable legal costs \u2014 you agree to repay it, and we may recover it from payouts we hold.',
      'None of this applies to an honest mistake you report to us promptly. If something goes wrong and you tell us yourself, we take that into account.',
    ],
  },
  {
    heading: '9. Honest dealing, ratings, and reputation',
    paragraphs: [
      'You are free to disagree with us publicly, to say what you think of our commission, and to raise a dispute. Honest criticism is not a breach of this agreement.',
      'Dishonesty is. Please do not:',
      '• create fake orders, or ask friends or family to place orders you refund privately, to inflate your ratings or earnings;',
      '• write or arrange reviews of your own kitchen, or of another chef kitchen;',
      '• offer a customer money, free food, or a discount in exchange for changing or removing a review;',
      '• pressure, threaten, or repeatedly contact a customer about a review they left;',
      '• take a customer order off the platform to avoid commission after they found you here;',
      '• publish something about Fe3dr, a customer, or another chef that you know to be untrue and that damages them.',
      'If you do any of these we may remove the affected ratings, pause or close your account, withhold earnings obtained through the conduct, and pursue any legal remedy open to us.',
    ],
  },
  {
    heading: '10. Your responsibility cannot be passed to us',
    paragraphs: [
      'Nothing in this agreement makes Fe3dr the food business operator for your kitchen, and nothing in it moves your duties under food-safety law onto us. Those duties stay with you and cannot be transferred by contract.',
      'If a customer, a regulator, or anyone else brings a claim against Fe3dr because of food you prepared, how you described it, or your failure to hold a valid registration, you agree to cover what that costs us, including reasonable legal costs. This does not apply to anything caused by our own breach or negligence, and it does not apply where the law does not allow it.',
    ],
  },
  {
    heading: '11. Allergen and ingredient disclosure',
    paragraphs: [
      'You must accurately disclose the ingredients of each dish and any common allergens you are aware of. Customers rely on this information. Inaccurate or missing allergen disclosure that causes harm is your responsibility. Where a home kitchen handles many ingredients in a shared space, you should note the risk of cross-contact.',
    ],
  },
  {
    heading: '12. Order acceptance and cancellation duties',
    paragraphs: [
      'You should accept or decline orders promptly and prepare accepted orders within your stated prep time. You may decline an order when you are at capacity or an item is unavailable.',
      'If you cancel an order after accepting it, the customer receives a full refund. Repeated late cancellations, no-shows, or cancellations after a customer has paid harm customer trust and may lead to suspension. Genuine force-majeure events — power cuts, sudden illness, civic disruption — are handled fairly and are not treated as fault, but you should mark your kitchen closed as early as you can.',
    ],
  },
  {
    heading: '13. Taxes',
    paragraphs: [
      'You are responsible for your own tax affairs, including registering for and paying GST where applicable, and reporting your income. Fe3dr provides earnings statements and any statutory tax documents (such as TDS certificates) to help you, but does not act as your tax adviser.',
    ],
  },
  {
    heading: '14. Reviews and content',
    paragraphs: [
      'Customers may review your dishes. Reviews are the customer own honest opinion. Fe3dr hosts reviews as an intermediary and will act on reviews that breach our content rules or that a court directs us to remove. You may report a review you believe is false or abusive through the app.',
    ],
  },
  {
    heading: '15. Suspension and termination',
    paragraphs: [
      'You may stop using the platform and close your account at any time. We may suspend or terminate your listing or account for breach of this agreement, repeated food-safety or hygiene complaints, fraud, an expired or revoked FSSAI registration, or where required by law. Where practical we will tell you why and give you an opportunity to respond. Pending settled payouts for delivered orders remain payable to you.',
    ],
  },
  {
    heading: '16. Liability',
    paragraphs: [
      'As an independent business, you are responsible for the food you prepare and for your compliance with the law. To the extent permitted by law, Fe3dr liability to you in connection with an order is limited to the platform fees and commission attributable to that order. Neither party is liable for indirect or consequential loss except where the law requires otherwise.',
    ],
  },
  {
    heading: '17. Governing law and contact',
    paragraphs: [
      'Governing law. This agreement is governed by the laws of New South Wales, Australia, without regard to conflict-of-laws principles. The courts of New South Wales have exclusive jurisdiction, subject to any non-excludable consumer-protection forum rules in your jurisdiction.',
      'Dispute ladder. Before commencing proceedings, the parties will try to resolve any dispute in good faith for at least 30 days, then attempt mediation under the Rules of the Resolution Institute (Australia). Nothing in this clause prevents either party from seeking urgent interlocutory relief.',
      'This agreement is read alongside Indian food-safety (FSSAI), payments (RBI), and tax law that applies to your operation. We may update it from time to time, with notice in the app for material changes.',
      `For any question about this agreement, your payouts, or your account, contact ${LEGAL_SUPPORT_EMAIL}. For India data-protection grievances, our Grievance Officer is reachable at ${LEGAL_GRIEVANCE_EMAIL}.`,
    ],
  },
];

export default function VendorTermsPage() {
  return (
    <LegalPage
      title="Chef & Vendor Agreement"
      summary={SUMMARY}
      lastUpdated={LEGAL_LAST_UPDATED}
      sections={SECTIONS}
    />
  );
}
