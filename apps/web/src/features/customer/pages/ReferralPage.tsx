import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Gift, Copy, Check, Share2, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Card, Button } from '@/shared/components/ui';
import { useFormatPrice } from '@/shared/utils/format-price';

// Refer & Earn (#38) — invite friends with a unique code/link; the reward lands
// in the store-credit wallet. Reward amounts + stats come from the API.

interface ReferralInfo {
  code: string;
  link: string;
  enabled: boolean;
  referrerReward: number;
  refereeReward: number;
  currency: string;
  stats: { rewardedCount: number; pendingCount: number; totalEarned: number };
}
interface ReferralHistoryItem {
  refereeName: string;
  status: 'pending' | 'rewarded' | 'rejected';
  reward: number;
  createdAt: string;
}

export default function ReferralPage() {
  const fp = useFormatPrice();
  const [copied, setCopied] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ['referral'],
    queryFn: () => apiClient.get<ReferralInfo>('/customer/referral'),
  });
  const { data: history } = useQuery({
    queryKey: ['referral', 'history'],
    queryFn: () => apiClient.get<{ data: ReferralHistoryItem[] }>('/customer/referral/history'),
  });

  const copyLink = async () => {
    if (!data) return;
    try {
      await navigator.clipboard.writeText(data.link);
      setCopied(true);
      toast.success('Invite link copied');
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error('Could not copy — long-press the link to copy.');
    }
  };

  const share = async () => {
    if (!data) return;
    const text = `Join me on Fe3dr! Use my code ${data.code} and we both get ${fp(data.refereeReward)} in credit. ${data.link}`;
    if (navigator.share) {
      try {
        await navigator.share({ title: 'Fe3dr', text, url: data.link });
      } catch {
        /* cancelled */
      }
    } else {
      await copyLink();
    }
  };

  if (isLoading || !data) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" aria-hidden="true" />
      </div>
    );
  }

  const items = history?.data ?? [];

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-2xl space-y-6">
        {/* Hero */}
        <Card variant="filled" padding="lg" className="text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-herb-tint">
            <Gift className="h-6 w-6 text-herb" aria-hidden="true" />
          </div>
          <h1 className="mt-3 font-display text-display-sm text-ink">
            Give {fp(data.refereeReward)}, get {fp(data.referrerReward)}
          </h1>
          <p className="mt-2 text-ink-soft">
            Your friend gets {fp(data.refereeReward)} off their first order. You get{' '}
            {fp(data.referrerReward)} once they order.
          </p>
        </Card>

        {/* Code + actions */}
        <Card padding="lg" className="text-center">
          <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">Your code</p>
          <p className="mt-2 font-display text-3xl tracking-[0.2em] text-ink">{data.code}</p>
          <div className="mt-4 flex flex-col gap-2 sm:flex-row sm:justify-center">
            <Button variant="primary" onClick={share} leftIcon={<Share2 className="h-4 w-4" />}>
              Share invite
            </Button>
            <Button variant="outline" onClick={copyLink} leftIcon={copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}>
              {copied ? 'Copied' : 'Copy link'}
            </Button>
          </div>
        </Card>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-4">
          <Card className="text-center">
            <p className="font-display text-2xl text-ink tabular-nums">{data.stats.rewardedCount}</p>
            <p className="text-sm text-ink-soft">Friends joined</p>
          </Card>
          <Card className="text-center">
            <p className="font-display text-2xl text-ink tabular-nums">{fp(data.stats.totalEarned)}</p>
            <p className="text-sm text-ink-soft">Credit earned</p>
          </Card>
        </div>

        {/* History */}
        {items.length > 0 && (
          <Card padding="none" className="overflow-hidden">
            <h2 className="border-b border-mist px-4 py-3 text-sm font-semibold text-ink">Your referrals</h2>
            <ul className="divide-y divide-mist">
              {items.map((h, i) => (
                <li key={i} className="flex items-center justify-between px-4 py-3">
                  <span className="text-ink">{h.refereeName}</span>
                  <span className={`text-sm font-semibold ${h.status === 'rewarded' ? 'text-herb' : 'text-ink-soft'}`}>
                    {h.status === 'rewarded' ? `+${fp(h.reward)}` : h.status === 'pending' ? 'Pending' : '—'}
                  </span>
                </li>
              ))}
            </ul>
          </Card>
        )}
      </div>
    </div>
  );
}
