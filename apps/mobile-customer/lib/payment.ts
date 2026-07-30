// Shared payment launch flow. Used by the cart checkout (first attempt) and by
// "Retry payment" / "Pay now" on an unpaid order. Centralising it keeps the
// create-order → native-sheet hand-off identical everywhere.
//
// Uses the react-native-razorpay NATIVE checkout sheet (not a WebView) so the
// customer never sees a web page load — just our screens and the native sheet.

import RazorpayCheckout from 'react-native-razorpay';
import { router } from 'expo-router';
import { api } from './api';
import { RAZORPAY_DISPLAY_CONFIG } from './razorpay-config';
import { useCartStore } from '../store/cart-store';

export interface RazorpayPaymentData {
  // "wallet" + paid:true when credit covers the full total — no gateway sheet.
  provider?: string;
  paid?: boolean;
  walletApplied?: number;
  loyaltyApplied?: number;
  /** Server-authoritative amount still due at the gateway. */
  payable?: number;
  // Cashfree hands back a payment_session_id instead of an (order id, key id)
  // pair, plus the environment — sandbox and production are different hosts, so
  // unlike Razorpay's key prefix there is nothing for the client to infer it from.
  cashfreePaymentSessionId?: string;
  cashfreeOrderId?: string;
  cashfreeEnv?: string;
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

// react-native-razorpay ships no types — model the bits we use.
interface RazorpaySuccess {
  razorpay_payment_id: string;
  razorpay_order_id: string;
  razorpay_signature: string;
}
interface RazorpayError {
  code?: number; // 2 = PAYMENT_CANCELLED (user dismissed the sheet)
  description?: string;
}

/**
 * Create (or re-create) the Razorpay payment for an existing order, open the
 * NATIVE checkout sheet, and route to the result screen. Safe to call on a
 * pending order to retry — the server rejects already-paid orders with 400.
 *
 * The result screen is authoritative: it polls the order's real paymentStatus
 * (set by the verify below OR the payment.captured webhook), so a failed
 * client-side verify never shows a false failure.
 *
 * @param orderId  internal order id
 * @param credit   which credit rails to apply, and optionally how much of each
 *
 * The client sends INTENT, never a computed payable: the server re-runs the whole
 * allocation from the live balance and the real order, and its answer is what is
 * charged. Posting an amount is what let the screen show one figure while the
 * gateway took another.
 */
export async function startOrderPayment(
  orderId: string,
  credit: {
    useWallet: boolean;
    walletAmount?: number;
    useLoyalty: boolean;
    loyaltyPoints?: number;
  } = { useWallet: false, useLoyalty: false },
): Promise<void> {
  const resp = await api.post<{ data: RazorpayPaymentData }>(
    `/v1/payments/order/${orderId}/create`,
    credit,
  );
  const data = resp.data.data ?? (resp.data as unknown as RazorpayPaymentData);

  // Full-wallet order: store credit covered the total, so the server already
  // marked it paid — no gateway sheet. Go straight to the result poller.
  if (data.provider === 'wallet' || data.paid) {
    useCartStore.getState().clearCart();
    router.replace(`/payment/result?order_id=${orderId}`);
    return;
  }

  // Cashfree kitchens open the v3 web SDK in a WebView rather than a native
  // sheet. That is a deliberate trade: the Cashfree RN SDK is a native module, so
  // using it would gate the gateway behind a new EAS build and a store release,
  // whereas the WebView ships OTA. The sheet itself (including UPI intent) is the
  // same one the native SDK presents.
  if (data.provider === 'cashfree') {
    router.push(
      `/payment/cashfree?orderId=${encodeURIComponent(orderId)}` +
        `&paymentSessionId=${encodeURIComponent(data.cashfreePaymentSessionId ?? '')}` +
        `&cashfreeOrderId=${encodeURIComponent(data.cashfreeOrderId ?? '')}` +
        `&env=${encodeURIComponent(data.cashfreeEnv ?? '')}`,
    );
    return;
  }

  const options = {
    key: data.razorpayKeyId,
    order_id: data.razorpayOrderId,
    amount: data.amount,
    currency: data.currency ?? 'INR',
    name: 'Fe3dr',
    description: 'Order payment',
    prefill: {
      name: data.prefill?.name ?? '',
      email: data.prefill?.email ?? '',
      contact: data.prefill?.phone ?? '',
    },
    // UPI-first ordering (GPay/PhonePe/BHIM on top), cards/netbanking below.
    config: RAZORPAY_DISPLAY_CONFIG,
    theme: { color: '#FF385C' },
  };

  try {
    const result: RazorpaySuccess = await RazorpayCheckout.open(options);
    // Fast-path verify. The result screen polls the server status as a backstop
    // (webhook), so we swallow a verify failure here rather than surfacing it.
    try {
      await api.post(`/v1/payments/order/${orderId}/verify`, {
        razorpayPaymentId: result.razorpay_payment_id,
        razorpayOrderId: result.razorpay_order_id,
        razorpaySignature: result.razorpay_signature,
      });
      useCartStore.getState().clearCart();
    } catch {
      // ignore — the result screen confirms via polling
    }
    router.replace(`/payment/result?order_id=${orderId}`);
  } catch (err) {
    const e = err as RazorpayError;
    // User dismissed the sheet — return to where they were, instantly.
    if (e?.code === 2 || /cancel/i.test(e?.description ?? '')) {
      router.back();
      return;
    }
    // Genuine failure — the result screen shows status + a Retry option.
    router.replace(`/payment/result?order_id=${orderId}`);
  }
}
