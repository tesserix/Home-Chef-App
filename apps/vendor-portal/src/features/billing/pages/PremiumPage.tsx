import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import { Sparkles, Check, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

// Premium chef tier — upgrade flow + perks (#44). Prices/perks come from the API
// (admin-configurable); never hardcoded. Mirrors the mobile-vendor premium screen.

type Tier = 'standard' | 'premium';

interface Subscription {
  tier: Tier;
  status: string;
  billingInterval: string;
}
interface SubscriptionResponse {
  subscription: Subscription | null;
  status: string;
}
interface PlanOption {
  interval: 'monthly' | 'quarterly' | 'yearly';
  amount: number;
  currency: string;
  savingsPercent?: number;
}
interface PlansResponse {
  premiumPlans: PlanOption[];
  premiumPerks: string[];
  currency: string;
}

const CURRENCY_SYMBOL: Record<string, string> = { INR: '₹', USD: '$' };
const money = (amount: number, currency: string) =>
  `${CURRENCY_SYMBOL[currency] ?? ''}${Math.round(amount).toLocaleString('en-IN')}`;
const INTERVAL_LABEL: Record<PlanOption['interval'], string> = {
  monthly: 'Per month',
  quarterly: 'Per quarter',
  yearly: 'Per year',
};

export default function PremiumPage() {
  const queryClient = useQueryClient();
  const [pending, setPending] = useState<Tier | null>(null);

  const { data: sub, isLoading: subLoading } = useQuery<SubscriptionResponse>({
    queryKey: ['chef', 'subscription'],
    queryFn: () => apiClient.get<SubscriptionResponse>('/chef/subscription'),
  });
  const { data: plans, isLoading: plansLoading } = useQuery<PlansResponse>({
    queryKey: ['chef', 'subscription', 'plans'],
    queryFn: () => apiClient.get<PlansResponse>('/chef/subscription/plans'),
  });

  const changeTier = useMutation({
    mutationFn: (tier: Tier) => apiClient.put('/chef/subscription/tier', { tier }),
    onSuccess: (_data, tier) => {
      setPending(null);
      queryClient.invalidateQueries({ queryKey: ['chef', 'subscription'] });
      toast.success(tier === 'premium' ? 'Welcome to Premium — perks are active.' : 'Switched back to the standard tier.');
    },
    onError: (err: unknown) => {
      setPending(null);
      const status = (err as { status?: number })?.status;
      toast.error(status === 404 ? 'Start your subscription first, then upgrade.' : 'Could not change tier. Please try again.');
    },
  });

  const isPremium = sub?.subscription?.tier === 'premium';
  const loading = subLoading || plansLoading;

  const onChange = (tier: Tier) => {
    setPending(tier);
    changeTier.mutate(tier);
  };

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" aria-hidden="true" />
      </div>
    );
  }

  return (
    <motion.div variants={staggerContainer} initial="hidden" animate="visible" className="mx-auto max-w-3xl space-y-6 p-4 sm:p-6">
      {/* Hero */}
      <motion.div variants={fadeInUp}>
        <Card padding="lg" className={isPremium ? 'bg-herb text-paper' : ''}>
          <div className="flex flex-col items-center gap-2 text-center">
            <div className={`flex h-12 w-12 items-center justify-center rounded-full ${isPremium ? 'bg-paper/20' : 'bg-herb-tint'}`}>
              <Sparkles className={`h-6 w-6 ${isPremium ? 'text-paper' : 'text-herb'}`} aria-hidden="true" />
            </div>
            <h1 className={`font-display text-display-xs ${isPremium ? 'text-paper' : 'text-ink'}`}>
              {isPremium ? "You're on Premium" : 'Upgrade to Premium'}
            </h1>
            <p className={isPremium ? 'text-paper/80' : 'text-ink-soft'}>
              {isPremium
                ? 'Your Verified-Pro perks are active.'
                : 'Stand out, rank higher, and keep more of every order.'}
            </p>
          </div>
        </Card>
      </motion.div>

      {/* Perks */}
      <motion.div variants={fadeInUp}>
        <h2 className="mb-2 text-xs font-semibold uppercase tracking-wider text-ink-muted">What you get</h2>
        <Card>
          <ul className="space-y-3">
            {(plans?.premiumPerks ?? []).map((perk) => (
              <li key={perk} className="flex items-center gap-3">
                <Check className="h-5 w-5 shrink-0 text-herb" strokeWidth={2.5} aria-hidden="true" />
                <span className="text-ink">{perk}</span>
              </li>
            ))}
          </ul>
        </Card>
      </motion.div>

      {/* Pricing — admin-configurable, fetched from the API */}
      <motion.div variants={fadeInUp}>
        <h2 className="mb-2 text-xs font-semibold uppercase tracking-wider text-ink-muted">Premium pricing</h2>
        <Card>
          <div className="divide-y divide-mist">
            {(plans?.premiumPlans ?? []).map((p) => (
              <div key={p.interval} className="flex items-center justify-between py-3 first:pt-0 last:pb-0">
                <div>
                  <p className="text-ink">{INTERVAL_LABEL[p.interval]}</p>
                  {p.savingsPercent && p.savingsPercent > 0 ? (
                    <p className="text-sm font-medium text-herb">Save {Math.round(p.savingsPercent)}%</p>
                  ) : null}
                </div>
                <p className="text-lg font-semibold tabular-nums text-ink">{money(p.amount, plans?.currency ?? 'INR')}</p>
              </div>
            ))}
          </div>
          <p className="mt-3 text-xs text-ink-muted">
            Billed on your current cycle. Premium replaces the standard plan fee.
          </p>
        </Card>
      </motion.div>

      {/* CTA */}
      <motion.div variants={fadeInUp}>
        {isPremium ? (
          <Button variant="outline" fullWidth disabled={changeTier.isPending} onClick={() => onChange('standard')}>
            {pending === 'standard' ? 'Switching…' : 'Switch to Standard'}
          </Button>
        ) : (
          <Button variant="primary" fullWidth disabled={changeTier.isPending} onClick={() => onChange('premium')}>
            {pending === 'premium' ? 'Upgrading…' : 'Upgrade to Premium'}
          </Button>
        )}
      </motion.div>
    </motion.div>
  );
}
