import { useState } from 'react';
import { motion } from 'framer-motion';
import { BarChart3, TrendingUp, Clock, PieChart } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

interface AnalyticsData {
  summary?: { orders: number; revenue: number; aov: number; repeatRate: number; prevRevenue: number };
  orderTrends: { labels: string[]; data: number[] };
  revenueTrends: { labels: string[]; data: number[] };
  popularItems: { name: string; orders: number; percentage: number }[];
  peakHours: { hour: string; orders: number }[];
  revenueByCategory: { category: string; revenue: number; percentage: number }[];
}

interface SubscriptionMetrics {
  activePlans: number;
  subscribers: number;
  churnRate: number;
  adherenceRate: number;
}

interface DemandForecast {
  date: string;
  subscriptionMeals: number;
  subscriptionLunch: number;
  subscriptionDinner: number;
  alaCarteForecast: number;
  totalExpected: number;
  likelyDishes: { name: string; expected: number }[];
  basis: string;
}

const inr = (n: number) => `₹${Math.round(n).toLocaleString('en-IN')}`;

export default function AnalyticsPage() {
  const [period, setPeriod] = useState<'7d' | '30d' | '90d'>('7d');

  const { data: analytics, isLoading } = useQuery({
    queryKey: ['chef-analytics', period],
    queryFn: () => apiClient.get<AnalyticsData>('/chef/analytics', { period }),
  });
  const { data: subs } = useQuery({
    queryKey: ['chef-analytics-subscriptions'],
    queryFn: () => apiClient.get<SubscriptionMetrics>('/chef/analytics/subscriptions'),
  });
  const { data: forecast } = useQuery({
    queryKey: ['chef-analytics-forecast'],
    queryFn: () => apiClient.get<DemandForecast>('/chef/analytics/forecast'),
  });

  if (isLoading || !analytics) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-herb border-t-transparent" />
      </div>
    );
  }

  const orderData = analytics.orderTrends?.data ?? [];
  const revenueData = analytics.revenueTrends?.data ?? [];
  const peakHours = analytics.peakHours ?? [];
  const popularItems = analytics.popularItems ?? [];
  const revenueByCategory = analytics.revenueByCategory ?? [];

  const maxOrders = orderData.length > 0 ? Math.max(...orderData, 1) : 1;
  const maxRevenue = revenueData.length > 0 ? Math.max(...revenueData, 1) : 1;
  const maxPeakHour = peakHours.length > 0 ? Math.max(...peakHours.map((h) => h.orders ?? 0), 1) : 1;

  return (
    <motion.div variants={staggerContainer} initial="hidden" animate="visible" className="space-y-6">
      {/* Header */}
      <motion.div variants={fadeInUp} className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="page-header">
          <h1 className="page-title">Analytics</h1>
          <p className="page-description">Insights into your kitchen performance</p>
        </div>
        <div
          role="radiogroup"
          aria-label="Analytics period"
          className="flex rounded-lg border border-mist bg-bone p-1"
        >
          {(['7d', '30d', '90d'] as const).map((p) => {
            const isActive = period === p;
            const label = p === '7d' ? '7 Days' : p === '30d' ? '30 Days' : '90 Days';
            return (
              <button
                key={p}
                type="button"
                role="radio"
                aria-checked={isActive}
                onClick={() => setPeriod(p)}
                className={`rounded-md px-4 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
                  isActive ? 'bg-herb text-paper' : 'text-ink-soft hover:text-ink'
                }`}
              >
                {label}
              </button>
            );
          })}
        </div>
      </motion.div>

      {/* Summary stat cards (#228) */}
      {analytics.summary && (
        <motion.div variants={fadeInUp} className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard label="Revenue" value={inr(analytics.summary.revenue)} />
          <StatCard label="Orders" value={String(analytics.summary.orders)} />
          <StatCard label="Avg order" value={inr(analytics.summary.aov)} />
          <StatCard label="Repeat customers" value={`${analytics.summary.repeatRate}%`} sub="ordered 2+ times" />
        </motion.div>
      )}

      {/* Tomorrow's demand forecast (#230) */}
      {forecast && forecast.totalExpected > 0 && (
        <motion.div variants={fadeInUp} className="rounded-xl bg-herb-tint p-6">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-muted">Tomorrow's demand</h2>
          <p className="mt-1 font-display text-3xl font-semibold text-ink">~{forecast.totalExpected} meals</p>
          <p className="mt-1 text-sm text-ink-soft">
            {forecast.subscriptionMeals} subscription ({forecast.subscriptionLunch} lunch · {forecast.subscriptionDinner} dinner)
            {' '}+ ~{forecast.alaCarteForecast} à-la-carte
          </p>
          {forecast.likelyDishes.length > 0 && (
            <p className="mt-2 text-xs text-ink-soft">
              Likely: {forecast.likelyDishes.map((d) => `${d.name} (~${d.expected})`).join(', ')}
            </p>
          )}
          <p className="mt-2 text-xs text-ink-muted">{forecast.basis}</p>
        </motion.div>
      )}

      {/* Subscription health (#229) */}
      {subs && (subs.activePlans > 0 || subs.subscribers > 0) && (
        <motion.div variants={fadeInUp} className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard label="Active plans" value={String(subs.activePlans)} />
          <StatCard label="Subscribers" value={String(subs.subscribers)} />
          <StatCard label="Churn" value={`${subs.churnRate}%`} />
          <StatCard label="Adherence" value={`${subs.adherenceRate}%`} sub="days delivered" />
        </motion.div>
      )}

      {/* Order Trends */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
        <div className="mb-4 flex items-center gap-2">
          <BarChart3 aria-hidden="true" className="h-5 w-5 text-herb" />
          <h2 className="text-lg font-semibold text-ink">Order Trends</h2>
        </div>
        <div className="flex items-end gap-2" style={{ height: 200 }}>
          {(analytics.orderTrends?.labels ?? []).map((label, i) => (
            <div key={label} className="flex flex-1 flex-col items-center gap-1">
              <span className="text-xs font-medium text-ink-soft">
                {(orderData[i] ?? 0)}
              </span>
              <div
                className="w-full rounded-t-md bg-herb transition-all"
                style={{
                  height: `${((orderData[i] ?? 0) / maxOrders) * 160}px`,
                  minHeight: 4,
                }}
              />
              <span className="text-xs text-ink-muted">{label}</span>
            </div>
          ))}
        </div>
      </motion.div>

      {/* Revenue Trends */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
        <div className="mb-4 flex items-center gap-2">
          <TrendingUp aria-hidden="true" className="h-5 w-5 text-herb" />
          <h2 className="text-lg font-semibold text-ink">Revenue Trends</h2>
        </div>
        <div className="flex items-end gap-2" style={{ height: 200 }}>
          {(analytics.revenueTrends?.labels ?? []).map((label, i) => (
            <div key={label} className="flex flex-1 flex-col items-center gap-1">
              <span className="text-xs font-medium text-ink-soft">
                ${(revenueData[i] ?? 0)}
              </span>
              <div
                className="w-full rounded-t-md bg-herb transition-all"
                style={{
                  height: `${((revenueData[i] ?? 0) / maxRevenue) * 160}px`,
                  minHeight: 4,
                }}
              />
              <span className="text-xs text-ink-muted">{label}</span>
            </div>
          ))}
        </div>
      </motion.div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Popular Items */}
        <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
          <div className="mb-4 flex items-center gap-2">
            <PieChart className="h-5 w-5 text-info" />
            <h2 className="text-lg font-semibold text-ink">Popular Items</h2>
          </div>
          <div className="space-y-4">
            {popularItems.map((item, i) => (
              <div key={item.name} className="flex items-center gap-3">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-mist text-xs font-medium text-ink-soft">
                  {i + 1}
                </span>
                <div className="flex-1">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-ink">{item.name}</span>
                    <span className="text-sm text-ink-muted">{item.orders} orders</span>
                  </div>
                  <div className="mt-1 h-2 rounded-full bg-mist">
                    <div
                      className="h-2 rounded-full bg-info"
                      style={{ width: `${item.percentage}%` }}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </motion.div>

        {/* Revenue by Category */}
        <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
          <div className="mb-4 flex items-center gap-2">
            <PieChart className="h-5 w-5 text-herb" />
            <h2 className="text-lg font-semibold text-ink">Revenue by Category</h2>
          </div>
          <div className="space-y-4">
            {revenueByCategory.map((cat) => (
              <div key={cat.category}>
                <div className="flex items-center justify-between text-sm">
                  <span className="font-medium text-ink">{cat.category}</span>
                  <span className="text-ink-muted">${(cat.revenue ?? 0).toFixed(2)} ({cat.percentage ?? 0}%)</span>
                </div>
                <div className="mt-1 h-2 rounded-full bg-mist">
                  <div
                    className="h-2 rounded-full bg-herb"
                    style={{ width: `${cat.percentage}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </motion.div>
      </div>

      {/* Peak Hours */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-mist bg-bone p-6">
        <div className="mb-4 flex items-center gap-2">
          <Clock aria-hidden="true" className="h-5 w-5 text-amber" />
          <h2 className="text-lg font-semibold text-ink">Peak Hours</h2>
        </div>
        <div className="flex items-end gap-1 overflow-x-auto" style={{ height: 200 }}>
          {peakHours.map((item) => (
            // min-w-[40px] is intentional: keeps hour-by-hour bars legible on narrow viewports (overflow-x-auto handles spillover).
            <div key={item.hour} className="flex min-w-[40px] flex-1 flex-col items-center gap-1">
              <span className="text-xs font-medium text-ink-soft">{item.orders}</span>
              <div
                className={`w-full rounded-t-md transition-all ${
                  item.orders >= maxPeakHour * 0.8
                    ? 'bg-paprika'
                    : item.orders >= maxPeakHour * 0.5
                      ? 'bg-amber'
                      : 'bg-amber'
                }`}
                style={{
                  height: `${(item.orders / maxPeakHour) * 160}px`,
                  minHeight: 4,
                }}
              />
              <span className="text-[10px] text-ink-muted">{item.hour}</span>
            </div>
          ))}
        </div>
      </motion.div>
    </motion.div>
  );
}

// Reusable headline stat tile (#49).
function StatCard({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-xl border border-mist bg-bone p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">{label}</p>
      <p className="mt-1 font-display text-2xl font-semibold tabular-nums text-ink">{value}</p>
      {sub ? <p className="mt-0.5 text-xs text-ink-muted">{sub}</p> : null}
    </div>
  );
}
