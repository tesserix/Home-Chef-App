import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Gift, X } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { useAuth } from '@/app/providers/AuthProvider';

// Win-back offer banner (#42) — surfaces the customer's active offer at the top of
// the home page (the promo + push/email were issued server-side). Dismissible.

interface WinbackOffer {
  code: string;
  discountPercent: number;
  expiresAt: string;
}

function daysLeft(iso: string): number {
  const ms = new Date(iso).getTime() - Date.now();
  return Math.max(0, Math.ceil(ms / 86_400_000));
}

export function WinbackBanner() {
  const { isAuthenticated } = useAuth();
  const [dismissed, setDismissed] = useState(false);
  const { data } = useQuery({
    queryKey: ['winback', 'active'],
    queryFn: () => apiClient.get<{ offer: WinbackOffer | null }>('/winback/active'),
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000,
  });
  const offer = (data as unknown as { offer: WinbackOffer | null } | undefined)?.offer ?? null;
  if (!offer || dismissed) return null;

  const left = daysLeft(offer.expiresAt);
  return (
    <div className="container-app pt-6">
      <div className="flex items-center gap-3 rounded-xl border border-herb/30 bg-herb-tint px-4 py-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-bone">
          <Gift className="h-5 w-5 text-herb" aria-hidden="true" />
        </div>
        <div className="flex-1">
          <p className="text-sm font-semibold text-ink">
            We miss you — {Math.round(offer.discountPercent)}% off your next order
          </p>
          <p className="text-xs text-ink-soft">
            Use code <span className="font-semibold text-herb">{offer.code}</span> at checkout
            {left > 0 ? ` · ${left} day${left === 1 ? '' : 's'} left` : ''}
          </p>
        </div>
        <button
          type="button"
          onClick={() => setDismissed(true)}
          className="text-ink-soft hover:text-ink"
          aria-label="Dismiss offer"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
