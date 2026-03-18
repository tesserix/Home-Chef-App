import {
  BarChart3,
  TrendingUp,
  Users,
  DollarSign,
  ShoppingBag,
  ChefHat,
} from 'lucide-react';

export default function AnalyticsPage() {
  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Analytics</h1>
        <p className="page-description">Platform performance metrics and insights</p>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <SummaryCard icon={Users} label="Total Users" value="--" change="+--%" />
        <SummaryCard icon={ChefHat} label="Active Chefs" value="--" change="+--%" />
        <SummaryCard icon={ShoppingBag} label="Orders This Month" value="--" change="+--%" />
        <SummaryCard icon={DollarSign} label="Revenue This Month" value="₹--" change="+--%" />
      </div>

      {/* Charts Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        <ChartPlaceholder title="Orders Trend" icon={TrendingUp} />
        <ChartPlaceholder title="Revenue Trend" icon={DollarSign} />
        <ChartPlaceholder title="User Growth" icon={Users} />
        <ChartPlaceholder title="Top Cuisines" icon={BarChart3} />
      </div>
    </div>
  );
}

function SummaryCard({
  icon: Icon,
  label,
  value,
  change,
}: {
  icon: typeof Users;
  label: string;
  value: string;
  change: string;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <div className="flex items-center justify-between">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <Icon className="h-5 w-5 text-primary" />
        </div>
        <span className="text-sm text-success">{change}</span>
      </div>
      <p className="mt-3 text-2xl font-bold text-foreground">{value}</p>
      <p className="mt-1 text-sm text-muted-foreground">{label}</p>
    </div>
  );
}

function ChartPlaceholder({ title, icon: Icon }: { title: string; icon: typeof BarChart3 }) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <h3 className="text-lg font-semibold text-foreground">{title}</h3>
      <div className="mt-6 flex h-64 items-center justify-center rounded-lg border border-dashed border-border">
        <div className="text-center text-muted-foreground">
          <Icon className="mx-auto h-12 w-12" />
          <p className="mt-2 text-sm">Chart coming soon</p>
        </div>
      </div>
    </div>
  );
}
