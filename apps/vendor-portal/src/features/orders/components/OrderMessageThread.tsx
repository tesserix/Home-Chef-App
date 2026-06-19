import { useState } from 'react';
import { toast } from 'sonner';
import { MessageSquare, Send, ChevronDown, ChevronUp, ShieldCheck } from 'lucide-react';
import { useOrderMessages, useSendOrderMessage } from '@/features/orders/hooks/useOrderMessaging';

// Collapsible chef-side message thread for an order (#53). Admin-mediated: the
// chef sees the customer's relayed messages + their own; replies are reviewed by
// support before reaching the customer. Lazy — only fetches once expanded.

export function OrderMessageThread({ orderId }: { orderId: string }) {
  const [open, setOpen] = useState(false);
  const { data: messages = [], isLoading } = useOrderMessages(orderId, open);
  const send = useSendOrderMessage(orderId);
  const [text, setText] = useState('');

  const onSend = () => {
    const content = text.trim();
    if (!content || send.isPending) return;
    send.mutate(content, {
      onSuccess: () => setText(''),
      onError: () => toast.error('Could not send message'),
    });
  };

  return (
    <div className="border-t border-border pt-3">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between text-sm font-medium text-foreground"
      >
        <span className="flex items-center gap-2">
          <MessageSquare className="h-4 w-4 text-muted-foreground" /> Messages
          {messages.length > 0 && <span className="text-xs text-muted-foreground">({messages.length})</span>}
        </span>
        {open ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
      </button>

      {open && (
        <div className="mt-3 space-y-3">
          <p className="flex items-center gap-1.5 rounded-lg bg-secondary/50 px-3 py-2 text-xs text-muted-foreground">
            <ShieldCheck className="h-3.5 w-3.5" /> Reviewed by support before reaching the customer. Don't share contact details.
          </p>

          <div className="max-h-64 space-y-2 overflow-y-auto">
            {isLoading ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : messages.length === 0 ? (
              <p className="text-sm text-muted-foreground">No messages yet.</p>
            ) : (
              messages.map((m) => {
                const mine = m.senderRole === 'chef';
                return (
                  <div key={m.id} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
                    <div className={`max-w-[80%] rounded-2xl px-3 py-2 text-sm ${mine ? 'bg-primary text-primary-foreground' : 'bg-secondary text-foreground'}`}>
                      {!mine && (
                        <span className="mb-0.5 block text-[11px] font-semibold opacity-70">
                          {m.senderRole === 'customer' ? 'Customer (via support)' : 'Support'}
                        </span>
                      )}
                      {m.content}
                      {mine && m.relayStatus === 'pending' && (
                        <span className="mt-0.5 block text-[10px] opacity-70">under review</span>
                      )}
                    </div>
                  </div>
                );
              })
            )}
          </div>

          <div className="flex items-end gap-2">
            <textarea
              className="h-10 max-h-24 flex-1 resize-none rounded-lg border border-input bg-card px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              placeholder="Reply…"
              value={text}
              onChange={(e) => setText(e.target.value)}
            />
            <button
              type="button"
              onClick={onSend}
              disabled={!text.trim() || send.isPending}
              aria-label="Send message"
              className={`flex h-10 w-10 items-center justify-center rounded-full ${text.trim() ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground'}`}
            >
              <Send className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
