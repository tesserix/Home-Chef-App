import { useEffect, useRef, useState } from 'react';

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

async function api<T>(url: string, init?: RequestInit): Promise<T | null> {
  try {
    const r = await fetch(url, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...init?.headers },
    });
    if (!r.ok) return null;
    return (await r.json()) as T;
  } catch {
    return null;
  }
}

export function OttoChat() {
  const [open, setOpen] = useState(false);
  const [conv, setConv] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || conv) return;
    let cancelled = false;
    (async () => {
      const created = await api<Conversation>('/api/otto/conversations', {
        method: 'POST',
        body: JSON.stringify({
          intake: { reason: 'product_question', status: 'New question' },
        }),
      });
      if (!cancelled && created?.id) setConv(created);
    })();
    return () => {
      cancelled = true;
    };
  }, [open, conv]);

  useEffect(() => {
    if (!conv?.id || !open) return;
    let cancelled = false;
    const tick = async () => {
      const r = await api<{ items?: Message[] }>(
        `/api/otto/conversations/${conv.id}/messages`
      );
      if (!cancelled && r?.items) setMessages(r.items);
    };
    tick();
    const id = setInterval(tick, 3000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [conv?.id, open]);

  useEffect(() => {
    scrollRef.current?.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: 'smooth',
    });
  }, [messages.length]);

  const send = async () => {
    const text = input.trim();
    if (!text || !conv?.id || sending) return;
    setSending(true);
    setInput('');
    setMessages((m) => [
      ...m,
      {
        id: 'local-' + Date.now(),
        sender_type: 'customer',
        body: text,
        created_at: new Date().toISOString(),
      },
    ]);
    await api(`/api/otto/conversations/${conv.id}/messages`, {
      method: 'POST',
      body: JSON.stringify({ body: text }),
    });
    setSending(false);
  };

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        aria-label="Open support chat"
        className="fixed bottom-6 right-6 z-50 flex h-14 w-14 items-center justify-center rounded-full text-2xl text-white shadow-lg"
        style={{ background: 'linear-gradient(135deg, #16a34a 0%, #15803d 100%)' }}
      >
        💬
      </button>
    );
  }

  return (
    <div
      className="fixed bottom-6 right-6 z-50 flex flex-col overflow-hidden rounded-xl shadow-2xl"
      style={{
        width: 360,
        height: 520,
        background: '#0f172a',
        color: '#f1f5f9',
        fontFamily: 'system-ui, sans-serif',
      }}
    >
      <div
        className="flex items-center justify-between p-3 font-semibold"
        style={{
          background: 'linear-gradient(135deg, #16a34a 0%, #15803d 100%)',
        }}
      >
        <span>HomeChef support</span>
        <button
          onClick={() => setOpen(false)}
          aria-label="Close"
          className="cursor-pointer border-0 bg-transparent text-2xl leading-none text-white"
        >
          ×
        </button>
      </div>

      <div
        ref={scrollRef}
        className="flex flex-1 flex-col gap-2 overflow-y-auto p-4"
      >
        {messages.length === 0 && (
          <div className="mt-6 text-center text-sm text-slate-400">
            Ask anything — order status, delivery, refunds, chef availability.
          </div>
        )}
        {messages.map((m) => {
          const mine = m.sender_type === 'customer';
          const ai = m.sender_type === 'assistant';
          return (
            <div
              key={m.id}
              className="max-w-[82%] rounded-lg px-3 py-2 text-sm leading-snug"
              style={{
                alignSelf: mine ? 'flex-end' : 'flex-start',
                background: mine
                  ? 'linear-gradient(135deg, #16a34a 0%, #15803d 100%)'
                  : ai
                  ? '#1e293b'
                  : '#334155',
              }}
            >
              {!mine && (
                <div className="mb-1 text-[11px] text-slate-400">
                  {ai ? '🤖 Otto' : m.sender_name || 'Support'}
                </div>
              )}
              {m.body}
            </div>
          );
        })}
      </div>

      <div className="flex gap-2 border-t border-slate-700 p-3">
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && send()}
          placeholder={conv ? 'Type a message…' : 'Connecting…'}
          disabled={!conv || sending}
          className="flex-1 rounded-md border border-slate-700 px-3 py-2 text-sm outline-none"
          style={{ background: '#020617', color: '#f1f5f9' }}
        />
        <button
          onClick={send}
          disabled={!conv || sending || !input.trim()}
          className="rounded-md px-4 py-2 text-sm font-semibold text-white"
          style={{
            background:
              !conv || sending || !input.trim()
                ? '#475569'
                : 'linear-gradient(135deg, #16a34a 0%, #15803d 100%)',
            cursor:
              !conv || sending || !input.trim() ? 'not-allowed' : 'pointer',
          }}
        >
          Send
        </button>
      </div>
    </div>
  );
}
