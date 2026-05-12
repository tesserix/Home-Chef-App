import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Users,
  ChefHat,
  ShoppingBag,
  AlertTriangle,
  Clock,
  Loader2,
  ArrowUpRight,
  ArrowDownRight,
  ArrowRight,
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
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  const revenueChange = stats?.revenueChange;
  const positive = revenueChange !== undefined && revenueChange >= 0;
  const pendingChefs = stats?.pendingVerifications ?? 0;

  const formatINR = (amount: number) =>
    new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      maximumFractionDigits: 0,
    }).format(amount);

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
          Dashboard
        </h1>
        <p className="mt-1 text-sm text-ink-soft">Platform overview and analytics.</p>
      </header>

      {/* Lead block — Revenue today + pending verifications CTA */}
      <section className="grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:items-end">
        <div>
          <p className="text-sm text-ink-soft">Today's revenue</p>
          <p className="mt-1 text-4xl font-semibold tabular-nums tracking-tight text-foreground sm:text-5xl">
            {formatINR(stats?.revenueToday || 0)}
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

        {pendingChefs > 0 ? (
          <Link
            to="/chefs?status=pending"
            aria-label={`${pendingChefs} chef verifications pending`}
            className="group flex items-center justify-between gap-4 rounded-lg bg-amber px-5 py-4 text-foreground transition-colors hover:bg-amber/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber focus-visible:ring-offset-2"
          >
            <div>
              <p className="text-3xl font-semibold tabular-nums">
                {pendingChefs} chef{pendingChefs !== 1 ? 's' : ''}
              </p>
              <p className="mt-0.5 text-sm">Pending verification</p>
            </div>
            <ArrowRight className="h-5 w-5 transition-transform group-hover:translate-x-0.5" aria-hidden />
          </Link>
        ) : (
          <div className="rounded-lg border border-mist bg-bone px-5 py-4">
            <p className="text-3xl font-semibold tabular-nums text-foreground">All clear</p>
            <p className="mt-0.5 text-sm text-ink-soft">No pending chef applications.</p>
          </div>
        )}
      </section>

      {/* Status strip */}
      <section
        aria-label="Platform at a glance"
        className="grid grid-cols-2 divide-y divide-mist border-y border-mist sm:grid-cols-4 sm:divide-x sm:divide-y-0"
      >
        <StatRow
          label="Total users"
          value={stats?.totalUsers?.toLocaleString() ?? '0'}
          subtitle={`+${stats?.newUsersToday ?? 0} today`}
        />
        <StatRow
          label="Active chefs"
          value={stats?.totalChefs?.toLocaleString() ?? '0'}
        />
        <StatRow
          label="Orders today"
          value={stats?.ordersToday?.toLocaleString() ?? '0'}
          subtitle={
            stats?.ordersChange !== undefined
              ? `${stats.ordersChange >= 0 ? '+' : ''}${stats.ordersChange}% vs. last week`
              : undefined
          }
        />
        <StatRow
          label="Total revenue"
          value={formatINR(stats?.revenue || 0)}
          subtitle={`${stats?.totalOrders?.toLocaleString() ?? '0'} all-time`}
        />
      </section>

      {/* Work area */}
      <div className="grid gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <section aria-label="Recent activity" className="space-y-3">
          <div className="flex items-baseline justify-between">
            <h2 className="text-lg font-semibold text-foreground">Recent activity</h2>
            <Link to="/analytics" className="text-sm font-medium text-herb hover:underline">
              View all
            </Link>
          </div>

          {(!activities || activities.length === 0) ? (
            <div className="rounded-lg border border-mist bg-bone py-12 text-center">
              <Clock className="mx-auto h-10 w-10 text-ink-muted" aria-hidden />
              <p className="mt-3 font-medium text-foreground">No recent activity</p>
              <p className="mt-1 text-sm text-ink-soft">Platform events will appear here.</p>
            </div>
          ) : (
            <div className="divide-y divide-mist rounded-lg border border-mist bg-bone">
              {activities.slice(0, 8).map((activity) => (
                <div key={activity.id} className="flex items-start gap-3 px-4 py-3">
                  <ActivityIcon type={activity.type} />
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium text-foreground">{activity.title}</p>
                    <p className="truncate text-sm text-ink-soft">{activity.description}</p>
                  </div>
                  <span className="shrink-0 text-xs text-ink-muted tabular-nums">
                    {formatTimeAgo(activity.timestamp)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </section>

        <aside aria-label="Shortcuts" className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground">Shortcuts</h2>
          <nav className="divide-y divide-mist rounded-lg border border-mist bg-bone">
            <QuickAction to="/users" title="Manage users" subtitle="View all users" />
            <QuickAction to="/chefs" title="Chef verification" subtitle="Review applications" />
            <QuickAction to="/orders" title="Order management" subtitle="Track all orders" />
            <QuickAction to="/analytics" title="Analytics" subtitle="View reports" />
          </nav>
        </aside>
      </div>
    </div>
  );
}

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

function QuickAction({
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

function ActivityIcon({ type }: { type: string }) {
  const Icon =
    type === 'order'
      ? ShoppingBag
      : type === 'user'
        ? Users
        : type === 'chef'
          ? ChefHat
          : type === 'report'
            ? AlertTriangle
            : Clock;
  return <Icon className="h-4 w-4 shrink-0 text-ink-muted" aria-hidden />;
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
