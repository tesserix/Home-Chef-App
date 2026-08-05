// The Cashfree v3 web-SDK checkout document, shared by every app that embeds it
// in a WebView.
//
// Deliberately the web SDK rather than Cashfree's React Native SDK: that one is a
// native module, so adopting it would force a store build before anyone could
// pay. This opens the same payment sheet (UPI intent included on Android) and
// ships with an OTA update.

/** Cashfree splits sandbox from production by HOST, and the SDK has no key
 *  prefix to infer it from — so the environment is always passed in, resolved
 *  by the server, never guessed on the device. */
export type CashfreeMode = 'sandbox' | 'production';

/** Normalises the server's env label to the SDK's mode.
 *
 *  Anything unrecognised falls to production, matching the server's
 *  fail-toward-live asymmetry: a wrong "sandbox" would silently capture no real
 *  money, while a wrong "production" fails loudly against a sandbox session. */
export function cashfreeModeFromEnv(env: string | undefined | null): CashfreeMode {
  return String(env ?? '').toUpperCase() === 'SANDBOX' ? 'sandbox' : 'production';
}

/** What the checkout document posts back over the WebView bridge. */
export interface CashfreeBridgeMessage {
  type: 'settled' | 'error';
  message?: string;
}

/**
 * Builds the checkout document.
 *
 * The session id is injected as a JSON string literal so a value containing
 * quotes cannot break out of the script.
 */
export function buildCashfreeCheckoutHtml(opts: {
  paymentSessionId: string;
  mode: CashfreeMode;
}): string {
  return `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no" />
<!-- Loaded unpinned, without Subresource Integrity, deliberately: Cashfree ships
     security and compliance fixes to this SDK continuously, and an SRI hash would
     hard-fail checkout the moment they publish one. The mitigation is that the SDK
     is never trusted for the payment OUTCOME — the server re-fetches the captured
     payment from Cashfree before anything is marked paid. -->
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
    // _modal, not _self: _self navigates this document away to Cashfree, which
    // settles the promise immediately, so the app leaves the WebView before the
    // payer can pay. _modal keeps the sheet in-page and resolves on close.
    cashfree.checkout({
      paymentSessionId: ${JSON.stringify(opts.paymentSessionId)},
      redirectTarget: '_modal'
    }).then(function (result) {
      if (result && result.error) {
        post({ type: 'error', message: result.error.message });
        return;
      }
      // Any non-error completion: the server decides whether it was paid.
      post({ type: 'settled' });
    }).catch(function () {
      // A throw is not proof of failure — the payer may have paid and the sheet
      // errored afterwards. Settle and let the server adjudicate.
      post({ type: 'settled' });
    });
  } catch (e) {
    post({ type: 'error', message: 'Could not open checkout' });
  }
</script>
</body>
</html>`;
}
