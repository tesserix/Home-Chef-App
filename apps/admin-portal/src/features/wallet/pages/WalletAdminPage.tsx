import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Wallet, Search, ArrowDownLeft, ArrowUpRight, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

interface WalletTxn {
  id: string;
  type: 'credit' | 'debit';
  source: string;
  amount: number;
  balanceAfter: number;
  currency: string;
  reason?: string;
  createdAt: string;
}

interface AdminWalletResponse {
  balance: number;
  currency: string;
  transactions: WalletTxn[];
  count: number;
}

function fmt(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: currency || 'INR' }).format(amount);
  } catch {
    return `${currency || 'INR'} ${amount.toFixed(2)}`;
  }
}

function sourceLabel(s: string): string {
  return s.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function WalletAdminPage() {
  const queryClient = useQueryClient();
  const [userInput, setUserInput] = useState('');
  const [userId, setUserId] = useState('');

  // Adjust form
  const [adjType, setAdjType] = useState<'credit' | 'debit'>('credit');
  const [amount, setAmount] = useState('');
  const [reason, setReason] = useState('');

  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin-wallet', userId],
    queryFn: () => apiClient.get<AdminWalletResponse>(`/admin/wallet/${userId}`),
    enabled: userId !== '',
  });
  const wallet = data as unknown as AdminWalletResponse | undefined;

  const adjustMutation = useMutation({
    mutationFn: (payload: { type: 'credit' | 'debit'; amount: number; reason: string }) =>
      apiClient.post(`/admin/wallet/${userId}/adjust`, payload),
    onSuccess: () => {
      toast.success('Wallet adjusted');
      setAmount('');
      setReason('');
      queryClient.invalidateQueries({ queryKey: ['admin-wallet', userId] });
    },
    onError: (e: unknown) => {
      const msg =
        e && typeof e === 'object' && 'error' in e
          ? (e as { error?: { message?: string } }).error?.message
          : undefined;
      toast.error(msg || 'Failed to adjust wallet');
    },
  });

  const amountNum = Number(amount);
  const canAdjust = userId !== '' && amountNum > 0 && reason.trim().length >= 3 && !adjustMutation.isPending;

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Customer Wallets</h1>
        <p className="page-description">
          Look up a customer&apos;s store-credit balance and make an audited adjustment (goodwill
          credit, manual cashback, correction).
        </p>
      </div>

      {/* Lookup */}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          setUserId(userInput.trim());
        }}
        className="flex flex-col gap-3 sm:flex-row sm:items-end"
      >
        <div className="flex-1 max-w-md">
          <label htmlFor="wallet-user" className="mb-1 block text-sm font-medium text-foreground">
            Customer user ID
          </label>
          <div className="relative">
            <Search aria-hidden="true" className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              id="wallet-user"
              type="text"
              value={userInput}
              onChange={(e) => setUserInput(e.target.value)}
              placeholder="UUID of the customer"
              className="h-10 w-full rounded-lg border border-input bg-card pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
        </div>
        <Button type="submit" variant="primary" disabled={userInput.trim() === ''}>
          Look up
        </Button>
      </form>

      {userId !== '' && isLoading && (
        <div className="flex items-center justify-center py-12 text-muted-foreground">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
        </div>
      )}

      {userId !== '' && isError && (
        <div className="rounded-lg border border-border bg-card p-6 text-sm text-muted-foreground">
          No wallet found for that user id.
        </div>
      )}

      {wallet && (
        <>
          {/* Balance + adjust */}
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-xl border border-border bg-card p-6">
              <div className="flex items-center gap-3">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success/10 text-success">
                  <Wallet className="h-6 w-6" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Balance</p>
                  <p className="text-3xl font-semibold tabular-nums text-foreground">
                    {fmt(wallet.balance, wallet.currency)}
                  </p>
                </div>
              </div>
            </div>

            <div className="rounded-xl border border-border bg-card p-6">
              <h3 className="mb-3 text-sm font-semibold text-foreground">Adjust balance</h3>
              <div className="flex gap-2">
                {(['credit', 'debit'] as const).map((t) => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => setAdjType(t)}
                    className={`rounded-full px-4 py-1.5 text-sm font-medium capitalize ${
                      adjType === t ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    {t}
                  </button>
                ))}
              </div>
              <div className="mt-3 space-y-2">
                <input
                  type="number"
                  min={0}
                  step="0.01"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="Amount"
                  className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm tabular-nums text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
                <input
                  type="text"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="Reason (logged in the audit trail)"
                  className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
                <Button
                  variant="primary"
                  fullWidth
                  isLoading={adjustMutation.isPending}
                  disabled={!canAdjust}
                  onClick={() => adjustMutation.mutate({ type: adjType, amount: amountNum, reason: reason.trim() })}
                >
                  {adjType === 'credit' ? 'Add credit' : 'Deduct credit'}
                </Button>
              </div>
            </div>
          </div>

          {/* History */}
          <div>
            <h3 className="mb-3 text-sm font-semibold text-foreground">Recent transactions</h3>
            {wallet.transactions.length === 0 ? (
              <div className="rounded-lg border border-border bg-card p-6 text-sm text-muted-foreground">
                No transactions yet.
              </div>
            ) : (
              <div className="overflow-hidden rounded-xl border border-border bg-card">
                {wallet.transactions.map((t, i) => {
                  const credit = t.type === 'credit';
                  return (
                    <div
                      key={t.id}
                      className={`flex items-center justify-between px-4 py-3 ${i > 0 ? 'border-t border-border' : ''}`}
                    >
                      <div className="flex items-center gap-3">
                        <div className={`flex h-8 w-8 items-center justify-center rounded-full ${credit ? 'bg-success/10 text-success' : 'bg-muted text-muted-foreground'}`}>
                          {credit ? <ArrowDownLeft className="h-4 w-4" /> : <ArrowUpRight className="h-4 w-4" />}
                        </div>
                        <div>
                          <p className="text-sm font-medium text-foreground">{sourceLabel(t.source)}</p>
                          <p className="text-xs text-muted-foreground">
                            {new Date(t.createdAt).toLocaleString()}
                            {t.reason ? ` · ${t.reason}` : ''}
                          </p>
                        </div>
                      </div>
                      <p className={`text-sm font-medium tabular-nums ${credit ? 'text-success' : 'text-foreground'}`}>
                        {credit ? '+' : '−'}
                        {fmt(t.amount, t.currency)}
                      </p>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
