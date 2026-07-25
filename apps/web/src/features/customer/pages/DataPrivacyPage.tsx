import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { useAuth } from '@/app/providers/AuthProvider';
import {
  useDeactivateAccount,
  useDeleteAccount,
  useDeletionEligibility,
  useExportMyData,
  useReactivateAccount,
} from '@/features/customer/hooks/useDataPrivacy';
import {
  ACCOUNT_BLOCKED_EVENT,
  type AccountBlockedStatus,
} from '@/shared/services/api-client';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  Card,
  Input,
} from '@/shared/components/ui';

/**
 * DPDP Act 2023 access and erasure rights, plus the reversible pause.
 *
 * Deletion eligibility is fetched on mount rather than checked on submit, so
 * an account with money or work in flight says so up front instead of failing
 * after the user has typed their email.
 */
export default function DataPrivacyPage() {
  const { user } = useAuth();
  const [confirmEmail, setConfirmEmail] = useState('');
  const [confirmPauseOpen, setConfirmPauseOpen] = useState(false);
  const [purgeAfter, setPurgeAfter] = useState<string | null>(null);
  // Set directly on a successful pause, and also from ACCOUNT_BLOCKED_EVENT —
  // which the eligibility query below will trigger on mount if the account
  // is already paused (e.g. a paused user reloading this page). Either way,
  // once set this replaces the rest of the page: every other action on it
  // 403s for a paused account anyway (middleware/bff_auth.go only lets
  // /me/reactivate through), so showing them is just noise.
  const [blockedStatus, setBlockedStatus] = useState<AccountBlockedStatus | null>(null);

  const eligibility = useDeletionEligibility();
  const exportData = useExportMyData();
  const deactivate = useDeactivateAccount();
  const reactivate = useReactivateAccount();
  const deleteAccount = useDeleteAccount();

  useEffect(() => {
    function handler(e: Event) {
      const status = (e as CustomEvent<{ status?: AccountBlockedStatus }>).detail?.status;
      if (status) setBlockedStatus(status);
    }
    window.addEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    return () => window.removeEventListener(ACCOUNT_BLOCKED_EVENT, handler);
  }, []);

  const accountEmail = user?.email ?? '';
  const emailMatches =
    confirmEmail.trim().toLowerCase() === accountEmail.toLowerCase() &&
    accountEmail.length > 0;
  const blockers = eligibility.data?.blockers ?? [];
  const deletable = eligibility.data?.deletable === true;

  function handleExport() {
    exportData.mutate(undefined, {
      onSuccess: (data) => {
        // Hand the user a file rather than dumping JSON on screen — this is
        // their record to keep.
        const blob = new Blob([JSON.stringify(data, null, 2)], {
          type: 'application/json',
        });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = 'fe3dr-my-data.json';
        link.click();
        URL.revokeObjectURL(url);
      },
      onError: () => toast.error('Could not prepare your data — try again'),
    });
  }

  function handlePauseConfirmed() {
    deactivate.mutate(undefined, {
      onSuccess: () => {
        setConfirmPauseOpen(false);
        setBlockedStatus('account_deactivated');
        toast.success('Account paused — turn it back on any time from this page.');
      },
      onError: () => toast.error('Could not pause your account — try again'),
    });
  }

  function handleReactivate() {
    reactivate.mutate(undefined, {
      onSuccess: () => {
        setBlockedStatus(null);
        toast.success('Welcome back — your account is active again.');
      },
      onError: () => toast.error('Could not reactivate your account — try again'),
    });
  }

  function handleDelete() {
    deleteAccount.mutate(confirmEmail.trim(), {
      onSuccess: (result) => setPurgeAfter(result.purgeAfter),
      onError: () => toast.error('Could not delete your account — try again'),
    });
  }

  if (purgeAfter) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <h1 className="text-2xl font-semibold text-foreground">Account deleted</h1>
        <p className="mt-3 text-muted-foreground">
          Your sign-in no longer works. We keep your data until{' '}
          {new Date(purgeAfter).toLocaleDateString()} — sign up again with the
          same email before then and you can restore it. After that it is erased
          for good.
        </p>
      </div>
    );
  }

  if (blockedStatus) {
    const deleted = blockedStatus === 'account_deleted';
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <Card className="p-6">
          <h1 className="text-2xl font-semibold text-foreground">
            {deleted ? 'This account was deleted' : 'Your account is paused'}
          </h1>
          <p className="mt-3 text-muted-foreground">
            {deleted
              ? 'You can come back any time — sign up again with the same email within 180 days and your history comes with you.'
              : 'Everything is safe and waiting for you — your orders, saved addresses and wallet balance are exactly as you left them. Turn it back on whenever you like; it takes effect straight away.'}
          </p>
          {!deleted && (
            <Button
              className="mt-6"
              onClick={handleReactivate}
              disabled={reactivate.isPending}
            >
              {reactivate.isPending ? 'Reactivating…' : 'Turn my account back on'}
            </Button>
          )}
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <h1 className="text-2xl font-semibold text-foreground">Your data</h1>

      <Card className="mt-8 p-6">
        <h2 className="text-lg font-medium text-foreground">Download my data</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          A machine-readable copy of your profile, orders and related records.
        </p>
        <Button
          className="mt-4"
          onClick={handleExport}
          disabled={exportData.isPending}
        >
          {exportData.isPending ? 'Preparing…' : 'Download my data'}
        </Button>
      </Card>

      <Card className="mt-4 p-6">
        <h2 className="text-lg font-medium text-foreground">Pause my account</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Hides your profile and stops notifications. Nothing is deleted, and
          there is no time limit — turn it back on from this page whenever you
          want to come back.
        </p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={() => setConfirmPauseOpen(true)}
          disabled={deactivate.isPending}
        >
          {deactivate.isPending ? 'Pausing…' : 'Pause my account'}
        </Button>
      </Card>

      <AlertDialog open={confirmPauseOpen} onOpenChange={setConfirmPauseOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Pause your account?</AlertDialogTitle>
            <AlertDialogDescription>
              Your profile is hidden and notifications stop straight away.
              Nothing is deleted — come back and turn it back on from this
              page whenever you like.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="outline">Keep my account active</Button>
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handlePauseConfirmed}
              disabled={deactivate.isPending}
            >
              {deactivate.isPending ? 'Pausing…' : 'Pause my account'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Card className="mt-4 p-6">
        <h2 className="text-lg font-medium text-foreground">Delete my account</h2>

        {eligibility.isPending ? (
          <p className="mt-2 text-sm text-muted-foreground">Checking…</p>
        ) : deletable ? (
          <>
            <p className="mt-2 text-sm text-muted-foreground">
              Your sign-in stops working straight away. We keep your data for{' '}
              {eligibility.data?.retentionDays ?? 180} days in case you change
              your mind, then erase it permanently.
            </p>
            <label
              htmlFor="confirm-email"
              className="mt-4 block text-sm font-medium text-foreground"
            >
              Type {accountEmail} to confirm
            </label>
            <Input
              id="confirm-email"
              type="email"
              autoComplete="off"
              value={confirmEmail}
              onChange={(e) => setConfirmEmail(e.target.value)}
              className="mt-2"
            />
            <Button
              variant="destructive"
              className="mt-4"
              onClick={handleDelete}
              disabled={!emailMatches || deleteAccount.isPending}
            >
              {deleteAccount.isPending ? 'Deleting…' : 'Delete my account'}
            </Button>
          </>
        ) : (
          <>
            <p className="mt-2 text-sm text-muted-foreground">
              Finish these first — deleting now would strand a payment or an
              order someone is waiting on.
            </p>
            <ul className="mt-3 space-y-2">
              {blockers.map((blocker) => (
                <li key={blocker.code} className="text-sm text-foreground">
                  {blocker.label}
                </li>
              ))}
            </ul>
            <Button variant="destructive" className="mt-4" disabled>
              Delete my account
            </Button>
          </>
        )}
      </Card>
    </div>
  );
}
