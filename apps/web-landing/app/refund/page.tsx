import type { Metadata } from 'next';
import { LegalPage } from '@/components/legal-page';

export const metadata: Metadata = {
  title: 'Refund & Cancellation Policy',
  description:
    'How cancellations and refunds work for Home Chef orders.',
  alternates: { canonical: '/refund/' },
};

export default function RefundPage() {
  return (
    <LegalPage
      title="Refund & Cancellation Policy"
      summary="You can cancel an order before the chef starts cooking for a full refund. If something goes wrong with a delivered order, our support team makes it right — fairly for you and for the chef."
    />
  );
}
