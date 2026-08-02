// Pre-payment hold (#hold) — the pause between placing an order and paying for
// it, so a customer who changes their mind in the first few seconds gets out
// without a charge and without a refund to chase.
//
// The order exists and the gateway payment has been created, but NOTHING is
// charged until the sheet is completed — cancelling here just cancels an unpaid
// pending order (POST /v1/orders/:id/cancel), which releases the chef's capacity
// and slot exactly as any other pre-accept cancellation does.
//
// The deadline is wall-clock, not a tick count: JS timers are throttled in the
// background, so counting ticks would silently extend the window for anyone who
// switched apps. Remaining time is recomputed from Date.now() on every tick and
// on every return to the foreground.

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  AppState,
  type AppStateStatus,
  BackHandler,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useAlert } from '@homechef/mobile-shared/ui';
import { api } from '../../lib/api';
import { friendlyErrorMessage } from '../../lib/errors';
import { clearPendingGateway, launchGateway, takePendingGateway } from '../../lib/payment';

const DEFAULT_SECONDS = 30;

export default function PaymentHold() {
  const params = useLocalSearchParams() as unknown as {
    orderId: string;
    seconds?: string;
  };
  const orderId = params.orderId;
  const total = Math.max(
    1,
    Math.min(120, Number(params.seconds) || DEFAULT_SECONDS),
  );

  const { showAlert } = useAlert();
  const [remaining, setRemaining] = useState(total);
  const [cancelling, setCancelling] = useState(false);
  // One-way latch: the timeout, the Pay-now tap and the cancel tap all race for
  // the same order. Whoever gets here first owns the outcome.
  const settledRef = useRef(false);
  const deadlineRef = useRef(Date.now() + total * 1000);

  const proceed = useCallback(() => {
    if (settledRef.current) return;
    settledRef.current = true;
    const data = takePendingGateway(orderId);
    if (!data) {
      // The payload is gone (a reload dropped module state). The order is real
      // and unpaid, so send them to the result screen, which offers Pay now.
      router.replace(`/payment/result?order_id=${orderId}`);
      return;
    }
    void launchGateway(orderId, data, { replace: true });
  }, [orderId]);

  const cancel = useCallback(async () => {
    if (settledRef.current) return;
    settledRef.current = true;
    setCancelling(true);
    try {
      await api.post(`/v1/orders/${orderId}/cancel`, {
        reason: 'Cancelled by customer before payment',
      });
      clearPendingGateway();
      // Back to the cart, with it still filled — someone who backed out at the
      // last moment usually wants to change something, not start over.
      router.replace('/cart');
    } catch (err: unknown) {
      // The cancel did not land, so the order is still live and payable. Undo
      // the latch and let them decide again rather than stranding them.
      settledRef.current = false;
      setCancelling(false);
      showAlert(
        'Could not cancel',
        friendlyErrorMessage(err, 'Please try again, or continue to payment.'),
      );
    }
  }, [orderId, showAlert]);

  // Wall-clock countdown. Recomputed from the deadline each tick so a throttled
  // or suspended timer can never hand back more time than the customer is due.
  useEffect(() => {
    function sync(): void {
      const left = Math.ceil((deadlineRef.current - Date.now()) / 1000);
      setRemaining(left > 0 ? left : 0);
      if (left <= 0) proceed();
    }
    const id = setInterval(sync, 250);
    const sub = AppState.addEventListener('change', (s: AppStateStatus) => {
      if (s === 'active') sync();
    });
    return () => {
      clearInterval(id);
      sub.remove();
    };
  }, [proceed]);

  // Android hardware back must not drop the customer onto checkout with a live
  // unpaid order behind them — make the two real choices the only exits.
  useEffect(() => {
    if (Platform.OS !== 'android') return;
    const sub = BackHandler.addEventListener('hardwareBackPress', () => true);
    return () => sub.remove();
  }, []);

  const pct = Math.max(0, Math.min(1, remaining / total));

  return (
    <SafeAreaView style={styles.screen} edges={['top', 'bottom']}>
      <View style={styles.body}>
        <Text style={styles.eyebrow}>ORDER PLACED</Text>
        <Text style={styles.title}>Changed your mind?</Text>
        <Text style={styles.blurb}>
          You haven’t been charged yet. Cancel now and nothing leaves your account
          — otherwise we’ll take you to payment.
        </Text>

        <View style={styles.countdown}>
          <Text style={styles.seconds}>{remaining}</Text>
          <Text style={styles.secondsLabel}>
            {remaining === 1 ? 'second' : 'seconds'}
          </Text>
        </View>

        {/* Draining bar rather than a spinner: it shows how much of the window
            is left, which is the only thing the customer needs to judge here. */}
        <View style={styles.track}>
          <View style={[styles.fill, { width: `${pct * 100}%` }]} />
        </View>
      </View>

      <View style={styles.actions}>
        <Pressable
          onPress={proceed}
          disabled={cancelling}
          accessibilityRole="button"
          accessibilityLabel="Skip the wait and pay now"
          style={({ pressed }) => [
            styles.payBtn,
            pressed && styles.pressed,
            cancelling && styles.disabled,
          ]}
        >
          <Text style={styles.payLabel}>Pay now</Text>
        </Pressable>

        <Pressable
          onPress={() => void cancel()}
          disabled={cancelling}
          accessibilityRole="button"
          accessibilityLabel="Cancel this order"
          style={({ pressed }) => [styles.cancelBtn, pressed && styles.pressed]}
        >
          {cancelling ? (
            <ActivityIndicator size="small" color={customerColors.charcoal.soft} />
          ) : (
            <Text style={styles.cancelLabel}>Cancel order</Text>
          )}
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: customerColors.canvas },
  body: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 24,
  },
  eyebrow: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    letterSpacing: 1,
    color: customerColors.success.DEFAULT,
    marginBottom: 12,
  },
  title: {
    fontFamily: 'Geist',
    fontSize: 26,
    color: customerColors.charcoal.DEFAULT,
    textAlign: 'center',
  },
  blurb: {
    fontFamily: 'Inter',
    fontSize: 15,
    lineHeight: 22,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    marginTop: 10,
    maxWidth: 320,
  },
  countdown: { alignItems: 'center', marginTop: 40 },
  seconds: {
    fontFamily: 'Geist-Bold',
    fontSize: 64,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  secondsLabel: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    marginTop: -4,
  },
  track: {
    height: 4,
    width: '100%',
    maxWidth: 320,
    borderRadius: 2,
    backgroundColor: customerColors.hairline,
    marginTop: 28,
    overflow: 'hidden',
  },
  fill: { height: 4, borderRadius: 2, backgroundColor: customerColors.coral.DEFAULT },
  actions: { paddingHorizontal: 24, paddingBottom: 12, gap: 8 },
  payBtn: {
    height: 52,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.coral.DEFAULT,
  },
  payLabel: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: '#FFFFFF' },
  cancelBtn: {
    height: 48,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
  },
  cancelLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: 15,
    color: customerColors.charcoal.soft,
  },
  pressed: { opacity: 0.85 },
  disabled: { opacity: 0.5 },
});
