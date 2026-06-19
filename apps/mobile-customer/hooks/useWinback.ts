import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Win-back offer (#42) — the customer's current active offer for the home banner.
// The promo + push/email were issued server-side; this just surfaces it in-app.

export interface WinbackOffer {
  code: string;
  discountPercent: number;
  expiresAt: string;
  audienceType: string;
  trigger: string;
}

export function useWinback() {
  return useQuery<WinbackOffer | null>({
    queryKey: ['winback', 'active'],
    queryFn: async () => {
      const r = await api.get<{ offer: WinbackOffer | null }>('/v1/customer/winback/active');
      return r.data?.offer ?? null;
    },
    staleTime: 5 * 60 * 1000,
  });
}
