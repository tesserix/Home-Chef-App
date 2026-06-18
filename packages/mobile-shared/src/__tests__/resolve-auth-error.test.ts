// resolve-auth-error.test.ts — locks in the err.code mapping (PR #181 fix #5).
// React Native Firebase puts the machine code on `err.code`, not always in
// `err.message`, so resolveAuthErrorMessage must match against both and never
// leak a raw `auth/...` / `auto_login_*` code to the user.

import { describe, it, expect } from 'vitest';

// resolveAuthErrorMessage lives in bff-session.ts, which imports expo-secure-store
// at module scope — mock it so the module loads in the node test env.
import { vi } from 'vitest';
vi.mock('expo-secure-store', () => ({
  getItemAsync: vi.fn(),
  setItemAsync: vi.fn(),
  deleteItemAsync: vi.fn(),
}));

import { resolveAuthErrorMessage } from '../auth/bff-session';

describe('resolveAuthErrorMessage', () => {
  it('maps Firebase code carried on err.code (not in message)', () => {
    // The exact shape RN Firebase throws: code set, message is generic prose.
    const err = { code: 'auth/email-already-in-use', message: 'Something failed.' };
    expect(resolveAuthErrorMessage(err)).toBe('An account already exists for that email.');
  });

  it('maps auth/invalid-credential to a friendly wrong-credentials message', () => {
    expect(resolveAuthErrorMessage({ code: 'auth/invalid-credential' })).toBe(
      'Wrong email or password. Please try again.',
    );
  });

  it('maps codes found only in the message string', () => {
    const err = new Error('Firebase: Error (auth/user-not-found).');
    expect(resolveAuthErrorMessage(err)).toBe('No account found for that email.');
  });

  it('still resolves auto_login_* codes thrown as Error messages', () => {
    expect(resolveAuthErrorMessage(new Error('auto_login_502'))).toBe(
      "We're having a hiccup on our end. Try again in a moment.",
    );
  });

  it('never leaks a raw code for unknown errors', () => {
    const msg = resolveAuthErrorMessage({ code: 'auth/some-new-code', message: 'weird' });
    expect(msg).toBe("We couldn't sign you in. Please try again.");
    expect(msg).not.toContain('auth/');
  });

  it('handles non-error inputs without throwing', () => {
    expect(resolveAuthErrorMessage(null)).toBe("We couldn't sign you in. Please try again.");
    expect(resolveAuthErrorMessage('plain string')).toBe(
      "We couldn't sign you in. Please try again.",
    );
  });
});
