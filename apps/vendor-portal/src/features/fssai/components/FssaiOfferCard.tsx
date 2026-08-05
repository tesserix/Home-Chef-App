import { Link } from 'react-router-dom';
import { BadgeCheck, ChevronRight } from 'lucide-react';
import { formatCurrency } from '@/shared/utils/format';
import { fssaiStatusLabel, isFssaiClosed, useFssaiQuote, useFssaiRequest } from '../hooks/useFssai';

// The FSSAI filing offer, wherever a chef might realise they need one — the web
// twin of apps/mobile-vendor/components/vendor/FssaiOfferCard.tsx.
//
// Self-hiding: renders nothing when the service is switched off, and turns into
// a tracker link once a request exists, so a surface can drop it in without
// knowing anything about the request's state.

export function FssaiOfferCard() {
  const quoteQuery = useFssaiQuote();
  const requestQuery = useFssaiRequest();

  const enabled = quoteQuery.data?.enabled ?? false;
  const request = requestQuery.data?.request ?? null;
  const live = request && !isFssaiClosed(request.status) ? request : null;

  if (!enabled && !live) return null;

  const total = quoteQuery.data?.quote.total;

  return (
    <Link
      to="/fssai"
      className="flex items-center gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-muted/50"
    >
      <BadgeCheck className="h-5 w-5 shrink-0 text-ink-soft" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-foreground">
          {live ? 'Your FSSAI request' : 'No FSSAI registration yet?'}
        </p>
        <p className="text-xs text-ink-soft">
          {live
            ? fssaiStatusLabel(live.status)
            : total
              ? `We can obtain it for you. ${formatCurrency(total)} all in, including the government fee.`
              : 'We can complete the whole application for you.'}
        </p>
      </div>
      <ChevronRight className="h-4 w-4 shrink-0 text-ink-soft" />
    </Link>
  );
}
