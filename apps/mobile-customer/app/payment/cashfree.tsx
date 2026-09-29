// In-app Cashfree checkout, embedded in a WebView.
//
// Deliberately a WebView rather than the Cashfree React Native SDK: the SDK is a
// native module, so adopting it would force a new EAS build before any customer
// could pay through Cashfree. The v3 web SDK opens the same payment sheet
// (including UPI intent on Android) and ships with an OTA update, so the gateway
// can go live without a store release.
//
// Mirrors app/payment/checkout.tsx so both entry points feel
// identical to the customer. Two things genuinely differ:
//
//  1. The environment must be passed explicitly — Cashfree splits sandbox from
//     production by host, and the SDK has no key prefix to infer it from. It
//     comes from the SERVER (cashfreeEnv), never guessed here.
//  2. There is no client-verifiable success payload. Cashfree hands back no
//     (payment_id, signature) pair, so ANY non-error close routes to the result
//     screen, which polls the server's real paymentStatus. The client's own view
//     of "paid" is never trusted.

import { useCallback, useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Platform, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { WebView, type WebViewMessageEvent } from 'react-native-webview';
import { buildCashfreeCheckoutHtml } from '@homechef/mobile-shared/payments';
import { router, useLocalSearchParams } from 'expo-router';
import { useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { chargeRefreshKeys } from '../../lib/payment';

const BACK_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

interface CashfreeCheckoutParams {
  orderId: string;
  paymentSessionId: string;
  cashfreeOrderId: string;
  /** "SANDBOX" | "PRODUCTION", resolved server-side. */
  env?: string;
  /** 'mealplan' settles a plan advance instead of an order. Absent = order. */
  kind?: string;
  mealPlanId?: string;
  groupId?: string;
  cateringId?: string;
  /** 'tip' settles a post-delivery tip against its own row, not the order. */
  tipId?: string;
}

type BridgeMessage =
  | { type: 'settled' }
  | { type: 'dismiss' }
  | { type: 'error'; message?: string };

// Self-contained HTML that opens Cashfree checkout and relays the outcome to
export default function CashfreeCheckoutScreen() {
  // Cast rather than the generic form, matching checkout.tsx/result.tsx: without
  // generated route types (a fresh checkout, which is what CI has) the generic is
  // constrained to a route path string, not a params record.
  const params = useLocalSearchParams() as unknown as CashfreeCheckoutParams;
  const [loading, setLoading] = useState(true);
  const [needsRecovery, setNeedsRecovery] = useState(false);
  const [checking, setChecking] = useState(false);
  const finished = useRef(false);
  const verifying = useRef<Promise<{ data: { status?: string } }> | null>(null);
  const leaving = useRef(false);
  const qc = useQueryClient();

  const orderId = String(params.orderId ?? '');
  const cashfreeOrderId = String(params.cashfreeOrderId ?? '');
  // Default to production on anything unrecognised, matching the server's
  // fail-toward-live asymmetry: a wrong "sandbox" would silently capture no real
  // money, while a wrong "production" fails loudly against a sandbox session.
  const mode = String(params.env ?? '').toUpperCase() === 'SANDBOX' ? 'sandbox' : 'production';

  // Non-order charges (a plan advance, a group share, a catering deposit) settle
  // against their own row: different verify endpoint, and they land back on that
  // row rather than the order result screen. Every one of them takes an EMPTY body
  // — Cashfree gives the client no payment id or signature, so the server reads the
  // capture from the gateway and binds it by the order id, which is the row's uuid.
  const kind = String(params.kind ?? '');
  const chargeId = String(
    params.mealPlanId ?? params.groupId ?? params.cateringId ?? params.tipId ?? '',
  );
  const isCharge = kind === 'mealplan' || kind === 'group' || kind === 'catering' || kind === 'tip';
  const { verifyPath, doneRoute } =
    kind === 'mealplan'
      ? {
          verifyPath: `/v1/meal-plans/${chargeId}/verify-payment`,
          doneRoute: `/meal-plans/${chargeId}`,
        }
      : kind === 'group'
        ? {
            verifyPath: `/v1/group-orders/${chargeId}/pay/verify`,
            doneRoute: `/group-order/${chargeId}`,
          }
        : kind === 'catering'
          ? {
              verifyPath: `/v1/catering/requests/${chargeId}/deposit/verify`,
              doneRoute: `/catering/${chargeId}`,
            }
          : kind === 'tip'
            ? {
                verifyPath: `/v1/payments/tip/${chargeId}/verify`,
                doneRoute: `/order/${orderId}`,
              }
            : {
                verifyPath: `/v1/payments/order/${orderId}/verify`,
                doneRoute: `/payment/result?order_id=${orderId}`,
              };

  // The row we land back on is already mounted (we pushed from it), so replace()
  // reuses that screen and it keeps rendering its pre-payment snapshot unless the
  // cache is invalidated. The order result screen polls, so it needs nothing.
  const settle = useCallback(() => {
    if (finished.current) return;
    finished.current = true;
    for (const key of chargeRefreshKeys(kind, chargeId)) {
      void qc.invalidateQueries({ queryKey: key });
    }
    router.replace(doneRoute as never);
  }, [qc, kind, chargeId, doneRoute]);

  const verify = useCallback(() => {
    if (!verifying.current) {
      verifying.current = api
        .post<{
          status?: string;
        }>(verifyPath, isCharge ? {} : { cashfreeOrderId })
        .finally(() => {
          verifying.current = null;
        });
    }
    return verifying.current;
  }, [verifyPath, isCharge, cashfreeOrderId]);

  const finish = useCallback(async () => {
    if (leaving.current || finished.current) return;
    leaving.current = true;
    setChecking(true);
    try {
      await verify();
    } catch {
      // A timeout is an unknown outcome; the destination reads server status.
    }
    settle();
  }, [verify, settle]);

  useEffect(() => {
    finished.current = false;
    const timer = setTimeout(() => {
      setLoading(false);
      setNeedsRecovery(true);
    }, 60000);
    return () => {
      clearTimeout(timer);
      finished.current = true;
    };
  }, []);

  const recover = useCallback(() => {
    setLoading(false);
    setNeedsRecovery(true);
  }, []);

  // The bridge is not a reliable completion signal: the 3DS step navigates this
  // document away (popup fallback, bank redirect) and kills the script before it
  // can post, so the screen would otherwise hang on a paid order.
  //
  // Poll verify, not the order: paymentStatus only flips once verify or the
  // webhook runs, and the missing verify call is precisely the failure — reading
  // it back would wait on something nothing is going to do. verify re-fetches the
  // payment from Cashfree server-side and is idempotent, so it is safe to repeat.
  useEffect(() => {
    if (!orderId && !chargeId) return;
    const timer = setInterval(async () => {
      if (verifying.current || finished.current || leaving.current) return;
      try {
        const r = await verify();
        // The plan endpoint answers 200 only once the advance is confirmed, so
        // reaching here at all is the signal; the order endpoint reports a status.
        if (isCharge || (r.data?.status && r.data.status !== 'pending')) {
          clearInterval(timer);
          settle();
        }
      } catch {
        // Not captured yet, or transient — keep polling.
      }
    }, 4000);
    return () => clearInterval(timer);
  }, [orderId, chargeId, isCharge, verify, settle]);

  const onMessage = useCallback(
    (event: WebViewMessageEvent) => {
      let msg: BridgeMessage;
      try {
        msg = JSON.parse(event.nativeEvent.data) as BridgeMessage;
      } catch {
        return;
      }
      if (!['dismiss', 'settled', 'error'].includes(msg.type)) return;
      // 'settled' and 'error' both land on the result screen: even an error may
      // have followed a real capture, and only the server knows.
      void finish();
    },
    [finish],
  );

  if (!orderId || !params.paymentSessionId) {
    return (
      <SafeAreaView style={{ flex: 1, alignItems: 'center', justifyContent: 'center' }}>
        <Text style={{ color: customerColors.charcoal.DEFAULT }}>
          This payment session is no longer valid.
        </Text>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: '#fff' }} edges={['top']}>
      <View
        style={{
          flexDirection: 'row',
          alignItems: 'center',
          paddingHorizontal: 8,
          height: 48,
        }}
      >
        <Pressable
          onPress={() => {
            void finish();
          }}
          android_ripple={{ color: BACK_RIPPLE, borderless: true }}
          hitSlop={12}
          accessibilityRole="button"
          accessibilityLabel="Cancel payment"
          style={{ padding: 8 }}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text
          style={{
            fontSize: 16,
            fontWeight: '600',
            color: customerColors.charcoal.DEFAULT,
          }}
        >
          Secure checkout
        </Text>
      </View>

      {needsRecovery ? (
        <View style={{ padding: 16, backgroundColor: '#fff7ed' }}>
          <Text style={{ color: customerColors.charcoal.DEFAULT }}>
            Checkout taking too long? Check your payment status before trying again.
          </Text>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Check payment status"
            disabled={checking}
            onPress={finish}
            style={{ minHeight: 44, justifyContent: 'center' }}
          >
            <Text
              style={{
                color: customerColors.charcoal.DEFAULT,
                fontWeight: '600',
              }}
            >
              {checking ? 'Checking payment…' : 'Check payment status'}
            </Text>
          </Pressable>
        </View>
      ) : null}

      <WebView
        source={{
          html: buildCashfreeCheckoutHtml({
            paymentSessionId: String(params.paymentSessionId),
            mode,
          }),
          // A base URL is required for the SDK's own origin checks; without it
          // Android treats the document as opaque and the SDK refuses to open.
          baseUrl: 'https://sdk.cashfree.com',
        }}
        onMessage={onMessage}
        onLoadEnd={() => setLoading(false)}
        onError={recover}
        onHttpError={recover}
        javaScriptEnabled
        domStorageEnabled
        // UPI intent hands off to GPay/PhonePe via a custom scheme; without this
        // the WebView blocks the navigation and UPI silently does nothing.
        originWhitelist={['*']}
        setSupportMultipleWindows={false}
        style={{ flex: 1 }}
      />

      {loading ? (
        <View
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: '#fff',
          }}
          pointerEvents="none"
        >
          <ActivityIndicator
            size={Platform.OS === 'ios' ? 'large' : 48}
            color={customerColors.charcoal.DEFAULT}
          />
        </View>
      ) : null}
    </SafeAreaView>
  );
}
