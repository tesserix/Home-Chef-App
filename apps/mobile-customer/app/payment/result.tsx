// Deep link handler for homechef-customer://payment/result
//
// This screen is the callback_url target from Razorpay hosted checkout.
// Razorpay may append razorpay_payment_id / razorpay_order_id / razorpay_signature
// as query params, but we do NOT rely on them for payment confirmation.
//
// Strategy (Open Question 1 resolution from RESEARCH.md):
//   - Primary confirmation: server-side webhook → order status updated on server
//   - Client detection: checkout.tsx polls GET /v1/orders/:id every 3s for up to 60s
//   - This screen just navigates back so the checkout poller can detect the result
//
// Visual: white canvas, centered layout, safe-area aware.
// While navigating back (normal path) we show the confirming spinner.
// If navigation fails or the user lands here directly, a fallback UI with
// coral CTAs lets them recover gracefully.

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

// Razorpay appends these params on success; on failure the params may be absent
// or contain an error code. We read them but do NOT use them for confirmation —
// the server webhook is authoritative.
interface PaymentParams {
  razorpay_payment_id?: string;
  razorpay_order_id?: string;
  razorpay_signature?: string;
  error?: string; // present when Razorpay signals a client-side failure
  order_id?: string; // the internal order id we may pass when launching checkout
}

type ResultState = 'pending' | 'success' | 'failure';

export default function PaymentResult() {
  const router = useRouter();
  // expo-router's generic expects a Route-shaped param map; our param bag is a
  // plain string record, so read untyped and cast to the known shape.
  const params = useLocalSearchParams() as unknown as PaymentParams;

  // Determine client-side intent from params — NOT for payment confirmation.
  // Actual payment status comes from polling on the checkout screen.
  const hasClientError = Boolean(params.error);
  const hasPaymentId = Boolean(params.razorpay_payment_id);
  const initialState: ResultState = hasClientError
    ? 'failure'
    : hasPaymentId
      ? 'success'
      : 'pending';

  const [state, setState] = useState<ResultState>(initialState);

  useEffect(() => {
    if (state === 'pending') {
      // Normal path: navigate back immediately; the checkout poller picks up
      // the payment result and shows the final order screen.
      router.back();
    }
    // For success/failure we show the static screen and let the user act.
  }, [state, router]);

  // ── Pending / navigating back ───────────────────────────────────────────────
  if (state === 'pending') {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
          <Text style={styles.pendingLabel}>Confirming payment…</Text>
        </View>
      </SafeAreaView>
    );
  }

  // ── Success state ──────────────────────────────────────────────────────────
  if (state === 'success') {
    const orderId = params.order_id ?? params.razorpay_order_id;

    function handleViewOrder() {
      if (orderId) {
        router.replace(`/order/${orderId}`);
      } else {
        router.replace('/(tabs)/orders');
      }
    }

    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
        <View style={styles.centered}>
          {/* Success green check circle */}
          <View style={styles.successCircle}>
            <CheckCircle2
              size={48}
              color={customerColors.success.DEFAULT}
              strokeWidth={1.5}
            />
          </View>

          {/* Charcoal headline */}
          <Text style={styles.successTitle}>Payment confirmed</Text>
          <Text style={styles.successBody}>
            Your order has been placed successfully. We'll notify you when the chef starts preparing.
          </Text>

          {/* Coral primary CTA — visual styles on inner View (iOS Pressable bug) */}
          <Pressable
            onPress={handleViewOrder}
            accessibilityRole="button"
            accessibilityLabel="View order details"
            style={styles.ctaWrapper}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.ctaPrimary,
                  pressed && styles.ctaPressed,
                ]}
              >
                <Text style={styles.ctaPrimaryLabel}>View order</Text>
              </View>
            )}
          </Pressable>

          {/* Ghost secondary — go to orders list */}
          <Pressable
            onPress={() => router.replace('/(tabs)/orders')}
            accessibilityRole="button"
            accessibilityLabel="Go to My Orders"
          >
            <View style={styles.ctaGhost}>
              <Text style={styles.ctaGhostLabel}>My orders</Text>
            </View>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  // ── Failure state ───────────────────────────────────────────────────────────
  function handleTryAgain() {
    // Go back to checkout so the user can retry
    router.back();
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
      <View style={styles.centered}>
        {/* Destructive icon — charcoal-soft circle, red-X glyph */}
        <View style={styles.failureCircle}>
          <XCircle
            size={48}
            color={customerColors.charcoal.soft}
            strokeWidth={1.5}
          />
        </View>

        {/* Charcoal headline — no alarming red, calm and instructive */}
        <Text style={styles.failureTitle}>Payment not completed</Text>
        <Text style={styles.failureBody}>
          {params.error
            ? 'The payment was declined or cancelled. You can try a different payment method.'
            : 'Something went wrong during payment. No amount has been charged.'}
        </Text>

        {/* Coral "Try again" CTA */}
        <Pressable
          onPress={handleTryAgain}
          accessibilityRole="button"
          accessibilityLabel="Try payment again"
          style={styles.ctaWrapper}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.ctaPrimary,
                pressed && styles.ctaPressed,
              ]}
            >
              <Text style={styles.ctaPrimaryLabel}>Try again</Text>
            </View>
          )}
        </Pressable>

        {/* Ghost — go to orders to see if order was placed anyway */}
        <Pressable
          onPress={() => router.replace('/(tabs)/orders')}
          accessibilityRole="button"
          accessibilityLabel="Go to My Orders"
        >
          <View style={styles.ctaGhost}>
            <Text style={styles.ctaGhostLabel}>My orders</Text>
          </View>
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  // White canvas, flex-1 — matches all other customer screens
  root: {
    flex: 1,
    backgroundColor: customerColors.canvas,
  },

  // Centered single-column layout
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 32,
    gap: 16,
  },

  // ── Pending ──
  pendingLabel: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    marginTop: 12,
  },

  // ── Success icon circle ──
  successCircle: {
    width: 96,
    height: 96,
    borderRadius: 48,
    backgroundColor: customerColors.success.tint,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  successTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    color: customerColors.charcoal.DEFAULT,
    textAlign: 'center',
    letterSpacing: -0.3,
  },
  successBody: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 21,
    paddingHorizontal: 8,
  },

  // ── Failure icon circle ──
  failureCircle: {
    width: 96,
    height: 96,
    borderRadius: 48,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  failureTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    color: customerColors.charcoal.DEFAULT,
    textAlign: 'center',
    letterSpacing: -0.3,
  },
  failureBody: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 21,
    paddingHorizontal: 8,
  },

  // ── Buttons ──
  // ctaWrapper gives Pressable a predictable touch target without affecting
  // the inner View's visual layout (iOS Pressable array-style bug workaround).
  ctaWrapper: {
    width: '100%',
    maxWidth: 320,
  },

  // Primary coral button — spec §3: radius 8, minHeight 52, Inter-SemiBold
  ctaPrimary: {
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 24,
  },
  ctaPressed: {
    backgroundColor: customerColors.coral.pressed,
  },
  ctaPrimaryLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.canvas,
  },

  // Ghost / outline button — spec §3
  ctaGhost: {
    borderRadius: 8,
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 24,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
    width: '100%',
    maxWidth: 320,
  },
  ctaGhostLabel: {
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
});
