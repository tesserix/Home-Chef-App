import { useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import { CalendarDays, Users } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { formatCurrency } from '@/shared/utils/format';
import {
  useAvailableCateringRequests,
  useCateringBookings,
  useMyCateringQuotes,
  useSubmitCateringQuote,
  type CateringRequest,
} from '../hooks/useCatering';

// Chef catering — the web twin of apps/mobile-vendor/app/catering.tsx.
//
// Customers post an event brief, chefs quote, and an accepted quote becomes a
// booking. Web had none of this, so catering work only reached chefs who
// happened to be using the phone app.

type Tab = 'open' | 'quotes' | 'bookings';

const TABS: { id: Tab; label: string }[] = [
  { id: 'open', label: 'Open requests' },
  { id: 'quotes', label: 'My quotes' },
  { id: 'bookings', label: 'Bookings' },
];

export function CateringPage() {
  const [tab, setTab] = useState<Tab>('open');
  const requests = useAvailableCateringRequests();
  const quotes = useMyCateringQuotes();
  const bookings = useCateringBookings();

  const active =
    tab === 'open' ? requests : tab === 'quotes' ? quotes : bookings;

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Catering</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Event briefs customers have posted. Quote for the ones you can cook — an accepted quote
        becomes a booking.
      </p>

      <div className="mt-6 flex gap-2" role="tablist">
        {TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={tab === t.id}
            onClick={() => setTab(t.id)}
            className={`min-h-9 rounded-lg border px-3 py-1.5 text-sm transition-colors ${
              tab === t.id
                ? 'border-herb bg-herb-tint font-medium text-herb'
                : 'border-mist text-ink-soft hover:bg-paper'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {active.isLoading ? (
        <Skeleton className="mt-6 h-32 w-full" />
      ) : tab === 'open' ? (
        (requests.data ?? []).length === 0 ? (
          <Empty text="No open catering requests right now. New briefs appear here." />
        ) : (
          <div className="mt-6 flex flex-col gap-4">
            {(requests.data ?? []).map((r) => (
              <RequestCard key={r.id} req={r} />
            ))}
          </div>
        )
      ) : (
        (() => {
          const rows = (tab === 'quotes' ? quotes.data : bookings.data) ?? [];
          if (rows.length === 0) {
            return (
              <Empty
                text={
                  tab === 'quotes'
                    ? "You haven't quoted for anything yet."
                    : 'No confirmed bookings yet. Accepted quotes appear here.'
                }
              />
            );
          }
          return (
            <div className="mt-6 flex flex-col gap-3">
              {rows.map((q) => (
                <Card key={q.id} className="p-4">
                  <div className="flex items-start justify-between gap-3">
                    <p className="min-w-0 flex-1 text-sm text-foreground">{q.proposedMenu}</p>
                    <Badge variant={q.status === 'accepted' ? 'success' : 'warning'}>
                      {q.status}
                    </Badge>
                  </div>
                  <p className="mt-2 text-sm text-ink-soft tabular-nums">
                    {formatCurrency(q.pricePerPerson)}/head · {formatCurrency(q.totalPrice)} total
                  </p>
                </Card>
              ))}
            </div>
          );
        })()
      )}
    </div>
  );
}

function Empty({ text }: { text: string }) {
  return (
    <Card className="mt-6 p-6 text-center">
      <p className="text-sm text-ink-soft">{text}</p>
    </Card>
  );
}

function RequestCard({ req }: { req: CateringRequest }) {
  const submit = useSubmitCateringQuote();
  const [open, setOpen] = useState(false);
  const [menu, setMenu] = useState('');
  const [perHead, setPerHead] = useState('');

  // Total is derived from per-head × guests rather than asked for separately:
  // the server requires both, and two free-text money fields that must agree is
  // an invitation to quote one number and charge another.
  const total = (Number(perHead) || 0) * req.guestCount;

  function send() {
    if (!menu.trim() || !Number(perHead)) {
      toast.error('Describe the menu and give a price per head.');
      return;
    }
    submit.mutate(
      {
        requestId: req.id,
        proposedMenu: menu.trim(),
        pricePerPerson: Number(perHead),
        totalPrice: total,
      },
      {
        onSuccess: () => {
          toast.success('Quote sent.');
          setOpen(false);
        },
        onError: (err) =>
          toast.error(
            err?.httpStatus === 409
              ? "You've already quoted for this request."
              : 'Could not send that quote. Please try again.',
          ),
      },
    );
  }

  return (
    <Card className="p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-medium capitalize text-foreground">{req.eventType}</p>
          <p className="flex flex-wrap items-center gap-x-3 text-xs text-ink-muted tabular-nums">
            <span className="inline-flex items-center gap-1">
              <CalendarDays className="h-3.5 w-3.5" aria-hidden="true" />
              {format(new Date(req.eventDate), 'd MMM yyyy')}
              {req.eventTime ? ` · ${req.eventTime}` : ''}
            </span>
            <span className="inline-flex items-center gap-1">
              <Users className="h-3.5 w-3.5" aria-hidden="true" />
              {req.guestCount} guests
            </span>
          </p>
        </div>
        {req.budget ? (
          <p className="shrink-0 text-sm text-ink-soft tabular-nums">
            Budget {formatCurrency(req.budget)}
          </p>
        ) : null}
      </div>

      {req.description && <p className="mt-2 text-sm text-ink-soft">{req.description}</p>}
      {req.menuStyle && (
        <p className="mt-1 text-xs text-ink-muted capitalize">Style: {req.menuStyle}</p>
      )}
      {req.quoteDeadline && (
        <p className="mt-1 text-xs text-amber tabular-nums">
          Quotes close {format(new Date(req.quoteDeadline), 'd MMM yyyy')}
        </p>
      )}

      {!open ? (
        <Button size="sm" className="mt-3" onClick={() => setOpen(true)}>
          Quote for this
        </Button>
      ) : (
        <div className="mt-3 rounded-lg border border-mist p-3">
          <label htmlFor={`menu-${req.id}`} className="block text-sm font-medium text-ink-soft">
            What you&apos;d serve
          </label>
          <textarea
            id={`menu-${req.id}`}
            rows={3}
            value={menu}
            onChange={(e) => setMenu(e.target.value)}
            placeholder="Courses, dishes, anything the customer should know."
            className="input-base mt-1"
          />
          <label
            htmlFor={`price-${req.id}`}
            className="mt-3 block text-sm font-medium text-ink-soft"
          >
            Price per head (₹)
          </label>
          <input
            id={`price-${req.id}`}
            type="number"
            min={0}
            inputMode="numeric"
            value={perHead}
            onChange={(e) => setPerHead(e.target.value)}
            className="input-base mt-1 w-32 tabular-nums"
          />
          <p className="mt-2 text-sm text-ink-soft tabular-nums">
            Total for {req.guestCount} guests:{' '}
            <span className="font-semibold text-foreground">{formatCurrency(total)}</span>
          </p>
          <div className="mt-3 flex gap-2">
            <Button size="sm" onClick={send} isLoading={submit.isPending}>
              Send quote
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}

export default CateringPage;
