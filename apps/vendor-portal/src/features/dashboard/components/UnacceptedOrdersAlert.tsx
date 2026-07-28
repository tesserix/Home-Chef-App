import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { BellRing, ChevronRight } from 'lucide-react';


// UnacceptedOrdersAlert — the chef's "someone is waiting on you" banner.
//
// A paid order sitting at `pending` is the one state in this app where the
// customer has already been charged and is waiting on a human. It was
// previously indistinguishable from any other number on the dashboard: a small
// count in a sidebar the chef had to go looking for.
//
// So this sits at the top, above the stats, and is the only thing on the page
// that moves. The motion is opacity-only on a single dot at a 1.8s cycle —
// enough to catch a chef glancing across a kitchen, restrained enough not to
// become the flashing-red-banner every operator learns to ignore. The global
// prefers-reduced-motion rule in globals.css strips it entirely, leaving a
// perfectly readable static banner.
//
// Amber, not persimmon or paprika: this is "needs you", not "primary action"
// and not "something is broken".

/** Only what this banner reads. Kept structural rather than importing a full
 *  Order type so it works with the dashboard's narrower local shape and the
 *  shared one alike, and cannot break when either gains a field. */
export interface WaitingOrder {
  id: string;
  orderNumber: string;
  createdAt: string;
}

interface UnacceptedOrdersAlertProps {
  orders: WaitingOrder[];
}

/** Whole minutes since an ISO timestamp, floored at 0 so clock skew between the
 *  server and the chef's laptop can never render "waiting -3m". */
function minutesSince(iso: string): number {
  const ms = Date.now() - new Date(iso).getTime();
  return Math.max(0, Math.floor(ms / 60_000));
}

function waitLabel(mins: number): string {
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins} min`;
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  return m === 0 ? `${h}h` : `${h}h ${m}m`;
}

export function UnacceptedOrdersAlert({ orders }: UnacceptedOrdersAlertProps) {
  // Re-render once a minute so "waiting 3 min" doesn't sit frozen on a
  // dashboard the chef leaves open all service.
  const [, setTick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 60_000);
    return () => clearInterval(id);
  }, []);

  if (orders.length === 0) return null;

  // Oldest first — the honest number is how long the LONGEST-waiting customer
  // has been waiting, not the average or the newest.
  const oldest = orders.reduce((a, b) =>
    new Date(a.createdAt).getTime() <= new Date(b.createdAt).getTime() ? a : b
  );
  const waited = minutesSince(oldest.createdAt);
  const many = orders.length > 1;

  return (
    <div
      // Announced politely rather than assertively: it must reach a screen
      // reader without interrupting whatever the chef is currently doing.
      role="status"
      aria-live="polite"
      className="flex flex-col gap-3 rounded-xl border border-amber/40 bg-amber-tint p-4 sm:flex-row sm:items-center sm:justify-between"
    >
      <div className="flex items-start gap-3">
        <span className="relative mt-0.5 flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-amber/20">
          <BellRing className="h-4 w-4 text-amber" aria-hidden="true" />
          <span
            aria-hidden="true"
            className="animate-attention absolute -right-0.5 -top-0.5 h-2.5 w-2.5 rounded-full bg-amber ring-2 ring-bone"
          />
        </span>
        <div>
          <p className="text-sm font-semibold text-ink">
            {many ? `${orders.length} orders waiting to be accepted` : 'New order waiting to be accepted'}
          </p>
          <p className="mt-0.5 text-sm text-ink-soft">
            {many ? 'Oldest' : oldest.orderNumber} · waiting{' '}
            <span className="font-medium tabular-nums text-ink">{waitLabel(waited)}</span>
            {' · '}
            The customer has already paid.
          </p>
        </div>
      </div>

      <Link
        to="/orders"
        className="inline-flex flex-shrink-0 items-center justify-center gap-1 rounded-lg bg-ink px-4 py-2.5 text-sm font-semibold text-paper transition-colors hover:bg-ink-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ink focus-visible:ring-offset-2"
      >
        {many ? 'Review orders' : 'Review order'}
        <ChevronRight className="h-4 w-4" aria-hidden="true" />
      </Link>
    </div>
  );
}
