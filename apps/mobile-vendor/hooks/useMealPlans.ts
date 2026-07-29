import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// useMealPlans — chef-side tiffin meal-plan data (#195). Wraps the weekly-menu
// CRUD (#192) and the request → accept/cherry-pick handshake (#194/#196):
//   GET  /chef/weekly-menu          (incl. draft)
//   PUT  /chef/weekly-menu          (replace-all + publish)
//   GET  /chef/meal-plans?status=   (the chef's requests, scoped server-side)
//   POST /chef/meal-plans/:id/respond  (accept all / cherry-pick a subset)

export type MealSlot = 'lunch' | 'dinner';
export type MealVariant = 'veg' | 'nonveg';

export interface WeeklyMenuItem {
  id?: string;
  dayOfWeek: number; // 0=Sun .. 6=Sat
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  menuItemId?: string | null;
  /** When true the cell is a bundled thali/combo at one price. */
  isCombo?: boolean;
  /** The dishes the thali/combo includes (e.g. ["Rice","Dal","Sabji"]). */
  comboComponents?: string[];
  /** How much food, and for how many people. */
  portionSize?: string;
  serves?: number;
}

export interface WeeklyMenu {
  isPublished: boolean;
  publishedAt?: string | null;
  items: WeeklyMenuItem[];
}

export interface MealPlanDay {
  id: string;
  date: string; // ISO
  slot: MealSlot;
  variant: MealVariant;
  status: string;
  dishName?: string;
  price: number;
}

export interface MealPlan {
  id: string;
  mealPlanNumber: string;
  customerId: string;
  chefId: string;
  status: string;
  startDate: string;
  endDate: string;
  subtotal: number;
  total: number;
  currency?: string;
  days: MealPlanDay[];
  customer?: { firstName?: string; lastName?: string; email?: string } | null;
}

const WEEKDAY_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

/**
 * Mirrors the API's validatePublishableGrid (#1): a publishable week must have
 * every offered (day × slot) filled — no holes. Returns a human-readable message
 * for the first missing cell, or null when complete. Used to block publish
 * client-side so the chef sees the real reason, not a generic 400.
 */
export function weeklyMenuHole(items: WeeklyMenuItem[]): string | null {
  if (items.length === 0) return 'Add at least one dish before publishing.';
  const days = new Set<number>();
  const slots = new Set<MealSlot>();
  const present = new Set<string>();
  for (const it of items) {
    days.add(it.dayOfWeek);
    slots.add(it.slot);
    present.add(`${it.dayOfWeek}|${it.slot}`);
  }
  for (let day = 0; day < 7; day++) {
    if (!days.has(day)) continue;
    for (const slot of ['lunch', 'dinner'] as MealSlot[]) {
      if (!slots.has(slot)) continue;
      if (!present.has(`${day}|${slot}`)) {
        return `${WEEKDAY_SHORT[day]} is missing its ${slot} dish — every offered day needs a ${slot} dish.`;
      }
    }
  }
  return null;
}

/** The authed chef's weekly menu (published or draft). */
export function useWeeklyMenu() {
  return useQuery<WeeklyMenu>({
    queryKey: ['chef', 'weekly-menu'],
    queryFn: () => api.get<WeeklyMenu>('/chef/weekly-menu').then((r) => r.data),
    staleTime: 60_000,
  });
}

/** Replace-all save + publish toggle for the weekly menu. */
export function useSaveWeeklyMenu() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { isPublished: boolean; items: WeeklyMenuItem[] }) =>
      api.put<WeeklyMenu>('/chef/weekly-menu', body).then((r) => r.data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['chef', 'weekly-menu'] }),
  });
}

// Per-date dynamic menu (#405/#406): a date holds MULTIPLE dishes per slot, and a
// dish can be a combo (bundle) with its set price + component names.
export interface DailyMenuItemInput {
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  isCombo: boolean;
  comboComponents: string[];
  /** Set when the dish came from the chef's à-la-carte menu (DishPicker). */
  menuItemId?: string | null;
  /** How much food, and for how many people. */
  portionSize?: string;
  serves?: number;
  sortOrder: number;
}

export interface DailyMenuDay {
  date: string; // YYYY-MM-DD
  isPublished: boolean;
  publishedAt?: string | null;
  items: (DailyMenuItemInput & { id?: string })[];
}

/** The chef's own per-date menus (incl. drafts) over [from, to]. */
export function useMyDailyMenu(from: string, to: string) {
  return useQuery<{ days: DailyMenuDay[] }>({
    queryKey: ['chef', 'daily-menu', from, to],
    queryFn: () =>
      api
        .get<{ days: DailyMenuDay[] }>(`/chef/daily-menu?from=${from}&to=${to}`)
        .then((r) => r.data),
    staleTime: 60_000,
  });
}

/** Replace-all save + publish toggle for ONE date's menu. */
export function useSaveDailyMenu() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: {
      date: string;
      isPublished: boolean;
      items: DailyMenuItemInput[];
    }) =>
      api
        .put(`/chef/daily-menu/${vars.date}`, {
          isPublished: vars.isPublished,
          items: vars.items,
        })
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chef', 'daily-menu'] }),
  });
}

/**
 * The chef's meal plans in one or more statuses (comma-separated; default:
 * awaiting their response). The server filters — it does NOT return the full
 * history — so the states you want must be named explicitly.
 */
export function useChefMealPlanRequests(status: string = 'pending_chef') {
  return useQuery<{ data: MealPlan[] }>({
    queryKey: ['chef', 'meal-plans', status],
    queryFn: () =>
      api
        .get<{ data: MealPlan[] }>(`/chef/meal-plans?status=${encodeURIComponent(status)}`)
        .then((r) => r.data),
    refetchInterval: 30_000, // requests are time-boxed (24h) — keep the inbox fresh
    staleTime: 10_000,
  });
}

// ── Bulk subscription prep view (#50) ──────────────────────────────────────

export interface PrepManifestLine {
  slot: MealSlot;
  variant: MealVariant;
  dishName: string;
  total: number;
  prepared: number;
}

export interface PrepPackingRow {
  dayId: string;
  slot: MealSlot;
  variant: MealVariant;
  dishName: string;
  status: string;
  planNumber: string;
  customerName: string;
}

export interface PrepTotals {
  lunch: number;
  dinner: number;
  total: number;
  prepared: number;
}

export interface PrepManifest {
  date: string; // YYYY-MM-DD
  manifest: PrepManifestLine[];
  packingList: PrepPackingRow[];
  totals: PrepTotals;
}

/** The day's prep manifest (counts by dish) + packing list. date = YYYY-MM-DD. */
export function usePrepManifest(date: string) {
  return useQuery<PrepManifest>({
    queryKey: ['chef', 'prep', date],
    queryFn: () => api.get<PrepManifest>(`/chef/prep?date=${date}`).then((r) => r.data),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

/** Mark a whole dish (date+slot+variant+dish) or explicit days prepared. */
export function useMarkPrepared() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      date?: string;
      slot?: MealSlot;
      variant?: MealVariant;
      dishName?: string;
      dayIds?: string[];
    }) => api.post<{ prepared: number }>('/chef/prep/mark', body).then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chef', 'prep'] }),
  });
}

/** Accept all days, or cherry-pick a subset (the rest are declined). */
export function useRespondMealPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: {
      id: string;
      acceptAll: boolean;
      acceptedDayIds: string[];
    }) =>
      api
        .post(`/chef/meal-plans/${vars.id}/respond`, {
          acceptAll: vars.acceptAll,
          acceptedDayIds: vars.acceptedDayIds,
        })
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chef', 'meal-plans'] }),
  });
}

// ── Refund flow (docs/refund-policy-v3-spec.md, #834) ─────────────────────────
// A late skip/cancel routes to the chef, who sets HOW MUCH to refund — any amount from
// the day's lead-time floor up to 100%. The base is everything the customer paid for the
// day (food + GST + delivery), so the amounts must come from the server; they cannot be
// derived from foodPrice. The floor is enforced server-side.

export interface RefundDecisionDay {
  dayId: string;
  date: string; // YYYY-MM-DD
  slot: string;
  dishName: string;
  customerName: string;
  mealPlanNumber: string;
  foodPrice: number;
  /** The least this day may be refunded, and that floor in rupees. */
  minPercent: number;
  minRefund: number;
  /** 100% of what the customer paid for the day. */
  fullRefund: number;
  /** 50%. Retained from the pre-v3 fixed Full/Half pair. */
  halfRefund: number;
}

/** The refund at `percent`, interpolated from the server's 100% figure so the picker
 *  updates without a round-trip. The server recomputes it authoritatively on submit. */
export function refundAtPercent(day: RefundDecisionDay, percent: number): number {
  return Math.round(day.fullRefund * percent) / 100;
}

// Days awaiting THIS chef's decision.
export function useChefPendingRefundDecisions() {
  return useQuery<{ data: RefundDecisionDay[] }>({
    queryKey: ['chef', 'refund-decisions'],
    queryFn: () =>
      api
        .get<{ data: RefundDecisionDay[] }>('/chef/meal-plan-days/pending-refund-decisions')
        .then((r) => r.data),
  });
}

// The chef's decision: refund `percent` of what the customer paid, or decline (the day
// is cooked and the customer charged).
export function useChefRefundDecision() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { dayId: string; percent?: number; decline?: boolean }) =>
      api
        .post(`/chef/meal-plan-days/${vars.dayId}/refund-decision`, {
          percent: vars.percent ?? 0,
          decline: !!vars.decline,
        })
        .then((r) => r.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'refund-decisions'] });
      qc.invalidateQueries({ queryKey: ['chef', 'meal-plans'] });
    },
  });
}

// ── Next-24h dashboard window ───────────────────────────────────────────────

/** One meal the kitchen owes soon, whether or not it has become an order yet. */
export interface UpcomingMeal {
  dayId: string;
  planNumber: string;
  startsAt: string;
  slot: string;
  variant: string;
  dishName: string;
  status: string;
  customerName: string;
  /** Empty until the day's order locks — the signal it is live in the kitchen. */
  orderNumber?: string;
}

export interface UpcomingResponse {
  hours: number;
  total: number;
  lunch: number;
  dinner: number;
  meals: UpcomingMeal[];
}

/**
 * What the chef owes in the next `hours`.
 *
 * The dashboard read only live orders, so a pre-booked tiffin day was invisible
 * until its order locked 12h before service — a chef with a confirmed week still
 * saw an empty screen. This answers from the plan days themselves.
 */
export function useChefUpcoming(hours: number = 24) {
  return useQuery<UpcomingResponse>({
    queryKey: ['chef', 'upcoming', hours],
    queryFn: () =>
      api.get<UpcomingResponse>(`/chef/prep/upcoming?hours=${hours}`).then((r) => r.data),
    refetchInterval: 60_000, // time-sensitive; a dashboard left open must not go stale
    staleTime: 30_000,
  });
}
