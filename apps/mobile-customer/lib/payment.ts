// Shared payment launch flow. Used by the cart checkout (first attempt) and by
// "Retry payment" / "Pay now" on an unpaid order. Centralising it keeps the
// create-order → gateway hand-off identical everywhere.

import { router } from 'expo-router';
import { api } from './api';
import { useCartStore } from '../store/cart-store';

/** Caches a settled non-order charge invalidates. Both gateway screens replace()
 *  back onto a screen that is already mounted, so without this it re-renders its
 *  pre-payment snapshot. */
export function chargeRefreshKeys(kind: string, chargeId: string): string[][] {
  switch (kind) {
    case 'mealplan':
      return [['meal-plans']];
    case 'group':
      return [['group-order', chargeId]];
    case 'catering':
      return [['catering-request', chargeId], ['catering-requests']];
    default:
      return [];
  }
}

export interface GatewayPaymentData {
  // "wallet" + paid:true when credit covers the full total — no gateway sheet.
  provider?: string;
  paid?: boolean;
  walletApplied?: number;
  loyaltyApplied?: number;
  /** Server-authoritative amount still due at the gateway. */
  payable?: number;
  // Cashfree hands back a payment_session_id, plus the environment — sandbox and
  // production are different hosts and nothing in the session identifies which.
  cashfreePaymentSessionId?: string;
  cashfreeOrderId?: string;
  cashfreeEnv?: string;
  amount: number;
  currency: string;
  orderNumber?: string;
  prefill?: {
    name?: string;
    email?: string;
    phone?: string;
  };
}

// Hand-off slot for the resolved gateway payload while the customer sits in the
// pre-payment hold (#hold). Route params are strings; this is a whole object,
// and serialising it into the URL would put payment session ids in navigation
// state. One slot is enough — a customer can only be paying for one thing.
let pendingGateway: { orderId: string; data: GatewayPaymentData } | null = null;

export function takePendingGateway(orderId: string): GatewayPaymentData | null {
  if (!pendingGateway || pendingGateway.orderId !== orderId) return null;
  const { data } = pendingGateway;
  pendingGateway = null;
  return data;
}

export function clearPendingGateway(): void {
  pendingGateway = null;
}

/**
 * Open the gateway for a payment the server has already created, and route to
 * the result screen. Split out of startOrderPayment so the pre-payment hold can
 * own the "actually charge me" moment without re-creating the payment.
 */
export async function launchGateway(
  orderId: string,
  data: GatewayPaymentData,
  // From the hold screen, REPLACE — the hold must not survive in the back stack
  // for the customer to return to a countdown that has already elapsed.
  opts: { replace?: boolean } = {},
): Promise<void> {
  if (data.provider === 'cashfree') {
    // Built at runtime, so typedRoutes can't narrow it — `as never` is the cast
    // this app already uses for a composed href (see checkout's group-order push).
    const href = (`/payment/cashfree?orderId=${encodeURIComponent(orderId)}` +
      `&paymentSessionId=${encodeURIComponent(data.cashfreePaymentSessionId ?? '')}` +
      `&cashfreeOrderId=${encodeURIComponent(data.cashfreeOrderId ?? '')}` +
      `&env=${encodeURIComponent(data.cashfreeEnv ?? '')}`) as never;
    if (opts.replace) router.replace(href);
    else router.push(href);
    return;
  }

  // No second rail since #1086. A response the server minted on something other
  // than Cashfree cannot be opened here, so send the customer to the result screen
  // rather than silently doing nothing — it polls the real status and offers Retry.
  router.replace(`/payment/result?order_id=${orderId}`);
}

/**
 * Create (or re-create) the payment for an existing order, open the gateway, and
 * route to the result screen. Safe to call on a pending order to retry — the
 * server rejects already-paid orders with 400.
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
  // Seconds to hold before the gateway opens, giving the customer a window to
  // change their mind with nothing charged (#hold). Only for a first placement —
  // a retry on an unpaid order should go straight to paying. 0 = no hold.
  opts: { holdSeconds?: number } = {},
): Promise<void> {
  const resp = await api.post<{ data: GatewayPaymentData }>(
    `/v1/payments/order/${orderId}/create`,
    credit,
  );
  const data = resp.data.data ?? (resp.data as unknown as GatewayPaymentData);

  // Full-wallet order: store credit covered the total, so the server already
  // marked it paid — no gateway sheet. Go straight to the result poller.
  if (data.provider === 'wallet' || data.paid) {
    useCartStore.getState().clearCart();
    router.replace(`/payment/result?order_id=${orderId}`);
    return;
  }

  // Pre-payment hold: the payment now EXISTS at the gateway but nothing has been
  // charged — a session is not a charge — so cancelling here costs the customer
  // nothing. Held after /create rather than before it so a fully wallet-funded
  // order (already settled above) never sits through a countdown it can't act on.
  const holdSeconds = opts.holdSeconds ?? 0;
  if (holdSeconds > 0) {
    pendingGateway = { orderId, data };
    router.replace(
      `/payment/hold?orderId=${encodeURIComponent(orderId)}&seconds=${holdSeconds}`,
    );
    return;
  }

  // Cashfree kitchens open the v3 web SDK in a WebView rather than a native
  // sheet. That is a deliberate trade: the Cashfree RN SDK is a native module, so
  // using it would gate the gateway behind a new EAS build and a store release,
  // whereas the WebView ships OTA. The sheet itself (including UPI intent) is the
  // same one the native SDK presents.
  await launchGateway(orderId, data);
}
