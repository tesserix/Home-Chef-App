import { describe, expect, it } from 'vitest';

import { getPhoneRule, isValidPhone, sanitizePhoneInput } from '../validation/phone';

describe('phone rules per market', () => {
  it('validates Australian mobiles without the trunk 0', () => {
    expect(getPhoneRule('AU').dialCode).toBe('+61');
    expect(isValidPhone('412345678', 'AU')).toBe(true);
    expect(isValidPhone('0412345678', 'AU')).toBe(false);
    expect(isValidPhone('9876543210', 'AU')).toBe(false);
  });

  it('accepts New Zealand mobiles of 8 to 10 digits', () => {
    expect(getPhoneRule('nz').dialCode).toBe('+64');
    expect(isValidPhone('21234567', 'NZ')).toBe(true);
    expect(isValidPhone('2123456789', 'NZ')).toBe(true);
    expect(isValidPhone('912345678', 'NZ')).toBe(false);
  });

  it('caps typing at the country length', () => {
    expect(sanitizePhoneInput('412 345 6789', 'AU')).toBe('412345678');
    expect(sanitizePhoneInput('98765432101', 'IN')).toBe('9876543210');
  });
});
