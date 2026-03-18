import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useFormatPrice } from '@/shared/utils/format-price';
import {
  Users,
  ChefHat,
  ShoppingBag,
  DollarSign,
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  Clock,
  Loader2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';

interface AdminStats {
  totalUsers: number;
  newUsersToday: number;
  totalChefs: number;
  pendingVerifications: number;
  totalOrders: number;
  ordersToday: number;
  revenue: number;
  revenueToday: number;
  revenueChange: number;
  ordersChange: number;
}

interface RecentActivity {
  id: string;
  type: 'order' | 'user' | 'chef' | 'report';
  title: string;
  description: string;
  timestamp: string;
}

export default function AdminDashboardPage() {
  const fp = useFormatPrice();

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['admin-stats'],
    queryFn: () => apiClient.get<AdminStats>('/admin/stats'),
  });

  const { data: activities } = useQuery({
    queryKey: ['admin-activities'],
    queryFn: () => apiClient.get<RecentActivity[]>('/admin/activities'),
  });

  if (statsLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">Dashboard</h1>
        <p className="mt-1 text-gray-400">Platform overview and analytics</p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Users"
          value={stats?.totalUsers?.toLocaleString() || '0'}
          subtext={`+${stats?.newUsersToday || 0} today`}
          icon={Users}
          color="blue"
        />
        <StatCard
          title="Active Chefs"
          value={stats?.totalChefs?.toLocaleString() || '0'}
          subtext={`${stats?.pendingVerifications || 0} pending`}
          icon={ChefHat}
          color="purple"
          alert={stats?.pendingVerifications ? stats.pendingVerifications > 0 : false}
        />
        <StatCard
          title="Total Orders"
          value={stats?.totalOrders?.toLocaleString() || '0'}
          subtext={`${stats?.ordersToday || 0} today`}
          change={stats?.ordersChange}
          icon={ShoppingBag}
          color="green"
        />
        <StatCard
          title="Revenue"
          value={fp(stats?.revenue || 0)}
          subtext={`${fp(stats?.revenueToday || 0)} today`}
          change={stats?.revenueChange}
          icon={DollarSign}
          color="orange"
        />
      </div>

      {/* Alerts */}
      {stats?.pendingVerifications && stats.pendingVerifications > 0 && (
        <div className="flex items-center gap-3 rounded-lg bg-yellow-500/10 border border-yellow-500/20 p-4">
          <AlertTriangle className="h-5 w-5 text-yellow-500" />
          <div className="flex-1">
            <p className="font-medium text-yellow-500">
              {stats.pendingVerifications} chef verification{stats.pendingVerifications > 1 ? 's' : ''} pending
            </p>
            <p className="text-sm text-yellow-500/70">Review pending chef applications</p>
          </div>
          <Link to="/admin/chefs?status=pending" className="btn-base bg-yellow-500 text-black hover:bg-yellow-400">
            Review Now
          </Link>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Quick Actions */}
        <div className="rounded-xl bg-gray-800 p-6">
          <h2 className="text-lg font-semibold text-white">Quick Actions</h2>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <Link
              to="/admin/users"
              className="flex items-center gap-3 rounded-lg bg-gray-700/50 p-4 hover:bg-gray-700 transition-colors"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/20">
                <Users className="h-5 w-5 text-blue-400" />
              </div>
              <div>
                <p className="font-medium text-white">Manage Users</p>
                <p className="text-sm text-gray-400">View all users</p>
              </div>
            </Link>
            <Link
              to="/admin/chefs"
              className="flex items-center gap-3 rounded-lg bg-gray-700/50 p-4 hover:bg-gray-700 transition-colors"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/20">
                <ChefHat className="h-5 w-5 text-purple-400" />
              </div>
              <div>
                <p className="font-medium text-white">Chef Verification</p>
                <p className="text-sm text-gray-400">Review applications</p>
              </div>
            </Link>
            <Link
              to="/admin/orders"
              className="flex items-center gap-3 rounded-lg bg-gray-700/50 p-4 hover:bg-gray-700 transition-colors"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/20">
                <ShoppingBag className="h-5 w-5 text-green-400" />
              </div>
              <div>
                <p className="font-medium text-white">Order Management</p>
                <p className="text-sm text-gray-400">Track all orders</p>
              </div>
            </Link>
            <Link
              to="/admin/analytics"
              className="flex items-center gap-3 rounded-lg bg-gray-700/50 p-4 hover:bg-gray-700 transition-colors"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-orange-500/20">
                <TrendingUp className="h-5 w-5 text-orange-400" />
              </div>
              <div>
                <p className="font-medium text-white">Analytics</p>
                <p className="text-sm text-gray-400">View reports</p>
              </div>
            </Link>
          </div>
        </div>

        {/* Recent Activity */}
        <div className="rounded-xl bg-gray-800 p-6">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-white">Recent Activity</h2>
            <Link to="/admin/activities" className="text-sm text-brand-400 hover:text-brand-300">
              View all
            </Link>
          </div>

          <div className="mt-4 space-y-3">
            {activities?.slice(0, 5).map((activity) => (
              <div key={activity.id} className="flex items-start gap-3 rounded-lg bg-gray-700/50 p-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-600">
                  <ActivityIcon type={activity.type} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-white truncate">{activity.title}</p>
                  <p className="text-sm text-gray-400 truncate">{activity.description}</p>
                </div>
                <span className="text-xs text-gray-500 whitespace-nowrap">
                  {formatTimeAgo(activity.timestamp)}
                </span>
              </div>
            ))}
            {(!activities || activities.length === 0) && (
              <p className="text-center text-gray-500 py-4">No recent activity</p>
            )}
          </div>
        </div>
      </div>

      {/* Charts Placeholder */}
      <div className="grid gap-6 lg:grid-cols-2">
        <div className="rounded-xl bg-gray-800 p-6">
          <h2 className="text-lg font-semibold text-white">Orders Overview</h2>
          <div className="mt-6 h-64 flex items-center justify-center border border-dashed border-gray-700 rounded-lg">
            <div className="text-center text-gray-500">
              <TrendingUp className="mx-auto h-12 w-12" />
              <p className="mt-2">Orders chart</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl bg-gray-800 p-6">
          <h2 className="text-lg font-semibold text-white">Revenue Overview</h2>
          <div className="mt-6 h-64 flex items-center justify-center border border-dashed border-gray-700 rounded-lg">
            <div className="text-center text-gray-500">
              <DollarSign className="mx-auto h-12 w-12" />
              <p className="mt-2">Revenue chart</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({
  value,
  subtext,
  change,
  icon: Icon,
  color,
  alert,
}: {
  title?: string;
  value: string;
  subtext: string;
  change?: number;
  icon: typeof Users;
  color: 'blue' | 'purple' | 'green' | 'orange';
  alert?: boolean;
}) {
  const colorClasses = {
    blue: 'bg-blue-500/20 text-blue-400',
    purple: 'bg-purple-500/20 text-purple-400',
    green: 'bg-green-500/20 text-green-400',
    orange: 'bg-orange-500/20 text-orange-400',
  };

  return (
    <div className={`rounded-xl bg-gray-800 p-6 ${alert ? 'ring-2 ring-yellow-500' : ''}`}>
      <div className="flex items-center justify-between">
        <div className={`flex h-12 w-12 items-center justify-center rounded-lg ${colorClasses[color]}`}>
          <Icon className="h-6 w-6" />
        </div>
        {change !== undefined && (
          <span className={`flex items-center gap-1 text-sm ${change >= 0 ? 'text-green-400' : 'text-red-400'}`}>
            {change >= 0 ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
            {Math.abs(change)}%
          </span>
        )}
      </div>
      <p className="mt-4 text-3xl font-bold text-white">{value}</p>
      <p className="mt-1 text-sm text-gray-400">{subtext}</p>
    </div>
  );
}

function ActivityIcon({ type }: { type: string }) {
  switch (type) {
    case 'order':
      return <ShoppingBag className="h-4 w-4 text-green-400" />;
    case 'user':
      return <Users className="h-4 w-4 text-blue-400" />;
    case 'chef':
      return <ChefHat className="h-4 w-4 text-purple-400" />;
    case 'report':
      return <AlertTriangle className="h-4 w-4 text-yellow-400" />;
    default:
      return <Clock className="h-4 w-4 text-gray-400" />;
  }
}

function formatTimeAgo(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 60) return 'Just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}
