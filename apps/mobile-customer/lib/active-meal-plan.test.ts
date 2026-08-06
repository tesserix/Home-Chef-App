import { describe, it, expect } from '@jest/globals';
import { selectActiveMealPlanMeal } from './active-meal-plan';
import type { MealPlan, MealPlanDay } from '../hooks/useMealPlans';

// #1037 — the Home card must show today's or the next meal, never a past one,
// and must disappear entirely when a plan has nothing left to serve.

const TODAY = '2026-08-06';

let seq = 0;
function day(over: Partial<MealPlanDay> & { date: string }): MealPlanDay {
  seq += 1;
  return {
    id: `d${seq}`,
    slot: 'lunch',
    variant: 'veg',
    status: 'confirmed',
    dishName: 'Veg Thali',
    price: 200,
    ...over,
  } as MealPlanDay;
}

function plan(over: Partial<MealPlan> & { days: MealPlanDay[] }): MealPlan {
  return {
    id: 'p1',
    mealPlanNumber: 'MP-1',
    chefId: 'chef-1',
    status: 'confirmed',
    startDate: '2026-08-05',
    endDate: '2026-08-09',
    subtotal: 1000,
    total: 1000,
    chef: { businessName: 'Saffron Home Kitchen' },
    ...over,
  } as MealPlan;
}

// The dates below are bare YYYY-MM-DD; toLocalDateKey parses them as local
// midnight, so the keys round-trip regardless of the runner's timezone.
describe('selectActiveMealPlanMeal', () => {
  it('prefers today’s meal over a later one', () => {
    const got = selectActiveMealPlanMeal(
      [plan({ days: [day({ date: '2026-08-08' }), day({ date: TODAY })] })],
      TODAY,
    );
    expect(got?.isToday).toBe(true);
    expect(got && toKey(got.day.date)).toBe(TODAY);
  });

  it('falls forward to the next meal when nothing is booked today', () => {
    const got = selectActiveMealPlanMeal(
      [plan({ days: [day({ date: '2026-08-09' }), day({ date: '2026-08-07' })] })],
      TODAY,
    );
    expect(got?.isToday).toBe(false);
    expect(got && toKey(got.day.date)).toBe('2026-08-07');
  });

  it('never surfaces a past meal, even one still marked Scheduled (#1034)', () => {
    // The overdue sweep does not exist, so finished days sit at 'confirmed'
    // forever. Date wins over status.
    const got = selectActiveMealPlanMeal(
      [plan({ days: [day({ date: '2026-08-01', status: 'confirmed' })] })],
      TODAY,
    );
    expect(got).toBeNull();
  });

  it('skips a meal today has already lost — declined, skipped, cancelled', () => {
    const got = selectActiveMealPlanMeal(
      [
        plan({
          days: [
            day({ date: TODAY, slot: 'lunch', status: 'skipped' }),
            day({ date: TODAY, slot: 'dinner', dishName: 'Paneer' }),
          ],
        }),
      ],
      TODAY,
    );
    expect(got?.day.dishName).toBe('Paneer');
  });

  it('orders same-day meals by slot, so lunch shows before dinner', () => {
    const got = selectActiveMealPlanMeal(
      [
        plan({
          days: [
            day({ date: TODAY, slot: 'dinner', dishName: 'Dinner dish' }),
            day({ date: TODAY, slot: 'lunch', dishName: 'Lunch dish' }),
          ],
        }),
      ],
      TODAY,
    );
    expect(got?.day.dishName).toBe('Lunch dish');
  });

  it('keeps today’s delivered meal on screen once nothing is left to come', () => {
    const got = selectActiveMealPlanMeal(
      [plan({ days: [day({ date: TODAY, status: 'delivered' })] })],
      TODAY,
    );
    expect(got?.day.status).toBe('delivered');
    expect(got?.isToday).toBe(true);
  });

  it('hides once the last meal is delivered and the day has passed', () => {
    expect(
      selectActiveMealPlanMeal(
        [plan({ days: [day({ date: '2026-08-05', status: 'delivered' })] })],
        TODAY,
      ),
    ).toBeNull();
  });

  it('reports progress over standing meals only', () => {
    const got = selectActiveMealPlanMeal(
      [
        plan({
          days: [
            day({ date: '2026-08-04', status: 'delivered' }),
            day({ date: '2026-08-05', status: 'cancelled' }), // dropped from the count
            day({ date: TODAY }),
            day({ date: '2026-08-07' }),
          ],
        }),
      ],
      TODAY,
    );
    expect(got?.mealNumber).toBe(2); // delivered 4th, then today's
    expect(got?.totalMeals).toBe(3); // the cancelled meal is not counted
  });

  it('picks the soonest meal across several live plans', () => {
    const got = selectActiveMealPlanMeal(
      [
        plan({ id: 'later', days: [day({ date: '2026-08-08', dishName: 'Later' })] }),
        plan({ id: 'sooner', days: [day({ date: TODAY, dishName: 'Sooner' })] }),
      ],
      TODAY,
    );
    expect(got?.plan.id).toBe('sooner');
    expect(got?.day.dishName).toBe('Sooner');
  });

  it('ignores plans that are not live', () => {
    expect(
      selectActiveMealPlanMeal(
        [plan({ status: 'cancelled', days: [day({ date: TODAY })] })],
        TODAY,
      ),
    ).toBeNull();
    expect(
      selectActiveMealPlanMeal(
        [plan({ status: 'completed', days: [day({ date: TODAY })] })],
        TODAY,
      ),
    ).toBeNull();
  });

  it('handles no plans, and a plan with no days', () => {
    expect(selectActiveMealPlanMeal(undefined, TODAY)).toBeNull();
    expect(selectActiveMealPlanMeal([], TODAY)).toBeNull();
    expect(selectActiveMealPlanMeal([plan({ days: [] })], TODAY)).toBeNull();
  });
});

function toKey(iso: string): string {
  const d = new Date(iso);
  return `${d.getFullYear()}-${`${d.getMonth() + 1}`.padStart(2, '0')}-${`${d.getDate()}`.padStart(2, '0')}`;
}
