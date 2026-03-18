import { useState } from 'react';
import { motion } from 'framer-motion';
import { BarChart3, TrendingUp, Clock, PieChart } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

interface AnalyticsData {
  orderTrends: { labels: string[]; data: number[] };
  revenueTrends: { labels: string[]; data: number[] };
  popularItems: { name: string; orders: number; percentage: number }[];
  peakHours: { hour: string; orders: number }[];
  revenueByCategory: { category: string; revenue: number; percentage: number }[];
}

export default function AnalyticsPage() {
  const [period, setPeriod] = useState<'7d' | '30d' | '90d'>('7d');

  const { data: analytics, isLoading } = useQuery({
    queryKey: ['chef-analytics', period],
    queryFn: () => apiClient.get<AnalyticsData>('/chef/analytics', { period }),
  });

  if (isLoading || !analytics) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-500 border-t-transparent" />
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
        <div className="flex rounded-lg border border-gray-200 bg-white p-1">
          {(['7d', '30d', '90d'] as const).map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                period === p ? 'bg-brand-500 text-white' : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {p === '7d' ? '7 Days' : p === '30d' ? '30 Days' : '90 Days'}
            </button>
          ))}
        </div>
      </motion.div>

      {/* Order Trends */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
        <div className="mb-4 flex items-center gap-2">
          <BarChart3 className="h-5 w-5 text-brand-500" />
          <h2 className="text-lg font-semibold text-gray-900">Order Trends</h2>
        </div>
        <div className="flex items-end gap-2" style={{ height: 200 }}>
          {(analytics.orderTrends?.labels ?? []).map((label, i) => (
            <div key={label} className="flex flex-1 flex-col items-center gap-1">
              <span className="text-xs font-medium text-gray-700">
                {(orderData[i] ?? 0)}
              </span>
              <div
                className="w-full rounded-t-md bg-brand-500 transition-all"
                style={{
                  height: `${((orderData[i] ?? 0) / maxOrders) * 160}px`,
                  minHeight: 4,
                }}
              />
              <span className="text-xs text-gray-500">{label}</span>
            </div>
          ))}
        </div>
      </motion.div>

      {/* Revenue Trends */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
        <div className="mb-4 flex items-center gap-2">
          <TrendingUp className="h-5 w-5 text-green-500" />
          <h2 className="text-lg font-semibold text-gray-900">Revenue Trends</h2>
        </div>
        <div className="flex items-end gap-2" style={{ height: 200 }}>
          {(analytics.revenueTrends?.labels ?? []).map((label, i) => (
            <div key={label} className="flex flex-1 flex-col items-center gap-1">
              <span className="text-xs font-medium text-gray-700">
                ${(revenueData[i] ?? 0)}
              </span>
              <div
                className="w-full rounded-t-md bg-green-500 transition-all"
                style={{
                  height: `${((revenueData[i] ?? 0) / maxRevenue) * 160}px`,
                  minHeight: 4,
                }}
              />
              <span className="text-xs text-gray-500">{label}</span>
            </div>
          ))}
        </div>
      </motion.div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Popular Items */}
        <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
          <div className="mb-4 flex items-center gap-2">
            <PieChart className="h-5 w-5 text-purple-500" />
            <h2 className="text-lg font-semibold text-gray-900">Popular Items</h2>
          </div>
          <div className="space-y-4">
            {popularItems.map((item, i) => (
              <div key={item.name} className="flex items-center gap-3">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-gray-100 text-xs font-bold text-gray-600">
                  {i + 1}
                </span>
                <div className="flex-1">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-900">{item.name}</span>
                    <span className="text-sm text-gray-500">{item.orders} orders</span>
                  </div>
                  <div className="mt-1 h-2 rounded-full bg-gray-100">
                    <div
                      className="h-2 rounded-full bg-purple-500"
                      style={{ width: `${item.percentage}%` }}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </motion.div>

        {/* Revenue by Category */}
        <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
          <div className="mb-4 flex items-center gap-2">
            <PieChart className="h-5 w-5 text-brand-500" />
            <h2 className="text-lg font-semibold text-gray-900">Revenue by Category</h2>
          </div>
          <div className="space-y-4">
            {revenueByCategory.map((cat) => (
              <div key={cat.category}>
                <div className="flex items-center justify-between text-sm">
                  <span className="font-medium text-gray-900">{cat.category}</span>
                  <span className="text-gray-500">${(cat.revenue ?? 0).toFixed(2)} ({cat.percentage ?? 0}%)</span>
                </div>
                <div className="mt-1 h-2 rounded-full bg-gray-100">
                  <div
                    className="h-2 rounded-full bg-brand-500"
                    style={{ width: `${cat.percentage}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </motion.div>
      </div>

      {/* Peak Hours */}
      <motion.div variants={fadeInUp} className="rounded-xl border border-gray-200 bg-white p-6">
        <div className="mb-4 flex items-center gap-2">
          <Clock className="h-5 w-5 text-amber-500" />
          <h2 className="text-lg font-semibold text-gray-900">Peak Hours</h2>
        </div>
        <div className="flex items-end gap-1 overflow-x-auto" style={{ height: 200 }}>
          {peakHours.map((item) => (
            <div key={item.hour} className="flex min-w-[40px] flex-1 flex-col items-center gap-1">
              <span className="text-xs font-medium text-gray-700">{item.orders}</span>
              <div
                className={`w-full rounded-t-md transition-all ${
                  item.orders >= maxPeakHour * 0.8
                    ? 'bg-red-500'
                    : item.orders >= maxPeakHour * 0.5
                      ? 'bg-amber-500'
                      : 'bg-amber-300'
                }`}
                style={{
                  height: `${(item.orders / maxPeakHour) * 160}px`,
                  minHeight: 4,
                }}
              />
              <span className="text-[10px] text-gray-500">{item.hour}</span>
            </div>
          ))}
        </div>
      </motion.div>
    </motion.div>
  );
}
