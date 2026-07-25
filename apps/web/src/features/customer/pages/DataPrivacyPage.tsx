import { useState } from 'react';
import { toast } from 'sonner';
import { useAuth } from '@/app/providers/AuthProvider';
import {
  useDeactivateAccount,
  useDeleteAccount,
  useDeletionEligibility,
  useExportMyData,
} from '@/features/customer/hooks/useDataPrivacy';
import { Button, Card, Input } from '@/shared/components/ui';

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
  const [purgeAfter, setPurgeAfter] = useState<string | null>(null);

  const eligibility = useDeletionEligibility();
  const exportData = useExportMyData();
  const deactivate = useDeactivateAccount();
  const deleteAccount = useDeleteAccount();

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

  function handleDeactivate() {
    deactivate.mutate(undefined, {
      onSuccess: () => toast.success('Account paused. Sign in again to resume.'),
      onError: () => toast.error('Could not pause your account — try again'),
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
          there is no time limit — sign in again whenever you want to come back.
        </p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={handleDeactivate}
          disabled={deactivate.isPending}
        >
          {deactivate.isPending ? 'Pausing…' : 'Pause my account'}
        </Button>
      </Card>

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
