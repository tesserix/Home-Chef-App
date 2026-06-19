import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { AlertTriangle, Loader2, Check, X, Image as ImageIcon, Settings2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

// Admin queue for customer-reported order issues (#262). Pending issues that
// exceeded the auto-approve cap land here for an assisted partial refund (paid
// to the customer's wallet via the same exactly-once path as the auto-refund).

type IssueStatus = 'pending' | 'auto_refunded' | 'resolved' | 'rejected';

interface OrderIssue {
  id: string;
  orderId: string;
  chefId: string;
  customerId: string;
  reason: string;
  description?: string;
  photoUrls: string[] | null;
  affectedItemIds: string[] | null;
  requestedAmount: number;
  refundAmount: number;
  status: IssueStatus;
  resolvedAt?: string;
  createdAt: string;
}

interface IssuesResponse {
  data: OrderIssue[];
  count: number;
}

interface IssueConfig {
  enabled: boolean;
  autoApproveCap: number;
}

const STATUS_TABS: { value: IssueStatus | 'all'; label: string }[] = [
  { value: 'pending', label: 'Pending' },
  { value: 'auto_refunded', label: 'Auto-refunded' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'all', label: 'All' },
];

const STATUS_STYLE: Record<IssueStatus, string> = {
  pending: 'bg-amber-500/10 text-amber-600',
  auto_refunded: 'bg-success/10 text-success',
  resolved: 'bg-success/10 text-success',
  rejected: 'bg-muted text-muted-foreground',
};

function money(n: number): string {
  try {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(n);
  } catch {
    return `₹${n.toFixed(2)}`;
  }
}

function titleCase(s: string): string {
  return s.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function OrderIssuesPage() {
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<IssueStatus | 'all'>('pending');
  // Per-row resolve amount (defaults to the requested amount).
  const [amounts, setAmounts] = useState<Record<string, string>>({});

  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin-order-issues', tab],
    queryFn: () =>
      apiClient.get<IssuesResponse>(
        tab === 'all' ? '/admin/order-issues' : `/admin/order-issues?status=${tab}`,
      ),
    refetchInterval: 30000,
  });
  const issues = (data as unknown as IssuesResponse | undefined)?.data ?? [];

  const { data: configData } = useQuery({
    queryKey: ['admin-order-issue-config'],
    queryFn: () => apiClient.get<IssueConfig>('/admin/order-issue/config'),
  });
  const config = configData as unknown as IssueConfig | undefined;

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-order-issues'] });
  };

  const resolveMutation = useMutation({
    mutationFn: ({ id, amount }: { id: string; amount: number }) =>
      apiClient.post<{ status: string; refundAmount: number }>(`/admin/order-issues/${id}/resolve`, { amount }),
    onSuccess: (res, vars) => {
      // The server caps the refund at the order's remaining refundable, so trust
      // its returned amount over what was requested.
      const actual = (res as unknown as { refundAmount?: number })?.refundAmount ?? vars.amount;
      toast.success(`Refunded ${money(actual)} to the customer's wallet`);
      invalidate();
    },
    onError: (e: unknown) => toast.error(errMsg(e) || 'Could not issue the refund'),
  });

  const rejectMutation = useMutation({
    mutationFn: (id: string) => apiClient.post(`/admin/order-issues/${id}/reject`, {}),
    onSuccess: () => {
      toast.success('Issue rejected');
      invalidate();
    },
    onError: (e: unknown) => toast.error(errMsg(e) || 'Could not reject the issue'),
  });

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Order Issues</h1>
        <p className="page-description">
          Customer-reported problems on paid orders. Small, clear cases are refunded to the wallet
          automatically; anything above the auto-approve cap waits here for review.
        </p>
      </div>

      <ConfigCard config={config} />

      {/* Status tabs */}
      <div className="flex flex-wrap gap-2">
        {STATUS_TABS.map((t) => (
          <button
            key={t.value}
            type="button"
            onClick={() => setTab(t.value)}
            className={`rounded-full px-4 py-1.5 text-sm font-medium ${
              tab === t.value ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {isLoading && (
        <div className="flex items-center justify-center py-12 text-muted-foreground">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-border bg-card p-6 text-sm text-muted-foreground">
          Could not load order issues.
        </div>
      )}

      {!isLoading && !isError && issues.length === 0 && (
        <div className="rounded-lg border border-border bg-card p-10 text-center text-sm text-muted-foreground">
          No {tab === 'all' ? '' : tab.replace(/_/g, ' ')} issues.
        </div>
      )}

      <div className="space-y-3">
        {issues.map((issue) => {
          const pending = issue.status === 'pending';
          const amountStr = amounts[issue.id] ?? String(issue.requestedAmount || '');
          const amountNum = Number(amountStr);
          const busy =
            (resolveMutation.isPending && resolveMutation.variables?.id === issue.id) ||
            (rejectMutation.isPending && rejectMutation.variables === issue.id);
          return (
            <div key={issue.id} className="rounded-xl border border-border bg-card p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <AlertTriangle className="h-4 w-4 text-amber-500" aria-hidden="true" />
                    <span className="font-medium text-foreground">{titleCase(issue.reason)}</span>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLE[issue.status]}`}>
                      {titleCase(issue.status)}
                    </span>
                  </div>
                  {issue.description && (
                    <p className="mt-1 text-sm text-muted-foreground">{issue.description}</p>
                  )}
                  <p className="mt-1 text-xs text-muted-foreground">
                    Order {issue.orderId.slice(0, 8)} · {new Date(issue.createdAt).toLocaleString()}
                    {issue.affectedItemIds && issue.affectedItemIds.length > 0
                      ? ` · ${issue.affectedItemIds.length} item(s) flagged`
                      : ''}
                  </p>
                  {issue.photoUrls && issue.photoUrls.length > 0 && (
                    <div className="mt-2 flex gap-2">
                      {issue.photoUrls.map((url) => (
                        <a
                          key={url}
                          href={url}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                        >
                          <ImageIcon className="h-3.5 w-3.5" aria-hidden="true" /> Photo
                        </a>
                      ))}
                    </div>
                  )}
                </div>
                <div className="text-right">
                  <p className="text-xs text-muted-foreground">Requested</p>
                  <p className="text-lg font-semibold tabular-nums text-foreground">
                    {money(issue.requestedAmount)}
                  </p>
                  {issue.refundAmount > 0 && issue.status !== 'pending' && (
                    <p className="text-xs text-success">Refunded {money(issue.refundAmount)}</p>
                  )}
                </div>
              </div>

              {pending && (
                <div className="mt-4 flex flex-wrap items-end gap-2 border-t border-border pt-3">
                  <div>
                    <label
                      htmlFor={`amt-${issue.id}`}
                      className="mb-1 block text-xs font-medium text-muted-foreground"
                    >
                      Refund amount
                    </label>
                    <input
                      id={`amt-${issue.id}`}
                      type="number"
                      min={0}
                      step="0.01"
                      value={amountStr}
                      onChange={(e) => setAmounts((p) => ({ ...p, [issue.id]: e.target.value }))}
                      className="h-9 w-32 rounded-lg border border-input bg-card px-3 text-sm tabular-nums text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                    />
                  </div>
                  <Button
                    variant="primary"
                    leftIcon={<Check className="h-4 w-4" />}
                    isLoading={resolveMutation.isPending && resolveMutation.variables?.id === issue.id}
                    disabled={busy || !(amountNum > 0)}
                    onClick={() => resolveMutation.mutate({ id: issue.id, amount: amountNum })}
                  >
                    Approve refund
                  </Button>
                  <Button
                    variant="ghost"
                    leftIcon={<X className="h-4 w-4" />}
                    disabled={busy}
                    onClick={() => rejectMutation.mutate(issue.id)}
                  >
                    Reject
                  </Button>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function ConfigCard({ config }: { config?: IssueConfig }) {
  const queryClient = useQueryClient();
  const [cap, setCap] = useState('');
  const [editing, setEditing] = useState(false);

  const updateMutation = useMutation({
    mutationFn: (payload: { enabled?: boolean; autoApproveCap?: number }) =>
      apiClient.put('/admin/order-issue/config', payload),
    onSuccess: () => {
      toast.success('Policy updated');
      setEditing(false);
      queryClient.invalidateQueries({ queryKey: ['admin-order-issue-config'] });
    },
    onError: () => toast.error('Could not update the policy'),
  });

  const capValue = useMemo(() => (editing ? cap : String(config?.autoApproveCap ?? '')), [editing, cap, config]);

  if (!config) return null;

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-4">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <Settings2 className="h-5 w-5" />
        </div>
        <div>
          <p className="text-sm font-medium text-foreground">Auto-approve policy</p>
          <p className="text-xs text-muted-foreground">
            {config.enabled ? 'Enabled' : 'Disabled'} · refunds up to{' '}
            <span className="font-medium text-foreground">{money(config.autoApproveCap)}</span> are paid
            automatically.
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => updateMutation.mutate({ enabled: !config.enabled })}
          isLoading={updateMutation.isPending}
        >
          {config.enabled ? 'Disable' : 'Enable'}
        </Button>
        {editing ? (
          <>
            <input
              type="number"
              min={0}
              step="1"
              value={capValue}
              onChange={(e) => setCap(e.target.value)}
              placeholder="Cap"
              className="h-9 w-24 rounded-lg border border-input bg-card px-2 text-sm tabular-nums text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <Button
              variant="primary"
              size="sm"
              isLoading={updateMutation.isPending}
              disabled={!(Number(cap) >= 0) || cap === ''}
              onClick={() => updateMutation.mutate({ autoApproveCap: Number(cap) })}
            >
              Save
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setEditing(false)}>
              Cancel
            </Button>
          </>
        ) : (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setCap(String(config.autoApproveCap));
              setEditing(true);
            }}
          >
            Edit cap
          </Button>
        )}
      </div>
    </div>
  );
}

function errMsg(e: unknown): string | undefined {
  if (e && typeof e === 'object' && 'error' in e) {
    return (e as { error?: { message?: string } }).error?.message;
  }
  return e instanceof Error ? e.message : undefined;
}
