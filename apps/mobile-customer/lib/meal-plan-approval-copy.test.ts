import { describe, it, expect } from '@jest/globals';

import { mealPlanApprovalCopy } from './meal-plan-approval-copy';

// #1036 — the banner claimed "Your chef revised this plan" above "They can cook
// 5 of 5 days". The heading must follow what the chef actually did, and the copy
// must say "meals" (#1040), because plan.days holds booked meals, not dates.

function meals(...statuses: string[]): { status: string }[] {
  return statuses.map((status) => ({ status }));
}

describe('mealPlanApprovalCopy', () => {
  it('says accepted, never revised, when every meal was taken', () => {
    const copy = mealPlanApprovalCopy(
      meals('accepted', 'accepted', 'confirmed', 'confirmed', 'requested'),
    );
    expect(copy.title).toBe('Your chef accepted your plan');
    expect(copy.title).not.toMatch(/revis/i);
    expect(copy.body).toContain('all 5 meals');
    expect(copy.body).not.toMatch(/revis|of 5/i);
    expect(copy.acceptedCount).toBe(5);
    expect(copy.declinedCount).toBe(0);
  });

  it('reads naturally for a single accepted meal', () => {
    const copy = mealPlanApprovalCopy(meals('accepted'));
    expect(copy.title).toBe('Your chef accepted your plan');
    expect(copy.body).toContain('the meal you asked for');
    expect(copy.body).not.toContain('1 meals');
  });

  it('says revised, and how many dropped, when one meal was declined', () => {
    const copy = mealPlanApprovalCopy(meals('accepted', 'accepted', 'declined'));
    expect(copy.title).toBe('Your chef revised this plan');
    expect(copy.body).toContain('2 of 3 meals');
    expect(copy.body).toContain('1 meal dropped');
    expect(copy.body).not.toContain('1 meals dropped');
    expect(copy.acceptedCount).toBe(2);
    expect(copy.declinedCount).toBe(1);
  });

  it('pluralises the dropped count when several meals were declined', () => {
    const copy = mealPlanApprovalCopy(
      meals('accepted', 'declined', 'skipped', 'cancelled'),
    );
    expect(copy.title).toBe('Your chef revised this plan');
    expect(copy.body).toContain('1 of 4 meals');
    expect(copy.body).toContain('3 meals dropped');
    expect(copy.declinedCount).toBe(3);
  });

  // isDeclinedDayStatus is the shared "will not be served" rule — refunded and
  // failed count too, so the banner can never disagree with the priced subset.
  it('counts every declined-day status, not just "declined"', () => {
    const copy = mealPlanApprovalCopy(
      meals('accepted', 'skipped', 'cancelled', 'refunded', 'failed'),
    );
    expect(copy.acceptedCount).toBe(1);
    expect(copy.declinedCount).toBe(4);
  });

  it('does not offer an approval when the chef took nothing', () => {
    const copy = mealPlanApprovalCopy(meals('declined', 'declined', 'declined'));
    expect(copy.title).toBe("Your chef can't cook this plan");
    expect(copy.body).toContain('any of the 3 meals');
    expect(copy.body).toContain('Reject to cancel');
    expect(copy.body).not.toMatch(/Approve/);
    expect(copy.acceptedCount).toBe(0);
  });

  it('reads naturally when the only meal was declined', () => {
    const copy = mealPlanApprovalCopy(meals('declined'));
    expect(copy.title).toBe("Your chef can't cook this plan");
    expect(copy.body).toContain('the meal you asked for');
  });

  it('stays neutral, and never claims a revision, with no meals at all', () => {
    for (const copy of [mealPlanApprovalCopy([]), mealPlanApprovalCopy()]) {
      expect(copy.title).toBe('Your approval needed');
      expect(copy.body).not.toMatch(/revis/i);
      expect(copy.body).not.toMatch(/\bNaN\b|undefined/);
      expect(copy.acceptedCount).toBe(0);
      expect(copy.declinedCount).toBe(0);
    }
  });

  it('never says "day" or "days" anywhere (#1040)', () => {
    const cases = [
      mealPlanApprovalCopy([]),
      mealPlanApprovalCopy(meals('accepted')),
      mealPlanApprovalCopy(meals('accepted', 'accepted')),
      mealPlanApprovalCopy(meals('accepted', 'declined')),
      mealPlanApprovalCopy(meals('declined')),
      mealPlanApprovalCopy(meals('declined', 'declined')),
    ];
    for (const copy of cases) {
      expect(`${copy.title} ${copy.body}`).not.toMatch(/\bdays?\b/i);
    }
  });
});
