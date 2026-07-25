// Account deletion instructions.
//
// Google Play's account-deletion policy requires a publicly reachable URL that
// explains how to delete an account and what happens to the data — reachable
// WITHOUT installing the app, because a user who has already uninstalled it
// still has the right to delete their account. This page is that URL, and it is
// the one submitted in the Play Console Data Safety form.
//
// Apple's equivalent requirement (5.1.1(v)) is satisfied in-app, on the
// Account screen of each of the three apps.

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import {
  LEGAL_GRIEVANCE_EMAIL,
  LEGAL_LAST_UPDATED,
  LEGAL_OPERATOR_FULL,
  LEGAL_SUPPORT_EMAIL,
} from '@/lib/site';

export const metadata: Metadata = {
  title: 'Delete Your Account',
  description:
    'How to delete your Fe3dr account and what happens to your data afterwards.',
  alternates: { canonical: '/account-deletion/' },
};

const SUMMARY =
  'You can delete your Fe3dr account from inside the app in under a minute — open Account, choose Delete account, and confirm with your email address. Your sign-in stops working immediately. We keep your data for 180 days so you can change your mind, then erase it permanently. If you no longer have the app installed, email us and we will do it for you.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Delete from inside the app',
    paragraphs: [
      'Deletion takes effect straight away and does not need a support ticket.',
      '• Customer app: open the Profile tab → Your Data → Delete my account.',
      '• Chef app: open Settings → Pause or delete account → Delete account.',
      '• Delivery partner app: open Settings → Pause or Delete Account → Delete account.',
      'On each screen you confirm by typing your account email address. Once you confirm, your sign-in stops working immediately and your profile disappears from the service.',
    ],
  },
  {
    heading: '2. If you have already uninstalled the app',
    paragraphs: [
      `Email ${LEGAL_SUPPORT_EMAIL} from the address on your account with the subject "Delete my account". We verify the request comes from you before acting on it, and we complete verified deletion requests within 30 days — usually far sooner.`,
      `If you would rather raise it formally, our Grievance Officer can be reached at ${LEGAL_GRIEVANCE_EMAIL}.`,
    ],
  },
  {
    heading: '3. What is deleted, and when',
    paragraphs: [
      'Immediately on deletion: your sign-in credential is destroyed, your profile stops being visible anywhere in the service, and — for chefs — your kitchen and menu are removed from the marketplace.',
      'For the next 180 days we retain your account data so that you can come back. Nothing in it is visible or usable by anyone during this window.',
      'After 180 days we erase your personal data permanently: your name, email address, phone number, delivery addresses, profile photo, documents, and messages.',
    ],
  },
  {
    heading: '4. What we keep, and why',
    paragraphs: [
      `${LEGAL_OPERATOR_FULL} is required by Indian tax law to retain a financial record of transactions for a statutory period. After the 180 days elapse we keep only that record: order totals, invoice references, and tax lines.`,
      'That record contains no name, email address, phone number, delivery address, or location data. It cannot be linked back to you.',
      'We may also retain specific records where the law separately requires it — for example a food-safety complaint under FSSAI traceability rules. If that applies to your account we tell you when you ask.',
    ],
  },
  {
    heading: '5. Changing your mind',
    paragraphs: [
      'If you sign up again using the same email address within 180 days, we recognise the account and offer to restore it. Choose Restore and your order history comes back as it was. Choose Start fresh and we erase the old account immediately and give you a brand-new one.',
      'Restoring does not restore your standing. Chefs and delivery partners return with approval reset: your kitchen or driver profile stays invisible to customers until our team approves it again, and you must re-upload your identity documents (PAN, Aadhaar, FSSAI licence, and similar) so nothing expired or withdrawn is reinstated by accident. Menu photos, descriptions, and order history are kept.',
      'After 180 days there is nothing to restore — signing up again simply creates a new account.',
    ],
  },
  {
    heading: '6. Pausing instead of deleting',
    paragraphs: [
      'If you only want a break, every app offers a reversible pause on the same screen. Pausing hides your profile and stops notifications; for chefs it closes the kitchen and stops new orders, and for delivery partners it takes you offline.',
      'A pause has no time limit, deletes nothing, and does not reset your approval. Sign in again whenever you want to pick up where you left off.',
    ],
  },
  {
    heading: '7. Before you delete',
    paragraphs: [
      'We cannot delete an account while money or work is still in flight, because deleting would either strand a payment or leave someone without the order they paid for. The app tells you exactly what is outstanding if this applies to you.',
      '• An order still in progress — wait for it to be delivered or cancel it.',
      '• A meal plan we are still holding payment for — let it finish or cancel it.',
      '• Store credit left in your wallet — spend it first, as it cannot be transferred out.',
      '• Chef earnings not yet paid out — wait for the payout to complete.',
      '• A delivery still assigned to you — complete or hand it off.',
      'Pausing your account is available immediately and is not blocked by any of these.',
    ],
  },
  {
    heading: '8. Get a copy of your data first',
    paragraphs: [
      'Deletion is permanent once the 180 days pass, so download your data before you delete if you want to keep it. Each app offers "Download my data" on the same screen, which produces a machine-readable copy of your profile, orders, and related records.',
      'This is your right of access under the DPDP Act, 2023.',
    ],
  },
];

export default function AccountDeletionPage() {
  return (
    <LegalPage
      title="Delete Your Account"
      lastUpdated={LEGAL_LAST_UPDATED}
      summary={SUMMARY}
      sections={SECTIONS}
    />
  );
}
