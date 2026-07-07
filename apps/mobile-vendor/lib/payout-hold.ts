import { theme } from '@homechef/mobile-shared/theme';

// Escrow payout-hold presentation for the vendor earnings screen (#617/#387).
// The chef sees where each delivered order's money sits in the escrow lifecycle
// (held awaiting the customer's confirmation → confirmed → released), plus the
// paid/pending state of a weekly settlement statement. Every hold is empty while
// the escrow flags are off, so the pills hide and the screen reads as it did
// pre-launch. Uber-monochrome tokens: ink carries actions, functional green =
// released/paid, amber = pending/awaiting, info = under review, destructive =
// withheld/reversed. Persimmon (`herb`) is never used for status.

// Mirrors models/payout_hold.go PayoutHoldStatus. '' (or undefined) = no hold.
export type PayoutHoldStatus =
  | ''
  | 'awaiting_customer_confirmation'
  | 'release_eligible'
  | 'released'
  | 'disputed'
  | 'withheld'
  | 'reversed';

export interface PillMeta {
  /** i18n key for the pill label; empty string ⇒ render no pill (no hold /
   *  escrow flags off). The caller resolves it via t(). */
  labelKey: string;
  fg: string;
  bg: string;
}

/** The escrow pill for one order on the earnings list. Empty labelKey when there
 *  is no hold, so the row renders exactly as it did before escrow. */
export function payoutHoldMeta(status?: PayoutHoldStatus): PillMeta {
  switch (status) {
    case 'awaiting_customer_confirmation':
      // Pending confirmation — amber tint + dark ink text (matches the app's
      // pending pill in admin-requests for contrast).
      return { labelKey: 'earnings.holdAwaiting', fg: theme.colors.ink.DEFAULT, bg: theme.colors.amber.tint };
    case 'release_eligible':
      return { labelKey: 'earnings.holdConfirmed', fg: theme.colors.success.soft, bg: theme.colors.success.tint };
    case 'released':
      return { labelKey: 'earnings.holdReleased', fg: theme.colors.success.soft, bg: theme.colors.success.tint };
    case 'disputed':
      return { labelKey: 'earnings.holdDisputed', fg: theme.colors.info.DEFAULT, bg: theme.colors.info.tint };
    case 'withheld':
      return { labelKey: 'earnings.holdWithheld', fg: theme.colors.destructive.DEFAULT, bg: theme.colors.destructive.tint };
    case 'reversed':
      return { labelKey: 'earnings.holdReversed', fg: theme.colors.destructive.DEFAULT, bg: theme.colors.destructive.tint };
    default:
      return { labelKey: '', fg: theme.colors.ink.muted, bg: theme.colors.mist.DEFAULT };
  }
}

/** The paid/pending pill for a weekly settlement statement. */
export function statementStatusMeta(status: 'pending' | 'paid'): PillMeta {
  return status === 'paid'
    ? { labelKey: 'earnings.statementPaid', fg: theme.colors.success.soft, bg: theme.colors.success.tint }
    : { labelKey: 'earnings.statementPending', fg: theme.colors.ink.DEFAULT, bg: theme.colors.amber.tint };
}
