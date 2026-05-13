import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useFormatPrice } from '@/shared/utils/format-price';
import {
  TrendingUp,
  TrendingDown,
  Users,
  ShoppingBag,
  DollarSign,
  Download,
  Loader2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';

interface AnalyticsData {
  overview: {
    totalRevenue: number;
    revenueChange: number;
    totalOrders: number;
    ordersChange: number;
    avgOrderValue: number;
    aovChange: number;
    activeUsers: number;
    usersChange: number;
  };
  topChefs: Array<{ id: string; name: string; orders: number; revenue: number }>;
  topCuisines: Array<{ name: string; orders: number; percentage: number }>;
  ordersByStatus: Record<string, number>;
  revenueByDay: Array<{ date: string; revenue: number; orders: number }>;
}

const TIME_PERIODS = [
  { value: '7d', label: 'Last 7 Days' },
  { value: '30d', label: 'Last 30 Days' },
  { value: '90d', label: 'Last 90 Days' },
  { value: 'year', label: 'This Year' },
];

export default function AdminAnalyticsPage() {
  const fp = useFormatPrice();

  const [period, setPeriod] = useState('30d');

  const { data, isLoading } = useQuery({
    queryKey: ['admin-analytics', period],
    queryFn: () => apiClient.get<AnalyticsData>('/admin/analytics', { period }),
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
          <h1 className="font-display text-2xl font-semibold text-paper">Analytics</h1>
          <p className="mt-1 text-ink-muted">Platform performance and insights</p>
        </div>
        <div className="flex gap-3">
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="rounded-lg bg-ink border-ink-soft text-paper"
          >
            {TIME_PERIODS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
          <Button
            variant="outline"
            leftIcon={<Download className="h-4 w-4"  aria-hidden="true" />}
            className="border-ink-soft text-paper hover:bg-ink-soft/30 hover:text-paper hover:border-ink-soft"
          >
            Export
          </Button>
        </div>
      </div>

      {/* Overview Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Revenue"
          value={fp(data?.overview.totalRevenue || 0)}
          change={data?.overview.revenueChange}
          icon={DollarSign}
        />
        <MetricCard
          title="Total Orders"
          value={data?.overview.totalOrders?.toLocaleString() || '0'}
          change={data?.overview.ordersChange}
          icon={ShoppingBag}
        />
        <MetricCard
          title="Avg. Order Value"
          value={fp(data?.overview.avgOrderValue || 0)}
          change={data?.overview.aovChange}
          icon={TrendingUp}
        />
        <MetricCard
          title="Active Users"
          value={data?.overview.activeUsers?.toLocaleString() || '0'}
          change={data?.overview.usersChange}
          icon={Users}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Revenue Chart */}
        <div className="rounded-xl bg-ink p-6">
          <h2 className="text-lg font-semibold text-paper">Revenue Overview</h2>
          <div className="mt-6 h-64 flex items-end gap-2">
            {data?.revenueByDay?.slice(-14).map((day, i) => {
              const maxRevenue = Math.max(...(data.revenueByDay?.map((d) => d.revenue) || [1]));
              const height = (day.revenue / maxRevenue) * 100;
              return (
                <div key={i} className="flex-1 flex flex-col items-center">
                  <div
                    className="w-full bg-herb rounded-t hover:bg-herb-soft transition-colors"
                    style={{ height: `${height}%` }}
                    title={fp(day.revenue)}
                  />
                  <span className="mt-2 text-xs text-ink-muted rotate-45">
                    {new Date(day.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                  </span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Orders by Status */}
        <div className="rounded-xl bg-ink p-6">
          <h2 className="text-lg font-semibold text-paper">Orders by Status</h2>
          <div className="mt-6 space-y-4">
            {Object.entries(data?.ordersByStatus || {}).map(([status, count]) => {
              const total = Object.values(data?.ordersByStatus || {}).reduce((a, b) => a + b, 0);
              const percentage = total > 0 ? (count / total) * 100 : 0;
              return (
                <div key={status}>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-ink-muted capitalize">{status.replace('_', ' ')}</span>
                    <span className="text-ink-muted">{count}</span>
                  </div>
                  <div className="h-2 bg-ink-soft rounded-full overflow-hidden">
                    <div
                      className="h-full bg-herb rounded-full"
                      style={{ width: `${percentage}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Top Chefs */}
        <div className="rounded-xl bg-ink p-6">
          <h2 className="text-lg font-semibold text-paper">Top Performing Chefs</h2>
          <div className="mt-4 space-y-3">
            {data?.topChefs?.map((chef, index) => (
              <div key={chef.id} className="flex items-center gap-4 rounded-lg bg-ink-soft/50 p-3">
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-herb/20 text-sm font-medium text-herb-soft">
                  {index + 1}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-paper truncate">{chef.name}</p>
                  <p className="text-sm text-ink-muted">{chef.orders} orders</p>
                </div>
                <p className="font-semibold text-paper">{fp(chef.revenue)}</p>
              </div>
            ))}
            {(!data?.topChefs || data.topChefs.length === 0) && (
              <p className="text-center text-ink-muted py-4">No data available</p>
            )}
          </div>
        </div>

        {/* Top Cuisines */}
        <div className="rounded-xl bg-ink p-6">
          <h2 className="text-lg font-semibold text-paper">Popular Cuisines</h2>
          <div className="mt-4 space-y-4">
            {data?.topCuisines?.map((cuisine) => (
              <div key={cuisine.name}>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-ink-muted">{cuisine.name}</span>
                  <span className="text-ink-muted">{cuisine.orders} orders</span>
                </div>
                <div className="h-2 bg-ink-soft rounded-full overflow-hidden">
                  <div
                    className="h-full bg-info rounded-full"
                    style={{ width: `${cuisine.percentage}%` }}
                  />
                </div>
              </div>
            ))}
            {(!data?.topCuisines || data.topCuisines.length === 0) && (
              <p className="text-center text-ink-muted py-4">No data available</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  change,
  icon: Icon,
}: {
  title: string;
  value: string;
  change?: number;
  icon: typeof DollarSign;
}) {
  return (
    <div className="rounded-xl bg-ink p-6">
      <div className="flex items-center justify-between">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-herb/20">
          <Icon className="h-5 w-5 text-herb-soft" />
        </div>
        {change !== undefined && (
          <span
            className={`flex items-center gap-1 text-sm ${
              change >= 0 ? 'text-herb' : 'text-paprika'
            }`}
          >
            {change >= 0 ? <TrendingUp className="h-4 w-4"  aria-hidden="true" /> : <TrendingDown className="h-4 w-4"  aria-hidden="true" />}
            {Math.abs(change)}%
          </span>
        )}
      </div>
      <p className="mt-4 font-display text-2xl font-semibold text-paper">{value}</p>
      <p className="mt-1 text-sm text-ink-muted">{title}</p>
    </div>
  );
}
