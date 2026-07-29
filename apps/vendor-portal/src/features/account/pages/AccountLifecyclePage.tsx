import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';
import { AlertTriangle, PauseCircle, Trash2 } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { useAuth } from '@/app/providers/AuthProvider';
import {
  useDeactivateAccount,
  useDeleteAccount,
  useDeletionEligibility,
} from '../hooks/useAccountLifecycle';

// Pause or delete the kitchen — the web twin of
// apps/mobile-vendor/app/account-lifecycle.tsx.
//
// Two deliberately different weights on one page. Pausing is presented as an
// ordinary operational action because it IS one and it is reversible. Deletion
// is gated behind the server's own eligibility check and a typed email
// confirmation, because it resets approval and starts a retention clock.

export function AccountLifecyclePage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { data: eligibility, isLoading } = useDeletionEligibility();
  const deactivate = useDeactivateAccount();
  const del = useDeleteAccount();

  const [pauseReason, setPauseReason] = useState('');
  const [confirmEmail, setConfirmEmail] = useState('');

  const userEmail = user?.email ?? '';
  // Case-insensitive: the server compares the same way, and forcing a chef to
  // reproduce their own capitalisation is a pointless obstacle.
  const emailMatches =
    confirmEmail.trim().toLowerCase() === userEmail.trim().toLowerCase() && userEmail !== '';

  function onPause() {
    deactivate.mutate(pauseReason.trim() || undefined, {
      onSuccess: () => {
        toast.success('Your kitchen is paused. Reactivate any time from here.');
        navigate('/dashboard');
      },
      onError: () => toast.error('Could not pause your account. Please try again.'),
    });
  }

  function onDelete() {
    del.mutate(confirmEmail.trim(), {
      onSuccess: (res) => {
        toast.success(
          `Account deleted. You can restore it until ${new Date(res.purgeAfter).toLocaleDateString()}.`,
        );
        navigate('/login');
      },
      onError: () => toast.error('Could not delete your account. Please try again.'),
    });
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">
        Pause or delete account
      </h1>
      <p className="mt-1 text-sm text-ink-soft">
        Pausing is reversible and keeps everything. Deleting is not the same thing — read the
        second section before you start it.
      </p>

      {/* ── Pause ─────────────────────────────────────────────────────────── */}
      <Card className="mt-6 p-5">
        <h2 className="flex items-center gap-2 font-semibold text-foreground">
          <PauseCircle className="h-5 w-5 text-ink-soft" aria-hidden="true" />
          Pause your kitchen
        </h2>
        <p className="mt-1 text-sm text-ink-soft">
          Closes the kitchen, hides it from customers and turns off schedule-driven auto-open so it
          can&apos;t quietly reopen. Nothing is deleted, your approval is untouched, and you can
          reactivate whenever you like.
        </p>
        <label htmlFor="pause-reason" className="mt-4 block text-sm font-medium text-ink-soft">
          Reason (optional)
        </label>
        <input
          id="pause-reason"
          value={pauseReason}
          onChange={(e) => setPauseReason(e.target.value)}
          placeholder="Taking a break, travelling, etc."
          className="input-base mt-1"
        />
        <Button
          variant="secondary"
          className="mt-4"
          onClick={onPause}
          isLoading={deactivate.isPending}
        >
          Pause my kitchen
        </Button>
      </Card>

      {/* ── Delete ────────────────────────────────────────────────────────── */}
      <Card className="mt-6 border-destructive/30 p-5">
        <h2 className="flex items-center gap-2 font-semibold text-destructive">
          <Trash2 className="h-5 w-5" aria-hidden="true" />
          Delete your account
        </h2>

        {isLoading ? (
          <Skeleton className="mt-4 h-20 w-full" />
        ) : eligibility && !eligibility.deletable ? (
          // Blockers first, and the delete control is not rendered at all —
          // showing a disabled button next to a list of reasons invites people
          // to click it and conclude the page is broken.
          <div className="mt-3">
            <p className="flex items-start gap-2 text-sm font-medium text-amber">
              <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
              You can&apos;t delete your account just yet
            </p>
            <p className="mt-1 text-sm text-ink-soft">
              These need to finish first — they involve orders or money that are still in play:
            </p>
            <ul className="mt-2 space-y-1">
              {eligibility.blockers.map((b) => (
                <li key={b.code} className="text-sm text-ink-soft">
                  • {b.label}
                  {b.count != null ? ` (${b.count})` : ''}
                </li>
              ))}
            </ul>
          </div>
        ) : (
          <>
            <p className="mt-1 text-sm text-ink-soft">
              Your kitchen is removed from the platform and your menu stops being visible. We keep
              your data for{' '}
              <strong className="text-foreground">{eligibility?.retentionDays ?? 180} days</strong>{' '}
              so you can restore the account by signing in again — after that it is permanently
              purged. Restoring resets your approval, so you would be re-verified before trading.
            </p>

            <label htmlFor="confirm-email" className="mt-4 block text-sm font-medium text-ink-soft">
              Type <span className="font-semibold text-foreground">{userEmail}</span> to confirm
            </label>
            <input
              id="confirm-email"
              value={confirmEmail}
              onChange={(e) => setConfirmEmail(e.target.value)}
              autoComplete="off"
              placeholder={userEmail}
              className="input-base mt-1"
            />

            <Button
              variant="destructive"
              className="mt-4"
              disabled={!emailMatches || del.isPending}
              isLoading={del.isPending}
              onClick={onDelete}
            >
              Delete my account
            </Button>
          </>
        )}
      </Card>
    </div>
  );
}

export default AccountLifecyclePage;
