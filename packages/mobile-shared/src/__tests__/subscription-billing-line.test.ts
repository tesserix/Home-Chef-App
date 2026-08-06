import { describe, it, expect } from 'vitest';

// #1042: the subscription card printed cadence and amount but never when the
// next charge lands, so a customer on a ₹900/week trial could not tell when the
// trial ended or when they would first be charged.

import { subscriptionBillingLine } from '../utils/subscription-billing-line';

const base = {
  cycleAmount: 900,
  cadence: 'weekly',
  currentPeriodEnd: '2026-08-12T18:30:00Z',
};

describe('subscriptionBillingLine', () => {
  it('names the day the trial ends and what is charged then', () => {
    expect(subscriptionBillingLine({ ...base, status: 'trialing' })).toBe(
      'Trial ends 13 Aug · then ₹900/week',
    );
  });

  it('names the renewal date and amount on an active subscription', () => {
    expect(subscriptionBillingLine({ ...base, status: 'active' })).toBe(
      'Renews 13 Aug · ₹900/week',
    );
  });

  it('says a monthly cadence in the same breath as the amount', () => {
    expect(subscriptionBillingLine({ ...base, status: 'active', cadence: 'monthly' })).toBe(
      'Renews 13 Aug · ₹900/month',
    );
  });

  // A paused subscription is not billed, so naming a renewal date would be a lie.
  it('states that a paused subscription is not being charged', () => {
    expect(subscriptionBillingLine({ ...base, status: 'paused' })).toBe(
      'Paused · no charge until you resume',
    );
  });

  it('flags an overdue payment with the date it was due', () => {
    expect(subscriptionBillingLine({ ...base, status: 'past_due' })).toBe(
      'Payment due since 13 Aug',
    );
  });

  it('says nothing about billing once cancelled', () => {
    expect(subscriptionBillingLine({ ...base, status: 'cancelled' })).toBeNull();
  });

  it('says nothing rather than guessing when the period end is missing', () => {
    expect(subscriptionBillingLine({ ...base, status: 'active', currentPeriodEnd: undefined })).toBeNull();
    expect(subscriptionBillingLine({ ...base, status: 'active', currentPeriodEnd: 'nonsense' })).toBeNull();
  });

  // The period end is an instant: 18:30 UTC is already the next IST day, and a
  // charge dated by the server in IST must not print a day early.
  it('dates the charge in IST, not in the device timezone', () => {
    expect(
      subscriptionBillingLine({ ...base, status: 'active', currentPeriodEnd: '2026-08-12T17:00:00Z' }),
    ).toBe('Renews 12 Aug · ₹900/week');
  });
});
