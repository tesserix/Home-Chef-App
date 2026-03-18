import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { Link } from 'react-router-dom';
import {
  Wifi,
  Truck,
  AlertCircle,
  CheckCircle2,
  Users,
  ShieldCheck,
  Clock,
  DollarSign,
  ArrowRight,
} from 'lucide-react';
import { PageLoader } from '@/shared/components/LoadingScreen';

interface FleetOverview {
  onlinePartners: number;
  activeDeliveries: number;
  unassignedOrders: number;
  todayCompleted: number;
  totalPartners: number;
  verifiedPartners: number;
  pendingVerification: number;
  todayEarnings: number;
}

export default function FleetOverviewPage() {
  const { data: overview, isLoading } = useQuery({
    queryKey: ['fleet-overview'],
    queryFn: () => apiClient.get<FleetOverview>('/delivery/staff/fleet/overview'),
    refetchInterval: 30000,
  });

  if (isLoading) return <PageLoader />;

  const liveStats = [
    {
      label: 'Online Partners',
      value: overview?.onlinePartners ?? 0,
      icon: Wifi,
      color: 'text-success',
      bg: 'bg-success/10',
    },
    {
      label: 'Active Deliveries',
      value: overview?.activeDeliveries ?? 0,
      icon: Truck,
      color: 'text-primary',
      bg: 'bg-primary/10',
    },
    {
      label: 'Unassigned Orders',
      value: overview?.unassignedOrders ?? 0,
      icon: AlertCircle,
      color: 'text-warning',
      bg: 'bg-warning/10',
    },
    {
      label: "Today's Completed",
      value: overview?.todayCompleted ?? 0,
      icon: CheckCircle2,
      color: 'text-success',
      bg: 'bg-success/10',
    },
  ];

  const partnerStats = [
    {
      label: 'Total Partners',
      value: overview?.totalPartners ?? 0,
      icon: Users,
    },
    {
      label: 'Verified',
      value: overview?.verifiedPartners ?? 0,
      icon: ShieldCheck,
    },
    {
      label: 'Pending Verification',
      value: overview?.pendingVerification ?? 0,
      icon: Clock,
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="page-title">Fleet Overview</h1>
          <p className="page-description">Monitor your delivery fleet in real time</p>
        </div>
        <Link
          to="/fleet/partners"
          className="flex items-center gap-2 rounded-xl border border-border px-4 py-2.5 text-sm font-medium text-foreground hover:bg-secondary"
        >
          View Partners
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>

      {/* Live Stats */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {liveStats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div
              key={stat.label}
              className="rounded-xl border border-border bg-card p-5"
            >
              <div className="flex items-center gap-2 text-muted-foreground mb-2">
                <div className={`rounded-lg p-1.5 ${stat.bg}`}>
                  <Icon className={`h-4 w-4 ${stat.color}`} />
                </div>
                <span className="text-xs font-medium">{stat.label}</span>
              </div>
              <p className="text-3xl font-bold text-foreground">{stat.value}</p>
            </div>
          );
        })}
      </div>

      {/* Partner Stats */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">
          Partner Summary
        </h2>
        <div className="grid grid-cols-3 gap-4 text-center">
          {partnerStats.map((stat) => {
            const Icon = stat.icon;
            return (
              <div key={stat.label}>
                <div className="flex justify-center mb-2">
                  <Icon className="h-5 w-5 text-muted-foreground" />
                </div>
                <p className="text-2xl font-bold text-foreground">{stat.value}</p>
                <p className="text-xs text-muted-foreground">{stat.label}</p>
              </div>
            );
          })}
        </div>
      </div>

      {/* Today's Earnings */}
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <DollarSign className="h-6 w-6 text-primary" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Today's Fleet Earnings</p>
            <p className="text-2xl font-bold text-foreground">
              ${(overview?.todayEarnings ?? 0).toFixed(2)}
            </p>
          </div>
        </div>
      </div>

      {/* Quick Links */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Link
          to="/fleet/partners"
          className="flex items-center gap-4 rounded-xl border border-border bg-card p-5 transition-colors hover:bg-secondary"
        >
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <Users className="h-6 w-6 text-primary" />
          </div>
          <div>
            <p className="font-semibold text-foreground">Manage Partners</p>
            <p className="text-sm text-muted-foreground">
              View, verify, and manage delivery partners
            </p>
          </div>
        </Link>
        {(overview?.pendingVerification ?? 0) > 0 && (
          <Link
            to="/fleet/partners"
            className="flex items-center gap-4 rounded-xl border-2 border-warning bg-warning/5 p-5 transition-colors hover:bg-warning/10"
          >
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-warning/10">
              <Clock className="h-6 w-6 text-warning" />
            </div>
            <div>
              <p className="font-semibold text-foreground">
                Pending Verifications
              </p>
              <p className="text-sm text-muted-foreground">
                {overview?.pendingVerification} partners awaiting review
              </p>
            </div>
          </Link>
        )}
      </div>
    </div>
  );
}
