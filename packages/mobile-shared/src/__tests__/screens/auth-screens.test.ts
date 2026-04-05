// Auth screen validation logic tests — vitest node environment
// These tests verify the Zod validation schemas used by the auth screens
// React Native rendering tests use jest-expo in each app's test suite (node env cannot render RN components)

import { describe, it, expect } from 'vitest';
import { z } from 'zod';

// ─── Duplicate the schemas from the screen files for isolated testing ─────────
// Keep in sync with packages/mobile-shared/src/screens/*.tsx

const loginSchema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(1, 'Password is required'),
});

const registerSchema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(8, 'Minimum 8 characters'),
  firstName: z.string().min(1, 'First name is required'),
  lastName: z.string().min(1, 'Last name is required'),
  phone: z.string().optional(),
});

const forgotPasswordSchema = z.object({
  email: z.string().email('Enter a valid email'),
});

// ─── LoginScreen validation ───────────────────────────────────────────────────

describe('loginSchema', () => {
  it('accepts valid email and password', () => {
    const result = loginSchema.safeParse({ email: 'user@test.com', password: 'secret' });
    expect(result.success).toBe(true);
  });

  it('rejects invalid email format', () => {
    const result = loginSchema.safeParse({ email: 'not-an-email', password: 'secret' });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].message).toBe('Enter a valid email');
    }
  });

  it('rejects empty password', () => {
    const result = loginSchema.safeParse({ email: 'user@test.com', password: '' });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].message).toBe('Password is required');
    }
  });

  it('rejects missing fields', () => {
    const result = loginSchema.safeParse({});
    expect(result.success).toBe(false);
  });
});

// ─── RegisterScreen validation ────────────────────────────────────────────────

describe('registerSchema', () => {
  const validData = {
    email: 'john@test.com',
    password: 'password123',
    firstName: 'John',
    lastName: 'Doe',
  };

  it('accepts valid registration data', () => {
    const result = registerSchema.safeParse(validData);
    expect(result.success).toBe(true);
  });

  it('accepts phone as optional field', () => {
    const result = registerSchema.safeParse({ ...validData, phone: '+1234567890' });
    expect(result.success).toBe(true);
  });

  it('accepts registration data without phone', () => {
    const result = registerSchema.safeParse(validData);
    expect(result.success).toBe(true);
  });

  it('rejects password shorter than 8 characters', () => {
    const result = registerSchema.safeParse({ ...validData, password: 'short' });
    expect(result.success).toBe(false);
    if (!result.success) {
      const pwIssue = result.error.issues.find((i) => i.path.includes('password'));
      expect(pwIssue?.message).toBe('Minimum 8 characters');
    }
  });

  it('rejects empty firstName', () => {
    const result = registerSchema.safeParse({ ...validData, firstName: '' });
    expect(result.success).toBe(false);
  });

  it('rejects empty lastName', () => {
    const result = registerSchema.safeParse({ ...validData, lastName: '' });
    expect(result.success).toBe(false);
  });

  it('rejects invalid email', () => {
    const result = registerSchema.safeParse({ ...validData, email: 'bad-email' });
    expect(result.success).toBe(false);
  });
});

// ─── ForgotPasswordScreen validation ─────────────────────────────────────────

describe('forgotPasswordSchema', () => {
  it('accepts a valid email', () => {
    const result = forgotPasswordSchema.safeParse({ email: 'reset@test.com' });
    expect(result.success).toBe(true);
  });

  it('rejects an invalid email', () => {
    const result = forgotPasswordSchema.safeParse({ email: 'not-valid' });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].message).toBe('Enter a valid email');
    }
  });

  it('rejects empty email', () => {
    const result = forgotPasswordSchema.safeParse({ email: '' });
    expect(result.success).toBe(false);
  });

  it('rejects missing email field', () => {
    const result = forgotPasswordSchema.safeParse({});
    expect(result.success).toBe(false);
  });
});

// ─── Screen contract assertions (verified at implementation time) ─────────────

describe('Screen file exports contract', () => {
  it('LoginScreen module exists and exports LoginScreen', async () => {
    const mod = await import('../../screens/LoginScreen');
    expect(typeof mod.LoginScreen).toBe('function');
  });

  it('RegisterScreen module exists and exports RegisterScreen', async () => {
    const mod = await import('../../screens/RegisterScreen');
    expect(typeof mod.RegisterScreen).toBe('function');
  });

  it('ForgotPasswordScreen module exists and exports ForgotPasswordScreen', async () => {
    const mod = await import('../../screens/ForgotPasswordScreen');
    expect(typeof mod.ForgotPasswordScreen).toBe('function');
  });

  it('screens barrel index exports all three screens', async () => {
    const mod = await import('../../screens');
    expect(typeof mod.LoginScreen).toBe('function');
    expect(typeof mod.RegisterScreen).toBe('function');
    expect(typeof mod.ForgotPasswordScreen).toBe('function');
  });
});
