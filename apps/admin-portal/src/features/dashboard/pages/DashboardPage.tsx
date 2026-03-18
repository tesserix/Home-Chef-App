import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
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

export default function DashboardPage() {
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
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="page-header">
        <h1 className="page-title">Dashboard</h1>
        <p className="page-description">Platform overview and analytics</p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Users"
          value={stats?.totalUsers?.toLocaleString() || '0'}
          subtext={`+${stats?.newUsersToday || 0} today`}
          icon={Users}
          color="info"
        />
        <StatCard
          title="Active Chefs"
          value={stats?.totalChefs?.toLocaleString() || '0'}
          subtext={`${stats?.pendingVerifications || 0} pending`}
          icon={ChefHat}
          color="primary"
          alert={stats?.pendingVerifications ? stats.pendingVerifications > 0 : false}
        />
        <StatCard
          title="Total Orders"
          value={stats?.totalOrders?.toLocaleString() || '0'}
          subtext={`${stats?.ordersToday || 0} today`}
          change={stats?.ordersChange}
          icon={ShoppingBag}
          color="success"
        />
        <StatCard
          title="Revenue"
          value={`₹${(stats?.revenue || 0).toLocaleString()}`}
          subtext={`₹${(stats?.revenueToday || 0).toLocaleString()} today`}
          change={stats?.revenueChange}
          icon={DollarSign}
          color="warning"
        />
      </div>

      {/* Alerts */}
      {stats?.pendingVerifications && stats.pendingVerifications > 0 && (
        <div className="flex items-center gap-3 rounded-xl border border-warning/30 bg-warning/5 p-4">
          <AlertTriangle className="h-5 w-5 text-warning" />
          <div className="flex-1">
            <p className="font-medium text-foreground">
              {stats.pendingVerifications} chef verification{stats.pendingVerifications > 1 ? 's' : ''} pending
            </p>
            <p className="text-sm text-muted-foreground">Review pending chef applications</p>
          </div>
          <Link
            to="/chefs?status=pending"
            className="inline-flex items-center rounded-lg bg-warning px-3 py-1.5 text-sm font-medium text-warning-foreground hover:bg-warning/90 transition-colors"
          >
            Review Now
          </Link>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Quick Actions */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground">Quick Actions</h2>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <QuickAction
              href="/users"
              icon={Users}
              title="Manage Users"
              description="View all users"
              color="info"
            />
            <QuickAction
              href="/chefs"
              icon={ChefHat}
              title="Chef Verification"
              description="Review applications"
              color="primary"
            />
            <QuickAction
              href="/orders"
              icon={ShoppingBag}
              title="Order Management"
              description="Track all orders"
              color="success"
            />
            <QuickAction
              href="/analytics"
              icon={TrendingUp}
              title="Analytics"
              description="View reports"
              color="warning"
            />
          </div>
        </div>

        {/* Recent Activity */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">Recent Activity</h2>
            <Link to="/analytics" className="text-sm text-primary hover:text-primary/80 transition-colors">
              View all
            </Link>
          </div>

          <div className="mt-4 space-y-3">
            {activities?.slice(0, 5).map((activity) => (
              <div key={activity.id} className="flex items-start gap-3 rounded-lg bg-secondary/50 p-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                  <ActivityIcon type={activity.type} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-foreground truncate">{activity.title}</p>
                  <p className="text-sm text-muted-foreground truncate">{activity.description}</p>
                </div>
                <span className="text-xs text-muted-foreground whitespace-nowrap">
                  {formatTimeAgo(activity.timestamp)}
                </span>
              </div>
            ))}
            {(!activities || activities.length === 0) && (
              <p className="text-center text-muted-foreground py-4">No recent activity</p>
            )}
          </div>
        </div>
      </div>

      {/* Charts Placeholder */}
      <div className="grid gap-6 lg:grid-cols-2">
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground">Orders Overview</h2>
          <div className="mt-6 h-64 flex items-center justify-center border border-dashed border-border rounded-lg">
            <div className="text-center text-muted-foreground">
              <TrendingUp className="mx-auto h-12 w-12" />
              <p className="mt-2">Orders chart coming soon</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground">Revenue Overview</h2>
          <div className="mt-6 h-64 flex items-center justify-center border border-dashed border-border rounded-lg">
            <div className="text-center text-muted-foreground">
              <DollarSign className="mx-auto h-12 w-12" />
              <p className="mt-2">Revenue chart coming soon</p>
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
  color: 'info' | 'primary' | 'success' | 'warning';
  alert?: boolean;
}) {
  const colorClasses = {
    info: 'bg-info/10 text-info',
    primary: 'bg-primary/10 text-primary',
    success: 'bg-success/10 text-success',
    warning: 'bg-warning/10 text-warning',
  };

  return (
    <div className={`rounded-xl border border-border bg-card p-6 shadow-card ${alert ? 'ring-2 ring-warning' : ''}`}>
      <div className="flex items-center justify-between">
        <div className={`flex h-12 w-12 items-center justify-center rounded-lg ${colorClasses[color]}`}>
          <Icon className="h-6 w-6" />
        </div>
        {change !== undefined && (
          <span className={`flex items-center gap-1 text-sm ${change >= 0 ? 'text-success' : 'text-destructive'}`}>
            {change >= 0 ? <TrendingUp className="h-4 w-4" /> : <TrendingDown className="h-4 w-4" />}
            {Math.abs(change)}%
          </span>
        )}
      </div>
      <p className="mt-4 text-3xl font-bold text-foreground">{value}</p>
      <p className="mt-1 text-sm text-muted-foreground">{subtext}</p>
    </div>
  );
}

function QuickAction({
  href,
  icon: Icon,
  title,
  description,
  color,
}: {
  href: string;
  icon: typeof Users;
  title: string;
  description: string;
  color: 'info' | 'primary' | 'success' | 'warning';
}) {
  const colorClasses = {
    info: 'bg-info/10 text-info',
    primary: 'bg-primary/10 text-primary',
    success: 'bg-success/10 text-success',
    warning: 'bg-warning/10 text-warning',
  };

  return (
    <Link
      to={href}
      className="flex items-center gap-3 rounded-lg border border-border bg-secondary/30 p-4 hover:bg-secondary/60 transition-colors"
    >
      <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${colorClasses[color]}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="font-medium text-foreground">{title}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </Link>
  );
}

function ActivityIcon({ type }: { type: string }) {
  switch (type) {
    case 'order':
      return <ShoppingBag className="h-4 w-4 text-success" />;
    case 'user':
      return <Users className="h-4 w-4 text-info" />;
    case 'chef':
      return <ChefHat className="h-4 w-4 text-primary" />;
    case 'report':
      return <AlertTriangle className="h-4 w-4 text-warning" />;
    default:
      return <Clock className="h-4 w-4 text-muted-foreground" />;
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
