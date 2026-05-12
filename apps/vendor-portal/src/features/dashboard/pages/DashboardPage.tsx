import { motion } from 'framer-motion';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { Link } from 'react-router-dom';
import {
  Plus,
  ClipboardList,
  UserCog,
  Wallet,
  ChevronRight,
  Check,
  X,
  UtensilsCrossed,
  TrendingUp,
} from 'lucide-react';
import { Badge, OrderStatusBadge } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface DashboardStats {
  todayRevenue: number;
  todayOrders: number;
  pendingOrders: number;
  weekOrders: number;
  weekRevenue: number;
  rating: number;
  totalReviews: number;
  totalOrders: number;
  acceptingOrders: boolean;
}

interface OrderItem {
  name: string;
  quantity: number;
  price: number;
}

interface Order {
  id: string;
  orderNumber: string;
  customerName: string;
  items: OrderItem[];
  total: number;
  status: string;
  createdAt: string;
  estimatedDelivery?: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatCurrency(amount: number) {
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return 'Just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function StatRow({
  label,
  value,
  subtitle,
}: {
  label: string;
  value: string | number;
  subtitle?: string;
}) {
  return (
    <div className="px-4 py-4 sm:px-5">
      <p className="text-sm text-ink-soft">{label}</p>
      <p className="mt-1.5 text-2xl font-semibold tabular-nums tracking-tight text-foreground">
        {value}
      </p>
      {subtitle && <p className="mt-0.5 text-xs text-ink-soft tabular-nums">{subtitle}</p>}
    </div>
  );
}

function WeeklyRevenueChart({
  data,
  labels,
}: {
  data: number[];
  labels: string[];
}) {
  const max = Math.max(...data, 1);

  return (
    <motion.div
      variants={fadeInUp}
      className="rounded-xl border border-mist bg-bone p-6 shadow-sm"
    >
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h3 className="text-base font-semibold text-ink">
            Weekly Revenue
          </h3>
          <p className="text-sm text-ink-muted">Last 7 days performance</p>
        </div>
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-herb-tint">
          <TrendingUp className="h-4 w-4 text-herb" />
        </div>
      </div>

      <div className="flex items-end gap-2">
        {data.map((value, i) => {
          const heightPercent = Math.max((value / max) * 100, 4);
          const isToday = i === data.length - 1;

          return (
            <div key={i} className="group flex flex-1 flex-col items-center gap-2">
              {/* Tooltip */}
              <div className="relative">
                <span className="absolute -top-8 left-1/2 -translate-x-1/2 whitespace-nowrap rounded bg-ink px-2 py-1 text-xs text-paper opacity-0 transition-opacity group-hover:opacity-100">
                  {formatCurrency(value)}
                </span>
              </div>
              {/* Bar */}
              <div className="relative w-full" style={{ height: '140px' }}>
                <motion.div
                  initial={{ height: 0 }}
                  animate={{ height: `${heightPercent}%` }}
                  transition={{ duration: 0.6, delay: i * 0.08, ease: [0.16, 1, 0.3, 1] }}
                  className={`absolute bottom-0 w-full rounded-t-md transition-colors ${
                    isToday
                      ? 'bg-herb'
                      : 'bg-herb-tint group-hover:bg-herb-tint'
                  }`}
                />
              </div>
              {/* Label */}
              <span
                className={`text-xs ${
                  isToday ? 'font-semibold text-herb' : 'text-ink-muted'
                }`}
              >
                {labels[i]}
              </span>
            </div>
          );
        })}
      </div>

      <div className="mt-4 flex items-center justify-between border-t border-mist pt-4">
        <p className="text-sm text-ink-muted">
          Total:{' '}
          <span className="font-semibold text-ink">
            {formatCurrency(data.reduce((s, v) => s + v, 0))}
          </span>
        </p>
        <Link
          to="/earnings"
          className="inline-flex items-center gap-1 text-sm font-medium text-herb hover:text-herb"
        >
          View details
          <ChevronRight className="h-4 w-4" />
        </Link>
      </div>
    </motion.div>
  );
}

function PendingOrderCard({
  order,
  onAccept,
  onReject,
}: {
  order: Order;
  onAccept: (id: string) => void;
  onReject: (id: string) => void;
}) {
  return (
    <motion.div
      variants={fadeInUp}
      className="rounded-xl border border-amber/20 bg-amber-tint/50 p-4"
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-ink">
              #{order.orderNumber}
            </span>
            <Badge variant="warning" size="sm">
              New
            </Badge>
          </div>
          <p className="mt-1 text-sm text-ink-soft">{order.customerName}</p>
          <div className="mt-2 space-y-0.5">
            {(order.items ?? []).slice(0, 3).map((item, i) => (
              <p key={i} className="text-xs text-ink-muted">
                {item.quantity}x {item.name}
              </p>
            ))}
            {(order.items ?? []).length > 3 && (
              <p className="text-xs text-ink-muted">
                +{(order.items ?? []).length - 3} more items
              </p>
            )}
          </div>
        </div>
        <div className="text-right">
          <p className="text-sm font-medium text-ink">
            {formatCurrency(order.total)}
          </p>
          <p className="mt-0.5 text-xs text-ink-muted">
            {timeAgo(order.createdAt)}
          </p>
        </div>
      </div>

      <div className="mt-4 flex gap-2">
        <Button
          size="sm"
          variant="default"
          className="flex-1"
          leftIcon={<Check className="h-4 w-4" />}
          onClick={() => onAccept(order.id)}
        >
          Accept
        </Button>
        <Button
          size="sm"
          variant="outline"
          className="flex-1"
          leftIcon={<X className="h-4 w-4" />}
          onClick={() => onReject(order.id)}
        >
          Reject
        </Button>
      </div>
    </motion.div>
  );
}

function RecentOrdersTable({ orders }: { orders: Order[] }) {
  return (
    <motion.div
      variants={fadeInUp}
      className="rounded-xl border border-mist bg-bone shadow-sm"
    >
      <div className="flex items-center justify-between border-b border-mist px-6 py-4">
        <div>
          <h3 className="text-base font-semibold text-ink">
            Recent Orders
          </h3>
          <p className="text-sm text-ink-muted">Your latest order activity</p>
        </div>
        <Link
          to="/orders"
          className="inline-flex items-center gap-1 text-sm font-medium text-herb hover:text-herb"
        >
          View all
          <ChevronRight className="h-4 w-4" />
        </Link>
      </div>

      {/* Desktop table */}
      <div className="hidden overflow-x-auto md:block">
        <table className="w-full">
          <thead>
            <tr className="border-b border-mist text-left">
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-ink-muted">
                Order
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-ink-muted">
                Customer
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-ink-muted">
                Items
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-ink-muted">
                Amount
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-ink-muted">
                Status
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-ink-muted">
                Time
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {orders.map((order) => (
              <tr
                key={order.id}
                className="transition-colors hover:bg-paper/50"
              >
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-ink">
                  #{order.orderNumber}
                </td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-ink-soft">
                  {order.customerName}
                </td>
                <td className="px-6 py-4 text-sm text-ink-muted">
                  {(order.items ?? []).length} item{(order.items ?? []).length !== 1 && 's'}
                </td>
                <td className="whitespace-nowrap px-6 py-4 text-sm font-semibold text-ink">
                  {formatCurrency(order.total)}
                </td>
                <td className="whitespace-nowrap px-6 py-4">
                  <OrderStatusBadge status={order.status} size="sm" />
                </td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-ink-muted">
                  {timeAgo(order.createdAt)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile card list */}
      <div className="divide-y divide-gray-50 md:hidden">
        {orders.map((order) => (
          <div key={order.id} className="flex items-center justify-between px-4 py-3">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-ink">
                  #{order.orderNumber}
                </span>
                <OrderStatusBadge status={order.status} size="sm" />
              </div>
              <p className="mt-0.5 text-xs text-ink-muted">
                {order.customerName} &middot; {(order.items ?? []).length} item
                {(order.items ?? []).length !== 1 && 's'}
              </p>
            </div>
            <div className="text-right">
              <p className="text-sm font-semibold text-ink">
                {formatCurrency(order.total)}
              </p>
              <p className="text-xs text-ink-muted">{timeAgo(order.createdAt)}</p>
            </div>
          </div>
        ))}
      </div>

      {orders.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <UtensilsCrossed className="h-10 w-10 text-ink-muted" />
          <p className="mt-3 text-sm text-ink-muted">No recent orders</p>
        </div>
      )}
    </motion.div>
  );
}

function QuickActions() {
  const actions = [
    {
      label: 'Add Menu Item',
      description: 'Create a new dish listing',
      href: '/menu/new',
      icon: Plus,
      iconBg: 'bg-herb-tint',
      iconColor: 'text-herb',
    },
    {
      label: 'View Orders',
      description: 'Manage live orders',
      href: '/orders',
      icon: ClipboardList,
      iconBg: 'bg-info/10',
      iconColor: 'text-info',
    },
    {
      label: 'Update Profile',
      description: 'Edit kitchen details',
      href: '/profile',
      icon: UserCog,
      iconBg: 'bg-info/10',
      iconColor: 'text-info',
    },
    {
      label: 'View Earnings',
      description: 'Track your revenue',
      href: '/earnings',
      icon: Wallet,
      iconBg: 'bg-herb-tint',
      iconColor: 'text-herb',
    },
  ];

  return (
    <motion.div variants={fadeInUp}>
      <h3 className="mb-3 text-base font-semibold text-ink">
        Quick Actions
      </h3>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <Link
              key={action.href}
              to={action.href}
              className="group rounded-xl border border-mist bg-bone p-4 shadow-sm transition-all hover:border-herb-tint hover:shadow-md"
            >
              <div
                className={`flex h-10 w-10 items-center justify-center rounded-lg ${action.iconBg} transition-transform group-hover:opacity-95 `}
              >
                <Icon className={`h-5 w-5 ${action.iconColor}`} />
              </div>
              <p className="mt-3 text-sm font-semibold text-ink">
                {action.label}
              </p>
              <p className="mt-0.5 text-xs text-ink-muted">
                {action.description}
              </p>
            </Link>
          );
        })}
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  // Fetch dashboard stats
  const {
    data: stats,
    isLoading: statsLoading,
  } = useQuery<DashboardStats>({
    queryKey: ['chef', 'dashboard', 'stats'],
    queryFn: () => apiClient.get('/chef/dashboard'),
  });

  // Fetch active orders (pending + accepted + preparing)
  const {
    data: orders,
    isLoading: ordersLoading,
  } = useQuery<Order[]>({
    queryKey: ['chef', 'orders', 'active'],
    queryFn: () =>
      apiClient.get('/chef/orders', { status: 'pending,accepted,preparing' }),
  });

  const pendingOrders = orders?.filter((o) => o.status === 'pending') ?? [];
  const recentOrders = orders?.slice(0, 8) ?? [];

  // Action handlers (would call mutation in production)
  const handleAcceptOrder = (_orderId: string) => {
    // TODO: useMutation to PATCH /chef/orders/:id/accept
  };

  const handleRejectOrder = (_orderId: string) => {
    // TODO: useMutation to PATCH /chef/orders/:id/reject
  };

  // Loading skeleton
  if (statsLoading || ordersLoading) {
    return (
      <div className="space-y-6">
        {/* Page header skeleton */}
        <div>
          <div className="h-7 w-48 animate-pulse rounded-lg bg-mist" />
          <div className="mt-2 h-4 w-72 animate-pulse rounded bg-mist" />
        </div>

        {/* Stats skeleton */}
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="h-32 animate-pulse rounded-xl border border-mist bg-bone"
            />
          ))}
        </div>

        {/* Chart skeleton */}
        <div className="h-64 animate-pulse rounded-xl border border-mist bg-bone" />

        {/* Table skeleton */}
        <div className="h-80 animate-pulse rounded-xl border border-mist bg-bone" />
      </div>
    );
  }

  const pendingCount = pendingOrders.length;

  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className="space-y-8"
    >
      {/* Page header */}
      <motion.header variants={fadeInUp} className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
            Dashboard
          </h1>
          <p className="mt-1 text-sm text-ink-soft">Here's your kitchen today.</p>
        </div>

        <Link to="/menu/new">
          <Button size="sm" leftIcon={<Plus className="h-4 w-4" />}>
            Add item
          </Button>
        </Link>
      </motion.header>

      {/* Lead block — Revenue + pending orders CTA */}
      <motion.section
        variants={fadeInUp}
        className="grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:items-end"
      >
        <div>
          <p className="text-sm text-ink-soft">Today's revenue</p>
          <p className="mt-1 text-5xl font-semibold tabular-nums tracking-tight text-foreground sm:text-6xl">
            {formatCurrency(stats?.todayRevenue ?? 0)}
          </p>
          <p className="mt-2 text-sm text-ink-soft tabular-nums">
            From {stats?.todayOrders ?? 0}{' '}
            {stats?.todayOrders === 1 ? 'order' : 'orders'} today
          </p>
        </div>

        {pendingCount > 0 ? (
          <Link
            to="/orders"
            aria-label={`${pendingCount} pending orders waiting`}
            className="group flex items-center justify-between gap-4 rounded-lg bg-herb px-5 py-4 text-paper transition-colors hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
          >
            <div>
              <p className="text-3xl font-semibold tabular-nums">
                {pendingCount} {pendingCount === 1 ? 'order' : 'orders'}
              </p>
              <p className="mt-0.5 text-sm text-paper/80">Waiting for you to accept</p>
            </div>
            <ChevronRight className="h-5 w-5 transition-transform group-hover:translate-x-0.5" aria-hidden />
          </Link>
        ) : (
          <div className="rounded-lg border border-mist bg-bone px-5 py-4">
            <p className="text-3xl font-semibold tabular-nums text-foreground">All caught up</p>
            <p className="mt-0.5 text-sm text-ink-soft">New orders will appear here.</p>
          </div>
        )}
      </motion.section>

      {/* Stats — hairline-divided */}
      <motion.section
        variants={fadeInUp}
        aria-label="Today at a glance"
        className="grid grid-cols-2 divide-y divide-mist border-y border-mist sm:grid-cols-4 sm:divide-x sm:divide-y-0"
      >
        <StatRow label="Today's orders" value={stats?.todayOrders ?? 0} />
        <StatRow
          label="Rating"
          value={(stats?.rating ?? 0).toFixed(1)}
          subtitle={stats?.totalReviews ? `${stats.totalReviews} reviews` : 'No reviews yet'}
        />
        <StatRow
          label="This week"
          value={stats?.weekOrders ?? 0}
          subtitle={formatCurrency(stats?.weekRevenue ?? 0)}
        />
        <StatRow label="All-time orders" value={stats?.totalOrders ?? 0} />
      </motion.section>

      {/* Weekly revenue chart + pending orders */}
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <WeeklyRevenueChart
            data={[0, 0, 0, 0, 0, 0, stats?.weekRevenue ?? 0]}
            labels={['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']}
          />
        </div>

        {/* Pending orders sidebar */}
        <motion.div variants={fadeInUp} className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-base font-semibold text-ink">
              Pending Orders
            </h3>
            {pendingOrders.length > 0 && (
              <Badge variant="warning" size="sm">
                {pendingOrders.length}
              </Badge>
            )}
          </div>

          {pendingOrders.length > 0 ? (
            <div className="space-y-3">
              {pendingOrders.slice(0, 3).map((order) => (
                <PendingOrderCard
                  key={order.id}
                  order={order}
                  onAccept={handleAcceptOrder}
                  onReject={handleRejectOrder}
                />
              ))}
              {pendingOrders.length > 3 && (
                <Link
                  to="/orders"
                  className="block rounded-lg border border-dashed border-mist py-3 text-center text-sm font-medium text-herb transition-colors hover:border-herb-tint hover:bg-herb-tint/50"
                >
                  View {pendingOrders.length - 3} more pending orders
                </Link>
              )}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-mist py-10 text-center">
              <Check className="h-10 w-10 text-herb-soft" />
              <p className="mt-3 text-sm font-medium text-ink-soft">
                All caught up!
              </p>
              <p className="mt-1 text-xs text-ink-muted">
                No pending orders right now
              </p>
            </div>
          )}
        </motion.div>
      </div>

      {/* Recent orders table */}
      <RecentOrdersTable orders={recentOrders} />

      {/* Quick actions */}
      <QuickActions />
    </motion.div>
  );
}
