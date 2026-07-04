import { useState } from 'react';
import { toast } from 'sonner';
import {
  CANCEL_REASONS,
  useAdminCancellations,
  useResolveCancellation,
  type AdminCancellationRequest,
} from '../hooks/useCancellations';

// Admin arbitration queue (#480). Customer disputes + vendor timeouts land here;
// the admin picks the correct tier and the refund is issued (timeout) or topped
// up to the difference (dispute).
const money = (paise: number) => `₹${(paise / 100).toFixed(0)}`;

export function CancellationsPage() {
  const { data, isLoading } = useAdminCancellations();
  const requests = data?.data ?? [];

  return (
    <div className="mx-auto max-w-3xl px-4 py-6">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Cancellation arbitration</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Disputes and vendor timeouts. Pick the tier that fits what actually happened — the platform fee
        is never refundable, and you can only raise a refund (never claw back).
      </p>

      {isLoading ? (
        <div className="mt-8 text-sm text-ink-soft">Loading…</div>
      ) : requests.length === 0 ? (
        <div className="mt-8 rounded-xl border border-mist bg-bone p-6 text-center text-sm text-ink-soft">
          Nothing awaiting review.
        </div>
      ) : (
        <div className="mt-6 flex flex-col gap-4">
          {requests.map((r) => (
            <ArbitrationCard key={r.id} req={r} />
          ))}
        </div>
      )}
    </div>
  );
}

function ArbitrationCard({ req }: { req: AdminCancellationRequest }) {
  const [reason, setReason] = useState<string | null>(null);
  const [note, setNote] = useState('');
  const resolve = useResolveCancellation();

  function onResolve() {
    if (!reason) return;
    resolve.mutate(
      { id: req.id, reason, note: note || undefined },
      {
        onSuccess: () => toast.success('Resolved — any additional refund has been issued.'),
        onError: () => toast.error('Could not resolve. Please try again.'),
      },
    );
  }

  return (
    <div className="rounded-xl border border-mist bg-bone p-5 shadow-sm">
      <div className="flex items-center justify-between">
        <p className="font-semibold text-foreground">Order #{req.orderId.slice(0, 8)}</p>
        <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800">
          {req.status === 'disputed' ? 'Disputed' : 'Vendor timeout'}
        </span>
      </div>
      {req.customerReason ? (
        <p className="mt-1 text-sm text-ink-soft">Customer: “{req.customerReason}”</p>
      ) : null}
      {req.disputeReason ? (
        <p className="mt-0.5 text-sm italic text-ink-soft">Dispute: “{req.disputeReason}”</p>
      ) : null}
      {req.refundExecuted ? (
        <p className="mt-1 text-xs text-ink-soft tabular-nums">
          Already refunded {money(req.refundTotalPaise)} · vendor kept {money(req.vendorKeptPaise)}
        </p>
      ) : null}

      <p className="mt-4 text-sm font-semibold text-foreground">Correct tier</p>
      <div className="mt-2 flex flex-col gap-2">
        {CANCEL_REASONS.map((r) => {
          const active = reason === r.value;
          return (
            <button
              key={r.value}
              type="button"
              onClick={() => setReason(r.value)}
              aria-pressed={active}
              className={
                'rounded-lg border p-3 text-left transition-colors ' +
                (active ? 'border-herb bg-herb/10' : 'border-mist hover:border-ink-soft')
              }
            >
              <span className={'block text-sm font-semibold ' + (active ? 'text-herb' : 'text-foreground')}>
                {r.label}
              </span>
              <span className="mt-0.5 block text-xs text-ink-soft">{r.hint}</span>
            </button>
          );
        })}
      </div>

      <textarea
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder="Internal note (optional)"
        rows={2}
        className="mt-3 w-full rounded-lg border border-mist bg-white p-2 text-sm text-foreground"
      />

      <button
        type="button"
        onClick={onResolve}
        disabled={!reason || resolve.isPending}
        className="mt-3 w-full rounded-lg bg-herb px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"
      >
        {resolve.isPending ? 'Resolving…' : 'Resolve'}
      </button>
    </div>
  );
}

export default CancellationsPage;
