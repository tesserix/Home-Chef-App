import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';

// useCreateTip — post-delivery tip (#45). Creates the charge that pays 100% to
// the chef and/or rider; the caller then opens the sheet for whichever gateway
// the server used and verifies via /payments/tip/:id/verify.
//
// The tip rides the SAME gateway as the order it thanks, so exactly one of the
// two shapes below comes back: Cashfree returns a payment session and no key id
// (Easy Split routes the whole tip to the chef's vendor account), Razorpay the
// reverse. Both are optional here because the server, not the client, decides.
export interface CreateTipResponse {
  tipId: string;
  provider?: string;
  razorpayOrderId?: string;
  razorpayKeyId?: string;
  cashfreeOrderId?: string;
  cashfreePaymentSessionId?: string;
  mode?: string;
  amount: number;
  currency: string;
}

export function useCreateTip() {
  return useMutation({
    mutationFn: (vars: {
      orderId: string;
      chefAmount: number;
      riderAmount: number;
    }) =>
      api
        .post<CreateTipResponse>(`/v1/payments/order/${vars.orderId}/tip`, {
          chefAmount: vars.chefAmount,
          riderAmount: vars.riderAmount,
        })
        .then((r) => r.data),
  });
}
