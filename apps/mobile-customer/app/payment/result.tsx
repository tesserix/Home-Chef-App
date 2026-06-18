// Payment result screen.
//
// The client-side Razorpay callback is NOT authoritative — the session can
// expire while the user is in the payment sheet, so the in-app verify call may
// fail even though Razorpay captured the money. Instead of trusting the
// callback params, this screen polls the order's real `paymentStatus` (which
// the server sets via the synchronous verify OR the payment.captured webhook)
// and shows the actual outcome:
//   - completed            → success
//   - failed / still pending after a grace window → failure + Retry payment
//
// Visual: white canvas, centered layout, safe-area aware. Coral CTAs.

import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
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

interface PaymentParams {
  razorpay_payment_id?: string;
  razorpay_order_id?: string;
  order_id?: string; // the internal order id passed when launching checkout
  error?: string;
}

// How long to keep polling for a 'pending' order before declaring failure.
// The webhook / verify normally lands within a few seconds.
const CONFIRM_TIMEOUT_MS = 25_000;

export default function PaymentResult() {
  const router = useRouter();
  const params = useLocalSearchParams() as unknown as PaymentParams;
  const orderId = params.order_id ?? params.razorpay_order_id ?? '';

  // Authoritative: poll the server's payment status until terminal.
  const { data } = useOrder(orderId, { pollUntilPaid: true });
  const paymentStatus = data?.data?.paymentStatus;

  const [timedOut, setTimedOut] = useState(false);
  const [retrying, setRetrying] = useState(false);

  useEffect(() => {
    if (!orderId) return;
    const t = setTimeout(() => setTimedOut(true), CONFIRM_TIMEOUT_MS);
    return () => clearTimeout(t);
  }, [orderId]);

  // Derive the displayed state from the real payment status (+ grace window).
  const state: 'checking' | 'success' | 'failure' =
    paymentStatus === 'completed'
      ? 'success'
      : paymentStatus === 'failed' ||
          (timedOut && (!paymentStatus || paymentStatus === 'pending')) ||
          (!orderId && Boolean(params.error))
        ? 'failure'
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
      await startOrderPayment(orderId, { replace: true });
    } catch {
      // Stay on the failure screen; the order is unpaid and can be retried.
      setRetrying(false);
    }
  }

  function handleViewOrder() {
    router.replace(orderId ? `/order/${orderId}` : '/(tabs)/orders');
  }

  // ── Checking (polling) ──────────────────────────────────────────────────────
  if (state === 'checking') {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
          <Text style={styles.pendingLabel}>Confirming your payment…</Text>
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
          <Pressable onPress={handleViewOrder} accessibilityRole="button" accessibilityLabel="View order details" style={styles.ctaWrapper}>
            {({ pressed }) => (
              <View style={[styles.ctaPrimary, pressed && styles.ctaPressed]}>
                <Text style={styles.ctaPrimaryLabel}>View order</Text>
              </View>
            )}
          </Pressable>
          <Pressable onPress={() => router.replace('/(tabs)/orders')} accessibilityRole="button" accessibilityLabel="Go to My Orders">
            <View style={styles.ctaGhost}>
              <Text style={styles.ctaGhostLabel}>My orders</Text>
            </View>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  // ── Failure ─────────────────────────────────────────────────────────────────
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
      <View style={styles.centered}>
        <View style={styles.failureCircle}>
          <XCircle size={48} color={customerColors.charcoal.soft} strokeWidth={1.5} />
        </View>
        <Text style={styles.failureTitle}>Payment not completed</Text>
        <Text style={styles.failureBody}>
          We couldn't confirm your payment. If money was deducted it will be refunded automatically.
          You can retry the payment for this order.
        </Text>
        <Pressable
          onPress={handleRetry}
          disabled={retrying}
          accessibilityRole="button"
          accessibilityLabel="Retry payment"
          style={styles.ctaWrapper}
        >
          {({ pressed }) => (
            <View style={[styles.ctaPrimary, pressed && styles.ctaPressed, retrying && styles.ctaPressed]}>
              {retrying ? (
                <ActivityIndicator color={customerColors.canvas} />
              ) : (
                <Text style={styles.ctaPrimaryLabel}>Retry payment</Text>
              )}
            </View>
          )}
        </Pressable>
        <Pressable onPress={handleViewOrder} accessibilityRole="button" accessibilityLabel="Go to the order">
          <View style={styles.ctaGhost}>
            <Text style={styles.ctaGhostLabel}>{orderId ? 'View order' : 'My orders'}</Text>
          </View>
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 32, gap: 16 },
  pendingLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, marginTop: 12 },
  successCircle: {
    width: 96, height: 96, borderRadius: 48, backgroundColor: customerColors.success.tint,
    alignItems: 'center', justifyContent: 'center', marginBottom: 8,
  },
  successTitle: { fontFamily: 'Geist-Bold', fontSize: 24, color: customerColors.charcoal.DEFAULT, textAlign: 'center', letterSpacing: -0.3 },
  successBody: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 21, paddingHorizontal: 8 },
  failureCircle: {
    width: 96, height: 96, borderRadius: 48, backgroundColor: customerColors.surface.soft,
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
  ctaGhostLabel: { fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.DEFAULT },
});
