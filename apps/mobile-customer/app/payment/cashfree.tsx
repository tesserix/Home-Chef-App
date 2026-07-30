// In-app Cashfree checkout, embedded in a WebView.
//
// Deliberately a WebView rather than the Cashfree React Native SDK: the SDK is a
// native module, so adopting it would force a new EAS build before any customer
// could pay through Cashfree. The v3 web SDK opens the same payment sheet
// (including UPI intent on Android) and ships with an OTA update, so the gateway
// can go live without a store release.
//
// Mirrors app/payment/checkout.tsx (the Razorpay sheet) so both gateways feel
// identical to the customer. Two things genuinely differ:
//
//  1. The environment must be passed explicitly — Cashfree splits sandbox from
//     production by host, and the SDK has no key prefix to infer it from. It
//     comes from the SERVER (cashfreeEnv), never guessed here.
//  2. There is no client-verifiable success payload. Cashfree hands back no
//     (payment_id, signature) pair, so ANY non-error close routes to the result
//     screen, which polls the server's real paymentStatus. The client's own view
//     of "paid" is never trusted.

import { useCallback, useState } from 'react';
import { ActivityIndicator, Platform, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { WebView, type WebViewMessageEvent } from 'react-native-webview';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { useCartStore } from '../../store/cart-store';

const BACK_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

interface CashfreeCheckoutParams {
  orderId: string;
  paymentSessionId: string;
  cashfreeOrderId: string;
  /** "SANDBOX" | "PRODUCTION", resolved server-side. */
  env?: string;
}

type BridgeMessage =
  | { type: 'settled' }
  | { type: 'dismiss' }
  | { type: 'error'; message?: string };

// Self-contained HTML that opens Cashfree checkout and relays the outcome to
// React Native. The session id is injected as a JSON string literal so a value
// containing quotes cannot break out of the script.
function buildCashfreeHtml(opts: { paymentSessionId: string; mode: 'sandbox' | 'production' }): string {
  return `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no" />
<!-- Loaded unpinned, without Subresource Integrity, deliberately: Cashfree ships
     security and compliance fixes to this SDK continuously, and an SRI hash would
     hard-fail checkout the moment they publish one. Every major PG (Cashfree,
     Razorpay, Stripe) documents the unpinned CDN URL for this reason, and the
     Razorpay sheet in checkout.tsx loads the same way. The mitigation is that the
     SDK is never trusted for the payment OUTCOME — the server re-fetches the
     captured payment from Cashfree before any order is marked paid. -->
<script src="https://sdk.cashfree.com/js/v3/cashfree.js"></script>
<style>
  html, body { margin:0; padding:0; height:100%; background:#ffffff; }
  #status { font-family: -apple-system, system-ui, sans-serif; color:#6b6b6b;
            display:flex; align-items:center; justify-content:center; height:100%; }
</style>
</head>
<body>
<div id="status">Opening secure checkout…</div>
<script>
  function post(msg) {
    if (window.ReactNativeWebView) {
      window.ReactNativeWebView.postMessage(JSON.stringify(msg));
    }
  }
  try {
    var cashfree = Cashfree({ mode: ${JSON.stringify(opts.mode)} });
    cashfree.checkout({
      paymentSessionId: ${JSON.stringify(opts.paymentSessionId)},
      redirectTarget: '_self'
    }).then(function (result) {
      if (result && result.error) {
        post({ type: 'error', message: result.error.message });
        return;
      }
      // Any non-error completion: the server decides whether it was paid.
      post({ type: 'settled' });
    }).catch(function (e) {
      // A throw is not proof of failure — the customer may have paid and the
      // sheet errored afterwards. Settle and let the server adjudicate.
      post({ type: 'settled' });
    });
  } catch (e) {
    post({ type: 'error', message: 'Could not open checkout' });
  }
</script>
</body>
</html>`;
}

export default function CashfreeCheckoutScreen() {
  // Cast rather than the generic form, matching checkout.tsx/result.tsx: without
  // generated route types (a fresh checkout, which is what CI has) the generic is
  // constrained to a route path string, not a params record.
  const params = useLocalSearchParams() as unknown as CashfreeCheckoutParams;
  const [loading, setLoading] = useState(true);
  const clearCart = useCartStore((s) => s.clearCart);

  const orderId = String(params.orderId ?? '');
  const cashfreeOrderId = String(params.cashfreeOrderId ?? '');
  // Default to production on anything unrecognised, matching the server's
  // fail-toward-live asymmetry: a wrong "sandbox" would silently capture no real
  // money, while a wrong "production" fails loudly against a sandbox session.
  const mode = String(params.env ?? '').toUpperCase() === 'SANDBOX' ? 'sandbox' : 'production';

  const finish = useCallback(async () => {
    // Fast-path verify. The result screen polls server status as the backstop
    // (the webhook completes it regardless), so a failure here is swallowed
    // rather than shown as a payment failure.
    try {
      await api.post(`/v1/payments/order/${orderId}/verify`, {
        cashfreeOrderId,
      });
      clearCart();
    } catch {
      // ignore — the result screen confirms via polling
    }
    router.replace(`/payment/result?order_id=${orderId}`);
  }, [orderId, cashfreeOrderId, clearCart]);

  const onMessage = useCallback(
    (event: WebViewMessageEvent) => {
      let msg: BridgeMessage;
      try {
        msg = JSON.parse(event.nativeEvent.data) as BridgeMessage;
      } catch {
        return;
      }
      if (msg.type === 'dismiss') {
        router.back();
        return;
      }
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
      <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: 8, height: 48 }}>
        <Pressable
          onPress={() => router.back()}
          android_ripple={{ color: BACK_RIPPLE, borderless: true }}
          hitSlop={12}
          accessibilityRole="button"
          accessibilityLabel="Cancel payment"
          style={{ padding: 8 }}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={{ fontSize: 16, fontWeight: '600', color: customerColors.charcoal.DEFAULT }}>
          Secure checkout
        </Text>
      </View>

      <WebView
        source={{
          html: buildCashfreeHtml({
            paymentSessionId: String(params.paymentSessionId),
            mode,
          }),
          // A base URL is required for the SDK's own origin checks; without it
          // Android treats the document as opaque and the SDK refuses to open.
          baseUrl: 'https://sdk.cashfree.com',
        }}
        onMessage={onMessage}
        onLoadEnd={() => setLoading(false)}
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
