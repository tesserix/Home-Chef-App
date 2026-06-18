// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).
// Vendor (chef) Terms of Service — reachable from the More tab Legal group.

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '11 June 2026';

const INTRO =
  'These Terms of Service govern your use of the Fe3dr vendor app. Fe3dr is a product of Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia ("Fe3dr", "we", "us"). The full commercial terms of your relationship with the platform — commission, payouts, food-safety duties, and cancellation rules — are set out in the Chef & Vendor Agreement, also available from this menu.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. About Fe3dr',
    paragraphs: [
      'Fe3dr is an online marketplace that connects independent home chefs with customers who want home-cooked food, and with delivery partners who deliver it. As a chef, you use the vendor app to list your kitchen, manage your menu, receive orders, and track your earnings. Fe3dr facilitates these transactions and handles payments; you prepare the food and are responsible for its quality, safety, and description.',
    ],
  },
  {
    heading: '2. Your account',
    paragraphs: [
      'You must be at least 18 years old and able to enter into a binding contract to use the vendor app. You are responsible for keeping your account credentials secure and for all activity under your account. Provide accurate kitchen, contact, and payout details and keep them up to date.',
    ],
  },
  {
    heading: '3. Your relationship with the platform',
    paragraphs: [
      'You are an independent business, not an employee or agent of Fe3dr. You control your menu, pricing, and hours. The commercial terms — commission and platform fee, weekly payouts, FSSAI and food-safety responsibilities, allergen-disclosure duties, order acceptance and cancellation rules, and your own tax obligations — are set out in full in the Chef & Vendor Agreement. By listing a menu you accept that agreement.',
    ],
  },
  {
    heading: '4. Acceptable use',
    paragraphs: [
      'You agree to use the app lawfully and in good faith. Do not provide false information, misrepresent your dishes, evade fees, or behave abusively toward customers, delivery partners, or our staff. We may suspend or close accounts that breach these terms or the Chef & Vendor Agreement.',
    ],
  },
  {
    heading: '5. Limitation of liability',
    paragraphs: [
      'To the extent permitted by law, Fe3dr’s liability to you in connection with an order is limited to the platform fees and commission attributable to that order. We are not liable for indirect or consequential losses. Nothing in these terms excludes liability that cannot be excluded under applicable law.',
    ],
  },
  {
    heading: '6. Changes to these terms',
    paragraphs: [
      'We may update these terms from time to time. Material changes will be notified in the app or by email. Continuing to use the vendor app after changes take effect means you accept the updated terms.',
    ],
  },
  {
    heading: '7. Governing law and contact',
    paragraphs: [
      'These terms are governed by the laws of New South Wales, Australia, without regard to conflict-of-laws principles. The courts of New South Wales have exclusive jurisdiction, subject to any non-excludable consumer-protection forum rules in your jurisdiction. They are read alongside Indian food-safety (FSSAI), payments (RBI), and tax law that applies to your operation. For any question about these terms, contact support@fe3dr.com.',
    ],
  },
];

export default function VendorTermsScreen() {
  return (
    <LegalScreen
      title="Terms of Service"
      lastUpdated={LAST_UPDATED}
      intro={INTRO}
      sections={SECTIONS}
    />
  );
}
