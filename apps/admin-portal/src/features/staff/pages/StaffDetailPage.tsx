import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Mail,
  Phone,
  Shield,
  Calendar,
  Clock,
  Loader2,
  UserX,
  UserCheck,
  ChevronDown,
  Building2,
  Briefcase,
  Key,
  UserPlus,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';

// ---------- Types ----------

interface StaffDetail {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  phone?: string;
  avatar?: string;
  role: string;
  department?: string;
  title?: string;
  isActive: boolean;
  invitedBy?: string;
  invitedByName?: string;
  permissions?: string[];
  createdAt: string;
  updatedAt?: string;
  lastLoginAt?: string;
}

interface RoleDefinition {
  name: string;
  description: string;
  permissions: string[];
}

interface RolesResponse {
  data: RoleDefinition[];
}

// ---------- Constants ----------

const STAFF_ROLES: Record<string, { label: string; color: string; description: string }> = {
  super_admin: {
    label: 'Super Admin',
    color: 'bg-destructive/10 text-destructive',
    description: 'Full access to everything',
  },
  admin: {
    label: 'Admin',
    color: 'bg-warning/10 text-warning',
    description: 'Admin portal management',
  },
  fleet_manager: {
    label: 'Fleet Manager',
    color: 'bg-info/10 text-info',
    description: 'Delivery fleet management',
  },
  delivery_ops: {
    label: 'Delivery Ops',
    color: 'bg-primary/10 text-primary',
    description: 'Delivery operations',
  },
  support: {
    label: 'Support',
    color: 'bg-success/10 text-success',
    description: 'Read-only support access',
  },
};

// ---------- Main Component ----------

export default function StaffDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showRoleDropdown, setShowRoleDropdown] = useState(false);
  const [confirmAction, setConfirmAction] = useState<'deactivate' | 'reactivate' | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-staff-member', id],
    queryFn: () => apiClient.get<StaffDetail>(`/admin/staff/${id}`),
    enabled: !!id,
  });

  const { data: rolesData } = useQuery({
    queryKey: ['admin-staff-roles'],
    queryFn: () => apiClient.get<RolesResponse>('/admin/staff/roles'),
  });

  const staff = data as unknown as StaffDetail | undefined;
  const roles = (rolesData as unknown as RolesResponse)?.data ?? [];

  const updateRoleMutation = useMutation({
    mutationFn: (newRole: string) => apiClient.put(`/admin/staff/${id}/role`, { role: newRole }),
    onSuccess: () => {
      toast.success('Role updated successfully');
      queryClient.invalidateQueries({ queryKey: ['admin-staff-member', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-staff'] });
      setShowRoleDropdown(false);
    },
    onError: () => toast.error('Failed to update role'),
  });

  const deactivateMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/staff/${id}/deactivate`),
    onSuccess: () => {
      toast.success('Staff member deactivated');
      queryClient.invalidateQueries({ queryKey: ['admin-staff-member', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-staff'] });
      setConfirmAction(null);
    },
    onError: () => toast.error('Failed to deactivate staff member'),
  });

  const reactivateMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/staff/${id}/reactivate`),
    onSuccess: () => {
      toast.success('Staff member reactivated');
      queryClient.invalidateQueries({ queryKey: ['admin-staff-member', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-staff'] });
      setConfirmAction(null);
    },
    onError: () => toast.error('Failed to reactivate staff member'),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!staff) {
    return (
      <div className="py-20 text-center">
        <p className="text-muted-foreground">Staff member not found</p>
        <button
          onClick={() => navigate('/staff')}
          className="mt-4 text-sm text-primary hover:underline"
        >
          Back to Staff
        </button>
      </div>
    );
  }

  const currentRole = STAFF_ROLES[staff.role] || {
    label: staff.role,
    color: 'bg-muted text-muted-foreground',
    description: '',
  };

  // Get permissions for the current role from the API roles data, or from the staff object
  const currentRoleDef = roles.find((r) => r.name === staff.role);
  const permissions = staff.permissions || currentRoleDef?.permissions || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/staff')}
          className="rounded-lg p-2 hover:bg-secondary transition-colors"
        >
          <ArrowLeft className="h-5 w-5 text-muted-foreground" />
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-foreground">
            {staff.firstName} {staff.lastName}
          </h1>
          <p className="text-sm text-muted-foreground">{staff.email}</p>
        </div>
        <div className="flex items-center gap-2">
          {staff.isActive ? (
            <button
              onClick={() => setConfirmAction('deactivate')}
              className="inline-flex items-center gap-2 rounded-lg border border-destructive/30 px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
            >
              <UserX className="h-4 w-4" />
              Deactivate
            </button>
          ) : (
            <button
              onClick={() => setConfirmAction('reactivate')}
              className="inline-flex items-center gap-2 rounded-lg border border-success/30 px-4 py-2 text-sm font-medium text-success hover:bg-success/10 transition-colors"
            >
              <UserCheck className="h-4 w-4" />
              Reactivate
            </button>
          )}
        </div>
      </div>

      {/* Confirmation Dialog */}
      {confirmAction && (
        <div className="rounded-xl border border-warning/30 bg-warning/5 p-4">
          <p className="text-sm font-medium text-foreground">
            {confirmAction === 'deactivate'
              ? 'Are you sure you want to deactivate this staff member? They will lose access to all portals.'
              : 'Are you sure you want to reactivate this staff member? They will regain access based on their role.'}
          </p>
          <div className="flex items-center gap-3 mt-3">
            <button
              onClick={() =>
                confirmAction === 'deactivate'
                  ? deactivateMutation.mutate()
                  : reactivateMutation.mutate()
              }
              disabled={deactivateMutation.isPending || reactivateMutation.isPending}
              className={`inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-primary-foreground transition-colors disabled:opacity-50 ${
                confirmAction === 'deactivate' ? 'bg-destructive hover:bg-destructive/90' : 'bg-success hover:bg-success/90'
              }`}
            >
              {(deactivateMutation.isPending || reactivateMutation.isPending) && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Confirm {confirmAction === 'deactivate' ? 'Deactivation' : 'Reactivation'}
            </button>
            <button
              onClick={() => setConfirmAction(null)}
              className="rounded-lg border border-border px-4 py-2 text-sm font-medium hover:bg-secondary transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Main Content */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Left Column */}
        <div className="lg:col-span-2 space-y-6">
          {/* Profile Card */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Profile Information</h2>
            <div className="flex items-center gap-5 mb-6">
              <div className="flex h-20 w-20 items-center justify-center rounded-full bg-primary/10">
                {staff.avatar ? (
                  <img
                    src={staff.avatar}
                    alt={staff.firstName}
                    className="h-20 w-20 rounded-full object-cover"
                  />
                ) : (
                  <span className="text-3xl font-bold text-primary">
                    {(staff.firstName?.[0] || '?').toUpperCase()}
                  </span>
                )}
              </div>
              <div>
                <h3 className="text-xl font-semibold text-foreground">
                  {staff.firstName} {staff.lastName}
                </h3>
                <div className="flex items-center gap-3 mt-1">
                  <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${currentRole.color}`}>
                    {currentRole.label}
                  </span>
                  {staff.isActive ? (
                    <span className="inline-flex items-center rounded-full bg-success/10 px-2.5 py-0.5 text-xs font-medium text-success">
                      Active
                    </span>
                  ) : (
                    <span className="inline-flex items-center rounded-full bg-destructive/10 px-2.5 py-0.5 text-xs font-medium text-destructive">
                      Inactive
                    </span>
                  )}
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <InfoRow icon={Mail} label="Email" value={staff.email} />
              <InfoRow icon={Phone} label="Phone" value={staff.phone || 'Not provided'} />
              <InfoRow icon={Building2} label="Department" value={staff.department || 'Not assigned'} />
              <InfoRow icon={Briefcase} label="Title" value={staff.title || 'Not assigned'} />
              <InfoRow
                icon={Calendar}
                label="Joined"
                value={new Date(staff.createdAt).toLocaleDateString('en-IN', {
                  day: 'numeric',
                  month: 'long',
                  year: 'numeric',
                })}
              />
              <InfoRow
                icon={Clock}
                label="Last Login"
                value={
                  staff.lastLoginAt
                    ? new Date(staff.lastLoginAt).toLocaleString('en-IN')
                    : 'Never'
                }
              />
            </div>
          </div>

          {/* Role & Permissions */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-foreground">Role & Permissions</h2>
              <div className="relative">
                <button
                  onClick={() => setShowRoleDropdown(!showRoleDropdown)}
                  className="inline-flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm font-medium hover:bg-secondary transition-colors"
                >
                  Change Role
                  <ChevronDown className="h-4 w-4" />
                </button>
                {showRoleDropdown && (
                  <div className="absolute right-0 top-full mt-1 z-10 w-64 rounded-lg border border-border bg-card shadow-lg">
                    {Object.entries(STAFF_ROLES).map(([key, val]) => (
                      <button
                        key={key}
                        onClick={() => updateRoleMutation.mutate(key)}
                        disabled={key === staff.role || updateRoleMutation.isPending}
                        className={`flex w-full items-start gap-3 px-4 py-3 text-left hover:bg-secondary/50 transition-colors first:rounded-t-lg last:rounded-b-lg ${
                          key === staff.role ? 'bg-primary/5 cursor-default' : ''
                        }`}
                      >
                        <div className="flex-1">
                          <p className="text-sm font-medium text-foreground">
                            {val.label}
                            {key === staff.role && (
                              <span className="ml-2 text-xs text-muted-foreground">(current)</span>
                            )}
                          </p>
                          <p className="text-xs text-muted-foreground">{val.description}</p>
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <div className="mb-4">
              <div className="flex items-center gap-3 rounded-lg bg-muted/50 px-4 py-3">
                <Shield className="h-5 w-5 text-primary" />
                <div>
                  <p className="text-sm font-medium text-foreground">Current Role</p>
                  <p className="text-xs text-muted-foreground">{currentRole.label} - {currentRole.description}</p>
                </div>
              </div>
            </div>

            {permissions.length > 0 && (
              <div>
                <p className="text-sm font-medium text-foreground mb-3">Permissions</p>
                <div className="flex flex-wrap gap-2">
                  {permissions.map((perm) => (
                    <span
                      key={perm}
                      className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-secondary/30 px-3 py-1.5 text-xs font-medium text-foreground"
                    >
                      <Key className="h-3 w-3 text-muted-foreground" />
                      {perm.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {permissions.length === 0 && (
              <p className="text-sm text-muted-foreground">
                No specific permissions data available for this role.
              </p>
            )}
          </div>

          {/* System Details */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-3">System Details</h2>
            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">Staff ID</span>
                <code className="text-sm font-mono text-foreground">{staff.id}</code>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">Created At</span>
                <span className="text-sm text-foreground">{new Date(staff.createdAt).toISOString()}</span>
              </div>
              {staff.updatedAt && (
                <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                  <span className="text-sm text-muted-foreground">Updated At</span>
                  <span className="text-sm text-foreground">{new Date(staff.updatedAt).toISOString()}</span>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right Column */}
        <div className="space-y-6">
          {/* Status Card */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Account Status</h2>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground">Status</span>
                {staff.isActive ? (
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-success/10 px-2.5 py-0.5 text-xs font-medium text-success">
                    <span className="h-1.5 w-1.5 rounded-full bg-success" />
                    Active
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-destructive/10 px-2.5 py-0.5 text-xs font-medium text-destructive">
                    <span className="h-1.5 w-1.5 rounded-full bg-destructive" />
                    Inactive
                  </span>
                )}
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground">Role</span>
                <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${currentRole.color}`}>
                  {currentRole.label}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground">Last Login</span>
                <span className="text-sm text-muted-foreground">
                  {staff.lastLoginAt ? formatRelative(staff.lastLoginAt) : 'Never'}
                </span>
              </div>
            </div>
          </div>

          {/* Invited By Card */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Invitation Info</h2>
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info/10">
                  <UserPlus className="h-5 w-5 text-info" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Invited By</p>
                  <p className="text-sm font-medium text-foreground">
                    {staff.invitedByName || staff.invitedBy || 'System'}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                  <Calendar className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Join Date</p>
                  <p className="text-sm font-medium text-foreground">
                    {new Date(staff.createdAt).toLocaleDateString('en-IN', {
                      day: 'numeric',
                      month: 'long',
                      year: 'numeric',
                    })}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ---------- Shared Components ----------

function InfoRow({ icon: Icon, label, value }: { icon: typeof Mail; label: string; value: string }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/20 p-3">
      <Icon className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm font-medium text-foreground truncate">{value}</p>
      </div>
    </div>
  );
}

function formatRelative(dateStr: string): string {
  const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
  return new Date(dateStr).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' });
}
