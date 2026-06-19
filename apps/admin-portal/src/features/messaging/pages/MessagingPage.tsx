import { toast } from 'sonner';
import { MessageSquare, Loader2, ArrowRight, Ban, ShieldAlert } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';
import {
  useMediationInbox,
  useRelayMessage,
  useBlockMessage,
  type MediatedMessage,
} from '@/features/messaging/hooks/useMessaging';

// Admin mediation inbox (#53). The relay queue: customer↔chef messages are held
// pending here until an admin relays them to the recipient (or blocks them).
// This is the human-in-the-loop that makes the chat fully mediated — without it,
// messages never reach the other party.

const ROLE_LABEL: Record<string, string> = { customer: 'Customer', chef: 'Chef', admin: 'Admin' };

export default function MessagingPage() {
  const { data: inbox = [], isLoading } = useMediationInbox();
  const relay = useRelayMessage();
  const block = useBlockMessage();

  const act = (m: MediatedMessage, kind: 'relay' | 'block') => {
    const mut = kind === 'relay' ? relay : block;
    mut.mutate(m.id, {
      onSuccess: () => toast.success(kind === 'relay' ? 'Relayed to recipient' : 'Message blocked'),
      onError: () => toast.error('Action failed'),
    });
  };

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Message Mediation</h1>
        <p className="page-description">
          Customer↔chef messages route through here. Relay a message to deliver it to the recipient,
          or block it (e.g. abuse or shared contact details). PII is auto-redacted before it reaches you.
        </p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center rounded-xl border border-border bg-card py-10 text-muted-foreground">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading inbox…
        </div>
      ) : inbox.length === 0 ? (
        <div className="rounded-xl border border-border bg-card p-8 text-center text-muted-foreground">
          Inbox clear — no messages awaiting relay.
        </div>
      ) : (
        <div className="space-y-3">
          {inbox.map((m) => {
            const pending = relay.isPending || block.isPending;
            return (
              <div key={m.id} className="rounded-xl border border-border bg-card p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span className="rounded-full bg-muted px-2 py-0.5 font-medium text-foreground">
                        {ROLE_LABEL[m.senderRole]} <ArrowRight className="inline h-3 w-3" /> {ROLE_LABEL[m.recipientRole]}
                      </span>
                      <span>order {m.orderId.slice(0, 8)}</span>
                      <span>· {new Date(m.createdAt).toLocaleString()}</span>
                      {m.piiDetected && (
                        <span className="inline-flex items-center gap-1 rounded-full bg-warning/10 px-2 py-0.5 font-medium text-warning">
                          <ShieldAlert className="h-3 w-3" /> PII redacted
                        </span>
                      )}
                    </div>
                    <p className="mt-2 text-sm text-foreground">{m.content}</p>
                  </div>
                  <div className="flex flex-shrink-0 items-center gap-2">
                    <Button variant="primary" size="sm" isLoading={pending} onClick={() => act(m, 'relay')}>
                      <MessageSquare className="h-4 w-4" /> Relay
                    </Button>
                    <Button variant="ghost" size="sm" disabled={pending} onClick={() => act(m, 'block')}>
                      <Ban className="h-4 w-4" /> Block
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
