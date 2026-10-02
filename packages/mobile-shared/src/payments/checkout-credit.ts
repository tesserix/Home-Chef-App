export type CheckoutCreditIntent = {
  useWallet: boolean;
  useLoyalty: boolean;
  walletAmount?: number;
  loyaltyPoints?: number;
};

export function checkoutCreditIntent(provider: string | undefined, intent: CheckoutCreditIntent): CheckoutCreditIntent {
  if (provider === 'stripe') return { useWallet: false, useLoyalty: false };
  if (provider === 'cashfree') return intent;
  throw new Error('Please wait for your payment quote and try again.');
}
