// packages/mobile-shared/src/validation/phone.ts
//
// Country-aware phone rules. HomeChef is India-only today, but the platform is
// built to go multi-country — and every country has its own national-number
// length (India 10, US 10, UK 10–11, …). Centralise the rule so the input can
// HARD-CAP typing at the country's length (you physically can't enter an 11th
// digit for India) and validation stays in one place instead of a regex copied
// across screens.

export interface PhoneRule {
  /** ISO 3166-1 alpha-2 country code. */
  country: string;
  /** E.164 dial code, e.g. "+91". */
  dialCode: string;
  /** Exact national-number length (digits after the dial code). */
  length: number;
  /** Full validator for a complete national number. */
  pattern: RegExp;
  /** Placeholder / example national number. */
  example: string;
}

// Add a country here to support it — nothing else needs to change.
const PHONE_RULES: Record<string, PhoneRule> = {
  IN: { country: 'IN', dialCode: '+91', length: 10, pattern: /^[6-9]\d{9}$/, example: '9876543210' },
};

export const DEFAULT_PHONE_COUNTRY = 'IN';

/** The rule for a country, falling back to the default (India) for anything
 *  not yet configured. */
export function getPhoneRule(country?: string): PhoneRule {
  const key = (country ?? DEFAULT_PHONE_COUNTRY).toUpperCase();
  return PHONE_RULES[key] ?? PHONE_RULES[DEFAULT_PHONE_COUNTRY]!;
}

/** The hard input guard: strip everything that isn't a digit and cap at the
 *  country's national length. Feed every phone `onChangeText` through this so a
 *  user can never type past the allowed number of digits. */
export function sanitizePhoneInput(raw: string, country?: string): string {
  return raw.replace(/\D/g, '').slice(0, getPhoneRule(country).length);
}

/** True when `value` is a complete, valid national number for the country. */
export function isValidPhone(value: string, country?: string): boolean {
  return getPhoneRule(country).pattern.test(value);
}
