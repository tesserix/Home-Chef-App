import { describe, it, expect } from 'vitest';

// #1039: the meal-plan detail screen listed five meals summing ₹1,190 and then
// printed "Total ₹1,513.63" — ₹323.63 the customer could not account for. The
// pre-approval figure had the opposite problem: it was food only, so it
// understated what approving would charge.

import { mealPlanChargeLines, mealPlanApprovalEstimate } from '../utils/meal-plan-charge';

// The plan from the bug report.
const plan = { subtotal: 1190, tax: 145.03, total: 1513.63 };

describe('mealPlanChargeLines', () => {
  it('itemises the charge into lines that sum to the total', () => {
    const lines = mealPlanChargeLines(plan);

    expect(lines).toEqual([
      { label: 'Food subtotal', amount: 1190 },
      { label: 'Delivery', amount: 178.6 },
      { label: 'GST', amount: 145.03 },
    ]);
    const sum = lines.reduce((s, l) => s + l.amount, 0);
    expect(Math.round(sum * 100) / 100).toBe(plan.total);
  });

  // With escrow off the server returns total === subtotal, and inventing a
  // delivery or GST line for it would be a lie.
  it('omits lines the plan was not charged', () => {
    expect(mealPlanChargeLines({ subtotal: 900, tax: 0, total: 900 })).toEqual([
      { label: 'Food subtotal', amount: 900 },
    ]);
  });
});

describe('mealPlanApprovalEstimate', () => {
  it('scales delivery and GST to the days the chef accepted', () => {
    // Chef accepted 3 of 5 meals, worth ₹820 of the ₹1,190 of food.
    const estimate = mealPlanApprovalEstimate(plan, {
      acceptedFood: 820,
      acceptedDayCount: 3,
      totalDayCount: 5,
    });

    expect(estimate.food).toBe(820);
    expect(estimate.delivery).toBe(107.16);
    expect(estimate.gst).toBe(99.94);
    expect(estimate.total).toBe(1027.1);
  });

  it('returns the full charge when every day was accepted', () => {
    const estimate = mealPlanApprovalEstimate(plan, {
      acceptedFood: 1190,
      acceptedDayCount: 5,
      totalDayCount: 5,
    });

    expect(estimate.total).toBe(plan.total);
  });

  // Rejecting every day charges nothing — never a stray rounding residue.
  it('charges nothing when no day was accepted', () => {
    const estimate = mealPlanApprovalEstimate(plan, {
      acceptedFood: 0,
      acceptedDayCount: 0,
      totalDayCount: 5,
    });

    expect(estimate).toEqual({ food: 0, delivery: 0, gst: 0, total: 0 });
  });
});
