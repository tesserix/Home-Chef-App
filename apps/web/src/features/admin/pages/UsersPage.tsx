import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  MoreHorizontal,
  User,
  Mail,
  Phone,
  Calendar,
  Ban,
  CheckCircle,
  Loader2,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
import type { User as UserType, PaginatedResponse } from '@/shared/types';

const ROLE_FILTER = [
  { value: 'all', label: 'All Roles' },
  { value: 'customer', label: 'Customers' },
  { value: 'chef', label: 'Chefs' },
  { value: 'delivery', label: 'Delivery Partners' },
  { value: 'admin', label: 'Admins' },
];

const STATUS_FILTER = [
  { value: 'all', label: 'All Status' },
  { value: 'active', label: 'Active' },
  { value: 'suspended', label: 'Suspended' },
];

export default function AdminUsersPage() {
  const [searchQuery, setSearchQuery] = useState('');
  const [roleFilter, setRoleFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [page, setPage] = useState(1);
  const [selectedUser, setSelectedUser] = useState<UserType | null>(null);
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-users', { search: searchQuery, role: roleFilter, status: statusFilter, page }],
    queryFn: () =>
      apiClient.get<PaginatedResponse<UserType>>('/admin/users', {
        search: searchQuery || undefined,
        role: roleFilter !== 'all' ? roleFilter : undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        page,
        limit: 20,
      }),
  });

  const updateUserMutation = useMutation({
    mutationFn: ({ userId, action }: { userId: string; action: 'suspend' | 'activate' }) =>
      apiClient.put(`/admin/users/${userId}/${action}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      toast.success('User updated successfully');
      setSelectedUser(null);
    },
  });

  const users = data?.data || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-paper">Users</h1>
          <p className="mt-1 text-ink-muted">
            {data?.pagination.total || 0} total users
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1 max-w-md">
          <label htmlFor="admin-users-search" className="sr-only">Search users</label>
          <Search aria-hidden="true" className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-ink-muted" />
          <input
            id="admin-users-search"
            type="search"
            aria-label="Search users by name or email"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search by name or email..."
            className="w-full rounded-lg bg-ink border-ink-soft pl-10 pr-4 py-2.5 text-paper placeholder:text-ink-muted focus-visible:border-herb focus-visible:ring-herb"
          />
        </div>
        <div className="flex gap-3">
          <select
            value={roleFilter}
            onChange={(e) => setRoleFilter(e.target.value)}
            className="rounded-lg bg-ink border-ink-soft text-paper"
          >
            {ROLE_FILTER.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="rounded-lg bg-ink border-ink-soft text-paper"
          >
            {STATUS_FILTER.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Users Table */}
      <div className="rounded-xl bg-ink overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-herb"  aria-hidden="true" />
          </div>
        ) : users.length === 0 ? (
          <div className="p-12 text-center">
            <User className="mx-auto h-12 w-12 text-ink-soft"  aria-hidden="true" />
            <h3 className="mt-4 font-medium text-paper">No users found</h3>
            <p className="mt-2 text-ink-muted">Try adjusting your filters</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-ink-soft/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    User
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Role
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Joined
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-ink-muted uppercase">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                {users.map((user) => (
                  <tr key={user.id} className="hover:bg-ink-soft/50">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="h-10 w-10 rounded-full bg-ink-soft flex items-center justify-center">
                          {user.avatar ? (
                            <img
                              src={user.avatar}
                              alt=""
                              className="h-full w-full rounded-full object-cover" loading="lazy" decoding="async"
                            />
                          ) : (
                            <span className="text-lg font-medium text-ink-muted">
                              {user.firstName?.charAt(0)}
                            </span>
                          )}
                        </div>
                        <div>
                          <p className="font-medium text-paper">
                            {user.firstName} {user.lastName}
                          </p>
                          <p className="text-sm text-ink-muted">{user.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <RoleBadge role={user.role} />
                    </td>
                    <td className="px-6 py-4">
                      <StatusBadge status={user.isActive ? 'active' : 'suspended'} />
                    </td>
                    <td className="px-6 py-4 text-sm text-ink-muted">
                      {new Date(user.createdAt).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Actions for ${user.firstName} ${user.lastName}`}
                        onClick={() => setSelectedUser(user)}
                        className="text-ink-muted hover:bg-ink-soft/50 hover:text-paper"
                      >
                        <MoreHorizontal className="h-5 w-5"  aria-hidden="true" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {data && data.pagination.totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-ink-soft px-6 py-4">
            <p className="text-sm text-ink-muted">
              Showing {(page - 1) * 20 + 1} to {Math.min(page * 20, data.pagination.total)} of{' '}
              {data.pagination.total}
            </p>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="Previous page"
                disabled={!data.pagination.hasPrev}
                onClick={() => setPage(page - 1)}
                className="bg-ink-soft text-paper hover:bg-ink-soft/80 hover:text-paper"
              >
                <ChevronLeft className="h-5 w-5"  aria-hidden="true" />
              </Button>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="Next page"
                disabled={!data.pagination.hasNext}
                onClick={() => setPage(page + 1)}
                className="bg-ink-soft text-paper hover:bg-ink-soft/80 hover:text-paper"
              >
                <ChevronRight className="h-5 w-5"  aria-hidden="true" />
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* User Detail Modal */}
      {selectedUser && (
        <UserDetailModal
          user={selectedUser}
          onClose={() => setSelectedUser(null)}
          onSuspend={() =>
            updateUserMutation.mutate({
              userId: selectedUser.id,
              action: selectedUser.isActive ? 'suspend' : 'activate',
            })
          }
          isUpdating={updateUserMutation.isPending}
        />
      )}
    </div>
  );
}

function RoleBadge({ role }: { role: string }) {
  const config: Record<string, { label: string; className: string }> = {
    customer: { label: 'Customer', className: 'bg-info/20 text-info' },
    chef: { label: 'Chef', className: 'bg-info/20 text-info' },
    delivery: { label: 'Delivery', className: 'bg-info/20 text-info' },
    admin: { label: 'Admin', className: 'bg-herb/20 text-herb' },
    super_admin: { label: 'Super Admin', className: 'bg-paprika/20 text-paprika' },
  };

  const { label, className } = config[role] || { label: role, className: 'bg-ink-muted/20 text-ink-muted' };

  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}>
      {label}
    </span>
  );
}

function StatusBadge({ status }: { status: string }) {
  return status === 'active' ? (
    <span className="inline-flex items-center gap-1 rounded-full bg-herb/20 px-2.5 py-0.5 text-xs font-medium text-herb">
      <CheckCircle className="h-3 w-3"  aria-hidden="true" />
      Active
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 rounded-full bg-paprika/20 px-2.5 py-0.5 text-xs font-medium text-paprika">
      <Ban className="h-3 w-3" />
      Suspended
    </span>
  );
}

function UserDetailModal({
  user,
  onClose,
  onSuspend,
  isUpdating,
}: {
  user: UserType;
  onClose: () => void;
  onSuspend: () => void;
  isUpdating: boolean;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/70 p-4">
      <div className="w-full max-w-md rounded-xl bg-ink shadow-3">
        <div className="p-6">
          <div className="flex items-start gap-4">
            <div className="h-16 w-16 rounded-full bg-ink-soft flex items-center justify-center">
              {user.avatar ? (
                <img src={user.avatar} alt="" className="h-full w-full rounded-full object-cover" loading="lazy" decoding="async" />
              ) : (
                <span className="text-2xl font-medium text-ink-muted">
                  {user.firstName?.charAt(0)}
                  {user.lastName?.charAt(0)}
                </span>
              )}
            </div>
            <div>
              <h2 className="text-xl font-semibold text-paper">
                {user.firstName} {user.lastName}
              </h2>
              <div className="mt-1 flex items-center gap-2">
                <RoleBadge role={user.role} />
                <StatusBadge status={user.isActive ? 'active' : 'suspended'} />
              </div>
            </div>
          </div>

          <div className="mt-6 space-y-3">
            <div className="flex items-center gap-3 text-ink-muted">
              <Mail className="h-4 w-4 text-ink-muted"  aria-hidden="true" />
              {user.email}
            </div>
            {user.phone && (
              <div className="flex items-center gap-3 text-ink-muted">
                <Phone className="h-4 w-4 text-ink-muted"  aria-hidden="true" />
                {user.phone}
              </div>
            )}
            <div className="flex items-center gap-3 text-ink-muted">
              <Calendar className="h-4 w-4 text-ink-muted"  aria-hidden="true" />
              Joined {new Date(user.createdAt).toLocaleDateString()}
            </div>
          </div>
        </div>

        <div className="flex gap-3 border-t border-ink-soft p-4">
          <Button
            variant="outline"
            fullWidth
            onClick={onClose}
            className="border-ink-soft text-paper hover:bg-ink-soft/30 hover:text-paper hover:border-ink-soft"
          >
            Close
          </Button>
          <Button
            variant={user.isActive ? 'destructive' : 'success'}
            fullWidth
            isLoading={isUpdating}
            disabled={isUpdating}
            leftIcon={
              !isUpdating
                ? user.isActive
                  ? <Ban className="h-4 w-4" />
                  : <CheckCircle className="h-4 w-4"  aria-hidden="true" />
                : undefined
            }
            onClick={onSuspend}
          >
            {user.isActive ? 'Suspend User' : 'Activate User'}
          </Button>
        </div>
      </div>
    </div>
  );
}
