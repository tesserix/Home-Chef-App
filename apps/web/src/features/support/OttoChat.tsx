import { useEffect, useRef, useState } from 'react';
import { MessageCircle, X, Send, AlertCircle, Loader2, Sparkles } from 'lucide-react';

// HomeChef support chat widget — minimal client for the shared Otto chat
// service running in support-platform. The /api/otto/* path is proxied
// server-side by nginx (see apps/web/nginx.conf) to
// support-platform-otto.support-platform.svc.cluster.local:8089 with the
// X-Internal-Auth + X-Tenant-Id headers injected. Browser sees only
// same-origin /api/otto.

interface Message {
  id: string;
  sender_type: 'customer' | 'staff' | 'assistant' | 'system';
  sender_name?: string;
  body: string;
  created_at: string;
}

interface Conversation {
  id: string;
  case_id?: string;
}

type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; error: string };

async function api<T>(url: string, init?: RequestInit): Promise<ApiResult<T>> {
  try {
    const r = await fetch(url, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...init?.headers },
    });
    if (!r.ok) {
      return { ok: false, status: r.status, error: `Request failed (${r.status})` };
    }
    return { ok: true, data: (await r.json()) as T };
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Network error';
    return { ok: false, status: 0, error: msg };
  }
}

export function OttoChat() {
  const [open, setOpen] = useState(false);
  const [conv, setConv] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [connectError, setConnectError] = useState<string | null>(null);
  const [retryToken, setRetryToken] = useState(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  // Create a conversation when the chat opens. Re-runs on retry via retryToken.
  useEffect(() => {
    if (!open || conv) return;
    let cancelled = false;
    setConnectError(null);
    (async () => {
      const result = await api<Conversation>('/api/otto/conversations', {
        method: 'POST',
        body: JSON.stringify({
          intake: { reason: 'product_question', status: 'New question' },
        }),
      });
      if (cancelled) return;
      if (result.ok && result.data.id) {
        setConv(result.data);
      } else {
        setConnectError(
          result.ok ? 'Could not start a conversation.' : result.error
        );
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, conv, retryToken]);

  // Poll for messages every 3s while the chat is open.
  useEffect(() => {
    if (!conv?.id || !open) return;
    let cancelled = false;
    const tick = async () => {
      const r = await api<{ items?: Message[] }>(
        `/api/otto/conversations/${conv.id}/messages`
      );
      if (!cancelled && r.ok && r.data.items) setMessages(r.data.items);
    };
    tick();
    const id = setInterval(tick, 3000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [conv?.id, open]);

  // Auto-scroll on new messages. Reduced-motion users get an instant jump.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    el.scrollTo({ top: el.scrollHeight, behavior: reduced ? 'auto' : 'smooth' });
  }, [messages.length]);

  // Restore focus to the trigger when the panel closes — keeps keyboard
  // users oriented.
  useEffect(() => {
    if (!open) triggerRef.current?.focus();
  }, [open]);

  const send = async () => {
    const text = input.trim();
    if (!text || !conv?.id || sending) return;
    setSending(true);
    const optimisticId = 'local-' + Date.now();
    setInput('');
    setMessages((m) => [
      ...m,
      {
        id: optimisticId,
        sender_type: 'customer',
        body: text,
        created_at: new Date().toISOString(),
      },
    ]);
    const result = await api(`/api/otto/conversations/${conv.id}/messages`, {
      method: 'POST',
      body: JSON.stringify({ body: text }),
    });
    if (!result.ok) {
      // Drop the optimistic message and restore the input so the user can retry.
      setMessages((m) => m.filter((msg) => msg.id !== optimisticId));
      setInput(text);
    }
    setSending(false);
  };

  if (!open) {
    return (
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen(true)}
        aria-label="Open support chat"
        className="fixed bottom-6 right-6 z-50 flex h-14 w-14 items-center justify-center rounded-full bg-herb text-paper shadow-3 transition-transform duration-200 ease-premium hover:bg-herb-soft active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
      >
        <MessageCircle aria-hidden="true" className="h-6 w-6" />
      </button>
    );
  }

  return (
    <div
      role="dialog"
      aria-label="HomeChef support chat"
      className="fixed bottom-6 right-6 z-50 flex h-[520px] w-[360px] max-w-[calc(100vw-3rem)] flex-col overflow-hidden rounded-xl border border-mist bg-bone text-foreground shadow-3"
    >
      <div className="flex items-center justify-between gap-2 border-b border-mist bg-herb px-3 py-2.5 text-paper">
        <div className="flex items-center gap-2 font-semibold">
          <Sparkles aria-hidden="true" className="h-4 w-4" />
          <span>HomeChef support</span>
        </div>
        <button
          type="button"
          onClick={() => setOpen(false)}
          aria-label="Close support chat"
          className="flex h-7 w-7 items-center justify-center rounded-md transition-colors hover:bg-paper/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-paper/40"
        >
          <X aria-hidden="true" className="h-4 w-4" />
        </button>
      </div>

      <div
        ref={scrollRef}
        role="log"
        aria-live="polite"
        aria-label="Conversation messages"
        className="flex flex-1 flex-col gap-2 overflow-y-auto p-4"
      >
        {connectError ? (
          <div className="m-auto flex max-w-[80%] flex-col items-center gap-3 text-center">
            <AlertCircle aria-hidden="true" className="h-8 w-8 text-paprika" />
            <p className="text-sm text-ink-soft">{connectError}</p>
            <button
              type="button"
              onClick={() => setRetryToken((n) => n + 1)}
              className="rounded-md bg-herb px-3 py-1.5 text-sm font-medium text-paper transition-colors hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
            >
              Try again
            </button>
          </div>
        ) : !conv ? (
          <div className="m-auto flex flex-col items-center gap-3 text-center text-ink-muted">
            <Loader2 aria-hidden="true" className="h-5 w-5 animate-spin" />
            <span className="text-sm">Connecting…</span>
            <span className="sr-only">Loading support chat</span>
          </div>
        ) : messages.length === 0 ? (
          <p className="m-auto max-w-[80%] text-center text-sm text-ink-muted">
            Ask anything — order status, delivery, refunds, chef availability.
          </p>
        ) : (
          messages.map((m) => {
            const mine = m.sender_type === 'customer';
            const ai = m.sender_type === 'assistant';
            return (
              <div
                key={m.id}
                className={`max-w-[82%] rounded-lg px-3 py-2 text-sm leading-snug ${
                  mine
                    ? 'self-end bg-herb text-paper'
                    : ai
                    ? 'self-start bg-paper text-foreground'
                    : 'self-start bg-mist text-foreground'
                }`}
              >
                {!mine && (
                  <div className="mb-1 text-[11px] text-ink-muted">
                    {ai ? 'Otto · AI assistant' : m.sender_name || 'Support'}
                  </div>
                )}
                {m.body}
              </div>
            );
          })
        )}
      </div>

      <div className="flex gap-2 border-t border-mist bg-bone p-3">
        <label htmlFor="otto-input" className="sr-only">
          Message
        </label>
        <input
          id="otto-input"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              send();
            }
          }}
          placeholder={conv ? 'Type a message…' : 'Connecting…'}
          disabled={!conv || sending}
          maxLength={2000}
          className="flex-1 rounded-md border border-mist bg-paper px-3 py-2 text-sm text-foreground placeholder:text-ink-muted focus-visible:border-herb focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/30 disabled:opacity-60"
        />
        <button
          type="button"
          onClick={send}
          disabled={!conv || sending || !input.trim()}
          aria-label="Send message"
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-herb text-paper transition-colors hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:bg-mist-strong disabled:text-ink-muted"
        >
          {sending ? (
            <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />
          ) : (
            <Send aria-hidden="true" className="h-4 w-4" />
          )}
        </button>
      </div>
    </div>
  );
}
