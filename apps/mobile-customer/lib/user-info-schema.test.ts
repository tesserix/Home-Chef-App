import { describe, it, expect } from '@jest/globals';
import { buildUserInfoSchema } from './user-info-schema';

const base = { firstName: 'Ava', lastName: 'Nguyen' };

describe('buildUserInfoSchema', () => {
  it('accepts an Australian mobile for AU', () => {
    expect(buildUserInfoSchema('AU').safeParse({ ...base, phone: '412345678' }).success).toBe(true);
  });

  it('accepts a New Zealand mobile for NZ', () => {
    expect(buildUserInfoSchema('NZ').safeParse({ ...base, phone: '211234567' }).success).toBe(true);
  });

  it('rejects an Indian number for AU with AU copy', () => {
    const r = buildUserInfoSchema('AU').safeParse({ ...base, phone: '9876543210' });
    expect(r.success).toBe(false);
    expect(r.error?.issues[0]?.message).toBe('Enter a valid 9-digit mobile number, without the leading 0');
  });

  it('keeps the Indian rule for IN', () => {
    expect(buildUserInfoSchema('IN').safeParse({ ...base, phone: '9876543210' }).success).toBe(true);
    expect(buildUserInfoSchema('IN').safeParse({ ...base, phone: '412345678' }).success).toBe(false);
  });
});
