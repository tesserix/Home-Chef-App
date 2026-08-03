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

const DEFAULT_SECONDS = 10;

const RING_SIZE = 176;
const TICK_LENGTH = 14;
const TICK_WIDTH = 4;

// One tick per second, laid out radially by rotating a full-size container —
// transform only, so it needs no SVG dependency (which would have meant a
// native rebuild) and honours the "animate opacity and transform" rule.
// A tick per second reads as time remaining at a glance, which a bare number
// does not; the number stays in the middle for the exact count.
function CountdownRing({ total, remaining }: { total: number; remaining: number }) {
  const ticks = Math.max(4, Math.min(total, 12));
  const perTick = total / ticks;
  return (
    <View style={styles.ring} accessibilityRole="progressbar" accessibilityValue={{ now: remaining, min: 0, max: total }}>
      {Array.from({ length: ticks }, (_, i) => {
        // Ticks are spent clockwise from 12 o'clock as the window drains.
        const spent = i >= Math.ceil(remaining / perTick);
        return (
          <View
            key={i}
            style={[styles.tickOrbit, { transform: [{ rotate: `${(360 / ticks) * i}deg` }] }]}
            pointerEvents="none"
          >
            <View style={[styles.tick, spent ? styles.tickSpent : styles.tickLive]} />
          </View>
        );
      })}
      <View style={styles.ringCentre}>
        <Text style={styles.seconds}>{remaining}</Text>
        <Text style={styles.secondsLabel}>
          {remaining === 1 ? 'second' : 'seconds'}
        </Text>
      </View>
    </View>
  );
}

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

  return (
    <SafeAreaView style={styles.screen} edges={['top', 'bottom']}>
      <View style={styles.body}>
        <Text style={styles.eyebrow}>ORDER PLACED</Text>
        <Text style={styles.title}>Changed your mind?</Text>
        <Text style={styles.blurb}>
          You haven’t been charged yet. Cancel now and nothing leaves your account
          — otherwise we’ll take you to payment.
        </Text>

        <CountdownRing total={total} remaining={remaining} />
      </View>

      <View style={styles.actions}>
        <Pressable
          onPress={proceed}
          disabled={cancelling}
          accessibilityRole="button"
          accessibilityLabel="Skip the wait and go to payment now"
          style={[styles.payBtn, cancelling && styles.disabled]}
        >
          <Text style={styles.payLabel}>Skip to payment</Text>
        </Pressable>

        <Pressable
          onPress={() => void cancel()}
          disabled={cancelling}
          accessibilityRole="button"
          accessibilityLabel="Cancel this order"
          style={styles.cancelBtn}
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
  ring: {
    width: RING_SIZE,
    height: RING_SIZE,
    marginTop: 40,
    alignItems: 'center',
    justifyContent: 'center',
  },
  // Full-size overlay rotated about its own centre; the tick sits at its top
  // edge, so the rotation alone places the tick on the circle.
  tickOrbit: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    alignItems: 'center',
  },
  tick: {
    width: TICK_WIDTH,
    height: TICK_LENGTH,
    borderRadius: TICK_WIDTH / 2,
  },
  tickLive: { backgroundColor: customerColors.coral.DEFAULT },
  tickSpent: { backgroundColor: customerColors.hairline },
  ringCentre: { alignItems: 'center' },
  seconds: {
    fontFamily: 'Geist-Bold',
    fontSize: 56,
    lineHeight: 62,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  secondsLabel: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    marginTop: -2,
  },
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
  disabled: { opacity: 0.5 },
});
