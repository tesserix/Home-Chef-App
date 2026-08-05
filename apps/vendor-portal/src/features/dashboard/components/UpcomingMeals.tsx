import { Link } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { CalendarClock, ChevronRight } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';

// Next 24 hours on the chef dashboard.
//
// The dashboard only ever read /chef/orders, so a tiffin plan day was invisible
// until its order locked — 12h before service. A week booked in advance showed
// nothing, and a chef asking "what's coming?" got an empty screen. This reads
// the plan days directly, so a committed meal appears as soon as the plan is
// confirmed.

interface UpcomingMeal {
  dayId: string;
  planNumber: string;
  startsAt: string;
  slot: string;
  variant: string;
  dishName: string;
  status: string;
  customerName: string;
  /** Empty until the day's order locks — the signal it's now live in the kitchen. */
  orderNumber?: string;
}

interface UpcomingResponse {
  hours: number;
  total: number;
  lunch: number;
  dinner: number;
  meals: UpcomingMeal[];
}

/** "in 3h" / "in 40m" — a chef reads time-to-cook, not a clock face. */
function countdown(iso: string): string {
  const mins = Math.round((new Date(iso).getTime() - Date.now()) / 60000);
  if (mins <= 0) return 'now';
  if (mins < 60) return `in ${mins}m`;
  const h = Math.floor(mins / 60);
  return h < 24 ? `in ${h}h` : `in ${Math.floor(h / 24)}d`;
}

function clockTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-IN', { hour: 'numeric', minute: '2-digit' });
}

export function UpcomingMeals() {
  const { data, isLoading } = useQuery<UpcomingResponse>({
    queryKey: ['chef', 'upcoming', 24],
    queryFn: () => apiClient.get<UpcomingResponse>('/chef/prep/upcoming?hours=24'),
    // A dashboard left open should not go stale on something time-sensitive.
    refetchInterval: 60_000,
    staleTime: 30_000,
  });

  if (isLoading) return null;

  const meals = data?.meals ?? [];

  return (
    <Card className="p-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <CalendarClock className="h-4 w-4 text-ink-soft" aria-hidden="true" />
          <h2 className="font-semibold text-foreground">Next 24 hours</h2>
        </div>
        {meals.length > 0 && (
          <span className="text-sm text-ink-soft">
            {data?.lunch ?? 0} lunch · {data?.dinner ?? 0} dinner
          </span>
        )}
      </div>

      {meals.length === 0 ? (
        <p className="mt-3 text-sm text-ink-soft">
          Nothing to cook in the next 24 hours. Pre-booked plans appear here as soon as they&apos;re
          confirmed.
        </p>
      ) : (
        <ul className="mt-3 divide-y divide-border">
          {meals.map((m) => (
            <li key={m.dayId} className="flex items-center gap-3 py-3">
              <div className="min-w-0 flex-1">
                <p className="font-medium text-foreground">
                  {m.dishName || 'Meal'}{' '}
                  <span className="font-normal text-ink-soft">· {m.customerName}</span>
                </p>
                <p className="text-sm text-ink-soft">
                  <span className="capitalize">{m.slot}</span> · {clockTime(m.startsAt)} ·{' '}
                  {m.orderNumber ? m.orderNumber : m.planNumber}
                </p>
              </div>
              {m.status === 'prepared' ? (
                <Badge variant="success">Prepared</Badge>
              ) : (
                <Badge variant="secondary">{countdown(m.startsAt)}</Badge>
              )}
            </li>
          ))}
        </ul>
      )}

      <Link
        to="/prep"
        className="mt-3 inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
      >
        Open prep list
        <ChevronRight className="h-4 w-4" aria-hidden="true" />
      </Link>
    </Card>
  );
}

export default UpcomingMeals;
