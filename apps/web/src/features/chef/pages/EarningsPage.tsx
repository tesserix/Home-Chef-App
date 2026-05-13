import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  DollarSign,
  TrendingUp,
  TrendingDown,
  Calendar,
  Download,
  ArrowUpRight,
  ArrowDownRight,
  CreditCard,
  Loader2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Button } from '@/shared/components/ui';

interface EarningsData {
  balance: number;
  pendingBalance: number;
  totalEarnings: number;
  thisMonth: number;
  lastMonth: number;
  monthlyChange: number;
  ordersCount: number;
  avgOrderValue: number;
  topItems: Array<{ name: string; count: number; revenue: number }>;
  recentPayouts: Array<{ id: string; amount: number; date: string; status: string }>;
  dailyEarnings: Array<{ date: string; amount: number; orders: number }>;
}

const TIME_PERIODS = [
  { value: '7d', label: 'Last 7 Days' },
  { value: '30d', label: 'Last 30 Days' },
  { value: '90d', label: 'Last 90 Days' },
  { value: 'year', label: 'This Year' },
];

export default function ChefEarningsPage() {
  const [timePeriod, setTimePeriod] = useState('30d');
  const fp = useFormatPrice();

  const { data: earnings, isLoading } = useQuery({
    queryKey: ['chef-earnings', timePeriod],
    queryFn: () => apiClient.get<EarningsData>('/chef/earnings', { period: timePeriod }),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-herb"  aria-hidden="true" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink">Earnings</h1>
          <p className="mt-1 text-ink-soft">Track your income and payouts</p>
        </div>
        <div className="flex gap-3">
          <select
            value={timePeriod}
            onChange={(e) => setTimePeriod(e.target.value)}
            className="input-base"
          >
            {TIME_PERIODS.map((period) => (
              <option key={period.value} value={period.value}>
                {period.label}
              </option>
            ))}
          </select>
          <Button variant="outline" leftIcon={<Download className="h-4 w-4"  aria-hidden="true" />}>
            Export
          </Button>
        </div>
      </div>

      {/* Balance Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl bg-herb p-6 text-paper">
          <div className="flex items-center justify-between">
            <span className="text-herb-tint">Available Balance</span>
            <DollarSign className="h-5 w-5 text-herb-tint"  aria-hidden="true" />
          </div>
          <p className="mt-4 font-display text-3xl font-semibold tabular-nums">{fp(earnings?.balance || 0)}</p>
          <Button
            variant="ghost"
            size="sm"
            className="mt-4 bg-bone/20 text-paper hover:bg-bone/30 hover:text-paper"
          >
            Withdraw Funds
          </Button>
        </div>

        <div className="rounded-xl bg-bone p-6 shadow-1">
          <div className="flex items-center justify-between">
            <span className="text-ink-muted">Pending</span>
            <Calendar className="h-5 w-5 text-ink-muted"  aria-hidden="true" />
          </div>
          <p className="mt-4 font-display text-3xl font-semibold tabular-nums text-ink">
            {fp(earnings?.pendingBalance || 0)}
          </p>
          <p className="mt-1 text-sm text-ink-muted">From recent orders</p>
        </div>

        <div className="rounded-xl bg-bone p-6 shadow-1">
          <div className="flex items-center justify-between">
            <span className="text-ink-muted">This Month</span>
            {(earnings?.monthlyChange || 0) >= 0 ? (
              <TrendingUp className="h-5 w-5 text-herb"  aria-hidden="true" />
            ) : (
              <TrendingDown className="h-5 w-5 text-paprika"  aria-hidden="true" />
            )}
          </div>
          <p className="mt-4 font-display text-3xl font-semibold tabular-nums text-ink">
            {fp(earnings?.thisMonth || 0)}
          </p>
          <p className={`mt-1 flex items-center gap-1 text-sm ${
            (earnings?.monthlyChange || 0) >= 0 ? 'text-herb' : 'text-paprika'
          }`}>
            {(earnings?.monthlyChange || 0) >= 0 ? (
              <ArrowUpRight className="h-4 w-4"  aria-hidden="true" />
            ) : (
              <ArrowDownRight className="h-4 w-4"  aria-hidden="true" />
            )}
            {Math.abs(earnings?.monthlyChange || 0)}% vs last month
          </p>
        </div>

        <div className="rounded-xl bg-bone p-6 shadow-1">
          <div className="flex items-center justify-between">
            <span className="text-ink-muted">Total Earned</span>
            <DollarSign className="h-5 w-5 text-ink-muted"  aria-hidden="true" />
          </div>
          <p className="mt-4 font-display text-3xl font-semibold tabular-nums text-ink">
            {fp(earnings?.totalEarnings || 0)}
          </p>
          <p className="mt-1 text-sm text-ink-muted">All time</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Chart Placeholder */}
        <div className="lg:col-span-2 rounded-xl bg-bone p-6 shadow-1">
          <h2 className="text-lg font-semibold text-ink">Earnings Overview</h2>
          <div className="mt-6 h-64 flex items-center justify-center border-2 border-dashed border-mist rounded-lg">
            <div className="text-center text-ink-muted">
              <TrendingUp className="mx-auto h-12 w-12"  aria-hidden="true" />
              <p className="mt-2">Chart visualization</p>
              <p className="text-sm">Would display earnings over time</p>
            </div>
          </div>

          {/* Simple bar representation */}
          <div className="mt-6 grid grid-cols-7 gap-2">
            {earnings?.dailyEarnings?.slice(0, 7).map((day, i) => (
              <div key={i} className="text-center">
                <div className="h-24 flex flex-col justify-end">
                  <div
                    className="bg-herb rounded-t"
                    style={{
                      height: `${Math.max(10, (day.amount / (earnings.thisMonth / 7)) * 80)}%`,
                    }}
                  />
                </div>
                <p className="mt-2 text-xs text-ink-muted">
                  {new Date(day.date).toLocaleDateString(undefined, { weekday: 'short' })}
                </p>
              </div>
            ))}
          </div>
        </div>

        {/* Top Items */}
        <div className="rounded-xl bg-bone p-6 shadow-1">
          <h2 className="text-lg font-semibold text-ink">Top Selling Items</h2>
          <div className="mt-4 space-y-4">
            {earnings?.topItems?.map((item, index) => (
              <div key={item.name} className="flex items-center gap-3">
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-herb-tint text-sm font-medium text-herb">
                  {index + 1}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-ink truncate">{item.name}</p>
                  <p className="text-sm text-ink-muted">{item.count} orders</p>
                </div>
                <p className="font-semibold text-ink">{fp(item.revenue)}</p>
              </div>
            ))}
            {(!earnings?.topItems || earnings.topItems.length === 0) && (
              <p className="text-center text-ink-muted py-4">No data available</p>
            )}
          </div>
        </div>
      </div>

      {/* Stats Row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl bg-bone p-6 shadow-1">
          <p className="text-sm text-ink-muted">Total Orders</p>
          <p className="mt-2 font-display text-2xl font-semibold text-ink">{earnings?.ordersCount || 0}</p>
        </div>
        <div className="rounded-xl bg-bone p-6 shadow-1">
          <p className="text-sm text-ink-muted">Average Order Value</p>
          <p className="mt-2 font-display text-2xl font-semibold text-ink">
            {fp(earnings?.avgOrderValue || 0)}
          </p>
        </div>
        <div className="rounded-xl bg-bone p-6 shadow-1">
          <p className="text-sm text-ink-muted">Last Month</p>
          <p className="mt-2 font-display text-2xl font-semibold text-ink">
            {fp(earnings?.lastMonth || 0)}
          </p>
        </div>
        <div className="rounded-xl bg-bone p-6 shadow-1">
          <p className="text-sm text-ink-muted">Platform Fee</p>
          <p className="mt-2 font-display text-2xl font-semibold text-ink">15%</p>
        </div>
      </div>

      {/* Recent Payouts */}
      <div className="rounded-xl bg-bone p-6 shadow-1">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-ink">Recent Payouts</h2>
          <Button variant="link" size="sm">View all</Button>
        </div>

        <div className="mt-4 overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b text-left text-sm text-ink-muted">
                <th className="pb-3 font-medium">Date</th>
                <th className="pb-3 font-medium">Amount</th>
                <th className="pb-3 font-medium">Method</th>
                <th className="pb-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {earnings?.recentPayouts?.map((payout) => (
                <tr key={payout.id} className="text-sm">
                  <td className="py-4 text-ink-soft">
                    {new Date(payout.date).toLocaleDateString(undefined, {
                      month: 'short',
                      day: 'numeric',
                      year: 'numeric',
                    })}
                  </td>
                  <td className="py-4 font-semibold text-ink">
                    {fp(payout.amount)}
                  </td>
                  <td className="py-4 text-ink-soft">
                    <span className="flex items-center gap-2">
                      <CreditCard className="h-4 w-4"  aria-hidden="true" />
                      Bank Transfer
                    </span>
                  </td>
                  <td className="py-4">
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        payout.status === 'completed'
                          ? 'bg-herb-tint text-herb'
                          : payout.status === 'pending'
                          ? 'bg-amber-tint text-amber'
                          : 'bg-mist text-ink'
                      }`}
                    >
                      {payout.status}
                    </span>
                  </td>
                </tr>
              ))}
              {(!earnings?.recentPayouts || earnings.recentPayouts.length === 0) && (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-ink-muted">
                    No payouts yet
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Payment Settings */}
      <div className="rounded-xl bg-bone p-6 shadow-1">
        <h2 className="text-lg font-semibold text-ink">Payment Settings</h2>
        <div className="mt-4 flex items-center justify-between rounded-lg border p-4">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-mist">
              <CreditCard className="h-6 w-6 text-ink-soft"  aria-hidden="true" />
            </div>
            <div>
              <p className="font-medium text-ink">Bank Account ••••4242</p>
              <p className="text-sm text-ink-muted">Default payout method</p>
            </div>
          </div>
          <Button variant="outline" size="sm">Edit</Button>
        </div>
      </div>
    </div>
  );
}
