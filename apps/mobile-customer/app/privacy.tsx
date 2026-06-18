// Privacy Policy — static legal screen.
// Content is a starting template; have it reviewed by legal counsel before launch.

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '11 June 2026';

const INTRO =
  'This Privacy Policy explains how we collect, use, and protect your personal information when you use the Fe3dr customer app. Fe3dr is a product of Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia ("Fe3dr", "we", "us"). We handle personal data in line with the Australian Privacy Principles (APP) and, for our India operations, in alignment with India’s Digital Personal Data Protection Act, 2023 (DPDP).';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Information we collect',
    paragraphs: [
      '• Account details you provide — name, email, phone number, and password.',
      '• Order and delivery information — your saved addresses, order history, and notes to the chef.',
      '• Payment information — processed by our payment gateway; we receive confirmation and limited details, not your full card number.',
      '• Device and usage data — app interactions, device type, and diagnostic information used to keep the service reliable.',
      '• Location data — used to show nearby chefs and support delivery, when you grant permission.',
    ],
  },
  {
    heading: '2. How we use your information',
    paragraphs: [
      'We use your information to create and manage your account, process and deliver your orders, handle payments and refunds, provide customer support, keep the platform secure, and improve our service. With your consent, we may send you updates and offers — you can opt out at any time.',
    ],
  },
  {
    heading: '3. Who we share it with',
    paragraphs: [
      '• Home chefs — the order details and the delivery information needed to prepare and fulfil your order.',
      '• Delivery partners — your delivery address and contact details for the duration of the delivery.',
      '• Payment processor (Razorpay) — to process payments and refunds securely.',
      '• Service providers — who help us run the app (for example hosting, notifications, and analytics), under appropriate confidentiality obligations.',
      'We do not sell your personal data.',
    ],
  },
  {
    heading: '4. Location data',
    paragraphs: [
      'If you grant location permission, we use your location to surface nearby chefs and to support accurate delivery. You can turn location access off in your device settings; some features may then be limited.',
    ],
  },
  {
    heading: '5. Data retention',
    paragraphs: [
      'We keep your information for as long as your account is active and as needed to provide the service, comply with legal and tax obligations, resolve disputes, and enforce our agreements. When no longer required, we delete or anonymise it.',
    ],
  },
  {
    heading: '6. Security',
    paragraphs: [
      'We use reasonable technical and organisational measures to protect your data, including encryption in transit. No method of transmission or storage is completely secure, but we work to safeguard your information and to address any incident promptly.',
    ],
  },
  {
    heading: '7. Your rights',
    paragraphs: [
      'You may access, correct, or request deletion of your personal data, and withdraw consent for optional processing such as marketing. To exercise these rights, contact us at the address below. We will respond within the time required by applicable law.',
    ],
  },
  {
    heading: '8. Children',
    paragraphs: [
      'Fe3dr is intended for users aged 18 and over. We do not knowingly collect personal data from children. If you believe a child has provided us data, contact us and we will remove it.',
    ],
  },
  {
    heading: '9. Changes and contact',
    paragraphs: [
      'We may update this policy from time to time; material changes will be notified in the app or by email.',
      // TODO(ops): provision dpo@fe3dr.com mailbox + name a resident Grievance Officer (DPDP §13)
      'For India data-protection grievances, contact our Grievance Officer at dpo@fe3dr.com. For any other privacy question or request, contact our general support and legal team at support@fe3dr.com.',
    ],
  },
];

export default function PrivacyPolicyScreen() {
  return (
    <LegalScreen
      title="Privacy Policy"
      lastUpdated={LAST_UPDATED}
      intro={INTRO}
      sections={SECTIONS}
    />
  );
}
