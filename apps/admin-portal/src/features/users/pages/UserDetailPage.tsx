import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Mail,
  Phone,
  Shield,
  Calendar,
  ShoppingBag,
  IndianRupee,
  UserX,
  UserCheck,
  CheckCircle,
  XCircle,
  Loader2,
  Globe,
  Clock,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';

interface UserDetail {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  phone?: string;
  avatar?: string;
  role: string;
  authProvider?: string;
  isActive: boolean;
  emailVerified: boolean;
  phoneVerified: boolean;
  lastLoginAt?: string;
  totalOrders: number;
  totalSpent: number;
  lastOrderAt?: string;
  createdAt: string;
  updatedAt: string;
}

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-user', id],
    queryFn: () => apiClient.get<UserDetail>(`/admin/users/${id}`),
    enabled: !!id,
  });

  const user = data as unknown as UserDetail | undefined;

  const suspendMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/users/${id}/suspend`),
    onSuccess: () => {
      toast.success('User suspended');
      queryClient.invalidateQueries({ queryKey: ['admin-user', id] });
    },
    onError: () => toast.error('Failed to suspend user'),
  });

  const activateMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/users/${id}/activate`),
    onSuccess: () => {
      toast.success('User activated');
      queryClient.invalidateQueries({ queryKey: ['admin-user', id] });
    },
    onError: () => toast.error('Failed to activate user'),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="py-20 text-center">
        <p className="text-muted-foreground">User not found</p>
        <button onClick={() => navigate('/users')} className="mt-4 text-sm text-primary hover:underline">
          Back to Users
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button onClick={() => navigate('/users')}
          className="rounded-lg p-2 hover:bg-secondary transition-colors">
          <ArrowLeft className="h-5 w-5 text-muted-foreground" />
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-foreground">{user.firstName} {user.lastName}</h1>
          <p className="text-sm text-muted-foreground">{user.email}</p>
        </div>
        <div className="flex items-center gap-2">
          {user.isActive ? (
            <button onClick={() => suspendMutation.mutate()}
              className="inline-flex items-center gap-2 rounded-lg border border-destructive/30 px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors">
              <UserX className="h-4 w-4" />Suspend
            </button>
          ) : (
            <button onClick={() => activateMutation.mutate()}
              className="inline-flex items-center gap-2 rounded-lg border border-success/30 px-4 py-2 text-sm font-medium text-success hover:bg-success/10 transition-colors">
              <UserCheck className="h-4 w-4" />Activate
            </button>
          )}
        </div>
      </div>

      {/* Profile Card */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main Info */}
        <div className="lg:col-span-2 space-y-6">
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Profile Information</h2>
            <div className="flex items-center gap-5 mb-6">
              <div className="flex h-20 w-20 items-center justify-center rounded-full bg-primary/10">
                {user.avatar ? (
                  <img src={user.avatar} alt={user.firstName} className="h-20 w-20 rounded-full object-cover" />
                ) : (
                  <span className="text-3xl font-bold text-primary">
                    {(user.firstName?.[0] || '?').toUpperCase()}
                  </span>
                )}
              </div>
              <div>
                <h3 className="text-xl font-semibold text-foreground">{user.firstName} {user.lastName}</h3>
                <div className="flex items-center gap-3 mt-1">
                  <RoleBadge role={user.role} />
                  <StatusBadge active={user.isActive} />
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <InfoRow icon={Mail} label="Email" value={user.email} verified={user.emailVerified} />
              <InfoRow icon={Phone} label="Phone" value={user.phone || 'Not provided'} verified={user.phoneVerified} />
              <InfoRow icon={Globe} label="Auth Provider" value={user.authProvider || 'email'} />
              <InfoRow icon={Shield} label="Role" value={user.role} />
              <InfoRow icon={Calendar} label="Joined" value={new Date(user.createdAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'long', year: 'numeric' })} />
              <InfoRow icon={Clock} label="Last Login" value={user.lastLoginAt ? new Date(user.lastLoginAt).toLocaleString('en-IN') : 'Never'} />
            </div>
          </div>

          {/* User ID */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-3">System Details</h2>
            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">User ID</span>
                <code className="text-sm font-mono text-foreground">{user.id}</code>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">Created At</span>
                <span className="text-sm text-foreground">{new Date(user.createdAt).toISOString()}</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">Updated At</span>
                <span className="text-sm text-foreground">{user.updatedAt ? new Date(user.updatedAt).toISOString() : '--'}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Sidebar Stats */}
        <div className="space-y-6">
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Order Activity</h2>
            <div className="space-y-4">
              <StatCard icon={ShoppingBag} label="Total Orders" value={String(user.totalOrders || 0)} color="info" />
              <StatCard icon={IndianRupee} label="Total Spent" value={`₹${(user.totalSpent || 0).toLocaleString('en-IN')}`} color="success" />
              <StatCard icon={Clock} label="Last Order" value={user.lastOrderAt ? formatRelative(user.lastOrderAt) : 'No orders yet'} color="primary" />
            </div>
          </div>

          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Verification</h2>
            <div className="space-y-3">
              <VerifyRow label="Email" verified={user.emailVerified} />
              <VerifyRow label="Phone" verified={!!user.phoneVerified} />
              <VerifyRow label="Account Active" verified={user.isActive} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function InfoRow({ icon: Icon, label, value, verified }: { icon: typeof Mail; label: string; value: string; verified?: boolean }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/20 p-3">
      <Icon className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm font-medium text-foreground capitalize truncate">{value}</p>
      </div>
      {verified !== undefined && (
        verified ? <CheckCircle className="h-4 w-4 text-success shrink-0" /> : <XCircle className="h-4 w-4 text-muted-foreground/40 shrink-0" />
      )}
    </div>
  );
}

function StatCard({ icon: Icon, label, value, color }: { icon: typeof ShoppingBag; label: string; value: string; color: string }) {
  const colors: Record<string, string> = { info: 'bg-info/10 text-info', success: 'bg-success/10 text-success', primary: 'bg-primary/10 text-primary' };
  return (
    <div className="flex items-center gap-3">
      <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${colors[color] || colors.primary}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm font-semibold text-foreground">{value}</p>
      </div>
    </div>
  );
}

function VerifyRow({ label, verified }: { label: string; verified: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-foreground">{label}</span>
      {verified ? (
        <span className="flex items-center gap-1 text-xs text-success"><CheckCircle className="h-3.5 w-3.5" />Verified</span>
      ) : (
        <span className="flex items-center gap-1 text-xs text-muted-foreground"><XCircle className="h-3.5 w-3.5" />Not verified</span>
      )}
    </div>
  );
}

function RoleBadge({ role }: { role: string }) {
  const styles: Record<string, string> = { customer: 'bg-info/10 text-info', chef: 'bg-primary/10 text-primary', delivery: 'bg-success/10 text-success', admin: 'bg-warning/10 text-warning' };
  return <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${styles[role] || 'bg-muted text-muted-foreground'}`}>{role}</span>;
}

function StatusBadge({ active }: { active: boolean }) {
  return active
    ? <span className="inline-flex items-center rounded-full bg-success/10 px-2.5 py-0.5 text-xs font-medium text-success">Active</span>
    : <span className="inline-flex items-center rounded-full bg-destructive/10 px-2.5 py-0.5 text-xs font-medium text-destructive">Suspended</span>;
}

function formatRelative(dateStr: string): string {
  const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
  return new Date(dateStr).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' });
}
