// packages/mobile-shared/src/validation/password.ts
//
// Strict password policy for account creation. One source of truth so the live
// requirements checklist under the field and the submit-time validation can't
// drift apart. Kept deliberately simple to reason about — length + character
// classes — which is far stronger than the old "8 chars, mix letters/numbers".

export interface PasswordCheck {
  id: string;
  label: string;
  test: (value: string) => boolean;
}

export const PASSWORD_MIN_LENGTH = 8;

export const PASSWORD_CHECKS: PasswordCheck[] = [
  { id: 'length', label: `At least ${PASSWORD_MIN_LENGTH} characters`, test: (v) => v.length >= PASSWORD_MIN_LENGTH },
  { id: 'upper', label: 'An uppercase letter', test: (v) => /[A-Z]/.test(v) },
  { id: 'lower', label: 'A lowercase letter', test: (v) => /[a-z]/.test(v) },
  { id: 'number', label: 'A number', test: (v) => /\d/.test(v) },
  { id: 'symbol', label: 'A symbol (! @ # $ …)', test: (v) => /[^A-Za-z0-9]/.test(v) },
];

/** True only when EVERY policy requirement is met. */
export function isStrongPassword(value: string): boolean {
  return PASSWORD_CHECKS.every((c) => c.test(value));
}

/** Which requirements a candidate password currently satisfies — drives the
 *  live checklist shown under the password field. */
export function passwordCheckResults(value: string): Array<PasswordCheck & { met: boolean }> {
  return PASSWORD_CHECKS.map((c) => ({ ...c, met: c.test(value) }));
}
