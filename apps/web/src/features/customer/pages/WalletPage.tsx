import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Wallet as WalletIcon, ArrowDownLeft, ArrowUpRight, Loader2, Gift, ChevronRight } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Card } from '@/shared/components/ui';
import { useFormatPrice } from '@/shared/utils/format-price';
import { cn } from '@/shared/utils/cn';

// Mirrors GET /customer/wallet (flat balance) and /wallet/transactions (#33).
interface WalletBalanceResponse {
  balance: number;
  currency: string;
}

interface WalletTransaction {
  id: string;
  type: 'credit' | 'debit';
  source: string;
  amount: number;
  balanceAfter: number;
  currency: string;
  orderId?: string;
  reason?: string;
  createdAt: string;
}

interface WalletTxnPage {
  data: WalletTransaction[];
  pagination?: { total: number };
}

function sourceLabel(source: string): string {
  return source.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function WalletPage() {
  const fp = useFormatPrice();

  const { data: wallet, isLoading: balLoading } = useQuery({
    queryKey: ['wallet'],
    queryFn: () => apiClient.get<WalletBalanceResponse>('/customer/wallet'),
  });
  const { data: txnData, isLoading: txnLoading } = useQuery({
    queryKey: ['wallet-transactions'],
    queryFn: () =>
      apiClient.get<WalletTxnPage>('/customer/wallet/transactions', { page: 1, limit: 50 }),
  });

  const balance = wallet?.balance ?? 0;
  const currency = wallet?.currency ?? 'INR';
  const txns = txnData?.data ?? [];

  if (balLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-3xl">
        <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">Wallet</h1>
        <p className="mt-1 text-ink-muted">Your store credit and transaction history.</p>

        <Card className="mt-6 bg-bone p-6 shadow-1">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-herb-tint text-herb">
              <WalletIcon aria-hidden="true" className="h-6 w-6" />
            </div>
            <div>
              <p className="text-sm text-ink-muted">Available balance</p>
              <p className="font-display text-3xl font-semibold tabular-nums text-ink">
                {fp(balance, { currency })}
              </p>
            </div>
          </div>
        </Card>

        {/* Refer & Earn CTA (#38) — the reward lands here in the wallet. */}
        <Link
          to="/referral"
          className="mt-4 flex items-center gap-3 rounded-xl bg-herb-tint p-4 transition-colors hover:bg-herb-tint/80"
        >
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-paper text-herb">
            <Gift aria-hidden="true" className="h-5 w-5" />
          </div>
          <div className="flex-1">
            <p className="font-semibold text-ink">Refer &amp; Earn</p>
            <p className="text-sm text-ink-soft">Invite friends — you both get wallet credit.</p>
          </div>
          <ChevronRight aria-hidden="true" className="h-5 w-5 text-ink-muted" />
        </Link>

        <h2 className="mt-8 text-lg font-semibold text-ink">Transactions</h2>
        {txnLoading ? (
          <div className="mt-4 flex justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-herb" />
          </div>
        ) : txns.length === 0 ? (
          <Card className="mt-4 bg-bone p-8 text-center text-ink-muted shadow-1">
            No transactions yet. Refunds and credits will appear here.
          </Card>
        ) : (
          <div className="mt-4 space-y-2">
            {txns.map((t) => {
              const credit = t.type === 'credit';
              return (
                <Card key={t.id} className="flex items-center justify-between bg-bone p-4 shadow-1">
                  <div className="flex items-center gap-3">
                    <div
                      className={cn(
                        'flex h-9 w-9 items-center justify-center rounded-full',
                        credit ? 'bg-herb-tint text-herb' : 'bg-mist text-ink-soft'
                      )}
                    >
                      {credit ? (
                        <ArrowDownLeft aria-hidden="true" className="h-4 w-4" />
                      ) : (
                        <ArrowUpRight aria-hidden="true" className="h-4 w-4" />
                      )}
                    </div>
                    <div>
                      <p className="font-medium text-ink">{sourceLabel(t.source)}</p>
                      <p className="text-xs text-ink-muted">
                        {new Date(t.createdAt).toLocaleDateString()}
                        {t.reason ? ` · ${t.reason}` : ''}
                      </p>
                    </div>
                  </div>
                  <p className={cn('font-medium tabular-nums', credit ? 'text-herb' : 'text-ink')}>
                    {credit ? '+' : '−'}
                    {fp(t.amount, { currency: t.currency })}
                  </p>
                </Card>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
