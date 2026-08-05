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

import { useCallback, useEffect, useRef, useState } from 'react';
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

// Returns to the tracker WITHOUT stacking a second copy of it.
//
// This screen is pushed on top of /fssai, so it pops. Replacing would swap it
// for a duplicate /fssai and leave the chef's back button pointing at a spent
// checkout — the customer app replaces because its done route is a different
// screen (the result page); ours is the screen underneath.
function backToTracker() {
  if (router.canGoBack()) {
    router.back();
    return;
  }
  router.replace('/fssai');
}

export default function VendorCashfreeCheckoutScreen() {
  const params = useLocalSearchParams() as unknown as Params;
  const { show: showToast } = useToast();
  const [loading, setLoading] = useState(true);
  // Whichever of the bridge or the poll wins, the other must not also navigate.
  const settled = useRef(false);

  const requestId = String(params.requestId ?? '');
  const paymentSessionId = String(params.paymentSessionId ?? '');
  const mode = cashfreeModeFromEnv(params.env);

  // Leaves the sheet and asks the server to adjudicate. Confirm takes an EMPTY
  // body: Cashfree gives the client no payment id or signature, so the server
  // reads the capture from the gateway and binds it by the order id, which is
  // the request row's own uuid.
  const leave = useCallback(
    async (ok: boolean) => {
      if (settled.current) return;
      settled.current = true;
      if (ok && requestId) {
        try {
          await api.post(`/chef/fssai/requests/${requestId}/confirm`);
        } catch {
          // The tracker polls the server's real status — a failed fast-path
          // confirm must not tell a chef who paid that they did not.
        }
      }
      backToTracker();
    },
    [requestId],
  );

  // The bridge is not a reliable completion signal: net banking and 3DS navigate
  // this document away and kill the script before it can post, so the screen
  // would otherwise sit on a payment that has already gone through.
  //
  // Poll confirm, not the request: the status only flips once confirm or the
  // webhook runs, and the missing confirm is precisely the failure — reading the
  // request back would wait on something nothing is going to do. confirm
  // re-fetches the capture from Cashfree server-side and is idempotent, so it is
  // safe to repeat.
  useEffect(() => {
    if (!requestId) return;
    const timer = setInterval(async () => {
      try {
        const r = await api.post<{ request?: { status?: string } }>(
          `/chef/fssai/requests/${requestId}/confirm`,
        );
        const status = r.data?.request?.status;
        if (status && status !== 'awaiting_payment') {
          clearInterval(timer);
          settled.current = true;
          backToTracker();
        }
      } catch {
        // Not captured yet, or transient — keep polling.
      }
    }, 4000);
    return () => clearInterval(timer);
  }, [requestId]);

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
        source={{
          html: buildCashfreeCheckoutHtml({ paymentSessionId, mode }),
          // The SDK runs its own origin checks; without a base URL the document
          // is opaque and the sheet refuses to advance past bank selection.
          baseUrl: 'https://sdk.cashfree.com',
        }}
        onMessage={onMessage}
        onLoadEnd={() => setLoading(false)}
        javaScriptEnabled
        domStorageEnabled
        // UPI intent hands off to GPay/PhonePe via a custom scheme; without this
        // the WebView blocks the navigation and UPI silently does nothing.
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
        onPress={backToTracker}
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
