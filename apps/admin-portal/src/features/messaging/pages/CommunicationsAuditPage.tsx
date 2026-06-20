import { useState } from 'react';
import {
  MessagesSquare,
  Loader2,
  ShieldAlert,
  Paperclip,
  Download,
  ArrowRight,
  Search,
  X,
} from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';
import {
  useConversations,
  useConversationTranscript,
  type ConversationFilter,
  type ConversationTranscript,
} from '@/features/messaging/hooks/useMessaging';

// Communications audit (#312). Admins (only — the whole /admin surface is
// RequireAdmin-gated) can browse EVERY conversation and read the COMPLETE
// transcript, including pending + blocked messages no participant ever sees,
// plus export a JSON/CSV record for compliance. Read-only: this is the audit
// trail, not the mediation queue (that's the Messaging page).

const ROLE_LABEL: Record<string, string> = { customer: 'Customer', chef: 'Chef', admin: 'Admin' };
const STATUS_STYLE: Record<string, string> = {
  relayed: 'bg-success/10 text-success',
  pending: 'bg-warning/10 text-warning',
  blocked: 'bg-destructive/10 text-destructive',
};
const PAGE_SIZE = 25;
const INPUT =
  'w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring';

function csvCell(s: string): string {
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

function exportTranscript(t: ConversationTranscript, format: 'json' | 'csv'): void {
  let blob: Blob;
  let name: string;
  if (format === 'csv') {
    const header = ['createdAt', 'senderRole', 'recipientRole', 'relayStatus', 'piiDetected', 'content', 'attachment'];
    const rows = t.messages.map((m) => [
      m.createdAt,
      m.senderRole,
      m.recipientRole,
      m.relayStatus,
      String(m.piiDetected),
      m.content,
      m.filename ?? '',
    ]);
    const csv = [header, ...rows].map((r) => r.map(csvCell).join(',')).join('\n');
    blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
    name = `conversation-${t.conversation.id}.csv`;
  } else {
    blob = new Blob([JSON.stringify(t, null, 2)], { type: 'application/json' });
    name = `conversation-${t.conversation.id}.json`;
  }
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

export default function CommunicationsAuditPage() {
  const [draft, setDraft] = useState<ConversationFilter>({});
  const [filter, setFilter] = useState<ConversationFilter>({ limit: PAGE_SIZE, offset: 0 });
  const [selected, setSelected] = useState<string | null>(null);

  const { data: list, isLoading } = useConversations(filter);
  const { data: transcript, isLoading: loadingTranscript } = useConversationTranscript(selected);

  const conversations = list?.data ?? [];
  const total = list?.total ?? 0;
  const offset = filter.offset ?? 0;

  const applyFilters = (): void => {
    setSelected(null);
    setFilter({ ...draft, limit: PAGE_SIZE, offset: 0 });
  };
  const clearFilters = (): void => {
    setDraft({});
    setSelected(null);
    setFilter({ limit: PAGE_SIZE, offset: 0 });
  };

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Communications Audit</h1>
        <p className="page-description">
          Read-only audit of every customer↔chef conversation — full transcripts including pending and
          blocked messages no participant can see. Export any conversation as a JSON or CSV record. Admin-only.
        </p>
      </div>

      {/* Filters */}
      <div className="rounded-xl border border-border bg-card p-4">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <input
            className={INPUT}
            placeholder="Order ID"
            value={draft.orderId ?? ''}
            onChange={(e) => setDraft((d) => ({ ...d, orderId: e.target.value }))}
          />
          <input
            className={INPUT}
            placeholder="Customer ID"
            value={draft.customerId ?? ''}
            onChange={(e) => setDraft((d) => ({ ...d, customerId: e.target.value }))}
          />
          <input
            className={INPUT}
            placeholder="Chef ID"
            value={draft.chefId ?? ''}
            onChange={(e) => setDraft((d) => ({ ...d, chefId: e.target.value }))}
          />
          <select
            className={INPUT}
            value={draft.status ?? ''}
            onChange={(e) => setDraft((d) => ({ ...d, status: e.target.value as ConversationFilter['status'] }))}
          >
            <option value="">Any status</option>
            <option value="open">Open</option>
            <option value="closed">Closed</option>
          </select>
          <input
            type="date"
            className={INPUT}
            value={draft.from ?? ''}
            onChange={(e) => setDraft((d) => ({ ...d, from: e.target.value }))}
          />
          <input
            type="date"
            className={INPUT}
            value={draft.to ?? ''}
            onChange={(e) => setDraft((d) => ({ ...d, to: e.target.value }))}
          />
        </div>
        <div className="mt-3 flex items-center gap-2">
          <Button variant="primary" size="sm" onClick={applyFilters}>
            <Search className="h-4 w-4" /> Apply
          </Button>
          <Button variant="ghost" size="sm" onClick={clearFilters}>
            <X className="h-4 w-4" /> Clear
          </Button>
          <span className="ml-auto text-xs text-muted-foreground">{total} conversation(s)</span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,360px)_1fr]">
        {/* Conversation list */}
        <div className="rounded-xl border border-border bg-card">
          {isLoading ? (
            <div className="flex items-center justify-center py-10 text-muted-foreground">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
            </div>
          ) : conversations.length === 0 ? (
            <div className="p-8 text-center text-sm text-muted-foreground">No conversations match.</div>
          ) : (
            <ul className="divide-y divide-border">
              {conversations.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onClick={() => setSelected(c.id)}
                    className={`flex w-full flex-col items-start gap-1 px-4 py-3 text-left transition-colors hover:bg-muted/50 ${
                      selected === c.id ? 'bg-muted' : ''
                    }`}
                  >
                    <span className="font-mono text-xs text-foreground">order {c.orderId.slice(0, 8)}</span>
                    <span className="text-xs text-muted-foreground">
                      {c.status} · {new Date(c.lastMessageAt ?? c.createdAt).toLocaleString()}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
          {total > PAGE_SIZE && (
            <div className="flex items-center justify-between border-t border-border px-4 py-2 text-xs text-muted-foreground">
              <Button
                variant="ghost"
                size="sm"
                disabled={offset === 0}
                onClick={() => setFilter((f) => ({ ...f, offset: Math.max(0, offset - PAGE_SIZE) }))}
              >
                Prev
              </Button>
              <span>
                {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
              </span>
              <Button
                variant="ghost"
                size="sm"
                disabled={offset + PAGE_SIZE >= total}
                onClick={() => setFilter((f) => ({ ...f, offset: offset + PAGE_SIZE }))}
              >
                Next
              </Button>
            </div>
          )}
        </div>

        {/* Transcript */}
        <div className="rounded-xl border border-border bg-card">
          {!selected ? (
            <div className="flex h-full min-h-[240px] flex-col items-center justify-center gap-2 p-8 text-center text-muted-foreground">
              <MessagesSquare className="h-8 w-8" />
              <p className="text-sm">Select a conversation to read its full transcript.</p>
            </div>
          ) : loadingTranscript || !transcript ? (
            <div className="flex items-center justify-center py-10 text-muted-foreground">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading transcript…
            </div>
          ) : (
            <div>
              <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border p-4">
                <div className="min-w-0">
                  <p className="font-mono text-xs text-muted-foreground">
                    order {transcript.conversation.orderId}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {transcript.messages.length} message(s) · {transcript.conversation.status}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="ghost" size="sm" onClick={() => exportTranscript(transcript, 'json')}>
                    <Download className="h-4 w-4" /> JSON
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => exportTranscript(transcript, 'csv')}>
                    <Download className="h-4 w-4" /> CSV
                  </Button>
                </div>
              </div>
              <div className="space-y-3 p-4">
                {transcript.messages.map((m) => (
                  <div key={m.id} className="rounded-lg border border-border p-3">
                    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                      <span className="rounded-full bg-muted px-2 py-0.5 font-medium text-foreground">
                        {ROLE_LABEL[m.senderRole] ?? m.senderRole} <ArrowRight className="inline h-3 w-3" />{' '}
                        {ROLE_LABEL[m.recipientRole] ?? m.recipientRole}
                      </span>
                      <span className={`rounded-full px-2 py-0.5 font-medium ${STATUS_STYLE[m.relayStatus] ?? ''}`}>
                        {m.relayStatus}
                      </span>
                      <span>{new Date(m.createdAt).toLocaleString()}</span>
                      {m.piiDetected && (
                        <span className="inline-flex items-center gap-1 rounded-full bg-warning/10 px-2 py-0.5 font-medium text-warning">
                          <ShieldAlert className="h-3 w-3" /> PII redacted
                        </span>
                      )}
                    </div>
                    <p className="mt-2 whitespace-pre-wrap text-sm text-foreground">{m.content}</p>
                    {m.attachmentId && (
                      <p className="mt-2 inline-flex items-center gap-1 text-xs text-muted-foreground">
                        <Paperclip className="h-3 w-3" /> {m.filename ?? 'attachment'}
                        {m.contentType ? ` (${m.contentType})` : ''}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
