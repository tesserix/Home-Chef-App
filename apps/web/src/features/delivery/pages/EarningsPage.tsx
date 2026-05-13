import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useFormatPrice } from '@/shared/utils/format-price';
import {
  DollarSign,
  TrendingUp,
  TrendingDown,
  Download,
  Package,
  Clock,
  CreditCard,
  Loader2,
  ArrowUpRight,
  Wallet,
  PiggyBank,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';

interface EarningsData {
  currentBalance: number;
  pendingPayout: number;
  lastPayout: {
    amount: number;
    date: string;
    method: string;
  } | null;
  periodStats: {
    totalEarnings: number;
    deliveryFees: number;
    tips: number;
    bonuses: number;
    deductions: number;
    deliveryCount: number;
    averagePerDelivery: number;
    hoursOnline: number;
    earningsPerHour: number;
  };
  dailyEarnings: Array<{
    date: string;
    earnings: number;
    deliveries: number;
    tips: number;
  }>;
  payoutHistory: Array<{
    id: string;
    amount: number;
    date: string;
    method: string;
    status: 'completed' | 'pending' | 'failed';
  }>;
}

const TIME_PERIODS = [
  { value: '7d', label: 'Last 7 Days' },
  { value: '30d', label: 'Last 30 Days' },
  { value: 'month', label: 'This Month' },
  { value: 'year', label: 'This Year' },
];

export default function DeliveryEarningsPage() {
  const fp = useFormatPrice();
  const [period, setPeriod] = useState('30d');

  const { data, isLoading } = useQuery({
    queryKey: ['delivery-earnings', period],
    queryFn: () => apiClient.get<EarningsData>('/delivery/earnings', { period }),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-herb"  aria-hidden="true" />
      </div>
    );
  }

  const maxEarnings = Math.max(...(data?.dailyEarnings?.map((d) => d.earnings) || [1]));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink">Earnings</h1>
          <p className="mt-1 text-ink-muted">Track your income and payouts</p>
        </div>
        <div className="flex gap-3">
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="rounded-lg border-mist-strong text-sm"
          >
            {TIME_PERIODS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
          <Button variant="outline" size="sm" leftIcon={<Download className="h-4 w-4"  aria-hidden="true" />}>
            Export
          </Button>
        </div>
      </div>

      {/* Balance Cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl bg-herb p-6 text-paper">
          <div className="flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-bone/20">
              <Wallet className="h-5 w-5"  aria-hidden="true" />
            </div>
            <ArrowUpRight className="h-5 w-5 opacity-70"  aria-hidden="true" />
          </div>
          <p className="mt-4 font-display text-3xl font-semibold tabular-nums">{fp(data?.currentBalance || 0)}</p>
          <p className="mt-1 text-herb-soft">Available Balance</p>
          <Button
            variant="ghost"
            size="sm"
            fullWidth
            className="mt-4 bg-bone/20 text-paper hover:bg-bone/30 hover:text-paper"
          >
            Cash Out
          </Button>
        </div>

        <div className="rounded-xl bg-bone border border-mist p-6">
          <div className="flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-tint">
              <Clock className="h-5 w-5 text-amber"  aria-hidden="true" />
            </div>
          </div>
          <p className="mt-4 font-display text-2xl font-semibold text-ink">
            {fp(data?.pendingPayout || 0)}
          </p>
          <p className="mt-1 text-ink-muted">Pending Payout</p>
        </div>

        <div className="rounded-xl bg-bone border border-mist p-6">
          <div className="flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info/10">
              <CreditCard className="h-5 w-5 text-info"  aria-hidden="true" />
            </div>
          </div>
          <p className="mt-4 font-display text-2xl font-semibold text-ink">
            {fp(data?.lastPayout?.amount || 0)}
          </p>
          <p className="mt-1 text-ink-muted">
            Last Payout
            {data?.lastPayout && (
              <span className="text-xs ml-1">
                ({new Date(data.lastPayout.date).toLocaleDateString()})
              </span>
            )}
          </p>
        </div>
      </div>

      {/* Period Stats */}
      <div className="rounded-xl bg-bone border border-mist p-6">
        <h2 className="text-lg font-semibold text-ink">Period Summary</h2>

        <div className="mt-6 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <p className="text-sm text-ink-muted">Total Earnings</p>
            <p className="mt-1 font-display text-2xl font-semibold text-ink">
              {fp(data?.periodStats.totalEarnings || 0)}
            </p>
          </div>
          <div>
            <p className="text-sm text-ink-muted">Deliveries</p>
            <p className="mt-1 font-display text-2xl font-semibold text-ink">
              {data?.periodStats.deliveryCount || 0}
            </p>
          </div>
          <div>
            <p className="text-sm text-ink-muted">Avg per Delivery</p>
            <p className="mt-1 font-display text-2xl font-semibold text-ink">
              {fp(data?.periodStats.averagePerDelivery || 0)}
            </p>
          </div>
          <div>
            <p className="text-sm text-ink-muted">Earnings per Hour</p>
            <p className="mt-1 font-display text-2xl font-semibold text-ink">
              {fp(data?.periodStats.earningsPerHour || 0)}
            </p>
          </div>
        </div>

        {/* Earnings Breakdown */}
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div className="rounded-lg bg-paper p-4">
            <div className="flex items-center gap-2">
              <Package className="h-4 w-4 text-ink-muted"  aria-hidden="true" />
              <span className="text-sm text-ink-muted">Delivery Fees</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-ink">
              {fp(data?.periodStats.deliveryFees || 0)}
            </p>
          </div>
          <div className="rounded-lg bg-herb-tint p-4">
            <div className="flex items-center gap-2">
              <DollarSign className="h-4 w-4 text-herb"  aria-hidden="true" />
              <span className="text-sm text-herb">Tips</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-herb">
              +{fp(data?.periodStats.tips || 0)}
            </p>
          </div>
          <div className="rounded-lg bg-info/10 p-4">
            <div className="flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-info"  aria-hidden="true" />
              <span className="text-sm text-info">Bonuses</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-info">
              +{fp(data?.periodStats.bonuses || 0)}
            </p>
          </div>
          <div className="rounded-lg bg-paprika-tint p-4">
            <div className="flex items-center gap-2">
              <TrendingDown className="h-4 w-4 text-paprika"  aria-hidden="true" />
              <span className="text-sm text-paprika">Deductions</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-paprika">
              -{fp(data?.periodStats.deductions || 0)}
            </p>
          </div>
        </div>
      </div>

      {/* Daily Earnings Chart */}
      <div className="rounded-xl bg-bone border border-mist p-6">
        <h2 className="text-lg font-semibold text-ink">Daily Earnings</h2>
        <div className="mt-6">
          <div className="flex h-48 items-end gap-1">
            {data?.dailyEarnings?.slice(-14).map((day, i) => {
              const height = maxEarnings > 0 ? (day.earnings / maxEarnings) * 100 : 0;
              const tipHeight = day.tips > 0 ? (day.tips / day.earnings) * height : 0;
              return (
                <div key={i} className="group relative flex-1 flex flex-col justify-end">
                  <div
                    className="relative w-full rounded-t bg-herb hover:bg-herb-soft transition-colors cursor-pointer"
                    style={{ height: `${height}%`, minHeight: height > 0 ? '4px' : '0' }}
                  >
                    {tipHeight > 0 && (
                      <div
                        className="absolute bottom-0 left-0 right-0 bg-herb rounded-t"
                        style={{ height: `${tipHeight}%` }}
                      />
                    )}
                    {/* Tooltip */}
                    <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block z-10">
                      <div className="rounded-lg bg-ink px-3 py-2 text-xs text-paper whitespace-nowrap">
                        <p className="font-semibold">{fp(day.earnings)}</p>
                        <p className="text-ink-muted">
                          {day.deliveries} deliveries • {fp(day.tips)} tips
                        </p>
                      </div>
                    </div>
                  </div>
                  <span className="mt-2 text-xs text-ink-muted text-center">
                    {new Date(day.date).toLocaleDateString(undefined, { weekday: 'short' })}
                  </span>
                </div>
              );
            })}
          </div>
          <div className="mt-4 flex justify-center gap-6">
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded bg-herb" />
              <span className="text-sm text-ink-muted">Delivery Fees</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded bg-herb" />
              <span className="text-sm text-ink-muted">Tips</span>
            </div>
          </div>
        </div>
      </div>

      {/* Payout History */}
      <div className="rounded-xl bg-bone border border-mist overflow-hidden">
        <div className="p-6">
          <h2 className="text-lg font-semibold text-ink">Payout History</h2>
        </div>

        {data?.payoutHistory && data.payoutHistory.length > 0 ? (
          <div className="divide-y divide-mist">
            {data.payoutHistory.map((payout) => (
              <div
                key={payout.id}
                className="flex items-center justify-between px-6 py-4 hover:bg-paper"
              >
                <div className="flex items-center gap-4">
                  <div
                    className={`flex h-10 w-10 items-center justify-center rounded-full ${
                      payout.status === 'completed'
                        ? 'bg-herb-tint'
                        : payout.status === 'pending'
                        ? 'bg-amber-tint'
                        : 'bg-paprika-tint'
                    }`}
                  >
                    <CreditCard
                      className={`h-5 w-5 ${
                        payout.status === 'completed'
                          ? 'text-herb'
                          : payout.status === 'pending'
                          ? 'text-amber'
                          : 'text-paprika'
                      }`}
                     aria-hidden="true" />
                  </div>
                  <div>
                    <p className="font-medium text-ink">{payout.method}</p>
                    <p className="text-sm text-ink-muted">
                      {new Date(payout.date).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-ink">{fp(payout.amount)}</p>
                  <span
                    className={`text-xs font-medium ${
                      payout.status === 'completed'
                        ? 'text-herb'
                        : payout.status === 'pending'
                        ? 'text-amber'
                        : 'text-paprika'
                    }`}
                  >
                    {payout.status.charAt(0).toUpperCase() + payout.status.slice(1)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="px-6 pb-6 text-center text-ink-muted">
            <PiggyBank className="mx-auto h-12 w-12 text-ink-muted" />
            <p className="mt-2">No payouts yet</p>
          </div>
        )}
      </div>

      {/* Hours Summary */}
      <div className="rounded-xl bg-herb p-6 text-paper">
        <h2 className="text-lg font-semibold">Time Summary</h2>
        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-herb-tint">Hours Online</p>
            <p className="mt-1 font-display text-3xl font-semibold tabular-nums">{data?.periodStats.hoursOnline || 0}h</p>
          </div>
          <div>
            <p className="text-herb-tint">Earnings per Hour</p>
            <p className="mt-1 font-display text-3xl font-semibold tabular-nums">
              {fp(data?.periodStats.earningsPerHour || 0)}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
