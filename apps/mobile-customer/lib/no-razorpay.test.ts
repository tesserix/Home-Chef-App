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

// The legal screens name Razorpay as a data recipient. That copy is wrong too, but
// rewriting a privacy policy is a content change with its own review (#1121).
const CONTENT_ONLY = ['app/terms.tsx', 'app/privacy.tsx', 'app/refund.tsx'];

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.tsx?$/.test(entry) && !full.endsWith('no-razorpay.test.ts') ? [full] : [];
  });
}

describe('the Razorpay checkout is gone from the customer app', () => {
  const files = SCANNED.flatMap((d) => sourceFiles(join(ROOT, d))).filter(
    (f) => !CONTENT_ONLY.some((allowed) => f.endsWith(join(...allowed.split('/')))),
  );

  it('no module imports the SDK or handles its payload', () => {
    const offenders = files.filter((f) =>
      /react-native-razorpay|RazorpayCheckout|RAZORPAY_DISPLAY_CONFIG|checkout\.razorpay\.com|razorpay_payment_id|razorpayKeyId|razorpaySignature/.test(
        readFileSync(f, 'utf8'),
      ),
    );
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
