import { formatMoney } from './format';

// Where a refund actually landed, in the customer's words.
//
// The server divides a refund across the rails that funded the order
// (SplitRefundByFunding), so an order part-paid with credit returns part to the
// card and part to the wallet. The order screen used to name a single
// destination taken from the cancellation request, which describes only the
// CARD slice — so a refund of ₹377.07 that put ₹132.22 on the card and ₹244.85
// in the wallet was reported as "₹377 refunded to your card".
//
// Loyalty is folded into the wallet figure rather than named separately: points
// are returned as wallet rupees, not restored as points, so "to your wallet" is
// where that money genuinely is.

export function refundDestinationLine(
  totalPaise: number,
  walletRefunded?: number,
  loyaltyRefunded?: number,
  destination?: string,
): string {
  const total = totalPaise / 100;
  const toWallet = (walletRefunded ?? 0) + (loyaltyRefunded ?? 0);
  // Never claim more reached the gateway than the refund itself, and never a
  // negative slice, however the two sources disagree.
  const toCard = Math.min(total, Math.max(0, total - toWallet));
  const gateway = destination === 'wallet' ? 'wallet' : 'card';

  if (toWallet <= 0) return `${formatMoney(total)} refunded to your ${gateway}.`;
  if (toCard <= 0) return `${formatMoney(total)} refunded to your wallet.`;
  return `${formatMoney(total)} refunded — ${formatMoney(toCard)} to your ${gateway}, ${formatMoney(toWallet)} to your wallet.`;
}
