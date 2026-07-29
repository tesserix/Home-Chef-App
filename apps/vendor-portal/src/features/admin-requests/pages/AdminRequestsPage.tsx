import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { AlertTriangle, BellRing } from 'lucide-react';
import {
  untilLabel,
  useAdminRequests,
  useRemindAdminRequest,
  type AdminRequest,
  type AdminRequestStatus,
} from '../hooks/useAdminRequests';

// Admin verification / info requests — the web twin of
// apps/mobile-vendor/app/admin-requests.tsx.
//
// This screen did not exist on web. The /admin-requests route was pointed at
// the notifications page instead, so a chef on the portal could not see that an
// admin had asked them for information, could not read what was asked, and had
// no way to bump a request that had gone unanswered.

// Status presentation. `info_requested` is the only one that is the CHEF's move,
// so it is the only one styled as a call to action rather than a state report.
const STATUS_META: Record<AdminRequestStatus, { label: string; className: string }> = {
  pending: { label: 'Pending', className: 'bg-amber-tint text-amber' },
  approved: { label: 'Approved', className: 'bg-herb-tint text-herb' },
  rejected: { label: 'Rejected', className: 'bg-paprika-tint text-paprika' },
  info_requested: { label: 'Action needed', className: 'bg-amber-tint text-amber' },
  cancelled: { label: 'Cancelled', className: 'bg-mist text-ink-soft' },
};

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

// A minute is the right resolution for a 6-24h countdown: it keeps the label
// honest without re-rendering the list every second.
function useNowEveryMinute(): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 60_000);
    return () => clearInterval(t);
  }, []);
  return now;
}

export function AdminRequestsPage() {
  const { data: requests = [], isLoading } = useAdminRequests();
  const now = useNowEveryMinute();

  // Anything the chef must answer floats to the top — the rest is history, and
  // an approved menu item from last week should never bury a question about a
  // licence that is blocking the kitchen.
  const actionable = requests.filter((r) => r.status === 'info_requested');
  const rest = requests.filter((r) => r.status !== 'info_requested');

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Admin requests</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Verification and information requests from the Fe3dr team, and the status of everything
        you&apos;ve submitted for approval.
      </p>

      {isLoading ? (
        <div className="mt-8 text-sm text-ink-soft">Loading…</div>
      ) : requests.length === 0 ? (
        <div className="mt-8 rounded-xl border border-mist bg-bone p-6 text-center">
          <p className="font-semibold text-foreground">Nothing to show</p>
          <p className="mt-1 text-sm text-ink-soft">
            When the team needs something from you — or reviews a menu item or document you
            submitted — it appears here.
          </p>
        </div>
      ) : (
        <>
          {actionable.length > 0 && (
            <section className="mt-6">
              <h2 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                Needs your reply
              </h2>
              <div className="mt-2 flex flex-col gap-3">
                {actionable.map((r) => (
                  <RequestCard key={r.id} req={r} now={now} />
                ))}
              </div>
            </section>
          )}

          {rest.length > 0 && (
            <section className="mt-8">
              {actionable.length > 0 && (
                <h2 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                  Everything else
                </h2>
              )}
              <div className="mt-2 flex flex-col gap-3">
                {rest.map((r) => (
                  <RequestCard key={r.id} req={r} now={now} />
                ))}
              </div>
            </section>
          )}
        </>
      )}
    </div>
  );
}

function RequestCard({ req, now }: { req: AdminRequest; now: number }) {
  const meta = STATUS_META[req.status] ?? STATUS_META.pending;
  const escalated = req.reminderCount >= 3;

  return (
    <article className="rounded-xl border border-mist bg-bone p-4 shadow-1">
      <div className="flex items-start justify-between gap-3">
        <span
          className={`inline-flex shrink-0 items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${meta.className}`}
        >
          {meta.label}
        </span>
        <span className="text-xs text-ink-muted tabular-nums">{formatDate(req.createdAt)}</span>
      </div>

      <h3 className="mt-2 font-semibold text-foreground">{req.title}</h3>
      {req.description && <p className="mt-1 text-sm text-ink-soft">{req.description}</p>}

      {/* Admin notes are the actual question when status is info_requested —
          without them the chef is told to act but not what to do. */}
      {req.adminNotes && (
        <div className="mt-3 rounded-lg border border-mist bg-paper p-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
            Note from the team
          </p>
          <p className="mt-1 text-sm text-foreground">{req.adminNotes}</p>
        </div>
      )}

      {escalated && (
        <p className="mt-3 flex items-center gap-1.5 text-xs font-medium text-amber">
          <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
          Escalated — a senior reviewer has been notified.
        </p>
      )}

      {/* The bump control, shown only where it means something: an undecided
          request the chef is waiting on. */}
      {req.status === 'pending' && <RemindControl req={req} now={now} />}
    </article>
  );
}

function RemindControl({ req, now }: { req: AdminRequest; now: number }) {
  const remind = useRemindAdminRequest();

  // Trust the server's verdict, but re-check the deadline locally so the button
  // unlocks on the countdown rather than on the next refetch.
  const unlocked =
    req.canRemind || (!!req.nextRemindAt && new Date(req.nextRemindAt).getTime() <= now);
  const waitingFor = req.nextRemindAt ? untilLabel(req.nextRemindAt, now) : '';

  function onClick() {
    remind.mutate(req.id, {
      onSuccess: (res) => {
        toast.success(
          res.escalated
            ? 'Escalated — a senior reviewer has been notified.'
            : 'Reminder sent to the team.',
        );
      },
      onError: (err) => {
        // A 429 is the expected outcome of a drifted countdown, not a failure
        // worth alarming the chef about.
        const status = err?.response?.status ?? err?.status;
        toast.error(
          status === 429
            ? 'This request was reminded recently — the button unlocks again shortly.'
            : (err?.response?.data?.error ?? 'Could not send the reminder. Please try again.'),
        );
      },
    });
  }

  return (
    <div className="mt-3 flex items-center gap-3">
      <button
        type="button"
        onClick={onClick}
        disabled={!unlocked || remind.isPending}
        className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-mist px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-paper disabled:cursor-not-allowed disabled:opacity-50"
      >
        <BellRing className="h-4 w-4" aria-hidden="true" />
        {remind.isPending ? 'Sending…' : 'Send a reminder'}
      </button>
      {!unlocked && waitingFor && (
        <span className="text-xs text-ink-muted">Available again in {waitingFor}</span>
      )}
      {req.reminderCount > 0 && (
        <span className="text-xs text-ink-muted tabular-nums">
          {req.reminderCount} sent
        </span>
      )}
    </div>
  );
}

export default AdminRequestsPage;
