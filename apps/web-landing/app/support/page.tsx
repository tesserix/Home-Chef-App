// Public support page.
//
// This is the `supportUrl` declared on both App Store listings
// (apps/mobile-customer/store.config.json, apps/mobile-vendor/store.config.json)
// and it has to resolve — App Review guideline 1.5 requires a support URL where
// a user can actually find out how to get help. It must therefore work for
// someone who has NOT installed the app, which is why email leads and the
// in-app paths follow.
//
// Everything on this page is a commitment that already exists somewhere in the
// repo: the response times come from the Terms of Service and Privacy Policy,
// the refund windows from the Refund & Cancellation Policy, and the in-app
// paths from the screens themselves. Nothing here may be aspirational — no
// phone number, no hours, no SLA that is not written down elsewhere.

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import {
  CHEFS_EMAIL,
  CUSTOMER_APP,
  LEGAL_GRIEVANCE_EMAIL,
  LEGAL_SUPPORT_EMAIL,
  VENDOR_APP,
} from '@/lib/site';

export const metadata: Metadata = {
  title: 'Help & Support',
  description:
    'How to get help with a Fe3dr order, refund, or account — in the app or by email.',
  alternates: { canonical: '/support/' },
};

const SUMMARY =
  `The quickest way to get help is the support chat inside the app: it already knows who you are and which orders are yours. If you cannot get to the app — you have uninstalled it, or you cannot sign in — email ${LEGAL_SUPPORT_EMAIL} from the address on your account and we will pick it up from there.`;

const SECTIONS: LegalSection[] = [
  {
    heading: 'Email us',
    paragraphs: [
      `Write to ${LEGAL_SUPPORT_EMAIL} from the email address on your account. If your question is about an order, include the order number and tell us what you expected to happen.`,
      'We acknowledge complaints within 48 hours and aim to resolve them within 30 days, as set out in our Terms of Service.',
      `If you want to cook on Fe3dr rather than order from it, write to ${CHEFS_EMAIL} instead.`,
    ],
    links: [{ href: '/terms/', label: 'Terms of Service' }],
  },
  {
    heading: 'Get help inside the app',
    paragraphs: [
      'Support in the app carries your signed-in account with it, so you do not have to verify who you are before anyone can help you.',
      `• In the ${CUSTOMER_APP.name} app, open the Profile tab and tap Help & support to start a chat with our support team.`,
      `• In the ${VENDOR_APP.name} app, open More, then Support, to raise a ticket and follow its status.`,
      'In the customer chat you choose what it is about — order tracking or ETA, a delivery problem, a refund request, a question about a chef or a dish, an account or sign-in issue, or something else. A chat can be turned into a tracked support ticket, so a conversation you start on your phone does not get lost when you close the app.',
    ],
  },
  {
    heading: 'Refunds and cancellations',
    paragraphs: [
      'You can cancel free of charge until the chef starts cooking. After that, what we can refund depends on how far the order has got, because a home chef cannot resell a meal that is already made.',
      'Once a refund is approved it goes back to your original payment method within 7 working days — the maximum window the Reserve Bank of India sets for payment aggregators. Raise a refund request through the support chat or by email, and raise it within 24 hours of delivery so the chef and the delivery partner can still recall the order accurately.',
      'The full policy sets out what happens at each order stage, what is not refundable, and how catering orders differ.',
    ],
    links: [{ href: '/refund/', label: 'Refund & Cancellation Policy' }],
  },
  {
    heading: 'Delete your account, or get a copy of your data',
    paragraphs: [
      'You can delete your account from inside the app in under a minute — you do not need to contact support to do it. If you have already uninstalled the app, email us from the address on your account with the subject "Delete my account" and we will do it for you.',
      'Each app also offers "Download my data", which produces a machine-readable copy of your profile, orders, and related records.',
      'Full instructions — what is erased, what we are required to keep, and how long you have to change your mind — are on the account deletion page.',
    ],
    links: [{ href: '/account-deletion/', label: 'Delete Your Account' }],
  },
  {
    heading: 'Privacy questions and grievances',
    paragraphs: [
      `If your question is about your personal data — a copy of it, a correction, or erasure — our Grievance Officer for India can be reached at ${LEGAL_GRIEVANCE_EMAIL}. Our Privacy Policy sets the timelines for that channel: acknowledgement within 24 hours, and resolution within 15 days.`,
      'If we cannot resolve a complaint, you can take it further. Data-protection complaints go to the Data Protection Board of India. Consumer complaints can go to the National Consumer Helpline on 1915, or through the e-Daakhil portal.',
    ],
    links: [{ href: '/privacy/', label: 'Privacy Policy' }],
  },
  {
    heading: 'Everything we have put in writing',
    paragraphs: [
      'The policies below are the ones that govern your account, your orders, and your data.',
    ],
    links: [
      { href: '/privacy/', label: 'Privacy Policy' },
      { href: '/terms/', label: 'Terms of Service' },
      { href: '/refund/', label: 'Refund & Cancellation Policy' },
      { href: '/eula/', label: 'End User Licence Agreement' },
      { href: '/vendor-terms/', label: 'Chef & Vendor Agreement' },
      { href: '/account-deletion/', label: 'Delete Your Account' },
    ],
  },
];

export default function SupportPage() {
  return (
    <LegalPage
      title="Help & Support"
      summary={SUMMARY}
      sections={SECTIONS}
      contactPrompt="Cannot find what you need?"
    />
  );
}
