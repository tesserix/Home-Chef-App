import { describe, it, expect } from '@jest/globals';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

// #975: every mobile app applies `nativewind/babel`, which routes all JSX
// through react-native-css-interop, and that runtime drops Pressable's FUNCTION
// form of `style` — the view then renders with no style at all (#972: a black
// button with white text painted white-on-white on the refund screen).
//
// This guard is a source scan rather than a render test on purpose: the defect
// lives in the babel/interop pipeline, so a component rendered under jest's
// transform would style correctly and prove nothing.

const REPO_ROOT = resolve(__dirname, '../../..');

const SCANNED = [
  'apps/mobile-customer',
  'apps/mobile-vendor',
  'apps/mobile-admin',
  'apps/mobile-delivery',
  'packages/mobile-shared',
];

const SKIPPED = new Set(['node_modules', 'ios', 'android', '.expo', 'dist', 'build']);

// `style={(...) => ...}` in any spelling: destructured, named, or bare.
const FUNCTION_STYLE = /style=\{\s*\([^)]*\)\s*=>/;

function tsxFilesUnder(dir: string): string[] {
  let entries: string[];
  try {
    entries = readdirSync(dir);
  } catch {
    return [];
  }
  const found: string[] = [];
  for (const entry of entries) {
    if (SKIPPED.has(entry)) continue;
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      found.push(...tsxFilesUnder(full));
    } else if (entry.endsWith('.tsx')) {
      found.push(full);
    }
  }
  return found;
}

function functionStyleSites(): string[] {
  const sites: string[] = [];
  for (const app of SCANNED) {
    for (const file of tsxFilesUnder(join(REPO_ROOT, app))) {
      readFileSync(file, 'utf8')
        .split('\n')
        .forEach((line, i) => {
          if (FUNCTION_STYLE.test(line)) {
            sites.push(`${relative(REPO_ROOT, file)}:${i + 1}`);
          }
        });
    }
  }
  return sites;
}

describe('Pressable style props', () => {
  it('finds source to scan', () => {
    // Without this, a bad path or a changed layout turns the guard below into a
    // test that passes by scanning nothing.
    expect(tsxFilesUnder(join(REPO_ROOT, 'apps/mobile-customer/app')).length).toBeGreaterThan(20);
  });

  it('matches the function form and not a static one', () => {
    expect(FUNCTION_STYLE.test('style={({ pressed }) => [styles.a, pressed && styles.b]}')).toBe(
      true,
    );
    expect(FUNCTION_STYLE.test('style={(state) => styles.a}')).toBe(true);
    expect(FUNCTION_STYLE.test('style={styles.a}')).toBe(false);
    expect(FUNCTION_STYLE.test('style={[styles.a, active && styles.b]}')).toBe(false);
  });

  it('has no function-styled Pressable in any mobile app', () => {
    // Press feedback belongs on an inner View via the children-as-function form
    // (`<Pressable style={styles.x}>{({ pressed }) => ...}</Pressable>`), which
    // the interop runtime does honour.
    expect(functionStyleSites()).toEqual([]);
  });
});
