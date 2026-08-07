import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// #1086 — Cashfree is the only gateway. The server can no longer answer
// `provider: "razorpay"`, so any surviving client branch is not just dead: in
// CheckoutPage it was the `else` fallback, meaning an unrecognised provider
// opened a Razorpay modal against a key the API stopped issuing. This guards the
// deletion, because the next person to add a gateway will copy an existing branch.

const SRC = resolve(__dirname, '../..');

// Nothing is exempt: the legal pages were rewritten in #1121 and now name the
// processor that actually handles the money.

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.tsx?$/.test(entry) && !/no-razorpay\.test\.ts$/.test(entry) ? [full] : [];
  });
}

describe('the Razorpay checkout is gone from the web client', () => {
  const files = sourceFiles(SRC);

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

  // The SDK patterns above never matched prose, so the copy naming Razorpay as a
  // data recipient sat behind an exemption and stayed wrong for three releases.
  // These are the files where a bare mention is a factual claim to a user (#1121).
  it('no user-facing copy names Razorpay as the payment processor', () => {
    const copy = files.filter(
      (f) => f.includes(join('features', 'legal', 'pages')) || f.endsWith('RegisterPage.tsx'),
    );
    expect(copy.length).toBeGreaterThan(0);
    const offenders = copy.filter((f) => /razorpay/i.test(readFileSync(f, 'utf8')));
    expect(offenders.map((f) => f.slice(SRC.length + 1))).toEqual([]);
  });

  it('index.html loads no Razorpay script and allows it nowhere in the CSP', () => {
    const html = readFileSync(resolve(SRC, '../index.html'), 'utf8');
    expect(html).not.toMatch(/razorpay/i);
  });
});
