import { useState } from 'react';
import { toast } from 'sonner';
import {
  CANCEL_REASONS,
  useCancellationRequests,
  useConfirmCancellation,
  type CancellationRequest,
} from '../hooks/useCancellations';

// Vendor arbitration queue (#475) — the web twin of the mobile-vendor screen.
// Same API + same reason tiers; a customer asked to cancel and the chef picks
// the reason, which the API turns into a tiered refund (platform fee kept).
export function CancellationRequestsPage() {
  const { data, isLoading } = useCancellationRequests('pending_vendor');
  const requests = data?.data ?? [];

  return (
    <div className="mx-auto max-w-2xl px-4 py-6 sm:px-5">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Cancellation requests</h1>
      <p className="mt-1 text-sm text-ink-soft">
        A customer asked to cancel. Tell us where the order is and we'll issue the right refund — your
        material and prep costs are protected, and the platform fee is never refunded.
      </p>

      {isLoading ? (
        <div className="mt-8 text-sm text-ink-soft">Loading…</div>
      ) : requests.length === 0 ? (
        <div className="mt-8 rounded-xl border border-mist bg-bone p-6 text-center">
          <p className="font-semibold text-foreground">No requests right now</p>
          <p className="mt-1 text-sm text-ink-soft">
            When a customer asks to cancel an order you're preparing, it appears here for you to confirm.
          </p>
        </div>
      ) : (
        <div className="mt-6 flex flex-col gap-4">
          {requests.map((r) => (
            <RequestCard key={r.id} req={r} />
          ))}
        </div>
      )}
    </div>
  );
}

function RequestCard({ req }: { req: CancellationRequest }) {
  const [reason, setReason] = useState<string | null>(null);
  const confirm = useConfirmCancellation();

  function onConfirm() {
    if (!reason) return;
    confirm.mutate(
      { id: req.id, reason },
      {
        onSuccess: () => toast.success('Cancellation confirmed — the refund has been issued.'),
        onError: () => toast.error('Could not confirm. Please try again.'),
      },
    );
  }

  return (
    <div className="rounded-xl border border-mist bg-bone p-5 shadow-sm">
      <p className="font-semibold text-foreground">Order #{req.orderId.slice(0, 8)}</p>
      {req.customerReason ? <p className="mt-0.5 text-sm italic text-ink-soft">“{req.customerReason}”</p> : null}

      <p className="mt-4 text-sm font-semibold text-foreground">Where is this order?</p>
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

      <button
        type="button"
        onClick={onConfirm}
        disabled={!reason || confirm.isPending}
        className="mt-4 w-full rounded-lg bg-herb px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"
      >
        {confirm.isPending ? 'Confirming…' : 'Confirm cancellation'}
      </button>
    </div>
  );
}

export default CancellationRequestsPage;
