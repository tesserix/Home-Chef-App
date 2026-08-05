import { useState } from 'react';
import { Link } from 'react-router';
import { toast } from 'sonner';
import { format } from 'date-fns';
import { ChevronRight, LifeBuoy, Plus } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import {
  CHEF_TICKET_CATEGORIES,
  TICKET_STATUS_META,
  useCreateTicket,
  useMyTickets,
  type TicketCategory,
} from '../hooks/useSupport';

// Help & support — the web twin of apps/mobile-vendor/app/support/index.tsx
// and support/new.tsx, combined: the list is short enough that a separate
// "new ticket" route would be a navigation step for no reason.
//
// The portal had no help surface at all. A chef hitting a problem on the web had
// nowhere to report it and no way to read a reply.

export function SupportPage() {
  const { data: tickets = [], isLoading } = useMyTickets();
  const [composing, setComposing] = useState(false);

  // Anything waiting on the chef leads — it is the only state where nothing
  // moves until they act.
  const waiting = tickets.filter((t) => t.status === 'waiting_on_chef');
  const rest = tickets.filter((t) => t.status !== 'waiting_on_chef');
  const ordered = [...waiting, ...rest];

  return (
    <div className="mx-auto max-w-2xl">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">Help &amp; support</h1>
          <p className="mt-1 text-sm text-ink-soft">
            Report an issue or request a feature. We reply here, and you&apos;ll get a
            notification.
          </p>
        </div>
        {!composing && (
          <Button size="sm" onClick={() => setComposing(true)}>
            <Plus className="mr-1 h-4 w-4" aria-hidden="true" />
            New request
          </Button>
        )}
      </div>

      {composing && <NewTicketForm onDone={() => setComposing(false)} />}

      {isLoading ? (
        <Skeleton className="mt-6 h-32 w-full" />
      ) : ordered.length === 0 ? (
        !composing && (
          <Card className="mt-6 p-6 text-center">
            <LifeBuoy className="mx-auto h-8 w-8 text-ink-muted" aria-hidden="true" />
            <p className="mt-2 font-semibold text-foreground">No requests yet</p>
            <p className="mt-1 text-sm text-ink-soft">
              If something goes wrong or you need a hand, start a request and we&apos;ll pick it up.
            </p>
          </Card>
        )
      ) : (
        <div className="mt-6 flex flex-col gap-3">
          {ordered.map((t) => {
            const meta = TICKET_STATUS_META[t.status] ?? TICKET_STATUS_META.open;
            return (
              <Link key={t.id} to={`/support/${t.id}`} className="block">
                <Card className="flex items-center gap-3 p-4 transition-colors hover:bg-paper">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-medium text-foreground">{t.subject}</span>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </div>
                    <p className="text-xs text-ink-muted tabular-nums">
                      {t.ticketNumber} · {format(new Date(t.createdAt), 'd MMM yyyy')}
                    </p>
                  </div>
                  <ChevronRight className="h-5 w-5 shrink-0 text-ink-muted" aria-hidden="true" />
                </Card>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

function NewTicketForm({ onDone }: { onDone: () => void }) {
  const create = useCreateTicket();
  const [category, setCategory] = useState<TicketCategory>('order_issue');
  const [subject, setSubject] = useState('');
  const [description, setDescription] = useState('');

  function submit() {
    if (!subject.trim() || !description.trim()) {
      toast.error('Add a subject and describe what happened.');
      return;
    }
    create.mutate(
      { category, subject: subject.trim(), description: description.trim() },
      {
        onSuccess: () => {
          toast.success('Request sent — we’ll reply here.');
          onDone();
        },
        onError: () => toast.error('Could not send that request. Please try again.'),
      },
    );
  }

  return (
    <Card className="mt-6 p-5">
      <h2 className="font-semibold text-foreground">New request</h2>

      <label htmlFor="ticket-category" className="mt-4 block text-sm font-medium text-ink-soft">
        What is it about?
      </label>
      <select
        id="ticket-category"
        value={category}
        onChange={(e) => setCategory(e.target.value as TicketCategory)}
        className="input-base mt-1"
      >
        {CHEF_TICKET_CATEGORIES.map((c) => (
          <option key={c.value} value={c.value}>
            {c.label}
          </option>
        ))}
      </select>

      <label htmlFor="ticket-subject" className="mt-4 block text-sm font-medium text-ink-soft">
        Subject
      </label>
      <input
        id="ticket-subject"
        value={subject}
        onChange={(e) => setSubject(e.target.value)}
        placeholder="Short summary"
        className="input-base mt-1"
      />

      <label htmlFor="ticket-body" className="mt-4 block text-sm font-medium text-ink-soft">
        What happened?
      </label>
      <textarea
        id="ticket-body"
        rows={5}
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder="Include the order number if it's about a specific order — it's the fastest way for us to find it."
        className="input-base mt-1"
      />

      <div className="mt-4 flex gap-2">
        <Button onClick={submit} isLoading={create.isPending}>
          Send request
        </Button>
        <Button variant="ghost" onClick={onDone} disabled={create.isPending}>
          Cancel
        </Button>
      </div>
    </Card>
  );
}

export default SupportPage;
