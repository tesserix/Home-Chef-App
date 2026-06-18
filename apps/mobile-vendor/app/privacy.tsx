// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).
// Vendor (chef) Privacy Policy — reachable from the More tab Legal group.

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '11 June 2026';

const INTRO =
  'This Privacy Policy explains how we collect, use, and protect your personal information when you use the Fe3dr vendor app as a home chef. Fe3dr is a product of Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia ("Fe3dr", "we", "us"). We handle personal data in line with the Australian Privacy Principles (APP) and, for our India operations, in alignment with India’s Digital Personal Data Protection Act, 2023 (DPDP).';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Information we collect',
    paragraphs: [
      '• Profile details — your name, email, phone number, and password.',
      '• Kitchen details — your kitchen name, cuisines, description, address, operating hours, and service radius.',
      '• Verification documents — your ID proof, FSSAI registration, and (if provided) GSTIN, reviewed privately to verify your kitchen.',
      '• Payout and bank details — the bank account or UPI ID you register to receive payouts.',
      '• Order and earnings data — orders you receive, accept, and fulfil, and your earnings and payout history.',
      '• Device and usage data — app interactions, device type, and diagnostic information used to keep the service reliable.',
    ],
  },
  {
    heading: '2. How we use your information',
    paragraphs: [
      'We use your information to set up and verify your kitchen, list you to customers, route and manage orders, calculate and pay your earnings, provide support, keep the platform secure, and meet our legal and tax obligations. With your consent, we may send you updates — you can opt out at any time.',
    ],
  },
  {
    heading: '3. Who we share it with',
    paragraphs: [
      '• Customers — your kitchen name, menu, photos, and ratings are shown publicly; for each order, the customer sees the order details needed to receive it.',
      '• Delivery partners — the pickup details needed to collect an order from your kitchen.',
      '• Payment processor (Razorpay) — your payout details, to settle your earnings.',
      '• Service providers — hosting, notifications, and analytics partners, under confidentiality obligations.',
      'Your verification documents are reviewed privately and are not shared with customers. We do not sell your personal data.',
    ],
  },
  {
    heading: '4. Data retention',
    paragraphs: [
      'We keep your information for as long as your account is active and as needed to provide the service, comply with legal and tax obligations (including order and payout records), resolve disputes, and meet FSSAI traceability requirements. When no longer required, we delete or anonymise it.',
    ],
  },
  {
    heading: '5. Security',
    paragraphs: [
      'We use reasonable technical and organisational measures to protect your data, including encryption in transit. No method of transmission or storage is completely secure, but we work to safeguard your information and to address any incident promptly.',
    ],
  },
  {
    heading: '6. Your rights',
    paragraphs: [
      'You may access, correct, or request deletion of your personal data, and withdraw consent for optional processing such as marketing. Some data — for example, settled payout and tax records — we are required to retain. To exercise your rights, contact us at the address below.',
    ],
  },
  {
    heading: '7. Changes and contact',
    paragraphs: [
      'We may update this policy from time to time; material changes will be notified in the app or by email.',
      // TODO(ops): provision dpo@fe3dr.com mailbox + name a resident Grievance Officer (DPDP §13)
      'For India data-protection grievances, contact our Grievance Officer at dpo@fe3dr.com. For any other privacy question or request, contact our general support and legal team at support@fe3dr.com.',
    ],
  },
];

export default function VendorPrivacyScreen() {
  return (
    <LegalScreen
      title="Privacy Policy"
      lastUpdated={LAST_UPDATED}
      intro={INTRO}
      sections={SECTIONS}
    />
  );
}
