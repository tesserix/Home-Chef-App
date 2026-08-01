import { useState } from 'react';
import { motion } from 'framer-motion';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Gift, Copy, Check, Sparkles, Users } from 'lucide-react';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { formatCurrency } from '@/shared/utils/format';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

// ---- API contract types --------------------------------------------------
// GET /chef/rewards (handlers/chef_rewards.go GetChefRewards)

interface ReferralProgress {
  kitchenName: string;
  status: 'pending' | 'rewarded' | 'rejected';
  deliveredOrders: number;
  milestoneOrders: number;
  reward: number;
  createdAt: string;
  rewardedAt?: string;
}

interface RewardsResponse {
  referral: {
    enabled: boolean;
    code: string;
    link: string;
    referrerAmount: number;
    refereeAmount: number;
    milestoneOrders: number;
    currency: string;
    totalEarned: number;
    referrals: ReferralProgress[];
  };
  loyalty: {
    enabled: boolean;
    points: number;
    lifetimePoints: number;
    earnRate: number;
    redeemRate: number;
    minConvertPoints: number;
    canConvert: boolean;
    convertValue: number;
  };
  pendingPayoutCredit: number;
}

interface ConvertResponse {
  convertedPoints: number;
  cashback: number;
  message: string;
}

function CopyButton({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => {
        navigator.clipboard.writeText(value);
        setCopied(true);
        toast.success(`${label} copied`);
        setTimeout(() => setCopied(false), 2000);
      }}
      leftIcon={copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
    >
      {copied ? 'Copied' : 'Copy'}
    </Button>
  );
}

const referralStatusBadge: Record<ReferralProgress['status'], { label: string; variant: 'success' | 'warning' | 'destructive' }> = {
  rewarded: { label: 'Rewarded', variant: 'success' },
  pending: { label: 'In progress', variant: 'warning' },
  rejected: { label: 'Rejected', variant: 'destructive' },
};

export default function RewardsPage() {
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery<RewardsResponse>({
    queryKey: ['chef', 'rewards'],
    queryFn: () => apiClient.get<RewardsResponse>('/chef/rewards'),
  });

  const convertMutation = useMutation({
    mutationFn: () => apiClient.post<ConvertResponse>('/chef/rewards/convert'),
    onSuccess: (res) => {
      toast.success(`Converted ${res.convertedPoints.toLocaleString('en-IN')} points into ${formatCurrency(res.cashback)}`, {
        description: res.message,
      });
      queryClient.invalidateQueries({ queryKey: ['chef', 'rewards'] });
    },
    onError: (err: unknown) => {
      toast.error(err instanceof Error ? err.message : 'Could not convert your points');
    },
  });

  if (isLoading || !data) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  const { referral, loyalty } = data;
  const progressPct = Math.min(100, (loyalty.points / loyalty.minConvertPoints) * 100);

  return (
    <motion.div variants={staggerContainer} initial="hidden" animate="visible" className="space-y-6">
      <motion.div variants={fadeInUp}>
        <h1 className="font-display text-2xl font-semibold text-ink">Rewards</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Earn points on every delivered order and cash for every kitchen you bring on board
        </p>
      </motion.div>

      {data.pendingPayoutCredit > 0 && (
        <motion.div variants={fadeInUp}>
          <Card className="border-primary/30 bg-primary/5">
            <div className="flex items-center gap-3">
              <Sparkles className="h-5 w-5 text-primary" />
              <p className="text-sm text-ink">
                <span className="font-semibold">{formatCurrency(data.pendingPayoutCredit)}</span>{' '}
                in earned rewards will be added to your next weekly payout.
              </p>
            </div>
          </Card>
        </motion.div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Loyalty points */}
        <motion.div variants={fadeInUp}>
          <Card className="h-full">
            <div className="flex items-center gap-2">
              <Gift className="h-5 w-5 text-ink-muted" />
              <h2 className="text-lg font-semibold text-ink">Reward points</h2>
            </div>
            <p className="mt-4 font-display text-4xl font-semibold tabular-nums text-ink">
              {loyalty.points.toLocaleString('en-IN')}
            </p>
            <p className="mt-1 text-sm text-ink-muted">
              worth {formatCurrency(loyalty.convertValue)} · lifetime{' '}
              {loyalty.lifetimePoints.toLocaleString('en-IN')} pts
            </p>

            <div className="mt-4">
              <div className="h-2 overflow-hidden rounded-full bg-secondary">
                <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${progressPct}%` }} />
              </div>
              <p className="mt-2 text-xs text-ink-muted">
                Convert to cashback at {loyalty.minConvertPoints.toLocaleString('en-IN')} points — paid
                into your weekly payout
              </p>
            </div>

            <Button
              className="mt-4"
              disabled={!loyalty.canConvert || convertMutation.isPending}
              isLoading={convertMutation.isPending}
              onClick={() => convertMutation.mutate()}
            >
              {loyalty.canConvert
                ? `Convert to ${formatCurrency(loyalty.convertValue)} cashback`
                : `${Math.max(0, loyalty.minConvertPoints - loyalty.points).toLocaleString('en-IN')} points to go`}
            </Button>
            <p className="mt-3 text-xs text-ink-muted">
              You earn {loyalty.earnRate} point{loyalty.earnRate === 1 ? '' : 's'} per ₹1 of delivered
              orders.
            </p>
          </Card>
        </motion.div>

        {/* Referral */}
        <motion.div variants={fadeInUp}>
          <Card className="h-full">
            <div className="flex items-center gap-2">
              <Users className="h-5 w-5 text-ink-muted" />
              <h2 className="text-lg font-semibold text-ink">Refer a chef</h2>
            </div>
            <p className="mt-1 text-sm text-ink-muted">
              You earn <span className="font-medium text-ink">{formatCurrency(referral.referrerAmount)}</span>{' '}
              and they earn <span className="font-medium text-ink">{formatCurrency(referral.refereeAmount)}</span>{' '}
              once their kitchen completes {referral.milestoneOrders} orders.
            </p>

            <div className="mt-4 flex items-center gap-3 rounded-lg border border-border bg-secondary/40 px-4 py-3">
              <span className="flex-1 font-mono text-lg font-semibold tracking-widest text-ink">
                {referral.code}
              </span>
              <CopyButton value={referral.code} label="Code" />
            </div>
            <div className="mt-2 flex items-center gap-3">
              <p className="flex-1 truncate text-xs text-ink-muted">{referral.link}</p>
              <CopyButton value={referral.link} label="Link" />
            </div>

            {referral.totalEarned > 0 && (
              <p className="mt-4 text-sm text-ink">
                Earned so far:{' '}
                <span className="font-semibold tabular-nums">{formatCurrency(referral.totalEarned)}</span>
              </p>
            )}
          </Card>
        </motion.div>
      </div>

      {/* Referral progress list */}
      <motion.div variants={fadeInUp}>
        <Card>
          <h2 className="text-lg font-semibold text-ink">Your referrals</h2>
          {referral.referrals.length === 0 ? (
            <p className="mt-4 text-sm text-ink-muted">
              No referrals yet — share your code with a home chef you know.
            </p>
          ) : (
            <div className="mt-4 divide-y divide-border">
              {referral.referrals.map((r, i) => {
                const badge = referralStatusBadge[r.status];
                return (
                  <div key={`${r.kitchenName}-${i}`} className="flex items-center gap-4 py-3">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-ink">{r.kitchenName || 'New kitchen'}</p>
                      <p className="text-xs text-ink-muted">
                        Joined {format(new Date(r.createdAt), 'd MMM yyyy')}
                        {r.status === 'pending' &&
                          ` · ${r.deliveredOrders}/${r.milestoneOrders} orders delivered`}
                      </p>
                    </div>
                    {r.status === 'rewarded' && (
                      <span className="text-sm font-semibold tabular-nums text-ink">
                        {formatCurrency(r.reward)}
                      </span>
                    )}
                    <Badge variant={badge.variant}>{badge.label}</Badge>
                  </div>
                );
              })}
            </div>
          )}
        </Card>
      </motion.div>
    </motion.div>
  );
}
