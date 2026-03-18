import { motion } from 'framer-motion';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { Link } from 'react-router-dom';
import {
  IndianRupee,
  ShoppingBag,
  Clock,
  Star,
  Plus,
  ClipboardList,
  UserCog,
  Wallet,
  ChevronRight,
  Check,
  X,
  UtensilsCrossed,
  TrendingUp,
  ArrowUpRight,
  ArrowDownRight,
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

function StatsCard({
  title,
  value,
  icon: Icon,
  trend,
  trendLabel,
  iconBg,
  iconColor,
}: {
  title: string;
  value: string;
  icon: React.ElementType;
  trend?: number;
  trendLabel?: string;
  iconBg: string;
  iconColor: string;
}) {
  const isPositive = trend !== undefined && trend >= 0;

  return (
    <motion.div
      variants={fadeInUp}
      className="rounded-xl border border-gray-100 bg-white p-5 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="flex items-start justify-between">
        <div className={`flex h-11 w-11 items-center justify-center rounded-lg ${iconBg}`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
        {trend !== undefined && (
          <span
            className={`inline-flex items-center gap-0.5 rounded-full px-2 py-0.5 text-xs font-medium ${
              isPositive
                ? 'bg-green-50 text-green-700'
                : 'bg-red-50 text-red-700'
            }`}
          >
            {isPositive ? (
              <ArrowUpRight className="h-3 w-3" />
            ) : (
              <ArrowDownRight className="h-3 w-3" />
            )}
            {Math.abs(trend)}%
          </span>
        )}
      </div>
      <p className="mt-4 text-2xl font-bold text-gray-900">{value}</p>
      <p className="mt-1 text-sm text-gray-500">
        {title}
        {trendLabel && (
          <span className="ml-1 text-xs text-gray-400">{trendLabel}</span>
        )}
      </p>
    </motion.div>
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
      className="rounded-xl border border-gray-100 bg-white p-6 shadow-sm"
    >
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            Weekly Revenue
          </h3>
          <p className="text-sm text-gray-500">Last 7 days performance</p>
        </div>
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-50">
          <TrendingUp className="h-4 w-4 text-brand-600" />
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
                <span className="absolute -top-8 left-1/2 -translate-x-1/2 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100">
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
                      ? 'bg-brand-500'
                      : 'bg-brand-100 group-hover:bg-brand-200'
                  }`}
                />
              </div>
              {/* Label */}
              <span
                className={`text-xs ${
                  isToday ? 'font-semibold text-brand-600' : 'text-gray-400'
                }`}
              >
                {labels[i]}
              </span>
            </div>
          );
        })}
      </div>

      <div className="mt-4 flex items-center justify-between border-t border-gray-100 pt-4">
        <p className="text-sm text-gray-500">
          Total:{' '}
          <span className="font-semibold text-gray-900">
            {formatCurrency(data.reduce((s, v) => s + v, 0))}
          </span>
        </p>
        <Link
          to="/earnings"
          className="inline-flex items-center gap-1 text-sm font-medium text-brand-600 hover:text-brand-700"
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
      className="rounded-xl border border-amber-100 bg-amber-50/50 p-4"
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-gray-900">
              #{order.orderNumber}
            </span>
            <Badge variant="warning" size="sm">
              New
            </Badge>
          </div>
          <p className="mt-1 text-sm text-gray-600">{order.customerName}</p>
          <div className="mt-2 space-y-0.5">
            {(order.items ?? []).slice(0, 3).map((item, i) => (
              <p key={i} className="text-xs text-gray-500">
                {item.quantity}x {item.name}
              </p>
            ))}
            {(order.items ?? []).length > 3 && (
              <p className="text-xs text-gray-400">
                +{(order.items ?? []).length - 3} more items
              </p>
            )}
          </div>
        </div>
        <div className="text-right">
          <p className="text-sm font-bold text-gray-900">
            {formatCurrency(order.total)}
          </p>
          <p className="mt-0.5 text-xs text-gray-400">
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
      className="rounded-xl border border-gray-100 bg-white shadow-sm"
    >
      <div className="flex items-center justify-between border-b border-gray-100 px-6 py-4">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            Recent Orders
          </h3>
          <p className="text-sm text-gray-500">Your latest order activity</p>
        </div>
        <Link
          to="/orders"
          className="inline-flex items-center gap-1 text-sm font-medium text-brand-600 hover:text-brand-700"
        >
          View all
          <ChevronRight className="h-4 w-4" />
        </Link>
      </div>

      {/* Desktop table */}
      <div className="hidden overflow-x-auto md:block">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-100 text-left">
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-gray-400">
                Order
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-gray-400">
                Customer
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-gray-400">
                Items
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-gray-400">
                Amount
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-gray-400">
                Status
              </th>
              <th className="px-6 py-3 text-xs font-medium uppercase tracking-wider text-gray-400">
                Time
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {orders.map((order) => (
              <tr
                key={order.id}
                className="transition-colors hover:bg-gray-50/50"
              >
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">
                  #{order.orderNumber}
                </td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-600">
                  {order.customerName}
                </td>
                <td className="px-6 py-4 text-sm text-gray-500">
                  {(order.items ?? []).length} item{(order.items ?? []).length !== 1 && 's'}
                </td>
                <td className="whitespace-nowrap px-6 py-4 text-sm font-semibold text-gray-900">
                  {formatCurrency(order.total)}
                </td>
                <td className="whitespace-nowrap px-6 py-4">
                  <OrderStatusBadge status={order.status} size="sm" />
                </td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-400">
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
                <span className="text-sm font-medium text-gray-900">
                  #{order.orderNumber}
                </span>
                <OrderStatusBadge status={order.status} size="sm" />
              </div>
              <p className="mt-0.5 text-xs text-gray-500">
                {order.customerName} &middot; {(order.items ?? []).length} item
                {(order.items ?? []).length !== 1 && 's'}
              </p>
            </div>
            <div className="text-right">
              <p className="text-sm font-semibold text-gray-900">
                {formatCurrency(order.total)}
              </p>
              <p className="text-xs text-gray-400">{timeAgo(order.createdAt)}</p>
            </div>
          </div>
        ))}
      </div>

      {orders.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <UtensilsCrossed className="h-10 w-10 text-gray-300" />
          <p className="mt-3 text-sm text-gray-500">No recent orders</p>
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
      iconBg: 'bg-brand-50',
      iconColor: 'text-brand-600',
    },
    {
      label: 'View Orders',
      description: 'Manage live orders',
      href: '/orders',
      icon: ClipboardList,
      iconBg: 'bg-blue-50',
      iconColor: 'text-blue-600',
    },
    {
      label: 'Update Profile',
      description: 'Edit kitchen details',
      href: '/profile',
      icon: UserCog,
      iconBg: 'bg-purple-50',
      iconColor: 'text-purple-600',
    },
    {
      label: 'View Earnings',
      description: 'Track your revenue',
      href: '/earnings',
      icon: Wallet,
      iconBg: 'bg-green-50',
      iconColor: 'text-green-600',
    },
  ];

  return (
    <motion.div variants={fadeInUp}>
      <h3 className="mb-3 text-base font-semibold text-gray-900">
        Quick Actions
      </h3>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <Link
              key={action.href}
              to={action.href}
              className="group rounded-xl border border-gray-100 bg-white p-4 shadow-sm transition-all hover:border-brand-200 hover:shadow-md"
            >
              <div
                className={`flex h-10 w-10 items-center justify-center rounded-lg ${action.iconBg} transition-transform group-hover:scale-110`}
              >
                <Icon className={`h-5 w-5 ${action.iconColor}`} />
              </div>
              <p className="mt-3 text-sm font-semibold text-gray-900">
                {action.label}
              </p>
              <p className="mt-0.5 text-xs text-gray-500">
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
  const handleAcceptOrder = (orderId: string) => {
    // TODO: useMutation to PATCH /chef/orders/:id/accept
    console.log('Accept order', orderId);
  };

  const handleRejectOrder = (orderId: string) => {
    // TODO: useMutation to PATCH /chef/orders/:id/reject
    console.log('Reject order', orderId);
  };

  // Loading skeleton
  if (statsLoading || ordersLoading) {
    return (
      <div className="space-y-6">
        {/* Page header skeleton */}
        <div>
          <div className="h-7 w-48 animate-pulse rounded-lg bg-gray-200" />
          <div className="mt-2 h-4 w-72 animate-pulse rounded bg-gray-100" />
        </div>

        {/* Stats skeleton */}
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="h-32 animate-pulse rounded-xl border border-gray-100 bg-white"
            />
          ))}
        </div>

        {/* Chart skeleton */}
        <div className="h-64 animate-pulse rounded-xl border border-gray-100 bg-white" />

        {/* Table skeleton */}
        <div className="h-80 animate-pulse rounded-xl border border-gray-100 bg-white" />
      </div>
    );
  }

  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Page header */}
      <motion.div variants={fadeInUp} className="page-header">
        <div>
          <h1 className="page-title text-2xl font-bold text-gray-900">
            Dashboard
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Welcome back! Here's how your kitchen is performing today.
          </p>
        </div>

        <Link to="/menu/new">
          <Button size="sm" leftIcon={<Plus className="h-4 w-4" />}>
            Add Item
          </Button>
        </Link>
      </motion.div>

      {/* Pending orders alert */}
      {pendingOrders.length > 0 && (
        <motion.div
          variants={fadeInUp}
          className="flex items-center gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3"
        >
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-amber-100">
            <Clock className="h-4 w-4 text-amber-600" />
          </div>
          <div className="flex-1">
            <p className="text-sm font-semibold text-amber-800">
              {pendingOrders.length} order{pendingOrders.length !== 1 && 's'}{' '}
              waiting for your response
            </p>
            <p className="text-xs text-amber-600">
              Accept or reject before they expire
            </p>
          </div>
          <Link
            to="/orders"
            className="text-sm font-medium text-amber-700 hover:text-amber-800"
          >
            View all
          </Link>
        </motion.div>
      )}

      {/* Stats cards */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatsCard
          title="Today's Revenue"
          value={formatCurrency(stats?.todayRevenue ?? 0)}
          icon={IndianRupee}
          trend={12}
          trendLabel="vs yesterday"
          iconBg="bg-green-50"
          iconColor="text-green-600"
        />
        <StatsCard
          title="Today's Orders"
          value={String(stats?.todayOrders ?? 0)}
          icon={ShoppingBag}
          trend={8}
          trendLabel="vs yesterday"
          iconBg="bg-blue-50"
          iconColor="text-blue-600"
        />
        <StatsCard
          title="Pending Orders"
          value={String(stats?.pendingOrders ?? 0)}
          icon={Clock}
          iconBg="bg-amber-50"
          iconColor="text-amber-600"
        />
        <StatsCard
          title="Average Rating"
          value={`${(stats?.rating ?? 0).toFixed(1)} / 5`}
          icon={Star}
          trend={2}
          iconBg="bg-brand-50"
          iconColor="text-brand-600"
        />
      </div>

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
            <h3 className="text-base font-semibold text-gray-900">
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
                  className="block rounded-lg border border-dashed border-gray-200 py-3 text-center text-sm font-medium text-brand-600 transition-colors hover:border-brand-300 hover:bg-brand-50/50"
                >
                  View {pendingOrders.length - 3} more pending orders
                </Link>
              )}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-gray-200 py-10 text-center">
              <Check className="h-10 w-10 text-green-300" />
              <p className="mt-3 text-sm font-medium text-gray-600">
                All caught up!
              </p>
              <p className="mt-1 text-xs text-gray-400">
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
