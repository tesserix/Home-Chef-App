import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  ShieldAlert,
  Unlock,
  AlertTriangle,
  Loader2,
  Clock,
  Mail,
  X,
  RotateCcw,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

interface LockedChef {
  chefId: string;
  userId: string;
  businessName: string;
  fssaiExpiry: string | null;
  daysSinceExpiry: number;
  overrideUntil?: string | null;
  overrideReason?: string;
  overrideBy?: string | null;
}

interface FssaiLockedResponse {
  locked: LockedChef[];
  overridden: LockedChef[];
  lockedCount: number;
  overriddenCount: number;
  missingExpiryCount: number;
}

const OVERRIDE_MAX_DAYS = 30;

function errMessage(e: unknown, fallback: string): string {
  if (e && typeof e === 'object' && 'error' in e) {
    const er = (e as { error?: { message?: string } }).error;
    if (er?.message) return er.message;
  }
  return fallback;
}

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleDateString();
}

export default function FssaiLockedChefsPage() {
  const queryClient = useQueryClient();
  const [overrideChef, setOverrideChef] = useState<LockedChef | null>(null);
  const [reason, setReason] = useState('');
  const [days, setDays] = useState(7);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-fssai-locked'],
    queryFn: () => apiClient.get<FssaiLockedResponse>('/admin/chefs/fssai-locked'),
  });
  const resp = data as unknown as FssaiLockedResponse | undefined;
  const locked = resp?.locked ?? [];
  const overridden = resp?.overridden ?? [];
  const missingExpiryCount = resp?.missingExpiryCount ?? 0;

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['admin-fssai-locked'] });

  const closeDialog = () => {
    setOverrideChef(null);
    setReason('');
    setDays(7);
  };

  const overrideMutation = useMutation({
    mutationFn: (v: { chefId: string; reason: string; days: number }) =>
      apiClient.post(`/admin/chefs/${v.chefId}/fssai-override`, { reason: v.reason, days: v.days }),
    onSuccess: () => {
      toast.success('Override applied — chef is temporarily back online');
      closeDialog();
      invalidate();
    },
    onError: (e) => toast.error(errMessage(e, 'Failed to apply override')),
  });

  const revokeMutation = useMutation({
    mutationFn: (chefId: string) => apiClient.delete(`/admin/chefs/${chefId}/fssai-override`),
    onSuccess: () => {
      toast.success('Override revoked — lockout re-applied');
      invalidate();
    },
    onError: () => toast.error('Failed to revoke override'),
  });

  const backfillMutation = useMutation({
    mutationFn: () => apiClient.post<{ notified: number }>('/admin/fssai-expiry-backfill'),
    onSuccess: (r) => {
      const n = (r as unknown as { notified?: number })?.notified ?? 0;
      toast.success(`Confirm-licence prompt sent to ${n} chef${n === 1 ? '' : 's'}`);
      invalidate();
    },
    onError: () => toast.error('Failed to send confirm-licence prompts'),
  });

  const reasonValid = reason.trim().length >= 10;
  const daysValid = days >= 1 && days <= OVERRIDE_MAX_DAYS;

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">FSSAI Lockouts</h1>
        <p className="page-description">
          India chefs whose food-safety (FSSAI) licence has lapsed are locked out of new orders and
          payouts. Grant a time-boxed, reason-logged override only for genuine edge cases (e.g. a
          government renewal backlog) — every action is audited.
        </p>
      </div>

      {/* Backfill banner: chefs with a verified FSSAI doc but no recorded expiry */}
      {missingExpiryCount > 0 && (
        <div className="flex flex-col gap-3 rounded-xl border border-warning/40 bg-warning/10 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-start gap-3">
            <AlertTriangle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
            <div>
              <p className="text-sm font-medium text-foreground">
                {missingExpiryCount} chef{missingExpiryCount === 1 ? '' : 's'} have an FSSAI licence
                on file with no recorded expiry
              </p>
              <p className="text-sm text-muted-foreground">
                These legacy uploads can't be expiry-checked. Prompt the chefs to confirm their
                licence so the lockout can protect customers going forward.
              </p>
            </div>
          </div>
          <Button
            variant="outline"
            leftIcon={<Mail className="h-4 w-4" />}
            isLoading={backfillMutation.isPending}
            disabled={backfillMutation.isPending}
            onClick={() => backfillMutation.mutate()}
            className="shrink-0"
          >
            Send confirm-licence prompt
          </Button>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
        </div>
      ) : (
        <>
          {/* Locked chefs */}
          <section className="space-y-3">
            <h2 className="flex items-center gap-2 text-base font-semibold text-foreground">
              <ShieldAlert className="h-4 w-4 text-destructive" />
              Locked ({locked.length})
            </h2>
            {locked.length === 0 ? (
              <p className="rounded-lg border border-border bg-card p-6 text-sm text-muted-foreground">
                No chefs are currently locked out. 🎉
              </p>
            ) : (
              <div className="overflow-hidden rounded-xl border border-border bg-card">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-left text-xs uppercase tracking-wide text-muted-foreground">
                      <th className="px-4 py-3 font-medium">Kitchen</th>
                      <th className="px-4 py-3 font-medium">Licence expired</th>
                      <th className="px-4 py-3 font-medium tabular-nums">Days overdue</th>
                      <th className="px-4 py-3 text-right font-medium">Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {locked.map((chef) => (
                      <tr key={chef.chefId} className="border-b border-border last:border-0">
                        <td className="px-4 py-3 font-medium text-foreground">{chef.businessName}</td>
                        <td className="px-4 py-3 text-muted-foreground">{formatDate(chef.fssaiExpiry)}</td>
                        <td className="px-4 py-3 tabular-nums text-destructive">{chef.daysSinceExpiry}</td>
                        <td className="px-4 py-3 text-right">
                          <Button
                            variant="outline"
                            size="sm"
                            leftIcon={<Unlock className="h-4 w-4" />}
                            onClick={() => setOverrideChef(chef)}
                          >
                            Override
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          {/* Overridden chefs */}
          {overridden.length > 0 && (
            <section className="space-y-3">
              <h2 className="flex items-center gap-2 text-base font-semibold text-foreground">
                <Clock className="h-4 w-4 text-warning" />
                Active overrides ({overridden.length})
              </h2>
              <div className="overflow-hidden rounded-xl border border-border bg-card">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-left text-xs uppercase tracking-wide text-muted-foreground">
                      <th className="px-4 py-3 font-medium">Kitchen</th>
                      <th className="px-4 py-3 font-medium">Reason</th>
                      <th className="px-4 py-3 font-medium">Reprieve until</th>
                      <th className="px-4 py-3 text-right font-medium">Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {overridden.map((chef) => (
                      <tr key={chef.chefId} className="border-b border-border last:border-0">
                        <td className="px-4 py-3 font-medium text-foreground">{chef.businessName}</td>
                        <td className="px-4 py-3 max-w-xs truncate text-muted-foreground" title={chef.overrideReason}>
                          {chef.overrideReason || '—'}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">{formatDate(chef.overrideUntil)}</td>
                        <td className="px-4 py-3 text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            leftIcon={<RotateCcw className="h-4 w-4" />}
                            isLoading={revokeMutation.isPending && revokeMutation.variables === chef.chefId}
                            disabled={revokeMutation.isPending}
                            onClick={() => revokeMutation.mutate(chef.chefId)}
                            className="text-muted-foreground hover:text-destructive"
                          >
                            Revoke
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}
        </>
      )}

      {/* Override dialog */}
      {overrideChef && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <button
            type="button"
            aria-label="Close"
            className="fixed inset-0 bg-foreground/50"
            onClick={closeDialog}
          />
          <div className="relative z-50 mx-4 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-3">
            <div className="mb-4 flex items-start justify-between">
              <div>
                <h3 className="text-lg font-semibold text-foreground">Override FSSAI lockout</h3>
                <p className="text-sm text-muted-foreground">{overrideChef.businessName}</p>
              </div>
              <button
                type="button"
                aria-label="Close"
                onClick={closeDialog}
                className="text-muted-foreground hover:text-foreground"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <p className="mb-4 rounded-lg bg-warning/10 p-3 text-xs text-muted-foreground">
              This temporarily lets a chef with a lapsed licence accept orders and receive payouts.
              Use only for genuine edge cases — the grant is recorded in the audit log with your
              account, the reason, and the expiry.
            </p>

            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (reasonValid && daysValid) {
                  overrideMutation.mutate({ chefId: overrideChef.chefId, reason: reason.trim(), days });
                }
              }}
              className="space-y-4"
            >
              <div className="space-y-2">
                <label htmlFor="override-reason" className="text-sm font-medium text-foreground">
                  Reason <span className="text-muted-foreground">(min 10 chars, logged)</span>
                </label>
                <textarea
                  id="override-reason"
                  rows={3}
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="e.g. FoSCoS renewal filed 2026-06-01, acknowledgement on file, processing backlog"
                  className="w-full rounded-lg border border-input bg-card p-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>

              <div className="space-y-2">
                <label htmlFor="override-days" className="text-sm font-medium text-foreground">
                  Duration (days)
                </label>
                <input
                  id="override-days"
                  type="number"
                  min={1}
                  max={OVERRIDE_MAX_DAYS}
                  value={days}
                  onChange={(e) => setDays(Number(e.target.value))}
                  className="h-10 w-32 rounded-lg border border-input bg-card px-3 text-sm tabular-nums text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
                <p className="text-xs text-muted-foreground">
                  Max {OVERRIDE_MAX_DAYS} days. The lockout re-applies automatically when it lapses.
                </p>
              </div>

              <div className="flex items-center justify-end gap-3 pt-2">
                <Button type="button" variant="outline" onClick={closeDialog}>
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  isLoading={overrideMutation.isPending}
                  disabled={overrideMutation.isPending || !reasonValid || !daysValid}
                >
                  Grant override
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
