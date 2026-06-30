import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// Regression guard for the tiffin meal-plan endpoints (#196). The mobile API
// client's baseURL is the API root WITHOUT the version segment (`.../api`, see
// eas.json), so every hook must prepend `/v1/...` itself. These hooks once
// omitted it, so the weekly-menu read + the booking POST 404'd and the booking
// screen showed "No weekly menu yet" for a chef who HAD published one. Lock the
// exact paths so the prefix can't silently regress again.

// Capture the queryFn/mutationFn each hook hands to React Query so the test can
// invoke it and assert the URL the api client is called with.
jest.mock('@tanstack/react-query', () => ({
  useQuery: (opts: { queryFn: () => unknown }) => ({ queryFn: opts.queryFn }),
  useMutation: (opts: { mutationFn: (v: unknown) => unknown }) => ({
    mutationFn: opts.mutationFn,
  }),
  useQueryClient: () => ({ invalidateQueries: jest.fn() }),
}));

jest.mock('../lib/api', () => ({
  api: {
    get: jest.fn(async () => ({ data: { isPublished: true, items: [] } })),
    post: jest.fn(async () => ({ data: { mealPlan: {} } })),
    put: jest.fn(async () => ({ data: { mealPlan: {} } })),
  },
}));

import { api } from '../lib/api';
import {
  useChefWeeklyMenu,
  useMyMealPlans,
  useMealPlan,
  useCreateMealPlan,
  useSkipMealPlanDay,
  useFinalizeMealPlan,
} from './useMealPlans';

const mockApi = api as unknown as {
  get: jest.Mock;
  post: jest.Mock;
  put: jest.Mock;
};

beforeEach(() => {
  mockApi.get.mockClear();
  mockApi.post.mockClear();
  mockApi.put.mockClear();
});

describe('useMealPlans endpoints are versioned (/v1)', () => {
  it('useChefWeeklyMenu reads the published weekly menu under /v1', async () => {
    await (useChefWeeklyMenu('chef-1') as { queryFn: () => Promise<unknown> }).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith('/v1/chefs/chef-1/weekly-menu');
  });

  it('useMyMealPlans lists the customer plans under /v1', async () => {
    await (useMyMealPlans() as { queryFn: () => Promise<unknown> }).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith('/v1/meal-plans');
  });

  it('useMealPlan reads one plan under /v1', async () => {
    await (useMealPlan('plan-9') as { queryFn: () => Promise<unknown> }).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith('/v1/meal-plans/plan-9');
  });

  it('useCreateMealPlan books against /v1/meal-plans', async () => {
    const body = { chefId: 'chef-1', days: [{ date: '2026-07-01', slot: 'lunch', variant: 'veg' }] };
    await (
      useCreateMealPlan() as { mutationFn: (v: unknown) => Promise<unknown> }
    ).mutationFn(body);
    expect(mockApi.post).toHaveBeenCalledWith('/v1/meal-plans', body);
  });

  it('useSkipMealPlanDay skips a day under /v1', async () => {
    await (
      useSkipMealPlanDay() as { mutationFn: (v: unknown) => Promise<unknown> }
    ).mutationFn({ planId: 'p1', dayId: 'd2' });
    expect(mockApi.put).toHaveBeenCalledWith('/v1/meal-plans/p1/days/d2/skip');
  });

  it('useFinalizeMealPlan approves and rejects under /v1', async () => {
    const finalize = useFinalizeMealPlan() as {
      mutationFn: (v: unknown) => Promise<unknown>;
    };
    await finalize.mutationFn({ id: 'p1', approve: true });
    expect(mockApi.put).toHaveBeenCalledWith('/v1/meal-plans/p1/approve');

    await finalize.mutationFn({ id: 'p1', approve: false });
    expect(mockApi.put).toHaveBeenCalledWith('/v1/meal-plans/p1/reject');
  });
});
