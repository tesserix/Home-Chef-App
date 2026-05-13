import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { Link, useNavigate } from 'react-router-dom';
import {
  Navigation,
  Package,
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

const formatCurrency = (amount: number) =>
  new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
    .format(amount);

// ---------- Fleet staff dashboard ----------

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

  const pending = overview?.pendingVerification ?? 0;
  const unassigned = overview?.unassignedOrders ?? 0;

  // Pick the most urgent right-side CTA: unassigned orders > pending verifications
  const urgent = unassigned > 0
    ? { count: unassigned, label: `Unassigned ${unassigned === 1 ? 'order' : 'orders'}`, subtitle: 'Needs dispatcher attention', to: '/fleet' }
    : pending > 0
      ? { count: pending, label: `Pending ${pending === 1 ? 'verification' : 'verifications'}`, subtitle: 'Partners awaiting review', to: '/fleet/partners' }
      : null;

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
            Dashboard
          </h1>
          <p className="mt-1 text-sm text-ink-soft">{roleLabel} · Welcome back.</p>
        </div>
        <Link
          to="/fleet"
          className="inline-flex items-center gap-2 rounded-md border border-mist bg-bone px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
        >
          Fleet overview
          <ArrowRight className="h-4 w-4" />
        </Link>
      </header>

      {/* Lead block — Today's fleet earnings + urgent CTA */}
      <section className="grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:items-end">
        <div>
          <p className="text-sm text-ink-soft">Today's fleet earnings</p>
          <p className="mt-1 text-4xl font-semibold tabular-nums tracking-tight text-foreground sm:text-5xl">
            {formatCurrency(overview?.todayEarnings ?? 0)}
          </p>
          <p className="mt-2 text-sm text-ink-soft tabular-nums">
            {overview?.todayCompleted ?? 0}{' '}
            {overview?.todayCompleted === 1 ? 'delivery' : 'deliveries'} completed today
          </p>
        </div>

        {urgent ? (
          <Link
            to={urgent.to}
            aria-label={urgent.label}
            className="group flex items-center justify-between gap-4 rounded-lg bg-amber px-5 py-4 text-foreground transition-colors hover:bg-amber/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber focus-visible:ring-offset-2"
          >
            <div>
              <p className="text-3xl font-semibold tabular-nums">{urgent.count}</p>
              <p className="mt-0.5 text-sm">{urgent.label.charAt(0).toUpperCase() + urgent.label.slice(1).toLowerCase()}</p>
              <p className="mt-0.5 text-xs text-ink-soft">{urgent.subtitle}</p>
            </div>
            <ArrowRight className="h-5 w-5 transition-transform group-hover:translate-x-0.5" aria-hidden />
          </Link>
        ) : (
          <div className="rounded-lg border border-mist bg-bone px-5 py-4">
            <p className="text-3xl font-semibold tabular-nums text-foreground">All clear</p>
            <p className="mt-0.5 text-sm text-ink-soft">No unassigned orders or pending partners.</p>
          </div>
        )}
      </section>

      {/* Stats strip */}
      <section
        aria-label="Fleet at a glance"
        className="grid grid-cols-2 divide-y divide-mist border-y border-mist sm:grid-cols-4 sm:divide-x sm:divide-y-0"
      >
        <StatRow
          label="Online partners"
          value={overview?.onlinePartners ?? 0}
          subtitle={`${overview?.offlinePartners ?? 0} offline`}
        />
        <StatRow label="Active deliveries" value={overview?.activeDeliveries ?? 0} />
        <StatRow
          label="Verified partners"
          value={overview?.verifiedPartners ?? 0}
          subtitle={`of ${overview?.totalPartners ?? 0} total`}
        />
        <StatRow label="Completed today" value={overview?.todayCompleted ?? 0} />
      </section>

      {/* Shortcuts */}
      <section aria-label="Shortcuts" className="space-y-3">
        <h2 className="text-lg font-semibold text-foreground">Shortcuts</h2>
        <nav className="divide-y divide-mist rounded-lg border border-mist bg-bone">
          <QuickAction
            to="/fleet/partners"
            title="Manage partners"
            subtitle="View, verify, and manage delivery partners"
          />
          <QuickAction to="/fleet" title="Fleet overview" subtitle="Real-time fleet state" />
        </nav>
      </section>
    </div>
  );
}

// ---------- Delivery partner dashboard ----------

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
  const activeCount = stats?.active ?? 0;
  const availableCount = stats?.availableOrders ?? 0;

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
            Dashboard
          </h1>
          <p className="mt-1 flex items-center gap-2 text-sm text-ink-soft">
            <span
              className={`inline-block h-2 w-2 rounded-full ${isOnline ? 'bg-herb' : 'bg-ink-muted'}`}
              aria-hidden
            />
            {isOnline ? 'Ready for deliveries' : 'Currently offline'}
          </p>
        </div>
        <button
          type="button"
          onClick={() => toggleOnline.mutate(!isOnline)}
          disabled={toggleOnline.isPending}
          aria-pressed={isOnline}
          aria-label={isOnline ? 'Go offline' : 'Go online'}
          className={`inline-flex h-11 items-center justify-center rounded-full px-6 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2 disabled:opacity-50 ${
            isOnline
              ? 'bg-foreground text-background hover:bg-ink-soft'
              : 'border border-mist bg-bone text-foreground hover:bg-mist'
          }`}
        >
          {isOnline ? 'Go offline' : 'Go online'}
        </button>
      </header>

      {/* Lead block — Today's earnings + active delivery CTA if any */}
      <section className="grid gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:items-end">
        <div>
          <p className="text-sm text-ink-soft">Today's earnings</p>
          <p className="mt-1 text-4xl font-semibold tabular-nums tracking-tight text-foreground sm:text-5xl">
            {formatCurrency(stats?.today?.earnings ?? 0)}
          </p>
          <p className="mt-2 text-sm text-ink-soft tabular-nums">
            From {stats?.today?.deliveries ?? 0}{' '}
            {stats?.today?.deliveries === 1 ? 'delivery' : 'deliveries'} today
          </p>
        </div>

        {activeCount > 0 ? (
          <Link
            to="/active"
            aria-label="Continue active delivery"
            className="group flex items-center justify-between gap-4 rounded-lg bg-herb px-5 py-4 text-paper transition-colors hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
          >
            <div>
              <p className="text-3xl font-semibold tabular-nums">In progress</p>
              <p className="mt-0.5 text-sm text-paper/80">Continue active delivery</p>
            </div>
            <Navigation className="h-5 w-5 transition-transform group-hover:translate-x-0.5" aria-hidden />
          </Link>
        ) : availableCount > 0 ? (
          <Link
            to="/available"
            aria-label={`${availableCount} available orders`}
            className="group flex items-center justify-between gap-4 rounded-lg border border-mist bg-bone px-5 py-4 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
          >
            <div>
              <p className="text-3xl font-semibold tabular-nums text-foreground">
                {availableCount} {availableCount === 1 ? 'order' : 'orders'}
              </p>
              <p className="mt-0.5 text-sm text-ink-soft">Waiting nearby</p>
            </div>
            <Package className="h-5 w-5 text-herb transition-transform group-hover:translate-x-0.5" aria-hidden />
          </Link>
        ) : (
          <div className="rounded-lg border border-mist bg-bone px-5 py-4">
            <p className="text-3xl font-semibold tabular-nums text-foreground">
              {isOnline ? 'Standing by' : 'You\'re offline'}
            </p>
            <p className="mt-0.5 text-sm text-ink-soft">
              {isOnline ? 'New deliveries will appear here.' : 'Go online to receive requests.'}
            </p>
          </div>
        )}
      </section>

      {/* Stats strip */}
      <section
        aria-label="Performance at a glance"
        className="grid grid-cols-2 divide-y divide-mist border-y border-mist sm:grid-cols-4 sm:divide-x sm:divide-y-0"
      >
        <StatRow
          label="This week"
          value={formatCurrency(stats?.week?.earnings ?? 0)}
          subtitle={`${stats?.week?.deliveries ?? 0} deliveries`}
        />
        <StatRow
          label="This month"
          value={formatCurrency(stats?.month?.earnings ?? 0)}
          subtitle={`${stats?.month?.deliveries ?? 0} deliveries`}
        />
        <StatRow
          label="Rating"
          value={(stats?.partner?.rating ?? 0).toFixed(1)}
          subtitle={stats?.totalReviews ? `${stats.totalReviews} reviews` : 'No reviews yet'}
        />
        <StatRow
          label="All time"
          value={stats?.totalDeliveries ?? 0}
          subtitle="deliveries"
        />
      </section>
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

// ---------- Router shell ----------

export default function DashboardPage() {
  const navigate = useNavigate();

  const { data: staffProfile, isLoading: staffLoading } = useQuery({
    queryKey: ['delivery-staff-me'],
    queryFn: () => apiClient.get<StaffProfile>('/delivery/staff/me'),
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  const isStaff = !!staffProfile?.id;

  const { data: onboardingStatus, isLoading: onboardingLoading } = useQuery({
    queryKey: ['onboarding-status'],
    queryFn: () => apiClient.get<OnboardingStatusResponse>('/driver/onboarding/status'),
    enabled: !staffLoading && !isStaff,
    retry: false,
    staleTime: 60 * 1000,
  });

  useEffect(() => {
    if (staffLoading || isStaff) return;
    if (onboardingLoading || !onboardingStatus) return;

    const status = onboardingStatus.status;

    if (status === 'not_started' || status === 'in_progress') {
      navigate('/onboarding', { replace: true });
    } else if (status === 'submitted' || status === 'in_review' || status === 'rejected') {
      navigate('/onboarding/status', { replace: true });
    }
  }, [staffLoading, isStaff, onboardingLoading, onboardingStatus, navigate]);

  if (staffLoading) return <PageLoader />;

  if (isStaff) {
    return <StaffDashboard staffProfile={staffProfile} />;
  }

  if (onboardingLoading) return <PageLoader />;

  if (onboardingStatus) {
    const status = onboardingStatus.status;
    if (status !== 'approved') {
      return <PageLoader />;
    }
  }

  return <PartnerDashboard />;
}
