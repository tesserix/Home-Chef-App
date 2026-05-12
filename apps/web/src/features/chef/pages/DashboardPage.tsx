import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Star,
  ChefHat,
  ArrowUpRight,
  ArrowDownRight,
  ArrowRight,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { useAuth } from '@/app/providers/AuthProvider';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Badge } from '@/shared/components/ui';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';
import type { Order, PaginatedResponse } from '@/shared/types';

interface DashboardStats {
  todayOrders: number;
  todayRevenue: number;
  pendingOrders: number;
  weeklyOrders: number;
  weeklyRevenue: number;
  rating: number;
  totalReviews: number;
  revenueChange: number;
  ordersChange: number;
}

export default function ChefDashboardPage() {
  const { user } = useAuth();
  const fp = useFormatPrice();

  const { data: stats } = useQuery({
    queryKey: ['chef-dashboard-stats'],
    queryFn: () => apiClient.get<DashboardStats>('/chef/dashboard/stats'),
  });

  const { data: recentOrders } = useQuery({
    queryKey: ['chef-recent-orders'],
    queryFn: () =>
      apiClient.get<PaginatedResponse<Order>>('/chef/orders', {
        limit: 5,
        sort: 'createdAt',
        order: 'desc',
      }),
  });

  const { data: pendingOrders } = useQuery({
    queryKey: ['chef-pending-orders'],
    queryFn: () =>
      apiClient.get<PaginatedResponse<Order>>('/chef/orders', {
        status: 'pending,accepted',
        limit: 10,
      }),
  });

  const pendingCount = stats?.pendingOrders ?? 0;
  const revenueChange = stats?.revenueChange;
  const positive = revenueChange !== undefined && revenueChange >= 0;

  return (
    <motion.div
      initial="hidden"
      animate="visible"
      variants={staggerContainer}
      className="space-y-8"
    >
      {/* Header */}
      <motion.header variants={fadeInUp}>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
          Welcome back, {user?.firstName}
        </h1>
        <p className="mt-1 text-sm text-ink-soft">Here's your kitchen today.</p>
      </motion.header>

      {/* Lead block — one prominent metric + pending-orders action */}
      <motion.section
        variants={fadeInUp}
        className="grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:items-end"
      >
        <div>
          <p className="text-sm text-ink-soft">Today's revenue</p>
          <p className="mt-1 text-5xl font-semibold tabular-nums tracking-tight text-foreground sm:text-6xl">
            {fp(stats?.todayRevenue || 0)}
          </p>
          {revenueChange !== undefined && (
            <p className="mt-2 flex items-center gap-1.5 text-sm">
              {positive ? (
                <ArrowUpRight className="h-4 w-4 text-herb" aria-hidden />
              ) : (
                <ArrowDownRight className="h-4 w-4 text-paprika" aria-hidden />
              )}
              <span className={`font-medium tabular-nums ${positive ? 'text-herb' : 'text-paprika'}`}>
                {positive ? '+' : ''}
                {revenueChange}%
              </span>
              <span className="text-ink-soft">vs. yesterday</span>
            </p>
          )}
        </div>

        {pendingCount > 0 ? (
          <Link
            to="/chef/orders"
            aria-label={`${pendingCount} pending orders need attention`}
            className="group flex items-center justify-between gap-4 rounded-lg bg-herb px-5 py-4 text-paper transition-colors hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
          >
            <div>
              <p className="text-3xl font-semibold tabular-nums">
                {pendingCount} {pendingCount === 1 ? 'order' : 'orders'}
              </p>
              <p className="mt-0.5 text-sm text-paper/80">Waiting for you to accept</p>
            </div>
            <ArrowRight className="h-5 w-5 transition-transform group-hover:translate-x-0.5" aria-hidden />
          </Link>
        ) : (
          <div className="rounded-lg border border-mist bg-bone px-5 py-4">
            <p className="text-3xl font-semibold tabular-nums text-foreground">All caught up</p>
            <p className="mt-0.5 text-sm text-ink-soft">New orders will appear here.</p>
          </div>
        )}
      </motion.section>

      {/* Stats — clean grid, no card chrome, hairline-divided */}
      <motion.section
        variants={fadeInUp}
        aria-label="Today at a glance"
        className="grid grid-cols-2 divide-y divide-mist border-y border-mist sm:grid-cols-4 sm:divide-x sm:divide-y-0"
      >
        <StatRow label="Today's orders" value={stats?.todayOrders ?? 0} />
        <StatRow
          label="Rating"
          value={stats?.rating !== undefined ? stats.rating.toFixed(1) : '—'}
          icon={Star}
          subtitle={stats?.totalReviews ? `${stats.totalReviews} reviews` : 'No reviews yet'}
        />
        <StatRow
          label="This week"
          value={stats?.weeklyOrders ?? 0}
          subtitle={fp(stats?.weeklyRevenue || 0)}
        />
        <StatRow
          label="vs. last week"
          value={
            stats?.ordersChange !== undefined
              ? `${stats.ordersChange >= 0 ? '+' : ''}${stats.ordersChange}%`
              : '—'
          }
          subtitle="Order change"
        />
      </motion.section>

      {/* Work area — asymmetric 2:1, pending orders dominant */}
      <div className="grid gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        {/* Pending Orders */}
        <motion.section variants={fadeInUp} aria-label="Pending orders" className="space-y-3">
          <div className="flex items-baseline justify-between">
            <h2 className="text-lg font-semibold text-foreground">Pending orders</h2>
            <Link to="/chef/orders" className="text-sm font-medium text-herb hover:underline">
              View all
            </Link>
          </div>

          {(pendingOrders?.data ?? []).length === 0 ? (
            <div className="rounded-lg border border-mist bg-bone py-12 text-center">
              <ChefHat className="mx-auto h-10 w-10 text-ink-muted" aria-hidden />
              <p className="mt-3 font-medium text-foreground">No pending orders</p>
              <p className="mt-1 text-sm text-ink-soft">
                New orders will appear here as customers place them.
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {(pendingOrders?.data ?? []).slice(0, 4).map((order) => (
                <PendingOrderCard key={order.id} order={order} />
              ))}
            </div>
          )}
        </motion.section>

        {/* Shortcuts */}
        <motion.aside variants={fadeInUp} aria-label="Shortcuts" className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground">Shortcuts</h2>
          <nav className="divide-y divide-mist rounded-lg border border-mist bg-bone">
            <QuickActionLink to="/chef/menu" title="Manage menu" subtitle="Add or edit items" />
            <QuickActionLink to="/chef/orders" title="View orders" subtitle="Manage all orders" />
            <QuickActionLink to="/chef/earnings" title="Earnings" subtitle="Track your income" />
            <QuickActionLink to="/chef/social" title="Social feed" subtitle="Share your creations" />
          </nav>
        </motion.aside>
      </div>

      {/* Recent Orders — clean table, hairline rows */}
      <motion.section variants={fadeInUp} aria-label="Recent orders" className="space-y-3">
        <div className="flex items-baseline justify-between">
          <h2 className="text-lg font-semibold text-foreground">Recent orders</h2>
          <Link to="/chef/orders" className="text-sm font-medium text-herb hover:underline">
            View all
          </Link>
        </div>

        <div className="overflow-x-auto rounded-lg border border-mist bg-bone">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-mist text-left text-sm text-ink-soft">
                <th className="px-4 py-3 font-medium">Order</th>
                <th className="px-4 py-3 font-medium">Items</th>
                <th className="px-4 py-3 text-right font-medium tabular-nums">Total</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium tabular-nums">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-mist">
              {(recentOrders?.data ?? []).map((order) => (
                <tr key={order.id} className="transition-colors hover:bg-mist/40">
                  <td className="px-4 py-3">
                    <Link
                      to={`/chef/orders/${order.id}`}
                      className="font-medium tabular-nums text-foreground hover:text-herb"
                    >
                      #{order.orderNumber}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-ink-soft">
                    {order.items.length} {order.items.length === 1 ? 'item' : 'items'}
                  </td>
                  <td className="px-4 py-3 text-right font-medium tabular-nums text-foreground">
                    {fp(order.total)}
                  </td>
                  <td className="px-4 py-3">
                    <OrderStatusBadge status={order.status} />
                  </td>
                  <td className="px-4 py-3 text-ink-soft tabular-nums">
                    {new Date(order.createdAt).toLocaleTimeString(undefined, {
                      hour: 'numeric',
                      minute: '2-digit',
                    })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </motion.section>
    </motion.div>
  );
}

/** Stat cell — clean sans, no card chrome, tabular numerals. */
function StatRow({
  label,
  value,
  subtitle,
  icon: Icon,
}: {
  label: string;
  value: string | number;
  subtitle?: string;
  icon?: typeof Star;
}) {
  return (
    <div className="px-4 py-4 sm:px-5">
      <p className="text-sm text-ink-soft">{label}</p>
      <p className="mt-1.5 flex items-baseline gap-1.5 text-2xl font-semibold tabular-nums tracking-tight text-foreground">
        {Icon && <Icon className="h-4 w-4 text-ink-muted" aria-hidden />}
        {value}
      </p>
      {subtitle && <p className="mt-0.5 text-xs text-ink-soft tabular-nums">{subtitle}</p>}
    </div>
  );
}

/** Shortcut list item — clean, single accent on hover via chevron. */
function QuickActionLink({
  to,
  title,
  subtitle,
}: {
  to: string;
  title: string;
  subtitle: string;
}) {
  return (
    <Link
      to={to}
      className="group flex items-center justify-between gap-3 px-4 py-3 transition-colors hover:bg-mist/60 first:rounded-t-lg last:rounded-b-lg"
    >
      <div className="min-w-0">
        <p className="font-medium text-foreground">{title}</p>
        <p className="truncate text-sm text-ink-soft">{subtitle}</p>
      </div>
      <ArrowRight
        className="h-4 w-4 shrink-0 text-ink-muted transition-transform group-hover:translate-x-0.5 group-hover:text-herb"
        aria-hidden
      />
    </Link>
  );
}

function PendingOrderCard({ order }: { order: Order }) {
  const fp = useFormatPrice();
  const isNew = order.status === 'pending';

  return (
    <Link
      to={`/chef/orders/${order.id}`}
      className={`flex items-center justify-between rounded-xl border p-4 transition-all hover:bg-paper ${
        isNew ? 'border-amber/30 bg-amber-tint hover:bg-amber-tint' : 'border-mist'
      }`}
    >
      <div className="flex items-center gap-3">
        {isNew && (
          <span className="flex h-2 w-2 rounded-full bg-amber animate-pulse" />
        )}
        <div>
          <p className="font-medium text-ink">#{order.orderNumber}</p>
          <p className="text-sm text-ink-muted">
            {order.items.length} items - {fp(order.total)}
          </p>
        </div>
      </div>
      <div className="text-right">
        <OrderStatusBadge status={order.status} />
        <p className="mt-1 text-xs text-ink-muted">
          {new Date(order.createdAt).toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
          })}
        </p>
      </div>
    </Link>
  );
}

function OrderStatusBadge({ status }: { status: string }) {
  const config: Record<string, { label: string; variant: 'warning' | 'info' | 'success' | 'default' | 'error' }> = {
    pending: { label: 'New', variant: 'warning' },
    accepted: { label: 'Accepted', variant: 'info' },
    preparing: { label: 'Preparing', variant: 'info' },
    ready: { label: 'Ready', variant: 'success' },
    picked_up: { label: 'Picked Up', variant: 'info' },
    delivering: { label: 'Delivering', variant: 'warning' },
    delivered: { label: 'Delivered', variant: 'default' },
    cancelled: { label: 'Cancelled', variant: 'error' },
  };

  const { label, variant } = config[status] || { label: status, variant: 'default' as const };

  return (
    <Badge variant={variant} size="sm">
      {label}
    </Badge>
  );
}
