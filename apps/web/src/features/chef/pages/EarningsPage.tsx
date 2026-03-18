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
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Earnings</h1>
          <p className="mt-1 text-gray-600">Track your income and payouts</p>
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
          <button className="btn-outline">
            <Download className="h-4 w-4" />
            Export
          </button>
        </div>
      </div>

      {/* Balance Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl bg-gradient-to-br from-brand-500 to-brand-600 p-6 text-white">
          <div className="flex items-center justify-between">
            <span className="text-brand-100">Available Balance</span>
            <DollarSign className="h-5 w-5 text-brand-200" />
          </div>
          <p className="mt-4 text-3xl font-bold">{fp(earnings?.balance || 0)}</p>
          <button className="mt-4 rounded-lg bg-white/20 px-4 py-2 text-sm font-medium hover:bg-white/30 transition-colors">
            Withdraw Funds
          </button>
        </div>

        <div className="rounded-xl bg-white p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-gray-500">Pending</span>
            <Calendar className="h-5 w-5 text-gray-400" />
          </div>
          <p className="mt-4 text-3xl font-bold text-gray-900">
            {fp(earnings?.pendingBalance || 0)}
          </p>
          <p className="mt-1 text-sm text-gray-500">From recent orders</p>
        </div>

        <div className="rounded-xl bg-white p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-gray-500">This Month</span>
            {(earnings?.monthlyChange || 0) >= 0 ? (
              <TrendingUp className="h-5 w-5 text-green-500" />
            ) : (
              <TrendingDown className="h-5 w-5 text-red-500" />
            )}
          </div>
          <p className="mt-4 text-3xl font-bold text-gray-900">
            {fp(earnings?.thisMonth || 0)}
          </p>
          <p className={`mt-1 flex items-center gap-1 text-sm ${
            (earnings?.monthlyChange || 0) >= 0 ? 'text-green-600' : 'text-red-600'
          }`}>
            {(earnings?.monthlyChange || 0) >= 0 ? (
              <ArrowUpRight className="h-4 w-4" />
            ) : (
              <ArrowDownRight className="h-4 w-4" />
            )}
            {Math.abs(earnings?.monthlyChange || 0)}% vs last month
          </p>
        </div>

        <div className="rounded-xl bg-white p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-gray-500">Total Earned</span>
            <DollarSign className="h-5 w-5 text-gray-400" />
          </div>
          <p className="mt-4 text-3xl font-bold text-gray-900">
            {fp(earnings?.totalEarnings || 0)}
          </p>
          <p className="mt-1 text-sm text-gray-500">All time</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Chart Placeholder */}
        <div className="lg:col-span-2 rounded-xl bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900">Earnings Overview</h2>
          <div className="mt-6 h-64 flex items-center justify-center border-2 border-dashed border-gray-200 rounded-lg">
            <div className="text-center text-gray-400">
              <TrendingUp className="mx-auto h-12 w-12" />
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
                    className="bg-brand-500 rounded-t"
                    style={{
                      height: `${Math.max(10, (day.amount / (earnings.thisMonth / 7)) * 80)}%`,
                    }}
                  />
                </div>
                <p className="mt-2 text-xs text-gray-500">
                  {new Date(day.date).toLocaleDateString('en-US', { weekday: 'short' })}
                </p>
              </div>
            ))}
          </div>
        </div>

        {/* Top Items */}
        <div className="rounded-xl bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900">Top Selling Items</h2>
          <div className="mt-4 space-y-4">
            {earnings?.topItems?.map((item, index) => (
              <div key={item.name} className="flex items-center gap-3">
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-100 text-sm font-medium text-brand-600">
                  {index + 1}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-gray-900 truncate">{item.name}</p>
                  <p className="text-sm text-gray-500">{item.count} orders</p>
                </div>
                <p className="font-semibold text-gray-900">{fp(item.revenue)}</p>
              </div>
            ))}
            {(!earnings?.topItems || earnings.topItems.length === 0) && (
              <p className="text-center text-gray-500 py-4">No data available</p>
            )}
          </div>
        </div>
      </div>

      {/* Stats Row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl bg-white p-6 shadow-sm">
          <p className="text-sm text-gray-500">Total Orders</p>
          <p className="mt-2 text-2xl font-bold text-gray-900">{earnings?.ordersCount || 0}</p>
        </div>
        <div className="rounded-xl bg-white p-6 shadow-sm">
          <p className="text-sm text-gray-500">Average Order Value</p>
          <p className="mt-2 text-2xl font-bold text-gray-900">
            {fp(earnings?.avgOrderValue || 0)}
          </p>
        </div>
        <div className="rounded-xl bg-white p-6 shadow-sm">
          <p className="text-sm text-gray-500">Last Month</p>
          <p className="mt-2 text-2xl font-bold text-gray-900">
            {fp(earnings?.lastMonth || 0)}
          </p>
        </div>
        <div className="rounded-xl bg-white p-6 shadow-sm">
          <p className="text-sm text-gray-500">Platform Fee</p>
          <p className="mt-2 text-2xl font-bold text-gray-900">15%</p>
        </div>
      </div>

      {/* Recent Payouts */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">Recent Payouts</h2>
          <button className="text-sm text-brand-600 hover:text-brand-700">View all</button>
        </div>

        <div className="mt-4 overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b text-left text-sm text-gray-500">
                <th className="pb-3 font-medium">Date</th>
                <th className="pb-3 font-medium">Amount</th>
                <th className="pb-3 font-medium">Method</th>
                <th className="pb-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {earnings?.recentPayouts?.map((payout) => (
                <tr key={payout.id} className="text-sm">
                  <td className="py-4 text-gray-600">
                    {new Date(payout.date).toLocaleDateString('en-US', {
                      month: 'short',
                      day: 'numeric',
                      year: 'numeric',
                    })}
                  </td>
                  <td className="py-4 font-semibold text-gray-900">
                    {fp(payout.amount)}
                  </td>
                  <td className="py-4 text-gray-600">
                    <span className="flex items-center gap-2">
                      <CreditCard className="h-4 w-4" />
                      Bank Transfer
                    </span>
                  </td>
                  <td className="py-4">
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        payout.status === 'completed'
                          ? 'bg-green-100 text-green-800'
                          : payout.status === 'pending'
                          ? 'bg-yellow-100 text-yellow-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      {payout.status}
                    </span>
                  </td>
                </tr>
              ))}
              {(!earnings?.recentPayouts || earnings.recentPayouts.length === 0) && (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-gray-500">
                    No payouts yet
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Payment Settings */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">Payment Settings</h2>
        <div className="mt-4 flex items-center justify-between rounded-lg border p-4">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100">
              <CreditCard className="h-6 w-6 text-gray-600" />
            </div>
            <div>
              <p className="font-medium text-gray-900">Bank Account ••••4242</p>
              <p className="text-sm text-gray-500">Default payout method</p>
            </div>
          </div>
          <button className="btn-outline">Edit</button>
        </div>
      </div>
    </div>
  );
}
