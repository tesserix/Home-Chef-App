import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  Filter,
  Loader2,
  Eye,
  Mail,
  UserPlus,
  X,
  Copy,
  Check,
  Ban,
  Send,
  Calendar,
  Clock,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

// ---------- Types ----------

interface StaffMember {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  phone?: string;
  role: string;
  department?: string;
  title?: string;
  isActive: boolean;
  invitedBy?: string;
  createdAt: string;
}

interface StaffListResponse {
  data: StaffMember[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
}

interface Invitation {
  id: string;
  email: string;
  role: string;
  department?: string;
  title?: string;
  status: string;
  invitedBy: string;
  invitedByName?: string;
  message?: string;
  inviteUrl?: string;
  createdAt: string;
  expiresAt: string;
}

interface InvitationListResponse {
  data: Invitation[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
}

interface RoleDefinition {
  name: string;
  description: string;
  permissions: string[];
}

interface RolesResponse {
  data: RoleDefinition[];
}

interface InviteResponse {
  data: {
    id: string;
    inviteUrl: string;
  };
}

// ---------- Constants ----------

const STAFF_ROLES: Record<string, { label: string; color: string }> = {
  super_admin: { label: 'Super Admin', color: 'bg-destructive/10 text-destructive' },
  admin: { label: 'Admin', color: 'bg-warning/10 text-warning' },
  fleet_manager: { label: 'Fleet Manager', color: 'bg-info/10 text-info' },
  delivery_ops: { label: 'Delivery Ops', color: 'bg-primary/10 text-primary' },
  support: { label: 'Support', color: 'bg-success/10 text-success' },
};

const INVITATION_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: 'Pending', color: 'bg-warning/10 text-warning' },
  accepted: { label: 'Accepted', color: 'bg-success/10 text-success' },
  expired: { label: 'Expired', color: 'bg-muted text-muted-foreground' },
  revoked: { label: 'Revoked', color: 'bg-destructive/10 text-destructive' },
};

// ---------- Main Component ----------

export default function StaffPage() {
  const [activeTab, setActiveTab] = useState<'members' | 'invitations'>('members');
  const [showInviteDialog, setShowInviteDialog] = useState(false);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="page-header">
          <h1 className="page-title">Staff Management</h1>
          <p className="page-description">
            Manage staff members and invitations
          </p>
        </div>
        <Button
          variant="primary"
          leftIcon={<UserPlus className="h-4 w-4" />}
          onClick={() => setShowInviteDialog(true)}
        >
          Invite Staff
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 rounded-lg border border-border bg-muted/50 p-1 w-fit">
        <button
          onClick={() => setActiveTab('members')}
          className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'members'
              ? 'bg-card text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          Staff Members
        </button>
        <button
          onClick={() => setActiveTab('invitations')}
          className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'invitations'
              ? 'bg-card text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          Invitations
        </button>
      </div>

      {/* Content */}
      {activeTab === 'members' ? <StaffMembersView /> : <InvitationsView />}

      {/* Invite Dialog */}
      {showInviteDialog && (
        <InviteStaffDialog onClose={() => setShowInviteDialog(false)} />
      )}
    </div>
  );
}

// ---------- Staff Members View ----------

function StaffMembersView() {
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<string>('all');
  const [page, setPage] = useState(1);
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-staff', search, roleFilter, page],
    queryFn: () =>
      apiClient.get<StaffListResponse>('/admin/staff', {
        search: search || undefined,
        role: roleFilter !== 'all' ? roleFilter : undefined,
        page,
        limit: 20,
      }),
  });

  const resp = data as unknown as StaffListResponse | undefined;
  const staff = resp?.data ?? [];
  const pagination = resp?.pagination;

  return (
    <>
      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <label htmlFor="staff-search" className="sr-only">Search staff</label>
          <Search aria-hidden="true" className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            id="staff-search"
            type="search"
            placeholder="Search staff by name or email..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="h-10 w-full rounded-lg border border-input bg-card pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
          <label htmlFor="staff-role-filter" className="sr-only">Filter by role</label>
          <select
            id="staff-role-filter"
            value={roleFilter}
            onChange={(e) => { setRoleFilter(e.target.value); setPage(1); }}
            className="h-10 rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="all">All Roles</option>
            <option value="super_admin">Super Admin</option>
            <option value="admin">Admin</option>
            <option value="fleet_manager">Fleet Manager</option>
            <option value="delivery_ops">Delivery Ops</option>
            <option value="support">Support</option>
          </select>
        </div>
      </div>

      {/* Staff count */}
      {pagination && (
        <p className="text-sm text-muted-foreground">
          {pagination.total} staff member{pagination.total !== 1 ? 's' : ''}
        </p>
      )}

      {/* Staff Table */}
      <div className="rounded-xl border border-border bg-card shadow-card">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Staff Member</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Role</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Department</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Title</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Joined</th>
                  <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-muted-foreground">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {staff.map((member) => (
                  <tr key={member.id} className="hover:bg-secondary/30 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                          <span className="text-sm font-semibold text-primary">
                            {(member.firstName?.[0] || member.email[0] || '?').toUpperCase()}
                          </span>
                        </div>
                        <div>
                          <p className="font-medium text-foreground">
                            {member.firstName} {member.lastName}
                          </p>
                          <p className="text-xs text-muted-foreground flex items-center gap-1">
                            <Mail className="h-3 w-3" />
                            {member.email}
                          </p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <RoleBadge role={member.role} />
                    </td>
                    <td className="px-6 py-4 text-sm text-foreground">
                      {member.department || <span className="text-muted-foreground">--</span>}
                    </td>
                    <td className="px-6 py-4 text-sm text-foreground">
                      {member.title || <span className="text-muted-foreground">--</span>}
                    </td>
                    <td className="px-6 py-4">
                      {member.isActive ? (
                        <span className="inline-flex items-center rounded-full bg-success/10 px-2.5 py-0.5 text-xs font-medium text-success">
                          Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center rounded-full bg-destructive/10 px-2.5 py-0.5 text-xs font-medium text-destructive">
                          Inactive
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-sm text-muted-foreground">
                      {new Date(member.createdAt).toLocaleDateString('en-IN', {
                        day: 'numeric',
                        month: 'short',
                        year: 'numeric',
                      })}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center justify-end">
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label="View staff member details"
                          title="View Details"
                          onClick={() => navigate(`/staff/${member.id}`)}
                          className="text-muted-foreground hover:bg-secondary hover:text-foreground"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
                {staff.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-6 py-20 text-center text-muted-foreground">
                      No staff members found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {pagination && pagination.totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-border px-6 py-3">
            <p className="text-sm text-muted-foreground">
              Page {pagination.page} of {pagination.totalPages}
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.hasPrev}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.hasNext}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

// ---------- Invitations View ----------

function InvitationsView() {
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-staff-invitations', statusFilter, page],
    queryFn: () =>
      apiClient.get<InvitationListResponse>('/admin/staff/invitations', {
        status: statusFilter !== 'all' ? statusFilter : undefined,
        page,
        limit: 20,
      }),
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => apiClient.put(`/admin/staff/invitations/${id}/revoke`),
    onSuccess: () => {
      toast.success('Invitation revoked');
      queryClient.invalidateQueries({ queryKey: ['admin-staff-invitations'] });
    },
    onError: () => toast.error('Failed to revoke invitation'),
  });

  const resendMutation = useMutation({
    mutationFn: (id: string) => apiClient.put(`/admin/staff/invitations/${id}/resend`),
    onSuccess: () => {
      toast.success('Invitation resent');
      queryClient.invalidateQueries({ queryKey: ['admin-staff-invitations'] });
    },
    onError: () => toast.error('Failed to resend invitation'),
  });

  const resp = data as unknown as InvitationListResponse | undefined;
  const invitations = resp?.data ?? [];
  const pagination = resp?.pagination;

  return (
    <>
      {/* Filters */}
      <div className="flex items-center gap-2">
        <Filter aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
        <label htmlFor="invitations-status-filter" className="sr-only">Filter by status</label>
        <select
          id="invitations-status-filter"
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
          className="h-10 rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
        >
          <option value="all">All Statuses</option>
          <option value="pending">Pending</option>
          <option value="accepted">Accepted</option>
          <option value="expired">Expired</option>
          <option value="revoked">Revoked</option>
        </select>
      </div>

      {/* Invitations Table */}
      <div className="rounded-xl border border-border bg-card shadow-card">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Email</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Role</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Invited By</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Created</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Expires</th>
                  <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-muted-foreground">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {invitations.map((inv) => (
                  <tr key={inv.id} className="hover:bg-secondary/30 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2 text-sm text-foreground">
                        <Mail className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                        {inv.email}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <RoleBadge role={inv.role} />
                    </td>
                    <td className="px-6 py-4">
                      <InvitationStatusBadge status={inv.status} />
                    </td>
                    <td className="px-6 py-4 text-sm text-muted-foreground">
                      {inv.invitedByName || inv.invitedBy || '--'}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                        <Calendar className="h-3.5 w-3.5" />
                        {new Date(inv.createdAt).toLocaleDateString('en-IN', {
                          day: 'numeric',
                          month: 'short',
                          year: 'numeric',
                        })}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                        <Clock className="h-3.5 w-3.5" />
                        {new Date(inv.expiresAt).toLocaleDateString('en-IN', {
                          day: 'numeric',
                          month: 'short',
                          year: 'numeric',
                        })}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center justify-end gap-1">
                        {(inv.status === 'pending' || inv.status === 'expired') && (
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            disabled={resendMutation.isPending}
                            title="Resend Invitation"
                            aria-label="Resend invitation"
                            onClick={() => resendMutation.mutate(inv.id)}
                            className="text-muted-foreground hover:bg-info/10 hover:text-info"
                          >
                            <Send className="h-4 w-4" />
                          </Button>
                        )}
                        {inv.status === 'pending' && (
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            disabled={revokeMutation.isPending}
                            title="Revoke Invitation"
                            aria-label="Revoke invitation"
                            onClick={() => revokeMutation.mutate(inv.id)}
                            className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                          >
                            <Ban className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
                {invitations.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-6 py-20 text-center text-muted-foreground">
                      No invitations found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {pagination && pagination.totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-border px-6 py-3">
            <p className="text-sm text-muted-foreground">
              Page {pagination.page} of {pagination.totalPages}
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.hasPrev}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.hasNext}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

// ---------- Invite Staff Dialog ----------

function InviteStaffDialog({ onClose }: { onClose: () => void }) {
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('');
  const [department, setDepartment] = useState('');
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [inviteUrl, setInviteUrl] = useState('');
  const [copied, setCopied] = useState(false);
  const queryClient = useQueryClient();

  const { data: rolesData } = useQuery({
    queryKey: ['admin-staff-roles'],
    queryFn: () => apiClient.get<RolesResponse>('/admin/staff/roles'),
  });

  const roles = (rolesData as unknown as RolesResponse)?.data ?? [];

  const inviteMutation = useMutation({
    mutationFn: (body: { email: string; role: string; department?: string; title?: string; message?: string }) =>
      apiClient.post<InviteResponse>('/admin/staff/invitations', body),
    onSuccess: (data) => {
      const resp = data as unknown as InviteResponse;
      toast.success('Invitation sent successfully');
      setInviteUrl(resp.data?.inviteUrl || '');
      queryClient.invalidateQueries({ queryKey: ['admin-staff-invitations'] });
    },
    onError: () => toast.error('Failed to send invitation'),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !role) return;
    inviteMutation.mutate({
      email,
      role,
      department: department || undefined,
      title: title || undefined,
      message: message || undefined,
    });
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(inviteUrl);
      setCopied(true);
      toast.success('Invite URL copied to clipboard');
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Failed to copy URL');
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <button type="button" aria-label="Close" className="fixed inset-0 bg-foreground/50" onClick={onClose} />
      <div className="relative z-50 w-full max-w-lg rounded-xl border border-border bg-card p-6 shadow-3 mx-4">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold text-foreground">Invite Staff Member</h2>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Close dialog"
            onClick={onClose}
            className="text-muted-foreground hover:bg-secondary hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </Button>
        </div>

        {inviteUrl ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-success/30 bg-success/5 p-4">
              <p className="text-sm font-medium text-success mb-2">Invitation created successfully!</p>
              <p className="text-sm text-muted-foreground">
                Share this link with the invitee to complete their registration.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <label htmlFor="staff-invite-url" className="sr-only">Invitation link</label>
              <input
                id="staff-invite-url"
                type="text"
                readOnly
                value={inviteUrl}
                className="h-10 flex-1 rounded-lg border border-input bg-muted/50 px-3 text-sm text-foreground font-mono"
              />
              <Button
                variant="outline"
                leftIcon={copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
                onClick={handleCopy}
              >
                {copied ? 'Copied' : 'Copy'}
              </Button>
            </div>
            <Button variant="primary" fullWidth onClick={onClose}>
              Done
            </Button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="invite-email" className="block text-sm font-medium text-foreground mb-1.5">
                Email <span aria-hidden="true" className="text-destructive">*</span>
              </label>
              <input
                id="invite-email"
                type="email"
                required
                aria-required="true"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="colleague@company.com"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>

            <div>
              <label htmlFor="invite-role" className="block text-sm font-medium text-foreground mb-1.5">
                Role <span aria-hidden="true" className="text-destructive">*</span>
              </label>
              <select
                id="invite-role"
                required
                aria-required="true"
                value={role}
                onChange={(e) => setRole(e.target.value)}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              >
                <option value="">Select a role...</option>
                {roles.length > 0
                  ? roles.map((r) => (
                      <option key={r.name} value={r.name}>
                        {STAFF_ROLES[r.name]?.label || r.name} - {r.description}
                      </option>
                    ))
                  : Object.entries(STAFF_ROLES).map(([key, val]) => (
                      <option key={key} value={key}>
                        {val.label}
                      </option>
                    ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="invite-department" className="block text-sm font-medium text-foreground mb-1.5">
                  Department
                </label>
                <input
                  id="invite-department"
                  type="text"
                  value={department}
                  onChange={(e) => setDepartment(e.target.value)}
                  placeholder="e.g. Operations"
                  className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>
              <div>
                <label htmlFor="invite-title" className="block text-sm font-medium text-foreground mb-1.5">
                  Title
                </label>
                <input
                  id="invite-title"
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="e.g. Operations Lead"
                  className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>
            </div>

            <div>
              <label htmlFor="invite-message" className="block text-sm font-medium text-foreground mb-1.5">
                Personal Message
              </label>
              <textarea
                id="invite-message"
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Optional message to include in the invitation email..."
                rows={3}
                className="w-full rounded-lg border border-input bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring resize-none"
              />
            </div>

            <div className="flex items-center justify-end gap-3 pt-2">
              <Button type="button" variant="outline" onClick={onClose}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                isLoading={inviteMutation.isPending}
                disabled={inviteMutation.isPending || !email || !role}
                leftIcon={!inviteMutation.isPending ? <Send className="h-4 w-4" /> : undefined}
              >
                Send Invitation
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

// ---------- Shared Components ----------

function RoleBadge({ role }: { role: string }) {
  const config = STAFF_ROLES[role] || { label: role, color: 'bg-muted text-muted-foreground' };
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${config.color}`}>
      {config.label}
    </span>
  );
}

function InvitationStatusBadge({ status }: { status: string }) {
  const config = INVITATION_STATUSES[status] || { label: status, color: 'bg-muted text-muted-foreground' };
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${config.color}`}>
      {config.label}
    </span>
  );
}
