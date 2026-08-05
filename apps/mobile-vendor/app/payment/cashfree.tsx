// In-app Cashfree checkout for the vendor app, embedded in a WebView.
//
// The chef used to be handed to an external browser session against
// fe3dr.com/pay, which showed an iOS "wants to use fe3dr.com to sign in"
// consent prompt an order checkout never shows. This is the same sheet the
// customer app opens for an order (app/payment/cashfree.tsx there), sharing one
// checkout document from @homechef/mobile-shared/payments.
//
// Cashfree hands the client no signed success payload, so ANY non-error close
// routes back and asks the SERVER whether the money arrived. The client's view
// of "paid" is never trusted.

import { useCallback, useState } from 'react';
import { ActivityIndicator, Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import type { WebViewMessageEvent } from 'react-native-webview';
import { getWebView } from '../../lib/webview-support';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import {
  buildCashfreeCheckoutHtml,
  cashfreeModeFromEnv,
  type CashfreeBridgeMessage,
} from '@homechef/mobile-shared/payments';
import { useToast } from '@homechef/mobile-shared/ui';
import { api } from '../../lib/api';

interface Params {
  /** The FSSAI request being paid for. */
  requestId?: string;
  paymentSessionId?: string;
  /** "SANDBOX" | "PRODUCTION", resolved server-side — never guessed here. */
  env?: string;
}

export default function VendorCashfreeCheckoutScreen() {
  const params = useLocalSearchParams() as unknown as Params;
  const { show: showToast } = useToast();
  const [loading, setLoading] = useState(true);

  const requestId = String(params.requestId ?? '');
  const paymentSessionId = String(params.paymentSessionId ?? '');
  const mode = cashfreeModeFromEnv(params.env);

  // Leaves the sheet and asks the server to adjudicate. Confirm takes an EMPTY
  // body: Cashfree gives the client no payment id or signature, so the server
  // reads the capture from the gateway and binds it by the order id, which is
  // the request row's own uuid.
  const leave = useCallback(
    async (settled: boolean) => {
      if (settled && requestId) {
        try {
          await api.post(`/chef/fssai/requests/${requestId}/confirm`);
        } catch {
          // The tracker polls the server's real status — a failed fast-path
          // confirm must not tell a chef who paid that they did not.
        }
      }
      router.replace('/fssai');
    },
    [requestId],
  );

  const onMessage = useCallback(
    (e: WebViewMessageEvent) => {
      let msg: CashfreeBridgeMessage;
      try {
        msg = JSON.parse(e.nativeEvent.data) as CashfreeBridgeMessage;
      } catch {
        return;
      }
      if (msg.type === 'error') {
        showToast({ message: msg.message || 'Checkout could not open.', tone: 'error' });
        void leave(false);
        return;
      }
      void leave(true);
    },
    [leave, showToast],
  );

  const WebView = getWebView();

  // No native module in this binary — the sheet cannot render. The caller only
  // routes here when it can, so this is the belt to that braces.
  if (!WebView) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <Header />
        <View style={styles.centered}>
          <Text style={styles.body}>
            This build can't open the payment sheet. Update the app and try again.
          </Text>
        </View>
      </SafeAreaView>
    );
  }

  if (!paymentSessionId) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <Header />
        <View style={styles.centered}>
          <Text style={styles.body}>This payment session has expired. Try again.</Text>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <Header />
      <WebView
        originWhitelist={['*']}
        source={{ html: buildCashfreeCheckoutHtml({ paymentSessionId, mode }) }}
        onMessage={onMessage}
        onLoadEnd={() => setLoading(false)}
        javaScriptEnabled
        domStorageEnabled
        // UPI intent hands off to the payer's bank app and comes back.
        setSupportMultipleWindows={false}
        style={styles.web}
      />
      {loading ? (
        <View style={styles.loadingOverlay}>
          <ActivityIndicator color={theme.colors.ink.DEFAULT} />
        </View>
      ) : null}
    </SafeAreaView>
  );
}

function Header() {
  return (
    <View style={styles.header}>
      <Pressable
        onPress={() => router.replace('/fssai')}
        hitSlop={12}
        accessibilityRole="button"
        accessibilityLabel="Cancel payment"
        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
      >
        {({ pressed }) => (
          <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : undefined}>
            <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} strokeWidth={1.75} />
          </View>
        )}
      </Pressable>
      <Text style={styles.title}>Secure payment</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: theme.colors.ink.DEFAULT,
  },
  web: { flex: 1 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: theme.spacing[6] },
  body: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    textAlign: 'center',
  },
  loadingOverlay: {
    position: 'absolute',
    top: 0, left: 0, right: 0, bottom: 0,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.paper,
  },
});
