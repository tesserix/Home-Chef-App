// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import { LEGAL_LAST_UPDATED, LEGAL_OPERATOR, LEGAL_SUPPORT_EMAIL } from '@/lib/site';

export const metadata: Metadata = {
  title: 'Privacy Policy',
  description: 'How Home Chef collects, uses, and protects your data.',
  alternates: { canonical: '/privacy/' },
};

const SUMMARY =
  'We collect the minimum personal data we need to deliver food: your name, contact details, delivery address, and payment information. We do not sell your data. We share it only with the partners who help us run the service — payment processors, cloud hosts, communication providers — and only for the purpose we collected it. You can ask us for a copy, fix mistakes, or delete your account at any time. If something goes wrong, our Grievance Officer responds within 15 days.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Who we are',
    paragraphs: [
      `${LEGAL_OPERATOR} operates Fe3dr. In this policy we call ourselves "we", "us", or Fe3dr, and we call you "you".`,
      'Under the Digital Personal Data Protection Act, 2023 (the DPDP Act), we are the Data Fiduciary — we decide why and how your personal data is processed. You are the Data Principal, the person the data is about.',
      'When we say Personal Data, we mean any information that identifies you or could identify you — your name, your phone number, your address, your face in a profile photo. When we say Processing, we mean anything we do with that data: collecting it, storing it, using it, sharing it, or deleting it.',
    ],
  },
  {
    heading: '2. What we collect',
    paragraphs: [
      'We collect only what we need, and what we collect depends on how you use Fe3dr.',
      '• When you create an account: your full name, email address, mobile phone number, a password (stored only as a one-way hash), and date of birth so we can confirm you are 18 or older.',
      '• When you sign in with Google, Facebook, or Apple: your name, email, profile photo, and a unique account identifier from the provider. We do not receive your password.',
      '• When you place an order: your delivery address and access instructions, the items you order and your dietary notes, your payment instrument in tokenised form (we never store your full card number or CVV), and the coordinates of your delivery address so the driver can find it.',
      '• When you message a chef or driver: the text of in-app messages, so chefs can read special requests and we can investigate complaints.',
      '• When you use the app: device type, operating system, app version, IP address and approximate location, the screens you view and the buttons you tap, and crash reports.',
      'We do not collect your contacts, photos, microphone, or precise GPS location unless you grant a permission for a specific feature, such as location at checkout to find chefs near you.',
    ],
  },
  {
    heading: '3. Why we collect it',
    paragraphs: [
      'We collect each piece of data for a specific reason:',
      '• Your name and email to create your account and let you sign in.',
      '• Your phone number to send order updates by SMS — when the chef accepts, when the driver is on the way, when the food arrives.',
      '• Your delivery address to route your order to the right place.',
      '• Your payment details to charge you for orders and to refund you when needed.',
      '• Your order history to show past orders, suggest dishes you might like, and help chefs improve their menus.',
      '• Your device and app-usage data to fix bugs and improve the app.',
      '• Your IP address and login activity to detect fraud.',
      '• Your messages with chefs and drivers to investigate complaints if you raise one.',
      '• Your email and phone for marketing only if you opt in — you can withdraw that consent at any time.',
      'We do not process your personal data for any purpose other than those above. If we ever need a new purpose, we will ask for your consent first.',
    ],
  },
  {
    heading: '4. Who we share it with',
    paragraphs: [
      'We do not sell your personal data. We share it only with the partners who help us run Fe3dr, and only for the purpose we collected it:',
      '• Payment processors (Razorpay in India; Stripe for international cards) — to process payments and refunds.',
      '• The home chef — your first name, delivery address, dietary notes, and order items, so the chef can prepare and hand off your order.',
      '• The delivery partner — your first name, delivery address, and a masked phone number, so they can deliver and contact you if needed.',
      '• Cloud hosting, email, SMS, and push-notification providers — to run the application and deliver messages reliably.',
      '• Analytics providers — anonymised event data, disabled if you opt out of analytics.',
      '• Government, courts, and regulators — only what they lawfully require.',
      'Every partner is bound by a written contract that limits their use of your data to the purpose we share it for. They act as Data Processors on our instructions, not as separate fiduciaries.',
    ],
  },
  {
    heading: '5. Where it is stored',
    paragraphs: [
      'We store the bulk of your personal data on servers located in India. Some processing takes place outside India — for example, international card payments processed by Stripe, and some email and analytics processing.',
      'The DPDP Act permits transfer of personal data outside India unless the Central Government restricts a specific country. We monitor those notifications and will stop transfers if a country is restricted.',
    ],
  },
  {
    heading: '6. How long we keep it',
    paragraphs: [
      'We keep your personal data only as long as we need it for the purpose we collected it, plus any period the law requires us to hold it. Order and payment records are kept for the period required by Indian tax law; food-safety complaint records are kept for FSSAI traceability; closed-account data is deleted or anonymised within 90 days, except records the law requires us to keep.',
      'Once the period ends, we delete the data or anonymise it so it can no longer be linked back to you.',
    ],
  },
  {
    heading: '7. Your rights',
    paragraphs: [
      'As a Data Principal under the DPDP Act you have the right to access a summary of the data we hold about you and who we have shared it with; to correct data that is wrong and erase data we no longer need; to grievance redressal through our Grievance Officer; to nominate another person to exercise your rights; and to withdraw any consent you gave us.',
      'You can use most of these rights from inside the app, or by emailing our Grievance Officer. We verify your identity before acting on a request. We respond within 15 days for grievances and within 30 days for access or erasure requests. If we cannot honour a request because the law requires us to keep certain records, we will tell you why.',
    ],
  },
  {
    heading: '8. Security',
    paragraphs: [
      'We put reasonable safeguards in place to protect your personal data, including encryption of data in transit, encryption of sensitive data at rest, access controls limiting data to staff who need it, audit logs, regular independent security reviews, and vendor reviews before we add a new processor.',
      'No system is perfect, and we do not promise that data is 100% secure. If a personal data breach that could harm you occurs, we will notify you and the Data Protection Board of India without undue delay, and tell you what happened and what you can do.',
    ],
  },
  {
    heading: '9. Children',
    paragraphs: [
      'Fe3dr is intended for adults. We do not knowingly collect personal data from anyone under 18. You confirm you are 18 or older when you create an account. If we discover a user is under 18, we delete their account and personal data, unless the law requires us to keep specific records. If you are a parent and believe a child has signed up, please contact our Grievance Officer.',
    ],
  },
  {
    heading: '10. Grievance Officer',
    paragraphs: [
      'We have appointed a Grievance Officer under Section 10 of the DPDP Act and the IT (Intermediary Guidelines) Rules, 2021. They acknowledge complaints within 24 hours and aim to resolve them within 15 days.',
      // TODO(counsel): replace bracketed Grievance Officer name and phone with the real India-resident appointee before launch (DPDP §10/§13 requirement).
      '• Name: [Grievance Officer Name]',
      '• Phone: [Grievance Officer Phone] (Mon–Fri, 10:00–18:00 IST)',
      `• Email: ${LEGAL_SUPPORT_EMAIL}`,
      'If we do not resolve your grievance, you can complain to the Data Protection Board of India.',
    ],
  },
  {
    heading: '11. Changes and contact',
    paragraphs: [
      'We update this policy when our processing changes, when the law changes, or when a partner changes. For material changes we notify you by email and in the app at least 30 days before the change takes effect. For any privacy question or request, contact us at the address above.',
    ],
  },
];

export default function PrivacyPage() {
  return (
    <LegalPage
      title="Privacy Policy"
      summary={SUMMARY}
      lastUpdated={LEGAL_LAST_UPDATED}
      sections={SECTIONS}
    />
  );
}
