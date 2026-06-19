import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';

// useCreateTip — post-delivery tip (#45). Creates the Razorpay charge that
// Route-splits 100% to the chef and/or rider; the caller then opens the shared
// checkout sheet (kind='tip') to pay, which verifies via /payments/tip/:id/verify.
export interface CreateTipResponse {
  tipId: string;
  razorpayOrderId: string;
  razorpayKeyId: string;
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
