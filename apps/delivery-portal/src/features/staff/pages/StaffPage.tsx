import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import {
  UserPlus,
  Mail,
  Shield,
  Copy,
  Check,
  RotateCw,
  XCircle,
  Clock,
  CheckCircle2,
} from 'lucide-react';
import { toast } from 'sonner';
import { PageLoader } from '@/shared/components/LoadingScreen';

interface StaffMember {
  id: string;
  userId: string;
  email: string;
  firstName: string;
  lastName: string;
  staffRole: string;
  department: string;
  title: string;
  isActive: boolean;
  joinedAt: string;
  lastActiveAt?: string;
  permissions: string[];
}

interface StaffRole {
  role: string;
  name: string;
  description: string;
  permissions: string[];
  portals: string[];
}

interface Invitation {
  id: string;
  email: string;
  staffRole: string;
  status: string;
  inviteUrl?: string;
  createdAt: string;
  expiresAt: string;
  department: string;
  title: string;
}

interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}

interface InviteForm {
  email: string;
  role: string;
  department: string;
  title: string;
  message: string;
}

const initialForm: InviteForm = {
  email: '',
  role: 'delivery_ops',
  department: '',
  title: '',
  message: '',
};

export default function StaffPage() {
  const queryClient = useQueryClient();
  const [showInviteForm, setShowInviteForm] = useState(false);
  const [form, setForm] = useState<InviteForm>(initialForm);
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null);
  const [inviteUrl, setInviteUrl] = useState<string | null>(null);

  const { data: staffData, isLoading: staffLoading } = useQuery({
    queryKey: ['delivery-staff'],
    queryFn: () => apiClient.get<PaginatedResponse<StaffMember>>('/delivery/staff', { portal: 'delivery' }),
  });

  const { data: myProfile } = useQuery({
    queryKey: ['delivery-staff-me'],
    queryFn: () => apiClient.get<StaffMember>('/delivery/staff/me'),
    retry: false,
  });

  const { data: rolesData } = useQuery({
    queryKey: ['delivery-staff-roles'],
    queryFn: () => apiClient.get<{ roles: StaffRole[] }>('/delivery/staff/roles'),
  });

  const { data: invitationsData, isLoading: invitationsLoading } = useQuery({
    queryKey: ['delivery-staff-invitations'],
    queryFn: () => apiClient.get<PaginatedResponse<Invitation>>('/delivery/staff/invitations'),
  });

  const staff = staffData?.data ?? [];
  const roles = rolesData?.roles ?? [];
  const invitations = invitationsData?.data ?? [];

  // Only fleet_manager and super_admin can invite
  const canInvite = myProfile?.permissions?.includes('staff:invite') ?? false;

  const createInvitation = useMutation({
    mutationFn: (data: InviteForm) =>
      apiClient.post<Invitation>('/delivery/staff/invitations', data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['delivery-staff-invitations'] });
      setInviteUrl(data.inviteUrl ?? '');
      toast.success('Invitation created successfully');
    },
    onError: () => toast.error('Failed to create invitation'),
  });

  const revokeInvitation = useMutation({
    mutationFn: (id: string) =>
      apiClient.put(`/delivery/staff/invitations/${id}/revoke`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['delivery-staff-invitations'] });
      toast.success('Invitation revoked');
    },
    onError: () => toast.error('Failed to revoke invitation'),
  });

  const resendInvitation = useMutation({
    mutationFn: (id: string) =>
      apiClient.put(`/delivery/staff/invitations/${id}/resend`),
    onSuccess: () => toast.success('Invitation resent'),
    onError: () => toast.error('Failed to resend invitation'),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.email || !form.role) {
      toast.error('Email and role are required');
      return;
    }
    createInvitation.mutate(form);
  };

  const handleCopyUrl = async (url: string) => {
    await navigator.clipboard.writeText(url);
    setCopiedUrl(url);
    toast.success('Invite URL copied to clipboard');
    setTimeout(() => setCopiedUrl(null), 2000);
  };

  const resetForm = () => {
    setForm(initialForm);
    setInviteUrl(null);
    setShowInviteForm(false);
  };

  if (staffLoading) return <PageLoader />;

  const pendingInvitations = invitations.filter(
    (inv) => inv.status === 'pending'
  );

  const roleLabel = (role: string) =>
    role
      .split('_')
      .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
      .join(' ');

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="page-title">Staff Management</h1>
          <p className="page-description">Manage delivery portal staff members</p>
        </div>
        {canInvite && (
          <button
            onClick={() => setShowInviteForm(!showInviteForm)}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <UserPlus className="h-4 w-4" />
            Invite Staff
          </button>
        )}
      </div>

      {/* Invite Form */}
      {showInviteForm && (
        <div className="rounded-xl border border-border bg-card p-6">
          <h2 className="text-lg font-semibold text-foreground mb-4">
            Send Staff Invitation
          </h2>

          {inviteUrl ? (
            <div className="space-y-4">
              <div className="rounded-lg border border-success/30 bg-success/10 p-4">
                <div className="flex items-center gap-2 mb-2">
                  <CheckCircle2 className="h-5 w-5 text-success" />
                  <p className="text-sm font-medium text-success">
                    Invitation Created
                  </p>
                </div>
                <p className="text-xs text-muted-foreground mb-3">
                  Share this link with the new staff member:
                </p>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    readOnly
                    value={inviteUrl}
                    className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground"
                  />
                  <button
                    onClick={() => handleCopyUrl(inviteUrl)}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground hover:bg-secondary"
                  >
                    {copiedUrl === inviteUrl ? (
                      <Check className="h-4 w-4 text-success" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                    {copiedUrl === inviteUrl ? 'Copied' : 'Copy'}
                  </button>
                </div>
              </div>
              <button
                onClick={resetForm}
                className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-secondary"
              >
                Done
              </button>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-foreground">
                    Email *
                  </label>
                  <input
                    type="email"
                    required
                    value={form.email}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, email: e.target.value }))
                    }
                    placeholder="staff@example.com"
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-foreground">
                    Role *
                  </label>
                  <select
                    value={form.role}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, role: e.target.value }))
                    }
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  >
                    {roles.length > 0 ? (
                      roles
                        .filter((r) => {
                          const deliveryRoles = ['fleet_manager', 'delivery_ops'];
                          // Super admins can also invite super_admins
                          if (myProfile?.staffRole === 'super_admin') {
                            return [...deliveryRoles, 'super_admin'].includes(r.role);
                          }
                          return deliveryRoles.includes(r.role);
                        })
                        .map((r) => (
                          <option key={r.role} value={r.role}>
                            {roleLabel(r.role)} — {r.description}
                          </option>
                        ))
                    ) : (
                      <>
                        <option value="delivery_ops">Delivery Ops</option>
                        <option value="fleet_manager">Fleet Manager</option>
                        {myProfile?.staffRole === 'super_admin' && (
                          <option value="super_admin">Super Admin</option>
                        )}
                      </>
                    )}
                  </select>
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-foreground">
                    Department
                  </label>
                  <input
                    type="text"
                    value={form.department}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, department: e.target.value }))
                    }
                    placeholder="e.g. Operations"
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-foreground">
                    Title
                  </label>
                  <input
                    type="text"
                    value={form.title}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, title: e.target.value }))
                    }
                    placeholder="e.g. Fleet Coordinator"
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-foreground">
                  Message (optional)
                </label>
                <textarea
                  value={form.message}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, message: e.target.value }))
                  }
                  rows={3}
                  placeholder="Add a personal message to the invitation email..."
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none"
                />
              </div>
              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={createInvitation.isPending}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                >
                  <Mail className="h-4 w-4" />
                  {createInvitation.isPending
                    ? 'Sending...'
                    : 'Send Invitation'}
                </button>
                <button
                  type="button"
                  onClick={resetForm}
                  className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-secondary"
                >
                  Cancel
                </button>
              </div>
            </form>
          )}
        </div>
      )}

      {/* Staff List */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">
          Current Staff
        </h2>
        {staff.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            No staff members yet. Send an invitation to get started.
          </p>
        ) : (
          <div className="space-y-3">
            {staff.map((member) => (
              <div
                key={member.id}
                className="flex items-center justify-between rounded-lg border border-border p-4"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                    <Shield className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {[member.firstName, member.lastName].filter(Boolean).join(' ') || member.email}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {member.email}
                    </p>
                    {member.department && (
                      <p className="text-xs text-muted-foreground">
                        {member.department}{member.title ? ` · ${member.title}` : ''}
                      </p>
                    )}
                  </div>
                </div>
                <div className="text-right">
                  <span className="inline-block rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary">
                    {roleLabel(member.staffRole)}
                  </span>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {member.isActive ? 'Active' : 'Inactive'}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Pending Invitations */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">
          Pending Invitations
        </h2>
        {invitationsLoading ? (
          <div className="flex justify-center py-8">
            <RotateCw className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : pendingInvitations.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            No pending invitations.
          </p>
        ) : (
          <div className="space-y-3">
            {pendingInvitations.map((inv) => (
              <div
                key={inv.id}
                className="flex items-center justify-between rounded-lg border border-border p-4"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-warning/10">
                    <Clock className="h-5 w-5 text-warning" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {inv.email}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {roleLabel(inv.staffRole)} &middot; Sent{' '}
                      {new Date(inv.createdAt).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => resendInvitation.mutate(inv.id)}
                    disabled={resendInvitation.isPending}
                    className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary disabled:opacity-50"
                  >
                    <RotateCw className="h-3.5 w-3.5" />
                    Resend
                  </button>
                  <button
                    onClick={() => revokeInvitation.mutate(inv.id)}
                    disabled={revokeInvitation.isPending}
                    className="flex items-center gap-1.5 rounded-lg border border-destructive/30 px-3 py-1.5 text-xs font-medium text-destructive hover:bg-destructive/5 disabled:opacity-50"
                  >
                    <XCircle className="h-3.5 w-3.5" />
                    Revoke
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
