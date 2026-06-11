import type { Metadata } from 'next';
import { LegalPage } from '@/components/legal-page';

export const metadata: Metadata = {
  title: 'Terms of Service',
  description: 'The terms that govern your use of Home Chef.',
  alternates: { canonical: '/terms/' },
};

export default function TermsPage() {
  return (
    <LegalPage
      title="Terms of Service"
      summary="These terms govern your use of the Home Chef apps and fe3dr.com — ordering from independent home chefs, payments, delivery, and acceptable use of the platform."
    />
  );
}
