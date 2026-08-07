import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';

// useCreateTip — post-delivery tip (#45). Creates the charge that pays 100% to
// the chef and/or rider; the caller then opens the sheet for whichever gateway
// the server used and verifies via /payments/tip/:id/verify.
//
// A tip is a new charge, so it is always minted on Cashfree (#1086) — Easy Split
// routes the whole tip to the chef's vendor account. There is no key id to hand
// to a checkout sheet; the payment session is opened directly.
export interface CreateTipResponse {
  tipId: string;
  provider: string;
  cashfreeOrderId: string;
  cashfreePaymentSessionId: string;
  cashfreeEnv: string;
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
