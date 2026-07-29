import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { toast } from 'sonner';
import { format } from 'date-fns';
import { ArrowLeft } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { useAuth } from '@/app/providers/AuthProvider';
import {
  TICKET_STATUS_META,
  useAddTicketMessage,
  useCloseTicket,
  useTicket,
} from '../hooks/useSupport';

// One support conversation — the web twin of
// apps/mobile-vendor/app/support/[id].tsx.

export function SupportTicketPage() {
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const { data: ticket, isLoading } = useTicket(id);
  const addMessage = useAddTicketMessage(id ?? '');
  const close = useCloseTicket(id ?? '');
  const [reply, setReply] = useState('');

  if (isLoading) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="mx-auto max-w-2xl text-center">
        <p className="font-semibold text-foreground">Request not found</p>
        <Button asChild variant="secondary" className="mt-4">
          <Link to="/support">Back to support</Link>
        </Button>
      </div>
    );
  }

  const meta = TICKET_STATUS_META[ticket.status] ?? TICKET_STATUS_META.open;
  const isClosed = ticket.status === 'closed' || ticket.status === 'resolved';

  function send() {
    if (!reply.trim()) return;
    addMessage.mutate(reply.trim(), {
      onSuccess: () => setReply(''),
      onError: () => toast.error('Could not send that message. Please try again.'),
    });
  }

  return (
    <div className="mx-auto max-w-2xl">
      <Link
        to="/support"
        className="inline-flex items-center gap-1.5 text-sm text-ink-soft hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Support
      </Link>

      <div className="mt-4 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">{ticket.subject}</h1>
          <p className="text-xs text-ink-muted tabular-nums">
            {ticket.ticketNumber} · opened {format(new Date(ticket.createdAt), 'd MMM yyyy')}
          </p>
        </div>
        <Badge variant={meta.variant}>{meta.label}</Badge>
      </div>

      <Card className="mt-4 p-4">
        <p className="whitespace-pre-wrap text-sm text-foreground">{ticket.description}</p>
      </Card>

      {ticket.resolution && (
        <Card className="mt-3 border-herb/30 p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">Resolution</p>
          <p className="mt-1 whitespace-pre-wrap text-sm text-foreground">{ticket.resolution}</p>
        </Card>
      )}

      <div className="mt-4 flex flex-col gap-3">
        {(ticket.messages ?? []).map((m) => {
          // Own messages align right; anything else is the support team. Falls
          // back to left alignment when the sender is unknown, which reads as
          // "from them" — the safer default for a reply the chef must not miss.
          const mine = !!user?.id && m.senderId === user.id;
          return (
            <div key={m.id} className={mine ? 'flex justify-end' : 'flex justify-start'}>
              <div
                className={`max-w-[85%] rounded-xl px-3 py-2 ${
                  mine ? 'bg-herb-tint text-foreground' : 'border border-mist bg-bone'
                }`}
              >
                <p className="whitespace-pre-wrap text-sm text-foreground">{m.content}</p>
                <p className="mt-1 text-[11px] text-ink-muted tabular-nums">
                  {format(new Date(m.createdAt), 'd MMM, HH:mm')}
                </p>
              </div>
            </div>
          );
        })}
      </div>

      {isClosed ? (
        <p className="mt-6 text-sm text-ink-muted">
          This request is {meta.label.toLowerCase()}. Start a new one if you need anything else.
        </p>
      ) : (
        <div className="mt-6">
          <label htmlFor="ticket-reply" className="block text-sm font-medium text-ink-soft">
            Reply
          </label>
          <textarea
            id="ticket-reply"
            rows={3}
            value={reply}
            onChange={(e) => setReply(e.target.value)}
            className="input-base mt-1"
          />
          <div className="mt-3 flex gap-2">
            <Button onClick={send} isLoading={addMessage.isPending} disabled={!reply.trim()}>
              Send
            </Button>
            <Button
              variant="ghost"
              disabled={close.isPending}
              onClick={() =>
                close.mutate(undefined, {
                  onSuccess: () => toast.success('Request closed.'),
                  onError: () => toast.error('Could not close that request.'),
                })
              }
            >
              Close request
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export default SupportTicketPage;
