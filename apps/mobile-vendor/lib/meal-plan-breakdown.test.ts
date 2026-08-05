import { describe, it, expect } from '@jest/globals';

import {
  chefEarningLines,
  customerChargeLines,
  planRespondable,
  planStatusLabel,
} from './meal-plan-breakdown';
import type { MealPlanChefEarnings } from '../hooks/useMealPlans';

function earnings(over: Partial<MealPlanChefEarnings> = {}): MealPlanChefEarnings {
  return {
    currency: 'INR',
    commissionRate: 0.06,
    tdsRate: 0.01,
    payableDays: 2,
    excludedDays: 0,
    foodSubtotal: 400,
    foodGst: 20,
    gross: 420,
    platformCommission: 24,
    cgst: 2.16,
    sgst: 2.16,
    tds: 4.2,
    netPayout: 391.8,
    customerSubtotal: 400,
    customerPlatformFee: 16,
    customerDelivery: 50,
    customerTaxFood: 20,
    customerTaxService: 2.88,
    customerTaxDelivery: 9,
    customerTax: 31.88,
    customerTotal: 497.88,
    refundedToCustomer: 0,
    days: [],
    ...over,
  };
}

describe('customerChargeLines', () => {
  it('lists every supply the customer was charged, in receipt order', () => {
    const lines = customerChargeLines(earnings());
    expect(lines.map((l) => l.label)).toEqual([
      'Food',
      'Platform fee',
      'Delivery',
      'GST on food',
      'GST on platform fee',
      'GST on delivery',
    ]);
  });

  it('sums to the customer total, to the paise', () => {
    const e = earnings();
    const sum = customerChargeLines(e).reduce((s, l) => s + l.amount, 0);
    expect(Math.round(sum * 100) / 100).toBe(e.customerTotal);
  });

  it('omits a supply that was not charged rather than showing a zero row', () => {
    const lines = customerChargeLines(
      earnings({
        customerDelivery: 0,
        customerTaxDelivery: 0,
        customerTax: 22.88,
        customerTotal: 438.88,
      }),
    );
    expect(lines.map((l) => l.label)).not.toContain('Delivery');
    expect(lines.map((l) => l.label)).not.toContain('GST on delivery');
  });

  it('shows a refund as its own negative line when days were returned', () => {
    const lines = customerChargeLines(earnings({ refundedToCustomer: 120.5 }));
    const refund = lines.find((l) => l.label === 'Refunded');
    expect(refund?.amount).toBe(-120.5);
  });
});

describe('chefEarningLines', () => {
  it('deducts commission and TDS from gross and lands on the net payout', () => {
    const e = earnings();
    const lines = chefEarningLines(e);
    expect(lines.map((l) => l.label)).toEqual([
      'Food',
      'GST on food',
      'Platform commission (6%)',
      'TDS (1%)',
    ]);
    const sum = lines.reduce((s, l) => s + l.amount, 0);
    expect(Math.round(sum * 100) / 100).toBe(e.netPayout);
  });

  it('renders the deductions negative so the arithmetic reads on screen', () => {
    const lines = chefEarningLines(earnings());
    expect(lines.find((l) => l.label.startsWith('Platform commission'))?.amount).toBe(-24);
    expect(lines.find((l) => l.label.startsWith('TDS'))?.amount).toBe(-4.2);
  });

  it('formats a fractional commission rate without a trailing zero', () => {
    const lines = chefEarningLines(earnings({ commissionRate: 0.075 }));
    expect(lines.map((l) => l.label)).toContain('Platform commission (7.5%)');
  });

  it('never shows delivery or the platform fee — they are not the chef’s', () => {
    const labels = chefEarningLines(earnings()).map((l) => l.label);
    expect(labels).not.toContain('Delivery');
    expect(labels).not.toContain('Platform fee');
  });

  it('handles a plan with nothing payable', () => {
    const lines = chefEarningLines(
      earnings({
        payableDays: 0,
        excludedDays: 2,
        foodSubtotal: 0,
        foodGst: 0,
        gross: 0,
        platformCommission: 0,
        tds: 0,
        netPayout: 0,
      }),
    );
    expect(lines).toEqual([]);
  });
});

describe('planRespondable', () => {
  it('is true only while the plan awaits this chef', () => {
    expect(planRespondable('pending_chef')).toBe(true);
    expect(planRespondable('confirmed')).toBe(false);
    expect(planRespondable('active')).toBe(false);
    expect(planRespondable('cancelled')).toBe(false);
    expect(planRespondable(undefined)).toBe(false);
  });
});

describe('planStatusLabel', () => {
  it('names every lifecycle state in the chef’s language', () => {
    expect(planStatusLabel('pending_chef')).toBe('Awaiting your response');
    expect(planStatusLabel('awaiting_customer')).toBe('Awaiting customer approval');
    expect(planStatusLabel('confirmed')).toBe('Confirmed');
    expect(planStatusLabel('active')).toBe('In progress');
    expect(planStatusLabel('completed')).toBe('Completed');
    expect(planStatusLabel('cancelled')).toBe('Cancelled');
    expect(planStatusLabel('expired')).toBe('Expired');
  });

  it('falls back to the raw status rather than rendering nothing', () => {
    expect(planStatusLabel('some_new_state')).toBe('some_new_state');
  });
});
