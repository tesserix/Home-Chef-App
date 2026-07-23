// In-app Razorpay Standard Checkout, embedded in a WebView.
//
// Replaces the old external-browser (expo-web-browser) flow: the customer pays
// inside the app. We load Razorpay's checkout.js against the server-created
// order_id, capture the result via the JS bridge, and confirm it server-side
// through POST /v1/payments/order/:orderId/verify (HMAC signature check) before
// routing to the result screen. The webhook remains the authoritative backstop.

import { useCallback, useState } from 'react';
import { ActivityIndicator, Platform, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { WebView, type WebViewMessageEvent } from 'react-native-webview';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { RAZORPAY_DISPLAY_CONFIG } from '../../lib/razorpay-config';
import { useCartStore } from '../../store/cart-store';
import { friendlyErrorMessage } from '../../lib/errors';

// Android ripple tint — translucent token, never a new literal colour.
const BACK_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

interface PaymentCheckoutParams {
  orderId: string;
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: string; // paise, as a string param
  currency: string;
  name?: string;
  email?: string;
  phone?: string;
  // kind='tip' reuses this sheet for a post-delivery tip charge (#45); kind='group'
  // for a group-order share (#46); kind='catering' for a catering deposit (#55,
  // verifies via /catering/requests/:cateringId/deposit/verify); kind='mealplan'
  // for a tiffin plan's full advance (#196, verifies via
  // /meal-plans/:mealPlanId/verify-payment).
  kind?: string;
  tipId?: string;
  groupId?: string;
  cateringId?: string;
  mealPlanId?: string;
}

// Bridge message shapes posted by the embedded checkout.js below.
type BridgeMessage =
  | { type: 'success'; razorpay_payment_id: string; razorpay_order_id: string; razorpay_signature: string }
  | { type: 'dismiss' }
  | { type: 'error'; message?: string };

// Builds the self-contained HTML that opens Razorpay Standard Checkout and
// relays the outcome back to React Native via window.ReactNativeWebView.
function buildCheckoutHtml(opts: {
  keyId: string;
  orderId: string;
  amount: number;
  currency: string;
  name?: string;
  email?: string;
  phone?: string;
}): string {
  // JSON.stringify keeps user/profile values safely escaped inside the script.
  const options = JSON.stringify({
    key: opts.keyId,
    order_id: opts.orderId,
    amount: opts.amount,
    currency: opts.currency || 'INR',
    name: 'Fe3dr',
    description: 'Order payment',
    prefill: {
      name: opts.name ?? '',
      email: opts.email ?? '',
      contact: opts.phone ?? '',
    },
    // UPI-first ordering (GPay/PhonePe/BHIM on top), cards/netbanking below.
    config: RAZORPAY_DISPLAY_CONFIG,
    theme: { color: '#FF385C' },
  });

  return `<!DOCTYPE html>
<html>
  <head>
    <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1" />
    <script src="https://checkout.razorpay.com/v1/checkout.js"></script>
  </head>
  <body style="background:#FFFFFF;margin:0">
    <script>
      function post(o) {
        if (window.ReactNativeWebView) {
          window.ReactNativeWebView.postMessage(JSON.stringify(o));
        }
      }
      function start() {
        try {
          var options = ${options};
          options.handler = function (r) {
            post({
              type: 'success',
              razorpay_payment_id: r.razorpay_payment_id,
              razorpay_order_id: r.razorpay_order_id,
              razorpay_signature: r.razorpay_signature,
            });
          };
          options.modal = {
            escape: true,
            ondismiss: function () { post({ type: 'dismiss' }); },
          };
          var rzp = new Razorpay(options);
          rzp.on('payment.failed', function (resp) {
            post({ type: 'error', message: (resp && resp.error && resp.error.description) || 'Payment failed' });
          });
          rzp.open();
        } catch (e) {
          post({ type: 'error', message: String(e && e.message ? e.message : e) });
        }
      }
      if (window.Razorpay) { start(); }
      else { window.addEventListener('load', start); }
    </script>
  </body>
</html>`;
}

export default function PaymentCheckout() {
  // expo-router's generic expects a Route-shaped map; our params are a plain
  // string record, so read untyped and cast (same as app/payment/result.tsx).
  const params = useLocalSearchParams() as unknown as PaymentCheckoutParams;
  const [verifying, setVerifying] = useState(false);

  const html = buildCheckoutHtml({
    keyId: params.razorpayKeyId ?? '',
    orderId: params.razorpayOrderId ?? '',
    amount: Number(params.amount ?? 0),
    currency: params.currency ?? 'INR',
    name: params.name,
    email: params.email,
    phone: params.phone,
  });

  const onMessage = useCallback(
    async (event: WebViewMessageEvent) => {
      let msg: BridgeMessage;
      try {
        msg = JSON.parse(event.nativeEvent.data) as BridgeMessage;
      } catch {
        return;
      }

      if (msg.type === 'dismiss') {
        // User closed the payment sheet — return to checkout to retry.
        router.back();
        return;
      }

      if (msg.type === 'error') {
        router.replace({
          pathname: '/payment/result',
          params: { error: msg.message || 'payment_failed' },
        });
        return;
      }

      // Success — confirm server-side before celebrating.
      setVerifying(true);
      try {
        // Post-delivery tip (#45): verify the tip charge, then show tip success.
        if (params.kind === 'tip') {
          await api.post(`/v1/payments/tip/${params.tipId}/verify`, {
            razorpayPaymentId: msg.razorpay_payment_id,
            razorpayOrderId: msg.razorpay_order_id,
          });
          router.replace({
            pathname: '/payment/result',
            params: { tip: '1', order_id: params.orderId },
          });
          return;
        }
        // Group / office order share (#46): verify the share, return to the group.
        if (params.kind === 'group') {
          await api.post(`/v1/group-orders/${params.groupId}/pay/verify`, {
            razorpayPaymentId: msg.razorpay_payment_id,
            razorpayOrderId: msg.razorpay_order_id,
          });
          router.replace(`/group-order/${params.groupId}` as never);
          return;
        }
        // Catering deposit (#55): verify the advance, return to the booking.
        if (params.kind === 'catering') {
          await api.post(`/v1/catering/requests/${params.cateringId}/deposit/verify`, {
            razorpayPaymentId: msg.razorpay_payment_id,
            razorpayOrderId: msg.razorpay_order_id,
          });
          router.replace(`/catering/${params.cateringId}` as never);
          return;
        }
        // Tiffin meal-plan advance (#196): verify the full escrow advance, then
        // show the customer their now-confirmed plan.
        if (params.kind === 'mealplan') {
          await api.post(`/v1/meal-plans/${params.mealPlanId}/verify-payment`, {
            razorpayPaymentId: msg.razorpay_payment_id,
            razorpaySignature: msg.razorpay_signature,
          });
          router.replace(`/meal-plans/${params.mealPlanId}` as never);
          return;
        }
        await api.post(`/v1/payments/order/${params.orderId}/verify`, {
          razorpayPaymentId: msg.razorpay_payment_id,
          razorpayOrderId: msg.razorpay_order_id,
          razorpaySignature: msg.razorpay_signature,
        });
        useCartStore.getState().clearCart();
        router.replace({
          pathname: '/payment/result',
          params: {
            order_id: params.orderId,
            razorpay_payment_id: msg.razorpay_payment_id,
          },
        });
      } catch (err: unknown) {
        router.replace({
          pathname: '/payment/result',
          params: {
            error: friendlyErrorMessage(err, 'We could not confirm your payment.'),
            order_id: params.orderId,
          },
        });
      }
    },
    [params.orderId, params.kind, params.tipId, params.groupId, params.cateringId, params.mealPlanId],
  );

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right', 'bottom']}>
      <View className="flex-row items-center bg-canvas border-b border-hairline px-4 pb-4 gap-3">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Cancel payment"
          className="p-1"
          android_ripple={{ color: BACK_RIPPLE, borderless: true, radius: 20 }}
        >
          {({ pressed }) => (
            <View className={pressed && Platform.OS === 'ios' ? 'opacity-60' : ''}>
              <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text className="text-xl font-semibold text-charcoal flex-1">Payment</Text>
      </View>

      {verifying ? (
        <View className="flex-1 items-center justify-center gap-3">
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
          <Text className="text-sm text-charcoal-soft">Confirming payment…</Text>
        </View>
      ) : (
        <WebView
          originWhitelist={['*']}
          source={{ html, baseUrl: 'https://fe3dr.com' }}
          onMessage={onMessage}
          javaScriptEnabled
          domStorageEnabled
          startInLoadingState
          renderLoading={() => (
            <View className="absolute inset-0 items-center justify-center bg-canvas">
              <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
            </View>
          )}
          style={{ flex: 1, backgroundColor: customerColors.canvas }}
        />
      )}
    </SafeAreaView>
  );
}
