import { describe, it, expect } from '@jest/globals';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';

// #1086 — Cashfree is the only gateway. This app shipped the react-native-razorpay
// NATIVE SDK, compiled into every build, for a rail that can no longer take money:
// the server stopped answering `provider: "razorpay"` and stopped issuing a key id,
// so every branch behind it was unreachable code carrying a payment SDK's weight.
// Guards the deletion, because the next gateway added here will be copied from one
// of these branches.

const ROOT = resolve(__dirname, '..');
const SCANNED = ['app', 'components', 'hooks', 'lib', 'store', 'types'];

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.tsx?$/.test(entry) && !full.endsWith('no-razorpay.test.ts') ? [full] : [];
  });
}

describe('the Razorpay checkout is gone from the customer app', () => {
  const files = SCANNED.flatMap((d) => sourceFiles(join(ROOT, d)));

  it('no module imports the SDK or handles its payload', () => {
    const offenders = files.filter((f) =>
      /react-native-razorpay|RazorpayCheckout|RAZORPAY_DISPLAY_CONFIG|checkout\.razorpay\.com|razorpay_payment_id|razorpayKeyId|razorpaySignature|razorpayOrderId|razorpayPaymentId/.test(
        readFileSync(f, 'utf8'),
      ),
    );
    expect(offenders.map((f) => f.slice(ROOT.length + 1))).toEqual([]);
  });

  // A bare mention in these screens is a factual claim to a user about who
  // receives their data, which the SDK patterns above never caught (#1121).
  it('no user-facing copy names Razorpay as the payment processor', () => {
    const copy = files.filter((f) => /app\/(terms|privacy|refund)\.tsx$/.test(f.replace(/\\/g, '/')));
    expect(copy.length).toBe(3);
    const offenders = copy.filter((f) => /razorpay/i.test(readFileSync(f, 'utf8')));
    expect(offenders.map((f) => f.slice(ROOT.length + 1))).toEqual([]);
  });

  it('the native SDK is not a dependency', () => {
    const pkg = JSON.parse(readFileSync(join(ROOT, 'package.json'), 'utf8')) as {
      dependencies?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };
    expect(Object.keys({ ...pkg.dependencies, ...pkg.devDependencies })).not.toContain(
      'react-native-razorpay',
    );
  });
});
