import { customerColors } from '@homechef/mobile-shared/theme';

// Escrow payout-hold presentation + gating for the customer "Confirm received"
// flow (#617/#387). The backend parks a delivered order / meal-plan day's hold at
// `awaiting_customer_confirmation` ONLY when the escrow flags are on; confirming
// advances it to `release_eligible` (or `disputed` when an open issue exists). So
// every surface gates purely on this status — with the flags off the field is
// empty/absent and the CTA simply never renders. Keeping the copy + colours here
// keeps the order-detail, orders-list, and meal-plan surfaces consistent.

// Mirrors models/payout_hold.go PayoutHoldStatus. '' (or undefined) = no hold.
export type PayoutHoldStatus =
  | ''
  | 'awaiting_customer_confirmation'
  | 'release_eligible'
  | 'released'
  | 'disputed'
  | 'withheld'
  | 'reversed';

/** A delivered fulfilment (order or meal-plan day) whose hold state we can act on. */
export interface Confirmable {
  status: string;
  payoutHoldStatus?: PayoutHoldStatus;
}

/**
 * Whether to show the "Confirm received" CTA. True only for a DELIVERED
 * fulfilment whose hold is awaiting the customer's confirmation — the sole state
 * the confirm endpoints accept. Undefined/'' (escrow off, or no hold) → false, so
 * the CTA is inert until the flags flip. Orders use status 'delivered'; meal-plan
 * days likewise use day.status 'delivered'.
 */
export function canConfirmReceipt(f: Confirmable): boolean {
  return f.status === 'delivered' && f.payoutHoldStatus === 'awaiting_customer_confirmation';
}

export interface PayoutHoldMeta {
  /** Short pill label; empty string when nothing should render. */
  label: string;
  color: string;
  bg: string;
}

/**
 * The non-actionable states worth surfacing as a pill on a delivered fulfilment:
 * confirmed (release_eligible/released → success green) and disputed (a calm
 * neutral — it is "under review", not an error). `awaiting_customer_confirmation`
 * returns an empty label because that state renders the CTA button instead, and
 * the terminal admin states (withheld/reversed) and no-hold return empty too.
 */
export function payoutHoldMeta(status?: PayoutHoldStatus): PayoutHoldMeta {
  switch (status) {
    case 'release_eligible':
    case 'released':
      return {
        label: 'Received',
        color: customerColors.success.DEFAULT,
        bg: customerColors.success.tint,
      };
    case 'disputed':
      return {
        label: 'Issue under review',
        color: customerColors.charcoal.soft,
        bg: customerColors.surface.soft,
      };
    default:
      return { label: '', color: customerColors.charcoal.soft, bg: customerColors.surface.soft };
  }
}
