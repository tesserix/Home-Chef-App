import { describe, it, expect } from '@jest/globals';
import { money, renewalLine } from './subscription-format';
import type { MealSubscription } from '../hooks/useMealSubscription';

// #1042: a subscription card that names an amount and a cadence but never a
// date leaves the customer unable to answer "when am I charged?" — worst on a
// trial, where the first charge is the whole question.

function sub(over: Partial<MealSubscription> = {}): MealSubscription {
  return {
    id: 's1',
    chefId: 'chef-1',
    slots: ['lunch'],
    days: [1, 2, 3, 4, 5],
    variant: 'veg',
    cadence: 'weekly',
    cycleAmount: 900,
    currency: 'INR',
    status: 'active',
    currentPeriodEnd: '2026-08-12T00:00:00Z',
    creditBalance: 0,
    ...over,
  };
}

describe('money', () => {
  it('renders whole rupees with Indian grouping', () => {
    expect(money(900)).toBe('₹900');
    expect(money(123456)).toBe('₹1,23,456');
    expect(money(899.6)).toBe('₹900');
  });
});

describe('renewalLine', () => {
  it('an active subscription states the date AND the recurring amount', () => {
    expect(renewalLine(sub())).toBe('Renews 12 Aug · ₹900/week');
  });

  it('a monthly cadence says month, not week', () => {
    expect(renewalLine(sub({ cadence: 'monthly' }))).toBe('Renews 12 Aug · ₹900/month');
  });

  it('a trial says when it ends and what happens next — the #1042 case', () => {
    expect(renewalLine(sub({ status: 'trialing' }))).toBe(
      'Trial ends 12 Aug · then ₹900/week',
    );
  });

  it('paused and past_due describe their own next event', () => {
    expect(renewalLine(sub({ status: 'paused' }))).toBe(
      'Paused · next charge 12 Aug if resumed',
    );
    expect(renewalLine(sub({ status: 'past_due' }))).toBe('Payment due · retried 12 Aug');
  });

  it('promises nothing when there is no next charge to promise', () => {
    expect(renewalLine(sub({ status: 'cancelled' }))).toBe('');
    expect(renewalLine(sub({ currentPeriodEnd: undefined }))).toBe('');
    expect(renewalLine(sub({ currentPeriodEnd: 'not-a-date' }))).toBe('');
  });
});
