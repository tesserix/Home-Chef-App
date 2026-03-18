import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { DollarSign, TrendingUp, Truck, Banknote } from 'lucide-react';
import type { EarningsData } from '@/shared/types';
import { PageLoader } from '@/shared/components/LoadingScreen';
import { useState } from 'react';

export default function EarningsPage() {
  const [period, setPeriod] = useState('week');

  const { data: earnings, isLoading } = useQuery({
    queryKey: ['delivery-earnings', period],
    queryFn: () => apiClient.get<EarningsData>('/delivery/earnings', { period }),
  });

  if (isLoading) return <PageLoader />;

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Earnings</h1>
        <p className="page-description">Track your delivery income</p>
      </div>

      {/* Period Selector */}
      <div className="flex gap-2 rounded-xl bg-muted p-1">
        {['day', 'week', 'month', 'all'].map((p) => (
          <button
            key={p}
            onClick={() => setPeriod(p)}
            className={`flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-all ${
              period === p
                ? 'bg-card text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {p.charAt(0).toUpperCase() + p.slice(1)}
          </button>
        ))}
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 text-muted-foreground mb-2">
            <DollarSign className="h-4 w-4" />
            <span className="text-xs font-medium">Total Earnings</span>
          </div>
          <p className="text-2xl font-bold text-foreground">
            ${(earnings?.totalEarnings ?? 0).toFixed(2)}
          </p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 text-muted-foreground mb-2">
            <Truck className="h-4 w-4" />
            <span className="text-xs font-medium">Deliveries</span>
          </div>
          <p className="text-2xl font-bold text-foreground">{earnings?.totalDeliveries ?? 0}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 text-muted-foreground mb-2">
            <Banknote className="h-4 w-4" />
            <span className="text-xs font-medium">Tips</span>
          </div>
          <p className="text-2xl font-bold text-foreground">
            ${(earnings?.totalTips ?? 0).toFixed(2)}
          </p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 text-muted-foreground mb-2">
            <TrendingUp className="h-4 w-4" />
            <span className="text-xs font-medium">Avg/Delivery</span>
          </div>
          <p className="text-2xl font-bold text-foreground">
            ${(earnings?.avgPerDelivery ?? 0).toFixed(2)}
          </p>
        </div>
      </div>

      {/* Daily Breakdown */}
      {earnings?.daily && earnings.daily.length > 0 && (
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="font-semibold text-foreground mb-4">Daily Breakdown</h3>
          <div className="space-y-3">
            {earnings.daily.map((day) => (
              <div key={day.date} className="flex items-center justify-between border-b border-border pb-3 last:border-0">
                <div>
                  <p className="text-sm font-medium text-foreground">{day.date}</p>
                  <p className="text-xs text-muted-foreground">{day.deliveries} deliveries</p>
                </div>
                <p className="text-sm font-semibold text-foreground">${day.earnings.toFixed(2)}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
