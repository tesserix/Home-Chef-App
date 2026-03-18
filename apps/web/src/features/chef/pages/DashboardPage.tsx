import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  DollarSign,
  ShoppingBag,
  Star,
  TrendingUp,
  Clock,
  ChefHat,
  ArrowUpRight,
  ArrowRight,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { useAuth } from '@/app/providers/AuthProvider';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Card, Badge, Button } from '@/shared/components/ui';
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

  return (
    <motion.div
      initial="hidden"
      animate="visible"
      variants={staggerContainer}
      className="space-y-6"
    >
      {/* Header */}
      <motion.div variants={fadeInUp}>
        <h1 className="font-display text-display-xs text-gray-900">
          Welcome back, {user?.firstName}!
        </h1>
        <p className="mt-1 text-gray-600">
          Here's what's happening with your kitchen today
        </p>
      </motion.div>

      {/* Stats Grid */}
      <motion.div variants={fadeInUp} className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Today's Revenue"
          value={fp(stats?.todayRevenue || 0)}
          change={stats?.revenueChange}
          icon={DollarSign}
          color="green"
        />
        <StatCard
          title="Today's Orders"
          value={stats?.todayOrders?.toString() || '0'}
          change={stats?.ordersChange}
          icon={ShoppingBag}
          color="blue"
        />
        <StatCard
          title="Pending Orders"
          value={stats?.pendingOrders?.toString() || '0'}
          icon={Clock}
          color="yellow"
          urgent={stats?.pendingOrders ? stats.pendingOrders > 0 : false}
        />
        <StatCard
          title="Your Rating"
          value={stats?.rating?.toFixed(1) || '0.0'}
          subtitle={`${stats?.totalReviews || 0} reviews`}
          icon={Star}
          color="golden"
        />
      </motion.div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Pending Orders */}
        <motion.div variants={fadeInUp}>
          <Card variant="default" padding="lg">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">Pending Orders</h2>
              <Link to="/chef/orders">
                <Button variant="link" size="sm">View all</Button>
              </Link>
            </div>

            {(pendingOrders?.data ?? []).length === 0 ? (
              <div className="mt-6 text-center py-8">
                <ChefHat className="mx-auto h-12 w-12 text-gray-300" />
                <p className="mt-3 text-gray-500">No pending orders</p>
                <p className="text-sm text-gray-400">New orders will appear here</p>
              </div>
            ) : (
              <div className="mt-4 space-y-3">
                {(pendingOrders?.data ?? []).slice(0, 4).map((order) => (
                  <PendingOrderCard key={order.id} order={order} />
                ))}
              </div>
            )}
          </Card>
        </motion.div>

        {/* Quick Actions */}
        <motion.div variants={fadeInUp} className="space-y-6">
          <Card variant="default" padding="lg">
            <h2 className="text-lg font-semibold text-gray-900">Quick Actions</h2>
            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <QuickActionLink
                to="/chef/menu"
                icon={ChefHat}
                color="brand"
                title="Manage Menu"
                subtitle="Add or edit items"
              />
              <QuickActionLink
                to="/chef/orders"
                icon={ShoppingBag}
                color="blue"
                title="View Orders"
                subtitle="Manage all orders"
              />
              <QuickActionLink
                to="/chef/earnings"
                icon={DollarSign}
                color="green"
                title="Earnings"
                subtitle="Track your income"
              />
              <QuickActionLink
                to="/chef/social"
                icon={TrendingUp}
                color="purple"
                title="Social Feed"
                subtitle="Share your creations"
              />
            </div>
          </Card>

          {/* Pro Tip */}
          <Card variant="premium" padding="lg" className="text-white">
            <h3 className="font-semibold">Pro Tip</h3>
            <p className="mt-2 text-brand-100">
              Customers love seeing photos of your dishes! Post on the social feed
              to increase visibility and attract more orders.
            </p>
            <Link to="/chef/social">
              <Button variant="ghost" size="sm" className="mt-4 bg-white/20 hover:bg-white/30 text-white">
                Create a post
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </Link>
          </Card>
        </motion.div>
      </div>

      {/* Recent Orders */}
      <motion.div variants={fadeInUp}>
        <Card variant="default" padding="lg">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Recent Orders</h2>
            <Link to="/chef/orders">
              <Button variant="link" size="sm" rightIcon={<ArrowRight className="h-4 w-4" />}>
                View all
              </Button>
            </Link>
          </div>

          <div className="mt-4 overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b text-left text-sm text-gray-500">
                  <th className="pb-3 font-medium">Order</th>
                  <th className="pb-3 font-medium">Items</th>
                  <th className="pb-3 font-medium">Total</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium">Time</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {(recentOrders?.data ?? []).map((order) => (
                  <tr key={order.id} className="text-sm">
                    <td className="py-3">
                      <Link
                        to={`/chef/orders/${order.id}`}
                        className="font-medium text-brand-600 hover:underline"
                      >
                        #{order.orderNumber}
                      </Link>
                    </td>
                    <td className="py-3 text-gray-600">
                      {order.items.length} item(s)
                    </td>
                    <td className="py-3 font-medium text-gray-900">
                      {fp(order.total)}
                    </td>
                    <td className="py-3">
                      <OrderStatusBadge status={order.status} />
                    </td>
                    <td className="py-3 text-gray-500">
                      {new Date(order.createdAt).toLocaleTimeString('en-US', {
                        hour: 'numeric',
                        minute: '2-digit',
                      })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </motion.div>
    </motion.div>
  );
}

function StatCard({
  title,
  value,
  change,
  subtitle,
  icon: Icon,
  color,
  urgent,
}: {
  title: string;
  value: string;
  change?: number;
  subtitle?: string;
  icon: typeof DollarSign;
  color: 'green' | 'blue' | 'yellow' | 'golden';
  urgent?: boolean;
}) {
  const colorClasses = {
    green: 'bg-fresh-100 text-fresh-600',
    blue: 'bg-blue-100 text-blue-600',
    yellow: 'bg-yellow-100 text-yellow-600',
    golden: 'bg-golden-100 text-golden-600',
  };

  return (
    <Card
      variant="default"
      padding="lg"
      className={urgent ? 'ring-2 ring-yellow-400' : ''}
    >
      <div className="flex items-center justify-between">
        <div className={`flex h-12 w-12 items-center justify-center rounded-xl ${colorClasses[color]}`}>
          <Icon className="h-6 w-6" />
        </div>
        {change !== undefined && (
          <span className={`flex items-center gap-1 text-sm font-medium ${change >= 0 ? 'text-fresh-600' : 'text-red-600'}`}>
            <ArrowUpRight className={`h-4 w-4 ${change < 0 ? 'rotate-180' : ''}`} />
            {Math.abs(change)}%
          </span>
        )}
      </div>
      <p className="mt-4 text-2xl font-bold text-gray-900">{value}</p>
      <p className="text-sm text-gray-500">{subtitle || title}</p>
    </Card>
  );
}

function QuickActionLink({
  to,
  icon: Icon,
  color,
  title,
  subtitle,
}: {
  to: string;
  icon: typeof ChefHat;
  color: 'brand' | 'blue' | 'green' | 'purple';
  title: string;
  subtitle: string;
}) {
  const colorClasses = {
    brand: 'bg-brand-100 text-brand-600',
    blue: 'bg-blue-100 text-blue-600',
    green: 'bg-fresh-100 text-fresh-600',
    purple: 'bg-purple-100 text-purple-600',
  };

  return (
    <Link
      to={to}
      className="flex items-center gap-3 rounded-xl border border-gray-200 p-4 hover:bg-gray-50 hover:border-gray-300 transition-all"
    >
      <div className={`flex h-10 w-10 items-center justify-center rounded-xl ${colorClasses[color]}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="font-medium text-gray-900">{title}</p>
        <p className="text-sm text-gray-500">{subtitle}</p>
      </div>
    </Link>
  );
}

function PendingOrderCard({ order }: { order: Order }) {
  const fp = useFormatPrice();
  const isNew = order.status === 'pending';

  return (
    <Link
      to={`/chef/orders/${order.id}`}
      className={`flex items-center justify-between rounded-xl border p-4 transition-all hover:bg-gray-50 ${
        isNew ? 'border-yellow-300 bg-yellow-50 hover:bg-yellow-100' : 'border-gray-200'
      }`}
    >
      <div className="flex items-center gap-3">
        {isNew && (
          <span className="flex h-2 w-2 rounded-full bg-yellow-500 animate-pulse" />
        )}
        <div>
          <p className="font-medium text-gray-900">#{order.orderNumber}</p>
          <p className="text-sm text-gray-500">
            {order.items.length} items - {fp(order.total)}
          </p>
        </div>
      </div>
      <div className="text-right">
        <OrderStatusBadge status={order.status} />
        <p className="mt-1 text-xs text-gray-400">
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
