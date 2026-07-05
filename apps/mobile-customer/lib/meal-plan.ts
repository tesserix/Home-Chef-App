import { customerColors } from '@homechef/mobile-shared/theme';

import type { MealPlan, MealPlanDay } from '../hooks/useMealPlans';

// Presentation helpers for tiffin meal-plan statuses (#196). Keeps the list +
// detail screens consistent and the status copy in one place.

// Statuses worth surfacing on the "Show my plan" chip — a plan still in flight
// (awaiting the chef or the customer) or running. Terminal statuses
// (completed/cancelled/expired) are hidden. Shared by the home + chef-page chips
// so the "is this plan live?" rule lives in one place (#434).
const LIVE_STATUSES = new Set([
  'pending_chef',
  'awaiting_customer',
  'chef_modified',
  'chef_accepted_full',
  'confirmed',
  'active',
]);

export function isLiveMealPlanStatus(status: string): boolean {
  return LIVE_STATUSES.has(status);
}

// Day statuses that mean "this day won't be served" — drives the struck-through
// row styling in the list AND the accepted-days / total calc on the approval
// screen, so the rule stays in one place (#434).
const DECLINED_DAY_STATUSES = new Set(['declined', 'skipped', 'cancelled', 'refunded']);

export function isDeclinedDayStatus(status: string): boolean {
  return DECLINED_DAY_STATUSES.has(status);
}

// summarizeLivePlan powers the compact "N days left · today: <status>" home chip
// (#434). daysLeft = days still to be served (not delivered, not declined/
// skipped/cancelled/refunded). todayStatus = the status of the day dated
// todayISO ('YYYY-MM-DD'), or null when no day falls on today. Pure (today is
// injected) so it's testable without mocking the clock.
export function summarizeLivePlan(
  days: MealPlanDay[],
  todayISO: string,
): { daysLeft: number; todayStatus: string | null } {
  const daysLeft = days.filter(
    (d) => !isDeclinedDayStatus(d.status) && d.status !== 'delivered',
  ).length;
  const today = days.find((d) => d.date === todayISO);
  return { daysLeft, todayStatus: today ? today.status : null };
}

// pickLiveMealPlan returns the single most-recent live plan to surface on a chip,
// optionally scoped to one chef (the chef-page chip). Most recent = latest
// startDate. Returns undefined when the customer has no live plan (chip hides).
export function pickLiveMealPlan(
  plans: MealPlan[] | undefined,
  chefId?: string,
): MealPlan | undefined {
  const live = (plans ?? []).filter(
    (p) => isLiveMealPlanStatus(p.status) && (chefId === undefined || p.chefId === chefId),
  );
  return live.sort((a, b) => (b.startDate ?? '').localeCompare(a.startDate ?? ''))[0];
}

export interface StatusMeta {
  label: string;
  color: string;
  bg: string;
  /** Customer action required (the chef trimmed the plan — approve/reject). */
  needsAction: boolean;
}

export function mealPlanStatusMeta(status: string): StatusMeta {
  switch (status) {
    case 'pending_chef':
      return {
        label: 'Waiting for chef',
        color: customerColors.charcoal.soft,
        bg: customerColors.surface.soft,
        needsAction: false,
      };
    case 'awaiting_customer':
    case 'chef_modified':
      return {
        label: 'Your approval needed',
        color: customerColors.coral.DEFAULT,
        bg: customerColors.coral.tint,
        needsAction: true,
      };
    case 'confirmed':
    case 'chef_accepted_full':
      return {
        label: 'Confirmed',
        color: customerColors.success.DEFAULT,
        bg: customerColors.success.tint,
        needsAction: false,
      };
    case 'active':
      return {
        label: 'In progress',
        color: customerColors.success.DEFAULT,
        bg: customerColors.success.tint,
        needsAction: false,
      };
    case 'completed':
      return {
        label: 'Completed',
        color: customerColors.charcoal.soft,
        bg: customerColors.surface.soft,
        needsAction: false,
      };
    case 'cancelled':
      return {
        label: 'Cancelled',
        color: customerColors.destructive.DEFAULT,
        bg: customerColors.destructive.tint,
        needsAction: false,
      };
    case 'expired':
      return {
        label: 'Expired',
        color: customerColors.destructive.DEFAULT,
        bg: customerColors.destructive.tint,
        needsAction: false,
      };
    default:
      return {
        label: status,
        color: customerColors.charcoal.soft,
        bg: customerColors.surface.soft,
        needsAction: false,
      };
  }
}

// Per-day status presentation (#50). `cooking` flags the live "being prepared"
// state that drives the animated cooking indicator the customer sees.
export interface DayStatusMeta extends StatusMeta {
  cooking: boolean;
}

export function mealPlanDayStatusMeta(status: string): DayStatusMeta {
  const base = { needsAction: false, cooking: false };
  switch (status) {
    case 'requested':
    case 'accepted':
    case 'confirmed':
      return { ...base, label: 'Scheduled', color: customerColors.charcoal.soft, bg: customerColors.surface.soft };
    case 'prepared':
      // The chef is cooking this dish right now — animate it.
      return { ...base, cooking: true, label: 'Being prepared', color: customerColors.coral.pressed, bg: customerColors.coral.tint };
    case 'delivered':
      return { ...base, label: 'Delivered', color: customerColors.success.DEFAULT, bg: customerColors.success.tint };
    case 'declined':
    case 'skipped':
    case 'cancelled':
    case 'refunded':
      return { ...base, label: status.charAt(0).toUpperCase() + status.slice(1), color: customerColors.charcoal.soft, bg: customerColors.surface.soft };
    default:
      return { ...base, label: status, color: customerColors.charcoal.soft, bg: customerColors.surface.soft };
  }
}

export function formatDateRange(start?: string, end?: string): string {
  if (!start || !end) return '';
  const fmt = (iso: string) =>
    new Date(iso).toLocaleDateString(undefined, {
      day: 'numeric',
      month: 'short',
    });
  return `${fmt(start)} – ${fmt(end)}`;
}
