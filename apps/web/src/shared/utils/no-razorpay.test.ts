import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// #1086 — Cashfree is the only gateway. The server can no longer answer
// `provider: "razorpay"`, so any surviving client branch is not just dead: in
// CheckoutPage it was the `else` fallback, meaning an unrecognised provider
// opened a Razorpay modal against a key the API stopped issuing. This guards the
// deletion, because the next person to add a gateway will copy an existing branch.

const SRC = resolve(__dirname, '../..');

// The legal pages name Razorpay as a data recipient. That copy is factually wrong
// too, but rewriting a privacy policy is a content change with its own review —
// tracked in #1121, not smuggled in behind a grep.
const CONTENT_ONLY = ['features/legal/pages', 'features/auth/pages/RegisterPage.tsx'];

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.tsx?$/.test(entry) && !/no-razorpay\.test\.ts$/.test(entry) ? [full] : [];
  });
}

describe('the Razorpay checkout is gone from the web client', () => {
  const files = sourceFiles(SRC).filter(
    (f) => !CONTENT_ONLY.some((allowed) => f.includes(join(...allowed.split('/')))),
  );

  it('no module reaches for the Razorpay SDK or its payload', () => {
    const offenders = files.filter((f) => {
      const code = readFileSync(f, 'utf8');
      // Comments explaining why Cashfree differs from the rail it replaced are
      // history worth keeping; a live reference is not.
      return /window\.Razorpay|new Razorpay|RazorpayOptions|RazorpayPaymentResponse|openRazorpayCheckout|razorpay_payment_id|razorpayKeyId|razorpaySignature|razorpayOrderId|razorpayPaymentId/.test(
        code,
      );
    });
    expect(offenders.map((f) => f.slice(SRC.length + 1))).toEqual([]);
  });

  it('index.html loads no Razorpay script and allows it nowhere in the CSP', () => {
    const html = readFileSync(resolve(SRC, '../index.html'), 'utf8');
    expect(html).not.toMatch(/razorpay/i);
  });
});
