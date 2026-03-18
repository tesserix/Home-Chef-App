import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useQuery } from '@tanstack/react-query';
import {
  DollarSign,
  TrendingUp,
  Clock,
  Award,
  ArrowRight,
  ShoppingBag,
  ChevronRight,
} from 'lucide-react';
import { format } from 'date-fns';
import { apiClient } from '@/shared/services/api-client';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

interface DailyEarning {
  date: string;
  amount: number;
}

interface TopItem {
  name: string;
  orders: number;
  revenue: number;
}

interface Payout {
  id: string;
  amount: number;
  date: string;
  status: 'completed' | 'pending' | 'failed';
  method: string;
}

interface EarningsData {
  totalBalance: number;
  pendingPayout: number;
  lifetimeEarnings: number;
  thisMonthEarnings: number;
  lastMonthEarnings: number;
  dailyEarnings: DailyEarning[];
  topItems: TopItem[];
  recentPayouts: Payout[];
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

function getPayoutStatusVariant(status: string) {
  switch (status) {
    case 'completed':
      return 'success' as const;
    case 'pending':
      return 'warning' as const;
    case 'failed':
      return 'error' as const;
    default:
      return 'default' as const;
  }
}

function EarningsLoadingSkeleton() {
  return (
    <div className="space-y-6">
      {/* Balance cards skeleton */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i}>
            <div className="flex items-center gap-4">
              <Skeleton className="h-12 w-12 rounded-xl" />
              <div className="flex-1">
                <Skeleton className="mb-2 h-3 w-20" />
                <Skeleton className="h-6 w-28" />
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Chart skeleton */}
      <Card>
        <Skeleton className="mb-4 h-5 w-32" />
        <Skeleton className="h-48 w-full" />
      </Card>

      {/* Bottom sections skeleton */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <Skeleton className="mb-4 h-5 w-32" />
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        </Card>
        <Card>
          <Skeleton className="mb-4 h-5 w-32" />
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}

export default function EarningsPage() {
  const { data, isLoading, isError } = useQuery<EarningsData>({
    queryKey: ['chef', 'earnings'],
    queryFn: () => apiClient.get<EarningsData>('/chef/earnings'),
  });

  if (isLoading) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-gray-900">Earnings</h1>
          <p className="mt-1 text-sm text-gray-500">
            Track your revenue and payouts
          </p>
        </div>
        <EarningsLoadingSkeleton />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <DollarSign className="mb-4 h-12 w-12 text-gray-300" />
          <h3 className="text-lg font-semibold text-gray-900">
            Unable to load earnings
          </h3>
          <p className="mt-1 text-sm text-gray-500">
            Please try again later.
          </p>
        </div>
      </div>
    );
  }

  const dailyEarnings = data.dailyEarnings ?? [];
  const maxDailyAmount = dailyEarnings.length > 0
    ? Math.max(...dailyEarnings.map((d) => d.amount ?? 0), 1)
    : 1;

  const monthOverMonthChange =
    data.lastMonthEarnings > 0
      ? ((data.thisMonthEarnings - data.lastMonthEarnings) /
          data.lastMonthEarnings) *
        100
      : 0;

  const balanceCards = [
    {
      label: 'Available Balance',
      value: data.totalBalance,
      icon: DollarSign,
      iconBg: 'bg-brand-50',
      iconColor: 'text-brand-600',
    },
    {
      label: 'Pending Payout',
      value: data.pendingPayout,
      icon: Clock,
      iconBg: 'bg-amber-50',
      iconColor: 'text-amber-600',
    },
    {
      label: 'This Month',
      value: data.thisMonthEarnings,
      icon: TrendingUp,
      iconBg: 'bg-emerald-50',
      iconColor: 'text-emerald-600',
      change: monthOverMonthChange,
    },
    {
      label: 'Lifetime Earnings',
      value: data.lifetimeEarnings,
      icon: Award,
      iconBg: 'bg-purple-50',
      iconColor: 'text-purple-600',
    },
  ];

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
      {/* Page Header */}
      <motion.div
        variants={staggerContainer}
        initial="hidden"
        animate="visible"
        className="space-y-6"
      >
        <motion.div variants={fadeInUp}>
          <h1 className="text-2xl font-bold text-gray-900">Earnings</h1>
          <p className="mt-1 text-sm text-gray-500">
            Track your revenue and payouts
          </p>
        </motion.div>

        {/* Balance Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {balanceCards.map((card) => (
            <motion.div key={card.label} variants={fadeInUp}>
              <Card>
                <div className="flex items-center gap-4">
                  <div
                    className={`flex h-12 w-12 items-center justify-center rounded-xl ${card.iconBg}`}
                  >
                    <card.icon className={`h-6 w-6 ${card.iconColor}`} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium text-gray-500">
                      {card.label}
                    </p>
                    <p className="text-xl font-bold text-gray-900">
                      {formatCurrency(card.value)}
                    </p>
                    {card.change !== undefined && card.change !== 0 && (
                      <p
                        className={`mt-0.5 text-xs font-medium ${
                          card.change > 0
                            ? 'text-emerald-600'
                            : 'text-red-600'
                        }`}
                      >
                        {card.change > 0 ? '+' : ''}
                        {card.change.toFixed(1)}% vs last month
                      </p>
                    )}
                  </div>
                </div>
              </Card>
            </motion.div>
          ))}
        </div>

        {/* Daily Earnings Chart */}
        <motion.div variants={fadeInUp}>
          <Card>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">
                Daily Earnings
              </h2>
              <p className="text-sm text-gray-500">Last 14 days</p>
            </div>
            <div className="flex items-end gap-1.5 overflow-x-auto pb-2">
              {dailyEarnings.map((day) => {
                const heightPercent =
                  maxDailyAmount > 0
                    ? (day.amount / maxDailyAmount) * 100
                    : 0;
                return (
                  <div
                    key={day.date}
                    className="group flex min-w-[2.5rem] flex-1 flex-col items-center gap-1"
                  >
                    {/* Tooltip */}
                    <div className="invisible text-xs font-medium text-gray-700 group-hover:visible">
                      {formatCurrency(day.amount)}
                    </div>
                    {/* Bar */}
                    <div className="relative w-full" style={{ height: '160px' }}>
                      <div
                        className="absolute bottom-0 w-full rounded-t-md bg-brand-500 transition-all duration-300 group-hover:bg-brand-600"
                        style={{
                          height: `${Math.max(heightPercent, 2)}%`,
                        }}
                      />
                    </div>
                    {/* Date label */}
                    <p className="text-[10px] text-gray-400">
                      {format(new Date(day.date), 'dd')}
                    </p>
                    <p className="text-[9px] text-gray-400">
                      {format(new Date(day.date), 'MMM')}
                    </p>
                  </div>
                );
              })}
            </div>
          </Card>
        </motion.div>

        {/* Bottom Section: Top Selling Items + Recent Payouts */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {/* Top Selling Items */}
          <motion.div variants={fadeInUp}>
            <Card>
              <div className="mb-4 flex items-center justify-between">
                <h2 className="text-lg font-semibold text-gray-900">
                  Top Selling Items
                </h2>
                <ShoppingBag className="h-5 w-5 text-gray-400" />
              </div>
              {(data.topItems ?? []).length === 0 ? (
                <p className="py-8 text-center text-sm text-gray-400">
                  No sales data yet
                </p>
              ) : (
                <div className="divide-y divide-gray-100">
                  {(data.topItems ?? []).map((item, index) => (
                    <div
                      key={item.name}
                      className="flex items-center gap-3 py-3 first:pt-0 last:pb-0"
                    >
                      <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 text-xs font-bold text-gray-500">
                        {index + 1}
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-gray-900">
                          {item.name}
                        </p>
                        <p className="text-xs text-gray-500">
                          {item.orders} orders
                        </p>
                      </div>
                      <p className="text-sm font-semibold text-gray-900">
                        {formatCurrency(item.revenue)}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </motion.div>

          {/* Recent Payouts */}
          <motion.div variants={fadeInUp}>
            <Card>
              <div className="mb-4 flex items-center justify-between">
                <h2 className="text-lg font-semibold text-gray-900">
                  Recent Payouts
                </h2>
                <Link to="/earnings/payouts">
                  <Button variant="ghost" size="sm">
                    View All
                    <ArrowRight className="ml-1 h-4 w-4" />
                  </Button>
                </Link>
              </div>
              {(data.recentPayouts ?? []).length === 0 ? (
                <p className="py-8 text-center text-sm text-gray-400">
                  No payouts yet
                </p>
              ) : (
                <div className="divide-y divide-gray-100">
                  {(data.recentPayouts ?? []).map((payout) => (
                    <div
                      key={payout.id}
                      className="flex items-center gap-3 py-3 first:pt-0 last:pb-0"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium text-gray-900">
                            {formatCurrency(payout.amount)}
                          </p>
                          <Badge
                            variant={getPayoutStatusVariant(payout.status)}
                            size="sm"
                          >
                            {payout.status}
                          </Badge>
                        </div>
                        <p className="text-xs text-gray-500">
                          {format(new Date(payout.date), 'dd MMM yyyy')} via{' '}
                          {payout.method}
                        </p>
                      </div>
                      <ChevronRight className="h-4 w-4 text-gray-300" />
                    </div>
                  ))}
                </div>
              )}

              {(data.recentPayouts ?? []).length > 0 && (
                <div className="mt-4 border-t border-gray-100 pt-4">
                  <Link
                    to="/earnings/payouts"
                    className="flex items-center justify-center gap-1 text-sm font-medium text-brand-600 hover:text-brand-700"
                  >
                    View full payout history
                    <ArrowRight className="h-4 w-4" />
                  </Link>
                </div>
              )}
            </Card>
          </motion.div>
        </div>
      </motion.div>
    </div>
  );
}
