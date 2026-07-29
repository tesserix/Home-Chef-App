import { Link } from 'react-router-dom';
import { CalendarDays, ChefHat, Inbox, RotateCcw, UtensilsCrossed, ChevronRight } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import { useRefundDecisions } from '../hooks/useRefundDecisions';
import { usePendingPlanRequests } from '../hooks/usePlanRequests';
import { PlanRequestRow } from './PlanRequestPage';

// Tiffin plans hub — the web twin of apps/mobile-vendor/app/meal-plans/index.tsx.
//
// Web already had the weekly menu and the prep list, but scattered as unrelated
// sidebar entries with nothing naming the thing they belong to. Mobile presents
// them as one place a chef goes to run their tiffin service, which is how chefs
// actually think about it — and it is the only surface that shows a pending
// refund request needing a decision.

const CARDS = [
  {
    to: '/weekly-menu',
    icon: UtensilsCrossed,
    title: 'Weekly menu',
    detail: 'Set the dishes customers can pre-book, per day',
  },
  {
    to: '/daily-menu',
    icon: UtensilsCrossed,
    title: 'Daily menu',
    detail: 'Different dishes each day + a combo/thali per day',
  },
  {
    to: '/prep',
    icon: ChefHat,
    title: "Tomorrow's prep",
    detail: 'What you owe tomorrow, by dish — with a packing list',
  },
  {
    to: '/refund-requests',
    icon: RotateCcw,
    title: 'Refund requests',
    detail: 'Late skips/cancels — you decide how much to refund',
    /** Only this card can carry a count; the others are always-available tools. */
    badge: 'refunds' as const,
  },
];

export function TiffinPlansPage() {
  // Surfaced as a count on the card because it is the one entry here that is a
  // QUEUE — a customer is waiting on the chef's answer, and money is held until
  // they give it.
  const { data: pending = [] } = useRefundDecisions();
  // The other real queue: a customer has pre-booked and is waiting on the chef's
  // yes/no. Web had no surface for it at all, so these requests silently expired.
  const { data: requests } = usePendingPlanRequests();

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Tiffin plans</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Everything for your subscription meals — what you serve, what you owe tomorrow, and
        anything waiting on your decision.
      </p>

      <div className="mt-6 flex flex-col gap-3">
        {CARDS.map((c) => {
          const Icon = c.icon;
          const count = c.badge === 'refunds' ? pending.length : 0;
          return (
            <Link key={c.to} to={c.to} className="block">
              <Card className="flex items-center gap-4 p-4 transition-colors hover:bg-paper">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-mist">
                  <Icon className="h-5 w-5 text-ink-soft" aria-hidden="true" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="flex items-center gap-2">
                    <span className="font-medium text-foreground">{c.title}</span>
                    {count > 0 && <Badge variant="warning">{count}</Badge>}
                  </span>
                  <span className="block text-sm text-ink-soft">{c.detail}</span>
                </span>
                <ChevronRight className="h-5 w-5 shrink-0 text-ink-muted" aria-hidden="true" />
              </Card>
            </Link>
          );
        })}
      </div>

      <section className="mt-8">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
          Pending requests
        </h2>
        {requests.length === 0 ? (
          <Card className="mt-3 flex flex-col items-center gap-2 p-8 text-center">
            <Inbox className="h-7 w-7 text-ink-muted" aria-hidden="true" />
            <p className="font-medium text-foreground">No pending requests</p>
            <p className="text-sm text-ink-soft">
              When a customer pre-books a plan, it appears here for you to accept or adjust.
            </p>
          </Card>
        ) : (
          <div className="mt-3 flex flex-col gap-3">
            {requests.map((p) => (
              <PlanRequestRow key={p.id} plan={p} />
            ))}
          </div>
        )}
      </section>

      <p className="mt-6 flex items-center gap-2 text-xs text-ink-muted">
        <CalendarDays className="h-3.5 w-3.5" aria-hidden="true" />
        Customers pre-book from your published menus; accepted plans appear in your orders.
      </p>
    </div>
  );
}

export default TiffinPlansPage;
