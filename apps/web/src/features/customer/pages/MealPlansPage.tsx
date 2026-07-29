import { Link } from 'react-router-dom';
import { CalendarDays, Loader2 } from 'lucide-react';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Button } from '@/shared/components/ui';
import { useMealPlans, type MealPlan, type MealPlanStatus } from '@/features/customer/hooks/useMealPlans';

// My meal plans — the web twin of the mobile Plans tab.
//
// Distinct from /subscriptions: a plan is a one-off pre-booked week paid as an
// advance, a subscription is a recurring daily tiffin. They were conflated in the
// nav, so plans had no home on web at all.

const STATUS_LABEL: Record<MealPlanStatus, string> = {
  pending_chef: 'Waiting for chef',
  chef_accepted_full: 'Chef accepted',
  chef_modified: 'Chef adjusted',
  awaiting_customer: 'Your approval needed',
  confirmed: 'Confirmed',
  active: 'In progress',
  completed: 'Completed',
  cancelled: 'Cancelled',
  expired: 'Expired',
};

/** Only the states where the customer is the blocker get the accent treatment. */
function statusClass(status: MealPlanStatus): string {
  if (status === 'awaiting_customer') return 'bg-herb/10 text-herb';
  if (status === 'cancelled' || status === 'expired') return 'bg-mist text-ink-muted';
  if (status === 'completed') return 'bg-mist text-ink-soft';
  return 'bg-mist text-ink-soft';
}

function dateRange(plan: MealPlan): string {
  const fmt = (iso: string) =>
    new Date(iso).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
  return `${fmt(plan.startDate)} – ${fmt(plan.endDate)}`;
}

export default function MealPlansPage() {
  const fp = useFormatPrice();
  const { data: plans = [], isLoading } = useMealPlans();

  if (isLoading) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <h1 className="text-2xl font-semibold tracking-tight text-ink">My meal plans</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Weeks you&apos;ve pre-booked from a chef&apos;s published menu.
      </p>

      {plans.length === 0 ? (
        <div className="mt-10 flex flex-col items-center gap-3 text-center">
          <CalendarDays className="h-8 w-8 text-ink-muted" aria-hidden="true" />
          <p className="font-medium text-ink">No meal plans yet</p>
          <p className="max-w-sm text-sm text-ink-soft">
            Pick a chef and plan a week of lunches and dinners — you pay once, up front, and they
            cook to the schedule.
          </p>
          <Button asChild className="mt-1">
            <Link to="/chefs">Browse chefs</Link>
          </Button>
        </div>
      ) : (
        <ul className="mt-6 flex flex-col gap-3">
          {plans.map((plan) => {
            const count = plan.days?.length ?? 0;
            return (
              <li key={plan.id}>
                <Link
                  to={`/meal-plans/${plan.id}`}
                  className="block rounded-lg border border-mist bg-bone p-4 transition-colors hover:border-ink-muted"
                >
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-xs uppercase tracking-wide text-ink-muted">
                      {plan.mealPlanNumber}
                    </span>
                    <span
                      className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${statusClass(plan.status)}`}
                    >
                      {STATUS_LABEL[plan.status] ?? plan.status}
                    </span>
                  </div>
                  <p className="mt-1.5 font-semibold text-ink">
                    {plan.chef?.businessName ?? 'Your chef'}
                  </p>
                  <p className="mt-0.5 text-sm text-ink-soft">
                    {dateRange(plan)} · {count} {count === 1 ? 'meal' : 'meals'} ·{' '}
                    <span className="tabular-nums">{fp(plan.total)}</span>
                  </p>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
