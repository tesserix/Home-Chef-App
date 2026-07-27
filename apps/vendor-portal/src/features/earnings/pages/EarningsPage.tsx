import { useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useQuery } from '@tanstack/react-query';
import { DollarSign, TrendingUp, Clock, Award, ArrowRight } from 'lucide-react';
import { format } from 'date-fns';
import { apiClient } from '@/shared/services/api-client';
import { formatCurrency } from '@/shared/utils/format';
import { Card } from '@/shared/components/ui/Card';
import { Badge, type BadgeProps } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

// ---- API contract types --------------------------------------------------
// GET /chef/earnings/breakdown?period=week|month|cycle
// Mirrors the wire shape returned by GetEarningsBreakdown's c.JSON exactly
// (apps/api/handlers/chef_earnings.go:179-190) — the response object itself,
// not wrapped in {data, pagination}, so apiClient passes it through as-is.

type EarningsPeriod = 'week' | 'month' | 'cycle';

// Mirrors models/payout_hold.go PayoutHoldStatus. '' (or absent) = no hold.
type PayoutHoldStatus =
  | ''
  | 'awaiting_customer_confirmation'
  | 'release_eligible'
  | 'released'
  | 'disputed'
  | 'withheld'
  | 'reversed';

interface EarningsRates {
  platformCommission: number;
  gst: number;
  tds: number;
}

interface EarningsTotals {
  grossRevenue: number;
  platformCommission: number;
  cgst: number;
  sgst: number;
  igst: number;
  tds: number;
  netPayout: number;
  ordersCount: number;
  // Escrow split (#617) — net payout still held vs already released to the
  // chef. Both read 0 while the escrow flags are off; that is real data
  // (nothing is held), not a placeholder.
  held: number;
  released: number;
}

interface EarningsOrder {
  orderId: string;
  orderNumber: string;
  completedAt: string;
  itemRevenue: number;
  deliveryFee: number;
  tip: number;
  gross: number;
  platformCommission: number;
  cgst: number;
  sgst: number;
  igst: number;
  tds: number;
  netPayout: number;
  payoutHoldStatus?: PayoutHoldStatus;
}

interface EarningsBreakdown {
  cycleStart: string;
  cycleEnd: string;
  currency: string;
  rates: EarningsRates;
  totals: EarningsTotals;
  orders: EarningsOrder[];
}

const PERIOD_OPTIONS: { value: EarningsPeriod; label: string }[] = [
  { value: 'week', label: 'This Week' },
  { value: 'month', label: 'This Month' },
  { value: 'cycle', label: 'Billing Cycle' },
];

function formatPercent(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`;
}

function pluralOrders(count: number): string {
  return count === 1 ? '1 order' : `${count} orders`;
}

function holdBadge(status?: PayoutHoldStatus): { label: string; variant: BadgeProps['variant'] } | null {
  switch (status) {
    case 'awaiting_customer_confirmation':
      return { label: 'Awaiting confirmation', variant: 'warning' };
    case 'release_eligible':
      return { label: 'Confirmed', variant: 'success' };
    case 'released':
      return { label: 'Released', variant: 'success' };
    case 'disputed':
      return { label: 'Disputed', variant: 'info' };
    case 'withheld':
      return { label: 'Withheld', variant: 'error' };
    case 'reversed':
      return { label: 'Reversed', variant: 'error' };
    default:
      return null;
  }
}

interface PeriodSelectorProps {
  period: EarningsPeriod;
  onChange: (period: EarningsPeriod) => void;
}

function PeriodSelector({ period, onChange }: PeriodSelectorProps) {
  return (
    <div
      role="radiogroup"
      aria-label="Earnings period"
      className="flex rounded-lg border border-mist bg-bone p-1"
    >
      {PERIOD_OPTIONS.map((opt) => {
        const isActive = period === opt.value;
        return (
          <button
            key={opt.value}
            type="button"
            role="radio"
            aria-checked={isActive}
            onClick={() => onChange(opt.value)}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
              isActive ? 'bg-herb text-paper' : 'text-ink-soft hover:text-ink'
            }`}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}

function EarningsLoadingSkeleton() {
  return (
    <div className="space-y-6">
      {/* Stat cards skeleton */}
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

      {/* Breakdown skeleton */}
      <Card>
        <Skeleton className="mb-4 h-5 w-32" />
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      </Card>

      {/* Order list skeleton */}
      <Card>
        <Skeleton className="mb-4 h-5 w-32" />
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      </Card>
    </div>
  );
}

export default function EarningsPage() {
  const [period, setPeriod] = useState<EarningsPeriod>('week');

  const { data, isLoading, isError } = useQuery<EarningsBreakdown>({
    queryKey: ['chef', 'earnings', 'breakdown', period],
    // GET /chef/earnings/breakdown returns the breakdown object directly
    // (handlers/chef_earnings.go:179 GetEarningsBreakdown's c.JSON) — not
    // wrapped in {data, pagination}, so apiClient resolves it unchanged.
    queryFn: () =>
      apiClient.get<EarningsBreakdown>(`/chef/earnings/breakdown?period=${period}`),
  });

  const header = (
    <motion.div
      variants={fadeInUp}
      className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"
    >
      <div>
        <h1 className="font-display text-2xl font-semibold text-ink">Earnings</h1>
        <p className="mt-1 text-sm text-ink-muted">Track your revenue and payouts</p>
      </div>
      <PeriodSelector period={period} onChange={setPeriod} />
    </motion.div>
  );

  if (isLoading) {
    return (
      <div className="space-y-6">
        <motion.div
          variants={staggerContainer}
          initial="hidden"
          animate="visible"
          className="space-y-6"
        >
          {header}
          <EarningsLoadingSkeleton />
        </motion.div>
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="space-y-6">
        <motion.div
          variants={staggerContainer}
          initial="hidden"
          animate="visible"
          className="space-y-6"
        >
          {header}
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <DollarSign className="mb-4 h-12 w-12 text-ink-muted" />
            <h3 className="text-lg font-semibold text-ink">Unable to load earnings</h3>
            <p className="mt-1 text-sm text-ink-muted">Please try again later.</p>
          </div>
        </motion.div>
      </div>
    );
  }

  const { totals, rates, orders } = data;
  const interState = totals.igst > 0;
  const gstCombined = totals.cgst + totals.sgst;

  const statCards = [
    {
      label: 'Gross Revenue',
      value: totals.grossRevenue,
      icon: TrendingUp,
      iconBg: 'bg-herb-tint',
      iconColor: 'text-herb',
    },
    {
      label: 'Net Payout',
      value: totals.netPayout,
      icon: DollarSign,
      iconBg: 'bg-herb-tint',
      iconColor: 'text-herb',
    },
    {
      label: 'Held in Escrow',
      value: totals.held,
      icon: Clock,
      iconBg: 'bg-amber-tint',
      iconColor: 'text-amber',
    },
    {
      label: 'Released',
      value: totals.released,
      icon: Award,
      iconBg: 'bg-info/10',
      iconColor: 'text-info',
    },
  ];

  return (
    <div className="space-y-6">
      <motion.div
        variants={staggerContainer}
        initial="hidden"
        animate="visible"
        className="space-y-6"
      >
        {header}

        {/* Stat Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {statCards.map((card) => (
            <motion.div key={card.label} variants={fadeInUp}>
              <Card>
                <div className="flex items-center gap-4">
                  <div
                    className={`flex h-12 w-12 items-center justify-center rounded-xl ${card.iconBg}`}
                  >
                    <card.icon className={`h-6 w-6 ${card.iconColor}`} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium text-ink-muted">{card.label}</p>
                    <p className="text-xl font-semibold text-ink">
                      {formatCurrency(card.value)}
                    </p>
                  </div>
                </div>
              </Card>
            </motion.div>
          ))}
        </div>

        {/* Rate/Tax Breakdown */}
        <motion.div variants={fadeInUp}>
          <Card>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-ink">Breakdown</h2>
              <p className="text-sm text-ink-muted">
                {format(new Date(data.cycleStart), 'dd MMM')} –{' '}
                {format(new Date(data.cycleEnd), 'dd MMM yyyy')}
              </p>
            </div>
            <div className="divide-y divide-mist">
              <div className="flex items-center justify-between py-3 first:pt-0">
                <span className="text-sm text-ink-soft">Gross Revenue</span>
                <span className="text-sm font-semibold text-ink">
                  {formatCurrency(totals.grossRevenue)}
                </span>
              </div>
              <div className="flex items-center justify-between py-3">
                <span className="text-sm text-ink-soft">
                  Platform Commission{' '}
                  <span className="text-xs text-ink-muted">
                    ({formatPercent(rates.platformCommission)})
                  </span>
                </span>
                <span className="text-sm text-ink-muted">
                  − {formatCurrency(totals.platformCommission)}
                </span>
              </div>
              {interState ? (
                <div className="flex items-center justify-between py-3">
                  <span className="text-sm text-ink-soft">
                    IGST{' '}
                    <span className="text-xs text-ink-muted">({formatPercent(rates.gst)})</span>
                  </span>
                  <span className="text-sm text-ink-muted">− {formatCurrency(totals.igst)}</span>
                </div>
              ) : (
                <div className="flex items-center justify-between py-3">
                  <span className="text-sm text-ink-soft">
                    GST (CGST + SGST){' '}
                    <span className="text-xs text-ink-muted">({formatPercent(rates.gst)})</span>
                  </span>
                  <span className="text-sm text-ink-muted">
                    − {formatCurrency(gstCombined)}
                  </span>
                </div>
              )}
              <div className="flex items-center justify-between py-3">
                <span className="text-sm text-ink-soft">
                  TDS{' '}
                  <span className="text-xs text-ink-muted">({formatPercent(rates.tds)})</span>
                </span>
                <span className="text-sm text-ink-muted">− {formatCurrency(totals.tds)}</span>
              </div>
              <div className="flex items-center justify-between py-3 last:pb-0">
                <span className="text-sm font-semibold text-ink">Net Payout</span>
                <span className="text-lg font-semibold text-herb">
                  {formatCurrency(totals.netPayout)}
                </span>
              </div>
            </div>
          </Card>
        </motion.div>

        {/* Per-order breakdown */}
        <motion.div variants={fadeInUp}>
          <Card>
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-ink">Order Breakdown</h2>
                <p className="text-xs text-ink-muted">{pluralOrders(totals.ordersCount)}</p>
              </div>
              <Link to="/earnings/payouts">
                <Button variant="ghost" size="sm">
                  Payout History
                  <ArrowRight className="ml-1 h-4 w-4" />
                </Button>
              </Link>
            </div>
            {orders.length === 0 ? (
              <p className="py-8 text-center text-sm text-ink-muted">
                No orders in this period yet
              </p>
            ) : (
              <div className="divide-y divide-mist">
                {orders.map((order) => {
                  const badge = holdBadge(order.payoutHoldStatus);
                  return (
                    <div
                      key={order.orderId}
                      className="flex items-center gap-3 py-3 first:pt-0 last:pb-0"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <p className="truncate text-sm font-medium text-ink">
                            #{order.orderNumber}
                          </p>
                          {badge && (
                            <Badge variant={badge.variant} size="sm">
                              {badge.label}
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-ink-muted">
                          {format(new Date(order.completedAt), 'dd MMM yyyy')}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm font-semibold text-ink">
                          {formatCurrency(order.netPayout)}
                        </p>
                        <p className="text-xs text-ink-muted">
                          Gross {formatCurrency(order.gross)}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </motion.div>
      </motion.div>
    </div>
  );
}
