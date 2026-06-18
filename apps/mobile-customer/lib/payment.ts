// Shared payment launch flow. Used by the cart checkout (first attempt) and by
// "Retry payment" / "Pay now" on an unpaid order. Centralising it keeps the
// create-order → gateway-sheet hand-off identical everywhere.

import { router } from 'expo-router';
import { api } from './api';
import { useCartStore } from '../store/cart-store';

export interface RazorpayPaymentData {
  // "wallet" + paid:true when store credit covers the full total — no gateway sheet.
  provider?: string;
  paid?: boolean;
  walletApplied?: number;
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number;
  currency: string;
  orderNumber?: string;
  prefill?: {
    name?: string;
    email?: string;
    phone?: string;
  };
}

/**
 * Create (or re-create) the Razorpay payment for an existing order and open the
 * in-app checkout sheet. Safe to call on a pending order to retry — the server
 * rejects already-paid orders with 400 ("Order already paid").
 *
 * @param orderId  internal order id
 * @param opts.walletAmount  store credit to apply (#141)
 * @param opts.replace  use router.replace (e.g. retrying from the result screen)
 *                      instead of push, so back doesn't return to the dead screen
 */
export async function startOrderPayment(
  orderId: string,
  opts: { walletAmount?: number; replace?: boolean } = {},
): Promise<void> {
  const resp = await api.post<{ data: RazorpayPaymentData }>(
    `/v1/payments/order/${orderId}/create`,
    opts.walletAmount && opts.walletAmount > 0 ? { walletAmount: opts.walletAmount } : {},
  );
  const data = resp.data.data ?? (resp.data as unknown as RazorpayPaymentData);

  // Full-wallet order: store credit covered the total, so the server already
  // marked it paid — no gateway sheet. Go straight to the result poller.
  if (data.provider === 'wallet' || data.paid) {
    useCartStore.getState().clearCart();
    router.replace(`/payment/result?order_id=${orderId}`);
    return;
  }

  const target = {
    pathname: '/payment/checkout' as const,
    params: {
      orderId,
      razorpayOrderId: data.razorpayOrderId,
      razorpayKeyId: data.razorpayKeyId,
      amount: String(data.amount),
      currency: data.currency ?? 'INR',
      name: data.prefill?.name ?? '',
      email: data.prefill?.email ?? '',
      phone: data.prefill?.phone ?? '',
    },
  };
  if (opts.replace) router.replace(target);
  else router.push(target);
}
