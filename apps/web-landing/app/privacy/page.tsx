import type { Metadata } from 'next';
import { LegalPage } from '@/components/legal-page';

export const metadata: Metadata = {
  title: 'Privacy Policy',
  description: 'How Home Chef collects, uses, and protects your data.',
  alternates: { canonical: '/privacy/' },
};

export default function PrivacyPage() {
  return (
    <LegalPage
      title="Privacy Policy"
      summary="Home Chef collects only what's needed to connect you with home cooks and deliver your order — your account details, delivery address, and order history. We never sell your personal data."
    />
  );
}
