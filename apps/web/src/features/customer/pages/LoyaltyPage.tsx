import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Award, Flame, Sparkles, ArrowDownLeft, ArrowUpRight, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Card, Button } from '@/shared/components/ui';
import { useFormatPrice } from '@/shared/utils/format-price';
import { cn } from '@/shared/utils/cn';

// Mirrors the loyalty API (#40): GET /customer/loyalty (balance + tier + streak +
// config), /loyalty/transactions (points ledger), POST /loyalty/redeem (→ wallet).

interface LoyaltyConfig {
  enabled: boolean;
  redeemRate: number;
  minRedeem: number;
  streakThreshold: number;
  streakBonus: number;
  tierSilverAt: number;
  tierGoldAt: number;
}

interface LoyaltyResponse {
  balance: number;
  lifetimePoints: number;
  tier: 'bronze' | 'silver' | 'gold';
  currentStreak: number;
  longestStreak: number;
  config: LoyaltyConfig;
}

interface LoyaltyTransaction {
  id: string;
  type: 'credit' | 'debit';
  source: string;
  points: number;
  pointsAfter: number;
  orderId?: string;
  reason?: string;
  createdAt: string;
}

interface LoyaltyTxnPage {
  data: LoyaltyTransaction[];
}

interface RedeemResponse {
  pointsRedeemed: number;
  walletCredited: number;
}

const TIER_LABEL: Record<string, string> = { bronze: 'Bronze', silver: 'Silver', gold: 'Gold' };

function sourceLabel(source: string): string {
  return source.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatPoints(p: number): string {
  return Math.round(p).toLocaleString('en-IN');
}

export default function LoyaltyPage() {
  const fp = useFormatPrice();
  const queryClient = useQueryClient();

  const { data: loyalty, isLoading } = useQuery({
    queryKey: ['loyalty'],
    queryFn: () => apiClient.get<LoyaltyResponse>('/customer/loyalty'),
  });
  const { data: txnData, isLoading: txnLoading } = useQuery({
    queryKey: ['loyalty-transactions'],
    queryFn: () => apiClient.get<LoyaltyTxnPage>('/customer/loyalty/transactions', { page: 1, limit: 50 }),
  });

  const redeem = useMutation({
    mutationFn: (points: number) => apiClient.post<RedeemResponse>('/customer/loyalty/redeem', { points }),
    onSuccess: (res) => {
      toast.success(`Redeemed ${formatPoints(res.pointsRedeemed)} points for ${fp(res.walletCredited)} wallet credit`);
      queryClient.invalidateQueries({ queryKey: ['loyalty'] });
      queryClient.invalidateQueries({ queryKey: ['loyalty-transactions'] });
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
      queryClient.invalidateQueries({ queryKey: ['wallet-transactions'] });
    },
    onError: (e: unknown) => {
      const msg = e && typeof e === 'object' && 'error' in e ? (e as { error?: { message?: string } }).error?.message : undefined;
      toast.error(typeof (e as { error?: unknown }).error === 'string' ? (e as { error: string }).error : msg || 'Could not redeem points');
    },
  });

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  const balance = loyalty?.balance ?? 0;
  const cfg = loyalty?.config;
  const redeemable = Math.floor(balance);
  const minRedeem = cfg?.minRedeem ?? 100;
  const redeemRate = cfg?.redeemRate ?? 0.1;
  const canRedeem = (cfg?.enabled ?? true) && redeemable >= minRedeem;
  const redeemValue = Math.round(redeemable * redeemRate * 100) / 100;
  const pointsToMin = Math.max(0, Math.ceil(minRedeem - redeemable));
  const threshold = cfg?.streakThreshold ?? 7;
  const streak = loyalty?.currentStreak ?? 0;
  const daysToBonus = threshold > 0 ? threshold - (streak % threshold) : 0;
  const txns = txnData?.data ?? [];

  const onRedeem = () => {
    if (!canRedeem) return;
    if (!window.confirm(`Redeem ${formatPoints(redeemable)} points for ${fp(redeemValue)} of wallet credit?`)) return;
    redeem.mutate(redeemable);
  };

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-3xl">
        <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">Rewards</h1>
        <p className="mt-1 text-ink-muted">Earn points on every order and redeem them for wallet credit.</p>

        {/* Points balance + tier */}
        <Card className="mt-6 bg-bone p-6 shadow-1">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-herb-tint text-herb">
                <Sparkles aria-hidden="true" className="h-6 w-6" />
              </div>
              <div>
                <p className="text-sm text-ink-muted">Points balance</p>
                <p className="font-display text-3xl font-semibold tabular-nums text-ink">{formatPoints(balance)}</p>
                <p className="text-sm text-ink-soft">Worth {fp(redeemValue)} in wallet credit</p>
              </div>
            </div>
            <span className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-3 py-1 text-sm font-semibold text-herb">
              <Award aria-hidden="true" className="h-4 w-4" />
              {TIER_LABEL[loyalty?.tier ?? 'bronze']}
            </span>
          </div>

          <div className="mt-5 flex items-center gap-3 border-t border-mist pt-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-mist text-ink-soft">
              <Flame aria-hidden="true" className={cn('h-5 w-5', streak > 0 && 'text-herb')} />
            </div>
            <div className="flex-1">
              <p className="font-medium text-ink">
                {streak > 0 ? `${streak}-day meal streak` : 'No active streak'}
              </p>
              <p className="text-sm text-ink-muted">
                {streak > 0
                  ? `${daysToBonus} more ${daysToBonus === 1 ? 'day' : 'days'} for a ${formatPoints(cfg?.streakBonus ?? 50)}-point bonus`
                  : 'Keep a daily meal subscription going to earn streak bonuses'}
              </p>
            </div>
          </div>

          <div className="mt-5">
            <Button variant="primary" className="w-full" disabled={!canRedeem} isLoading={redeem.isPending} onClick={onRedeem}>
              {canRedeem
                ? `Redeem ${formatPoints(redeemable)} points → ${fp(redeemValue)}`
                : `Earn ${formatPoints(pointsToMin)} more points to redeem`}
            </Button>
          </div>
        </Card>

        <h2 className="mt-8 text-lg font-semibold text-ink">History</h2>
        {txnLoading ? (
          <div className="mt-4 flex justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-herb" />
          </div>
        ) : txns.length === 0 ? (
          <Card className="mt-4 bg-bone p-8 text-center text-ink-muted shadow-1">
            No points yet. You'll earn points on every delivered order.
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
                    {formatPoints(t.points)}
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
