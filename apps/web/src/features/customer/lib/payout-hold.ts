import type { PayoutHoldStatus } from '@/shared/types';

// Escrow payout-hold presentation + gating for the customer "Confirm received"
// flow (#617/#387). Web counterpart of apps/mobile-customer/lib/payout-hold.ts —
// same gate, same states, so a customer sees the same thing whichever surface
// they opened. The backend parks a delivered, gateway-charged order's hold at
// `awaiting_customer_confirmation`; confirming advances it to `release_eligible`
// (or `disputed` when an open issue exists). Every surface gates purely on this
// status, so when no hold exists the CTA simply never renders.

/** A delivered fulfilment whose hold state we can act on. */
export interface Confirmable {
  status: string;
  payoutHoldStatus?: PayoutHoldStatus;
}

/**
 * Whether to show the "Confirm received" CTA. True only for a DELIVERED order
 * whose hold is awaiting the customer's confirmation — the sole state the
 * confirm endpoint accepts. Undefined/'' (no hold) → false.
 */
export function canConfirmReceipt(f: Confirmable): boolean {
  return f.status === 'delivered' && f.payoutHoldStatus === 'awaiting_customer_confirmation';
}

export interface PayoutHoldMeta {
  /** Short label; empty string when nothing should render. */
  label: string;
  /** Tailwind classes for the inline status mark. */
  className: string;
}

/**
 * The non-actionable states worth surfacing on a delivered order: confirmed
 * (release_eligible/released) and disputed — a calm neutral, since it is "under
 * review", not an error. `awaiting_customer_confirmation` returns an empty label
 * because that state renders the CTA instead; the terminal admin states
 * (withheld/reversed) and no-hold return empty too.
 */
export function payoutHoldMeta(status?: PayoutHoldStatus): PayoutHoldMeta {
  switch (status) {
    case 'release_eligible':
    case 'released':
      return { label: 'Received', className: 'text-herb' };
    case 'disputed':
      return { label: 'Issue under review', className: 'text-ink-soft' };
    default:
      return { label: '', className: 'text-ink-soft' };
  }
}
