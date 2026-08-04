// Payment result screen.
//
// The client-side Razorpay callback is NOT authoritative — the session can
// expire while the user is in the payment sheet, so the in-app verify call may
// fail even though Razorpay captured the money. Instead of trusting the
// callback params, this screen polls the order's real `paymentStatus` (which
// the server sets via the synchronous verify OR the payment.captured webhook)
// and shows the actual outcome. There are two genuinely different
// non-success outcomes, and conflating them is how a customer gets charged
// twice:
//   - completed                              → success
//   - server-declined (failed/refunded), or
//     no order + gateway error, or
//     no order past grace with no error       → failure + Retry payment
//   - real order, still pending/unknown
//     once past the grace window              → confirming (no Retry — a
//                                                 captured-but-unconfirmed
//                                                 payment must never be
//                                                 retried)
//
// Visual: white canvas, centered layout, safe-area aware. Coral CTAs.

import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useLocalSearchParams } from 'expo-router';
import { CheckCircle2, XCircle } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder } from '../../hooks/useOrderHistory';
import { startOrderPayment } from '../../lib/payment';
import { useCartStore } from '../../store/cart-store';

// Android ripple tints — translucent tokens, never a new literal colour.
const PRIMARY_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

interface PaymentParams {
  razorpay_payment_id?: string;
  razorpay_order_id?: string;
  order_id?: string; // the internal order id passed when launching checkout
  error?: string;
  tip?: string; // '1' when this is a post-delivery tip charge (#45)
}

// How long to hold the fast poll before backing off to the slow cadence.
// The webhook / synchronous verify normally lands within a few seconds — this
// window is not a "give up" deadline, it's just the fast/slow poll boundary.
const CONFIRM_GRACE_MS = 30_000;
// After the grace window, back off from the 2s fast poll: the reconcile cron
// (apps/api/services/order_payment_reconcile_cron.go) can take up to ~10
// minutes worst case to settle a captured-but-unconfirmed order, and holding
// a 2s poll open that whole time is unnecessary battery/radio drain for a
// screen the customer is just glancing at. 15s still surfaces a late settle
// within 15s of it happening.
const SLOW_POLL_MS = 15_000;

export default function PaymentResult() {
  const router = useRouter();
  const params = useLocalSearchParams() as unknown as PaymentParams;
  const orderId = params.order_id ?? params.razorpay_order_id ?? '';

  const [pastGrace, setPastGrace] = useState(false);
  const [retrying, setRetrying] = useState(false);

  // Authoritative: poll the server's payment status until terminal. Back off
  // to the slow cadence once past grace (see SLOW_POLL_MS above) — undefined
  // before that keeps the existing fast 2000ms default from useOrderHistory.
  const { data } = useOrder(orderId, {
    pollUntilPaid: true,
    pollIntervalMs: pastGrace ? SLOW_POLL_MS : undefined,
  });
  const paymentStatus = data?.data?.paymentStatus;
  const orderStatus = data?.data?.status;

  // A cancelled order can never be paid, so retrying it is a dead end that
  // returns to this same screen (D-10). The platform auto-cancels an order whose
  // payment never completed, which is exactly the state this screen shows — so
  // "payment failed" and "retry is possible" are different questions and only
  // the order's own status answers the second. No order id means nothing to
  // retry against either.
  const canRetry =
    Boolean(orderId) &&
    orderStatus !== 'cancelled' &&
    orderStatus !== 'rejected' &&
    orderStatus !== 'refunded';

  useEffect(() => {
    // Always arm the grace timer — even with no orderId. Without this, a result
    // screen reached without a resolvable order id (or holding a status the poll
    // never turns terminal) spins on "Confirming your payment…" forever.
    const t = setTimeout(() => setPastGrace(true), CONFIRM_GRACE_MS);
    return () => clearTimeout(t);
  }, []);

  // Derive the displayed state from the real payment status (+ grace window).
  // Success always wins first, at any point — a late webhook/reconcile settle
  // flips 'confirming' straight to 'success' since the poll keeps running.
  //
  // Genuine, server-authoritative failure: the server says 'failed', OR the
  // server says 'refunded' (a captured-then-reversed payment — terminal and
  // known, not unknown, so it belongs with 'failure' and its safe-to-retry
  // copy rather than 'confirming'; there is no outstanding charge left to
  // double up on), OR the gateway errored before any order existed (nothing
  // was ever charged). The last failure case, `!orderId && pastGrace` with no
  // error, is the original grace timer's purpose: there's no order to poll or
  // link to, so there's nothing left to "confirm" — it must still terminate.
  //
  // Everything else still not resolved once pastGrace is true (a real orderId
  // whose paymentStatus is 'pending' or undefined) is 'confirming', not
  // 'failure' — the outcome is genuinely unknown, not declined, so no "failed"
  // language and no Retry (retrying a possibly-captured payment risks a
  // double charge).
  const state: 'checking' | 'confirming' | 'success' | 'failure' =
    paymentStatus === 'completed'
      ? 'success'
      : paymentStatus === 'failed' ||
          paymentStatus === 'refunded' ||
          (!orderId && Boolean(params.error))
        ? 'failure'
        : !orderId && pastGrace
          ? 'failure'
          : pastGrace
            ? 'confirming'
            : 'checking';

  // Clear the cart once payment is confirmed (the verify path may not have run).
  useEffect(() => {
    if (state === 'success') useCartStore.getState().clearCart();
  }, [state]);

  async function handleRetry() {
    if (!orderId) {
      router.replace('/(tabs)/orders');
      return;
    }
    setRetrying(true);
    try {
      await startOrderPayment(orderId);
    } catch {
      // Stay on the failure screen; the order is unpaid and can be retried.
      setRetrying(false);
    }
  }

  // Post-purchase the checkout stack has to go (#874). `replace` swaps only
  // THIS screen, leaving checkout → chef → cart intact underneath — so back
  // from the order detail drops the customer into a cart for an order they
  // have already paid for, which reads as "the payment didn't go through".
  // Pop to the tabs root first, then push, so the stack is Home → Order.
  function resetToHome() {
    if (router.canDismiss()) router.dismissAll();
  }

  function handleViewOrder() {
    resetToHome();
    if (orderId) router.push(`/order/${orderId}`);
    else router.push('/(tabs)/orders');
  }

  function handleGoToOrders() {
    resetToHome();
    router.push('/(tabs)/orders');
  }

  // ── Tip success (#45) ─────────────────────────────────────────────────────
  // A tip is its own charge already verified by the checkout sheet — no order
  // payment to poll. Show a static thank-you.
  if (params.tip === '1') {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <View style={styles.centered}>
          <View style={styles.successCircle}>
            <CheckCircle2 size={48} color={customerColors.success.DEFAULT} strokeWidth={1.5} />
          </View>
          <Text style={styles.successTitle}>Tip sent! 🎉</Text>
          <Text style={styles.successBody}>
            Thank you for supporting your chef and rider — 100% of your tip goes
            straight to them.
          </Text>
          <Pressable
            onPress={handleViewOrder}
            accessibilityRole="button"
            accessibilityLabel="Back to order"
            style={styles.ctaWrapper}
            android_ripple={{ color: PRIMARY_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[styles.ctaPrimary, pressed && Platform.OS === 'ios' && styles.ctaPressed]}
              >
                <Text style={styles.ctaPrimaryLabel}>Done</Text>
              </View>
            )}
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  // ── Checking (polling) ──────────────────────────────────────────────────────
  // Branded pending state — same 96pt circle language as success/failure below,
  // holding a spinner instead of a static glyph while the outcome is unknown.
  // Copy unchanged (Global Constraints §2).
  if (state === 'checking') {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <View style={styles.centered}>
          <View style={styles.pendingCircle}>
            <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
          </View>
          <Text style={styles.pendingLabel}>Confirming your payment…</Text>
        </View>
      </SafeAreaView>
    );
  }

  // ── Confirming (past grace, outcome still unknown) ──────────────────────────
  // Deliberately honest, not alarming: no "failed" language, no raw status or
  // error string, and no Retry — retrying a possibly-captured payment is the
  // double-charge trap this state exists to avoid. Primary path routes the
  // customer forward to watch the order resolve, same CTAs as success since
  // that's the most likely outcome.
  if (state === 'confirming') {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <View style={styles.centered}>
          <View style={styles.pendingCircle}>
            <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
          </View>
          <Text style={styles.confirmingTitle}>Still confirming your payment</Text>
          <Text style={styles.confirmingBody}>
            This can take a few minutes. We'll update your order automatically the moment it's
            confirmed — there's no need to retry.
          </Text>
          <Pressable
            onPress={handleViewOrder}
            accessibilityRole="button"
            accessibilityLabel="View order details"
            style={styles.ctaWrapper}
            android_ripple={{ color: PRIMARY_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[styles.ctaPrimary, pressed && Platform.OS === 'ios' && styles.ctaPressed]}
              >
                <Text style={styles.ctaPrimaryLabel}>View order</Text>
              </View>
            )}
          </Pressable>
          <Pressable
            onPress={handleGoToOrders}
            accessibilityRole="button"
            accessibilityLabel="Go to My Orders"
            android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[styles.ctaGhost, pressed && Platform.OS === 'ios' && styles.ctaGhostPressed]}
              >
                <Text style={styles.ctaGhostLabel}>My orders</Text>
              </View>
            )}
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  // ── Success ─────────────────────────────────────────────────────────────────
  if (state === 'success') {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <View style={styles.centered}>
          <View style={styles.successCircle}>
            <CheckCircle2 size={48} color={customerColors.success.DEFAULT} strokeWidth={1.5} />
          </View>
          <Text style={styles.successTitle}>Payment confirmed</Text>
          <Text style={styles.successBody}>
            Your order has been placed successfully. We'll notify you when the chef starts preparing.
          </Text>
          {/* Primary CTA — coral, radius 8, 52pt (R14: post-payment success always
              routes forward via "View order", never a dead end). */}
          <Pressable
            onPress={handleViewOrder}
            accessibilityRole="button"
            accessibilityLabel="View order details"
            style={styles.ctaWrapper}
            android_ripple={{ color: PRIMARY_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[styles.ctaPrimary, pressed && Platform.OS === 'ios' && styles.ctaPressed]}
              >
                <Text style={styles.ctaPrimaryLabel}>View order</Text>
              </View>
            )}
          </Pressable>
          <Pressable
            onPress={handleGoToOrders}
            accessibilityRole="button"
            accessibilityLabel="Go to My Orders"
            android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[styles.ctaGhost, pressed && Platform.OS === 'ios' && styles.ctaGhostPressed]}
              >
                <Text style={styles.ctaGhostLabel}>My orders</Text>
              </View>
            )}
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  // ── Failure ─────────────────────────────────────────────────────────────────
  // Unified iconography: the same 96pt circle as success, coloured with the
  // functional destructive token (never brand coral) so all three states read
  // as one system.
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
      <View style={styles.centered}>
        <View style={styles.failureCircle}>
          <XCircle size={48} color={customerColors.destructive.DEFAULT} strokeWidth={1.5} />
        </View>
        <Text style={styles.failureTitle}>Payment not completed</Text>
        <Text style={styles.failureBody}>
          {canRetry
            ? "We couldn't confirm your payment. If money was deducted it will be refunded automatically. You can retry the payment for this order."
            : "We couldn't confirm your payment, so this order was cancelled. Anything that was deducted is refunded automatically — place a new order whenever you're ready."}
        </Text>
        {canRetry ? (
          <Pressable
            onPress={handleRetry}
            disabled={retrying}
            accessibilityRole="button"
            accessibilityLabel="Retry payment"
            style={styles.ctaWrapper}
            android_ripple={retrying ? undefined : { color: PRIMARY_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.ctaPrimary,
                  (retrying || (pressed && Platform.OS === 'ios')) && styles.ctaPressed,
                ]}
              >
                {retrying ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text style={styles.ctaPrimaryLabel}>Retry payment</Text>
                )}
              </View>
            )}
          </Pressable>
        ) : null}
        {/* With no retry to offer, this is the screen's only way forward and
            carries the primary weight rather than sitting as a ghost. */}
        <Pressable
          onPress={handleViewOrder}
          accessibilityRole="button"
          accessibilityLabel="Go to the order"
          style={canRetry ? undefined : styles.ctaWrapper}
          android_ripple={{ color: canRetry ? GHOST_RIPPLE : PRIMARY_RIPPLE, borderless: false }}
        >
          {({ pressed }) =>
            canRetry ? (
              <View
                style={[styles.ctaGhost, pressed && Platform.OS === 'ios' && styles.ctaGhostPressed]}
              >
                <Text style={styles.ctaGhostLabel}>{orderId ? 'View order' : 'My orders'}</Text>
              </View>
            ) : (
              <View
                style={[styles.ctaPrimary, pressed && Platform.OS === 'ios' && styles.ctaPressed]}
              >
                <Text style={styles.ctaPrimaryLabel}>{orderId ? 'View order' : 'My orders'}</Text>
              </View>
            )
          }
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 32, gap: 16 },
  pendingLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, marginTop: 12 },
  // Same 96pt circle language as success/failure — holds the spinner while
  // the outcome is unresolved (unifies iconography across all three states).
  pendingCircle: {
    width: 96, height: 96, borderRadius: 48, backgroundColor: customerColors.coral.tint,
    alignItems: 'center', justifyContent: 'center', marginBottom: 8,
  },
  successCircle: {
    width: 96, height: 96, borderRadius: 48, backgroundColor: customerColors.success.tint,
    alignItems: 'center', justifyContent: 'center', marginBottom: 8,
  },
  successTitle: { fontFamily: 'Geist-Bold', fontSize: 24, color: customerColors.charcoal.DEFAULT, textAlign: 'center', letterSpacing: -0.3 },
  successBody: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 21, paddingHorizontal: 8 },
  // Same title/body treatment as success — 'confirming' is not alarming, it's
  // still "in progress" (same pendingCircle language as 'checking' above it).
  confirmingTitle: { fontFamily: 'Geist-Bold', fontSize: 24, color: customerColors.charcoal.DEFAULT, textAlign: 'center', letterSpacing: -0.3 },
  confirmingBody: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 21, paddingHorizontal: 8 },
  // Destructive tint/colour — a functional error signal, never decorative.
  failureCircle: {
    width: 96, height: 96, borderRadius: 48, backgroundColor: customerColors.destructive.tint,
    alignItems: 'center', justifyContent: 'center', marginBottom: 8,
  },
  failureTitle: { fontFamily: 'Geist-Bold', fontSize: 24, color: customerColors.charcoal.DEFAULT, textAlign: 'center', letterSpacing: -0.3 },
  failureBody: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 21, paddingHorizontal: 8 },
  ctaWrapper: { width: '100%', maxWidth: 320 },
  ctaPrimary: {
    backgroundColor: customerColors.coral.DEFAULT, borderRadius: 8, minHeight: 52,
    alignItems: 'center', justifyContent: 'center', paddingHorizontal: 24,
  },
  ctaPressed: { backgroundColor: customerColors.coral.pressed },
  ctaPrimaryLabel: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.canvas },
  ctaGhost: {
    borderRadius: 8, minHeight: 48, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 24,
    borderWidth: 1, borderColor: customerColors.hairline, backgroundColor: customerColors.canvas, width: '100%', maxWidth: 320,
  },
  ctaGhostPressed: { backgroundColor: customerColors.surface.soft },
  ctaGhostLabel: { fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.DEFAULT },
});
