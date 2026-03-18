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
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
      </div>
    );
  }

  const maxEarnings = Math.max(...(data?.dailyEarnings?.map((d) => d.earnings) || [1]));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Earnings</h1>
          <p className="mt-1 text-gray-500">Track your income and payouts</p>
        </div>
        <div className="flex gap-3">
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="rounded-lg border-gray-300 text-sm"
          >
            {TIME_PERIODS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
          <button className="btn-outline flex items-center gap-2 text-sm">
            <Download className="h-4 w-4" />
            Export
          </button>
        </div>
      </div>

      {/* Balance Cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl bg-gradient-to-br from-green-500 to-green-600 p-6 text-white">
          <div className="flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-white/20">
              <Wallet className="h-5 w-5" />
            </div>
            <ArrowUpRight className="h-5 w-5 opacity-70" />
          </div>
          <p className="mt-4 text-3xl font-bold">{fp(data?.currentBalance || 0)}</p>
          <p className="mt-1 text-green-100">Available Balance</p>
          <button className="mt-4 w-full rounded-lg bg-white/20 py-2 text-sm font-medium hover:bg-white/30 transition-colors">
            Cash Out
          </button>
        </div>

        <div className="rounded-xl bg-white border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-yellow-100">
              <Clock className="h-5 w-5 text-yellow-600" />
            </div>
          </div>
          <p className="mt-4 text-2xl font-bold text-gray-900">
            {fp(data?.pendingPayout || 0)}
          </p>
          <p className="mt-1 text-gray-500">Pending Payout</p>
        </div>

        <div className="rounded-xl bg-white border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100">
              <CreditCard className="h-5 w-5 text-blue-600" />
            </div>
          </div>
          <p className="mt-4 text-2xl font-bold text-gray-900">
            {fp(data?.lastPayout?.amount || 0)}
          </p>
          <p className="mt-1 text-gray-500">
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
      <div className="rounded-xl bg-white border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900">Period Summary</h2>

        <div className="mt-6 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <p className="text-sm text-gray-500">Total Earnings</p>
            <p className="mt-1 text-2xl font-bold text-gray-900">
              {fp(data?.periodStats.totalEarnings || 0)}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Deliveries</p>
            <p className="mt-1 text-2xl font-bold text-gray-900">
              {data?.periodStats.deliveryCount || 0}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Avg per Delivery</p>
            <p className="mt-1 text-2xl font-bold text-gray-900">
              {fp(data?.periodStats.averagePerDelivery || 0)}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Earnings per Hour</p>
            <p className="mt-1 text-2xl font-bold text-gray-900">
              {fp(data?.periodStats.earningsPerHour || 0)}
            </p>
          </div>
        </div>

        {/* Earnings Breakdown */}
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div className="rounded-lg bg-gray-50 p-4">
            <div className="flex items-center gap-2">
              <Package className="h-4 w-4 text-gray-500" />
              <span className="text-sm text-gray-500">Delivery Fees</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-gray-900">
              {fp(data?.periodStats.deliveryFees || 0)}
            </p>
          </div>
          <div className="rounded-lg bg-green-50 p-4">
            <div className="flex items-center gap-2">
              <DollarSign className="h-4 w-4 text-green-600" />
              <span className="text-sm text-green-600">Tips</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-green-700">
              +{fp(data?.periodStats.tips || 0)}
            </p>
          </div>
          <div className="rounded-lg bg-purple-50 p-4">
            <div className="flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-purple-600" />
              <span className="text-sm text-purple-600">Bonuses</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-purple-700">
              +{fp(data?.periodStats.bonuses || 0)}
            </p>
          </div>
          <div className="rounded-lg bg-red-50 p-4">
            <div className="flex items-center gap-2">
              <TrendingDown className="h-4 w-4 text-red-600" />
              <span className="text-sm text-red-600">Deductions</span>
            </div>
            <p className="mt-2 text-lg font-semibold text-red-700">
              -{fp(data?.periodStats.deductions || 0)}
            </p>
          </div>
        </div>
      </div>

      {/* Daily Earnings Chart */}
      <div className="rounded-xl bg-white border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900">Daily Earnings</h2>
        <div className="mt-6">
          <div className="flex h-48 items-end gap-1">
            {data?.dailyEarnings?.slice(-14).map((day, i) => {
              const height = maxEarnings > 0 ? (day.earnings / maxEarnings) * 100 : 0;
              const tipHeight = day.tips > 0 ? (day.tips / day.earnings) * height : 0;
              return (
                <div key={i} className="group relative flex-1 flex flex-col justify-end">
                  <div
                    className="relative w-full rounded-t bg-brand-500 hover:bg-brand-400 transition-colors cursor-pointer"
                    style={{ height: `${height}%`, minHeight: height > 0 ? '4px' : '0' }}
                  >
                    {tipHeight > 0 && (
                      <div
                        className="absolute bottom-0 left-0 right-0 bg-green-400 rounded-t"
                        style={{ height: `${tipHeight}%` }}
                      />
                    )}
                    {/* Tooltip */}
                    <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block z-10">
                      <div className="rounded-lg bg-gray-900 px-3 py-2 text-xs text-white whitespace-nowrap">
                        <p className="font-semibold">{fp(day.earnings)}</p>
                        <p className="text-gray-400">
                          {day.deliveries} deliveries • {fp(day.tips)} tips
                        </p>
                      </div>
                    </div>
                  </div>
                  <span className="mt-2 text-xs text-gray-500 text-center">
                    {new Date(day.date).toLocaleDateString('en-US', { weekday: 'short' })}
                  </span>
                </div>
              );
            })}
          </div>
          <div className="mt-4 flex justify-center gap-6">
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded bg-brand-500" />
              <span className="text-sm text-gray-500">Delivery Fees</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-3 w-3 rounded bg-green-400" />
              <span className="text-sm text-gray-500">Tips</span>
            </div>
          </div>
        </div>
      </div>

      {/* Payout History */}
      <div className="rounded-xl bg-white border border-gray-200 overflow-hidden">
        <div className="p-6">
          <h2 className="text-lg font-semibold text-gray-900">Payout History</h2>
        </div>

        {data?.payoutHistory && data.payoutHistory.length > 0 ? (
          <div className="divide-y divide-gray-100">
            {data.payoutHistory.map((payout) => (
              <div
                key={payout.id}
                className="flex items-center justify-between px-6 py-4 hover:bg-gray-50"
              >
                <div className="flex items-center gap-4">
                  <div
                    className={`flex h-10 w-10 items-center justify-center rounded-full ${
                      payout.status === 'completed'
                        ? 'bg-green-100'
                        : payout.status === 'pending'
                        ? 'bg-yellow-100'
                        : 'bg-red-100'
                    }`}
                  >
                    <CreditCard
                      className={`h-5 w-5 ${
                        payout.status === 'completed'
                          ? 'text-green-600'
                          : payout.status === 'pending'
                          ? 'text-yellow-600'
                          : 'text-red-600'
                      }`}
                    />
                  </div>
                  <div>
                    <p className="font-medium text-gray-900">{payout.method}</p>
                    <p className="text-sm text-gray-500">
                      {new Date(payout.date).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-gray-900">{fp(payout.amount)}</p>
                  <span
                    className={`text-xs font-medium ${
                      payout.status === 'completed'
                        ? 'text-green-600'
                        : payout.status === 'pending'
                        ? 'text-yellow-600'
                        : 'text-red-600'
                    }`}
                  >
                    {payout.status.charAt(0).toUpperCase() + payout.status.slice(1)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="px-6 pb-6 text-center text-gray-500">
            <PiggyBank className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-2">No payouts yet</p>
          </div>
        )}
      </div>

      {/* Hours Summary */}
      <div className="rounded-xl bg-gradient-to-r from-brand-500 to-brand-600 p-6 text-white">
        <h2 className="text-lg font-semibold">Time Summary</h2>
        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-brand-100">Hours Online</p>
            <p className="mt-1 text-3xl font-bold">{data?.periodStats.hoursOnline || 0}h</p>
          </div>
          <div>
            <p className="text-brand-100">Earnings per Hour</p>
            <p className="mt-1 text-3xl font-bold">
              {fp(data?.periodStats.earningsPerHour || 0)}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
