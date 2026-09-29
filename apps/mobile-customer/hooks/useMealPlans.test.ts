import { describe, it, expect, jest, beforeEach } from "@jest/globals";

// Regression guard for the tiffin meal-plan endpoints (#196). The mobile API
// client's baseURL is the API root WITHOUT the version segment (`.../api`, see
// eas.json), so every hook must prepend `/v1/...` itself. These hooks once
// omitted it, so the weekly-menu read + the booking POST 404'd and the booking
// screen showed "No weekly menu yet" for a chef who HAD published one. Lock the
// exact paths so the prefix can't silently regress again.

// Capture the queryFn/mutationFn each hook hands to React Query so the test can
// invoke it and assert the URL the api client is called with. The mock replaces
// the real useQuery/useMutation at runtime; the casts below go through `unknown`
// because tsc still sees the real (richer) return types.
jest.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryFn: () => unknown }) => ({ queryFn: opts.queryFn }),
  useMutation: (opts: { mutationFn: (v: unknown) => unknown }) => ({
    mutationFn: opts.mutationFn,
  }),
  useQueryClient: () => ({ invalidateQueries: jest.fn() }),
}));

jest.mock("../lib/api", () => ({
  api: {
    get: jest.fn(async () => ({ data: { isPublished: true, items: [] } })),
    post: jest.fn(async () => ({ data: { mealPlan: {} } })),
    put: jest.fn(async () => ({ data: { mealPlan: {} } })),
  },
}));

import { api } from "../lib/api";
import {
  mealPlanAdvanceBreakdown,
  mealPlanSubsetBreakdown,
  useChefWeeklyMenu,
  useMyMealPlans,
  useMealPlan,
  useCreateMealPlan,
  useSkipMealPlanDay,
  useFinalizeMealPlan,
} from "./useMealPlans";

type MockFn = ReturnType<typeof jest.fn>;
const mockApi = api as unknown as {
  get: MockFn;
  post: MockFn;
  put: MockFn;
};

type CapturedQuery = { queryFn: () => Promise<unknown> };
type CapturedMutation = { mutationFn: (v: unknown) => Promise<unknown> };

const asQuery = (hook: unknown) => hook as unknown as CapturedQuery;
const asMutation = (hook: unknown) => hook as unknown as CapturedMutation;

beforeEach(() => {
  mockApi.get.mockClear();
  mockApi.post.mockClear();
  mockApi.put.mockClear();
});

describe("useMealPlans endpoints are versioned (/v1)", () => {
  it("useChefWeeklyMenu reads the published weekly menu under /v1", async () => {
    await asQuery(useChefWeeklyMenu("chef-1")).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith("/v1/chefs/chef-1/weekly-menu");
  });

  it("useMyMealPlans lists the customer plans under /v1", async () => {
    await asQuery(useMyMealPlans()).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith("/v1/meal-plans");
  });

  it("useMealPlan reads one plan under /v1", async () => {
    await asQuery(useMealPlan("plan-9")).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith("/v1/meal-plans/plan-9");
  });

  it("useCreateMealPlan books against /v1/meal-plans", async () => {
    const body = {
      chefId: "chef-1",
      days: [{ date: "2026-07-01", slot: "lunch", variant: "veg" }],
    };
    await asMutation(useCreateMealPlan()).mutationFn(body);
    expect(mockApi.post).toHaveBeenCalledWith("/v1/meal-plans", body);
  });

  it("useSkipMealPlanDay skips a day under /v1", async () => {
    await asMutation(useSkipMealPlanDay()).mutationFn({
      planId: "p1",
      dayId: "d2",
    });
    expect(mockApi.put).toHaveBeenCalledWith("/v1/meal-plans/p1/days/d2/skip");
  });

  it("useFinalizeMealPlan approves and rejects under /v1", async () => {
    const finalize = asMutation(useFinalizeMealPlan());
    await finalize.mutationFn({ id: "p1", approve: true });
    expect(mockApi.put).toHaveBeenCalledWith("/v1/meal-plans/p1/approve");

    await finalize.mutationFn({ id: "p1", approve: false });
    expect(mockApi.put).toHaveBeenCalledWith("/v1/meal-plans/p1/reject");
  });
});

// #402: the advance shown to the customer must equal the server charge (food + GST +
// per-day delivery), never the food-only selection sum.
describe("mealPlanAdvanceBreakdown", () => {
  it("separates the platform fee from delivery in the server charge", () => {
    const plan = { subtotal: 175, platformFee: 7.4, tax: 12.03, total: 233.43 };
    expect(mealPlanAdvanceBreakdown(plan)).toEqual({
      food: 175,
      platformFee: 7.4,
      gst: 12.03,
      delivery: 39,
      total: 233.43,
      amountPaise: 23343,
    });
  });
  it("splits the SERVER total into food / GST / delivery and derives delivery", () => {
    const b = mealPlanAdvanceBreakdown({
      subtotal: 2000,
      tax: 160,
      total: 2560,
    });
    expect(b).toEqual({
      food: 2000,
      platformFee: 0,
      gst: 160,
      delivery: 400, // 2560 − 2000 − 160
      total: 2560,
      amountPaise: 256000, // the full advance, not 200000 (food only)
    });
  });

  it("escrow-off plan (total == subtotal) charges food only, no GST/delivery", () => {
    const b = mealPlanAdvanceBreakdown({ subtotal: 1500, total: 1500 });
    expect(b.gst).toBe(0);
    expect(b.delivery).toBe(0);
    expect(b.amountPaise).toBe(150000);
  });

  it("never yields negative delivery and rounds to paise", () => {
    const b = mealPlanAdvanceBreakdown({
      subtotal: 999.99,
      tax: 80,
      total: 1179.99,
    });
    expect(b.delivery).toBe(100); // 1179.99 − 999.99 − 80
    expect(b.amountPaise).toBe(117999);
    const weird = mealPlanAdvanceBreakdown({
      subtotal: 100,
      tax: 50,
      total: 120,
    });
    expect(weird.delivery).toBe(0); // total below food+gst floors at 0, never negative
  });
});

describe("mealPlanSubsetBreakdown", () => {
  it('scales delivery GST by accepted meals when food prices differ', () => {
    const plan = {
      subtotal: 300, platformFee: 12, tax: 21.06,
      taxFood: 15, taxService: 2.16, taxDelivery: 3.9,
      total: 411.06, days: [1, 2],
    };
    expect(mealPlanSubsetBreakdown(plan, 100, 1)).toEqual({
      food: 100, platformFee: 4, gst: 7.67, delivery: 39,
      total: 150.67, amountPaise: 15067,
    });
  });

  // 5 days, ₹1,000 food, 5% GST, ₹40/day delivery → ₹1,250.
  const plan = { subtotal: 1000, tax: 50, total: 1250, days: [1, 2, 3, 4, 5] };

  it("every day accepted reproduces the plan exactly", () => {
    expect(mealPlanSubsetBreakdown(plan, 1000, 5)).toEqual(
      mealPlanAdvanceBreakdown(plan),
    );
  });

  it("a cherry-picked subset carries its GST and its delivery, not food alone (#1039)", () => {
    // 3 of 5 days, ₹600 of food: GST at the plan's 5%, delivery for 3 days.
    const b = mealPlanSubsetBreakdown(plan, 600, 3);
    expect(b.food).toBe(600);
    expect(b.gst).toBe(30);
    expect(b.delivery).toBe(120);
    expect(b.total).toBe(750); // NOT 600 — the bug approved against food only
    expect(b.food + b.gst + b.delivery).toBe(b.total); // the lines reach the total
    expect(b.amountPaise).toBe(75000);
  });

  it("rejecting every day charges nothing", () => {
    expect(mealPlanSubsetBreakdown(plan, 0, 0).total).toBe(0);
  });

  it("falls back to the plan when there is no base to scale against", () => {
    const escrowOff = { subtotal: 0, total: 0, days: [] };
    expect(mealPlanSubsetBreakdown(escrowOff, 0, 0)).toEqual(
      mealPlanAdvanceBreakdown(escrowOff),
    );
    const noDays = { subtotal: 500, tax: 25, total: 525, days: [] };
    expect(mealPlanSubsetBreakdown(noDays, 200, 1)).toEqual(
      mealPlanAdvanceBreakdown(noDays),
    );
  });
});
