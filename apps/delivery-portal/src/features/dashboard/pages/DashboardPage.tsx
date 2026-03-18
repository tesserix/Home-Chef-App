import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { Link, useNavigate } from 'react-router-dom';
import {
  Navigation,
  Package,
  DollarSign,
  Star,
  TrendingUp,
  Truck,
  Power,
  Wifi,
  AlertCircle,
  CheckCircle2,
  Users,
  ShieldCheck,
  Clock,
  ArrowRight,
} from 'lucide-react';
import type { DashboardStats } from '@/shared/types';
import { toast } from 'sonner';
import { PageLoader } from '@/shared/components/LoadingScreen';
import { useEffect } from 'react';

interface StaffProfile {
  id: string;
  staffRole: string;
  permissions: string[];
}

interface FleetOverview {
  onlinePartners: number;
  offlinePartners: number;
  activeDeliveries: number;
  unassignedOrders: number;
  todayCompleted: number;
  totalPartners: number;
  verifiedPartners: number;
  pendingVerification: number;
  todayEarnings: number;
}

interface OnboardingStatusResponse {
  step: number;
  status: string;
}

// Fleet manager / staff dashboard
function StaffDashboard({ staffProfile }: { staffProfile: StaffProfile }) {
  const { data: overview, isLoading } = useQuery({
    queryKey: ['fleet-overview'],
    queryFn: () => apiClient.get<FleetOverview>('/delivery/staff/fleet/overview'),
    refetchInterval: 30000,
  });

  if (isLoading) return <PageLoader />;

  const roleLabel = staffProfile.staffRole
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');

  const liveStats = [
    { label: 'Online Partners', value: overview?.onlinePartners ?? 0, icon: Wifi, color: 'text-success', bg: 'bg-success/10' },
    { label: 'Active Deliveries', value: overview?.activeDeliveries ?? 0, icon: Truck, color: 'text-primary', bg: 'bg-primary/10' },
    { label: 'Unassigned Orders', value: overview?.unassignedOrders ?? 0, icon: AlertCircle, color: 'text-warning', bg: 'bg-warning/10' },
    { label: "Today's Completed", value: overview?.todayCompleted ?? 0, icon: CheckCircle2, color: 'text-success', bg: 'bg-success/10' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="page-title">Dashboard</h1>
          <p className="page-description">Welcome back — {roleLabel}</p>
        </div>
        <Link
          to="/fleet"
          className="flex items-center gap-2 rounded-xl border border-border px-4 py-2.5 text-sm font-medium text-foreground hover:bg-secondary"
        >
          Fleet Overview
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>

      {/* Live Stats */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {liveStats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.label} className="rounded-xl border border-border bg-card p-5">
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

      {/* Partner Summary */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">Partner Summary</h2>
        <div className="grid grid-cols-3 gap-4 text-center">
          {[
            { label: 'Total Partners', value: overview?.totalPartners ?? 0, icon: Users },
            { label: 'Verified', value: overview?.verifiedPartners ?? 0, icon: ShieldCheck },
            { label: 'Pending Verification', value: overview?.pendingVerification ?? 0, icon: Clock },
          ].map((stat) => {
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

      {/* Today's Earnings + Quick Links */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
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
      </div>

      {(overview?.pendingVerification ?? 0) > 0 && (
        <Link
          to="/fleet/partners"
          className="block rounded-xl border-2 border-warning bg-warning/5 p-5 transition-colors hover:bg-warning/10"
        >
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-warning/10">
              <Clock className="h-6 w-6 text-warning" />
            </div>
            <div>
              <p className="font-semibold text-foreground">Pending Verifications</p>
              <p className="text-sm text-muted-foreground">
                {overview?.pendingVerification} partners awaiting review
              </p>
            </div>
          </div>
        </Link>
      )}
    </div>
  );
}

// Regular delivery partner dashboard
function PartnerDashboard() {
  const queryClient = useQueryClient();

  const { data: stats, isLoading } = useQuery({
    queryKey: ['delivery-stats'],
    queryFn: () => apiClient.get<DashboardStats>('/delivery/stats'),
    refetchInterval: 15000,
  });

  const toggleOnline = useMutation({
    mutationFn: (isOnline: boolean) => apiClient.put('/delivery/online', { isOnline }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['delivery-stats'] });
      toast.success(stats?.partner?.isOnline ? 'You are now offline' : 'You are now online');
    },
    onError: () => toast.error('Failed to update status'),
  });

  if (isLoading) return <PageLoader />;

  const isOnline = stats?.partner?.isOnline ?? false;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="page-title">Dashboard</h1>
          <p className="page-description">Welcome back, partner</p>
        </div>
        <button
          onClick={() => toggleOnline.mutate(!isOnline)}
          disabled={toggleOnline.isPending}
          className={`flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-medium transition-all ${
            isOnline
              ? 'bg-success text-success-foreground shadow-md'
              : 'bg-muted text-muted-foreground'
          }`}
        >
          <Power className="h-4 w-4" />
          {isOnline ? 'Online' : 'Offline'}
        </button>
      </div>

      {!isOnline && (
        <div className="rounded-xl border border-warning/30 bg-warning/10 p-4 text-center">
          <p className="text-sm font-medium text-warning">
            You are currently offline. Go online to receive delivery requests.
          </p>
        </div>
      )}

      {/* Quick Stats */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Truck className="h-4 w-4" />
            <span className="text-xs font-medium">Today</span>
          </div>
          <p className="mt-2 text-2xl font-bold text-foreground">{stats?.today?.deliveries ?? 0}</p>
          <p className="text-xs text-muted-foreground">deliveries</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <DollarSign className="h-4 w-4" />
            <span className="text-xs font-medium">Today's Earnings</span>
          </div>
          <p className="mt-2 text-2xl font-bold text-foreground">
            ${(stats?.today?.earnings ?? 0).toFixed(2)}
          </p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <TrendingUp className="h-4 w-4" />
            <span className="text-xs font-medium">This Week</span>
          </div>
          <p className="mt-2 text-2xl font-bold text-foreground">
            ${(stats?.week?.earnings ?? 0).toFixed(2)}
          </p>
          <p className="text-xs text-muted-foreground">{stats?.week?.deliveries ?? 0} deliveries</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Star className="h-4 w-4" />
            <span className="text-xs font-medium">Rating</span>
          </div>
          <p className="mt-2 text-2xl font-bold text-foreground">
            {(stats?.partner?.rating ?? 0).toFixed(1)}
          </p>
          <p className="text-xs text-muted-foreground">{stats?.totalReviews ?? 0} reviews</p>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {(stats?.active ?? 0) > 0 && (
          <Link
            to="/active"
            className="flex items-center gap-4 rounded-xl border-2 border-primary bg-primary/5 p-5 transition-colors hover:bg-primary/10"
          >
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
              <Navigation className="h-6 w-6 text-primary" />
            </div>
            <div>
              <p className="font-semibold text-foreground">Active Delivery</p>
              <p className="text-sm text-muted-foreground">You have an ongoing delivery</p>
            </div>
          </Link>
        )}

        <Link
          to="/available"
          className="flex items-center gap-4 rounded-xl border border-border bg-card p-5 transition-colors hover:bg-secondary"
        >
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-accent">
            <Package className="h-6 w-6 text-accent-foreground" />
          </div>
          <div>
            <p className="font-semibold text-foreground">Available Orders</p>
            <p className="text-sm text-muted-foreground">
              {stats?.availableOrders ?? 0} orders waiting
            </p>
          </div>
        </Link>
      </div>

      {/* Monthly Summary */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">This Month</h2>
        <div className="grid grid-cols-3 gap-4 text-center">
          <div>
            <p className="text-2xl font-bold text-foreground">{stats?.month?.deliveries ?? 0}</p>
            <p className="text-xs text-muted-foreground">Deliveries</p>
          </div>
          <div>
            <p className="text-2xl font-bold text-foreground">
              ${(stats?.month?.earnings ?? 0).toFixed(2)}
            </p>
            <p className="text-xs text-muted-foreground">Earnings</p>
          </div>
          <div>
            <p className="text-2xl font-bold text-foreground">{stats?.totalDeliveries ?? 0}</p>
            <p className="text-xs text-muted-foreground">Total All Time</p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function DashboardPage() {
  const navigate = useNavigate();

  // Check if user is a staff member (fleet_manager, delivery_ops, super_admin)
  const { data: staffProfile, isLoading: staffLoading } = useQuery({
    queryKey: ['delivery-staff-me'],
    queryFn: () => apiClient.get<StaffProfile>('/delivery/staff/me'),
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  const isStaff = !!staffProfile?.id;

  // Check onboarding status for non-staff users
  const { data: onboardingStatus, isLoading: onboardingLoading } = useQuery({
    queryKey: ['onboarding-status'],
    queryFn: () => apiClient.get<OnboardingStatusResponse>('/driver/onboarding/status'),
    enabled: !staffLoading && !isStaff,
    retry: false,
    staleTime: 60 * 1000,
  });

  // Redirect non-staff users based on onboarding status
  useEffect(() => {
    if (staffLoading || isStaff) return;
    if (onboardingLoading || !onboardingStatus) return;

    const status = onboardingStatus.status;

    if (status === 'not_started' || status === 'in_progress') {
      navigate('/onboarding', { replace: true });
    } else if (status === 'submitted' || status === 'in_review' || status === 'rejected') {
      navigate('/onboarding/status', { replace: true });
    }
    // If approved, stay on dashboard
  }, [staffLoading, isStaff, onboardingLoading, onboardingStatus, navigate]);

  // While checking staff status, show loader briefly
  if (staffLoading) return <PageLoader />;

  if (isStaff) {
    return <StaffDashboard staffProfile={staffProfile} />;
  }

  // While checking onboarding status for non-staff, show loader
  if (onboardingLoading) return <PageLoader />;

  // If status requires redirect, show loader while navigating
  if (onboardingStatus) {
    const status = onboardingStatus.status;
    if (status !== 'approved') {
      return <PageLoader />;
    }
  }

  return <PartnerDashboard />;
}
