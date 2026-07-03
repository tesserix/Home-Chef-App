// Shared Razorpay Standard Checkout display config. India-first: surface UPI
// (GPay / PhonePe / BHIM) as the top block, then Razorpay's default blocks
// (cards, netbanking, wallet, pay-later). Used by BOTH the native SDK sheet
// (lib/payment.ts) and the WebView checkout.js flow (app/payment/checkout.tsx)
// so the method ordering is identical everywhere.
//
// Caveat: UPI only *renders* when the checkout can launch a UPI app — a real
// Android device, or an iPhone with a UPI app (GPay/PhonePe) installed. It will
// not appear on the iOS Simulator (no UPI apps), regardless of this config.
export const RAZORPAY_DISPLAY_CONFIG = {
  display: {
    blocks: {
      upi: {
        name: 'Pay using UPI',
        instruments: [{ method: 'upi' }],
      },
    },
    sequence: ['block.upi'],
    // Keep cards/netbanking/wallet/pay-later visible below the UPI block.
    preferences: { show_default_blocks: true },
  },
} as const;
