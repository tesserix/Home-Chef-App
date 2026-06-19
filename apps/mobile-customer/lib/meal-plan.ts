import { customerColors } from '@homechef/mobile-shared/theme';

// Presentation helpers for tiffin meal-plan statuses (#196). Keeps the list +
// detail screens consistent and the status copy in one place.

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
