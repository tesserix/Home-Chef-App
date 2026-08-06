// What a cancelled order retained, split the way the customer was charged it.
//
// The retained remainder is not one thing: the chef keeps the food they cooked
// (#945), the platform keeps its fee, and the tax on both is withheld too.
// Printing the whole platform-side residual as "Platform fee" contradicted the
// order's own "Platform fee" row a few lines above (#1048).

const round2 = (n: number) => Math.round(n * 100) / 100;

export interface RetainedSplitInput {
  totalAmount: number;
  refundAmount: number;
  /** The chef's share of the retention, from the cancellation snapshot. */
  vendorKept: number;
  /** The order's own platform fee row, as charged at checkout. */
  platformFee?: number;
}

export interface RetainedSplit {
  platformFee: number;
  taxWithheld: number;
}

// The fee is capped at the residual so the two lines always sum to what was
// actually kept — a refund that reached into the fee must not print the full
// checkout fee back at the customer.
export function splitRetainedAmount(input: RetainedSplitInput): RetainedSplit {
  const residual = round2(input.totalAmount - input.refundAmount - input.vendorKept);
  if (residual <= 0) return { platformFee: 0, taxWithheld: 0 };

  const platformFee = round2(Math.min(Math.max(input.platformFee ?? 0, 0), residual));
  return { platformFee, taxWithheld: round2(residual - platformFee) };
}
