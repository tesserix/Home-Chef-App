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
          <h1 className="text-2xl font-bold text-white">Users</h1>
          <p className="mt-1 text-gray-400">
            {data?.pagination.total || 0} total users
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search by name or email..."
            className="w-full rounded-lg bg-gray-800 border-gray-700 pl-10 pr-4 py-2.5 text-white placeholder:text-gray-500 focus:border-brand-500 focus:ring-brand-500"
          />
        </div>
        <div className="flex gap-3">
          <select
            value={roleFilter}
            onChange={(e) => setRoleFilter(e.target.value)}
            className="rounded-lg bg-gray-800 border-gray-700 text-white"
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
            className="rounded-lg bg-gray-800 border-gray-700 text-white"
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
      <div className="rounded-xl bg-gray-800 overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
          </div>
        ) : users.length === 0 ? (
          <div className="p-12 text-center">
            <User className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 font-medium text-white">No users found</h3>
            <p className="mt-2 text-gray-400">Try adjusting your filters</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-700/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    User
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Role
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Joined
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                {users.map((user) => (
                  <tr key={user.id} className="hover:bg-gray-700/50">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="h-10 w-10 rounded-full bg-gray-700 flex items-center justify-center">
                          {user.avatar ? (
                            <img
                              src={user.avatar}
                              alt=""
                              className="h-full w-full rounded-full object-cover"
                            />
                          ) : (
                            <span className="text-lg font-medium text-gray-400">
                              {user.firstName?.charAt(0)}
                            </span>
                          )}
                        </div>
                        <div>
                          <p className="font-medium text-white">
                            {user.firstName} {user.lastName}
                          </p>
                          <p className="text-sm text-gray-400">{user.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <RoleBadge role={user.role} />
                    </td>
                    <td className="px-6 py-4">
                      <StatusBadge status={user.isActive ? 'active' : 'suspended'} />
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-400">
                      {new Date(user.createdAt).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <button
                        onClick={() => setSelectedUser(user)}
                        className="p-2 text-gray-400 hover:text-white"
                      >
                        <MoreHorizontal className="h-5 w-5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {data && data.pagination.totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-gray-700 px-6 py-4">
            <p className="text-sm text-gray-400">
              Showing {(page - 1) * 20 + 1} to {Math.min(page * 20, data.pagination.total)} of{' '}
              {data.pagination.total}
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(page - 1)}
                disabled={!data.pagination.hasPrev}
                className="p-2 rounded-lg bg-gray-700 text-white disabled:opacity-50"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
              <button
                onClick={() => setPage(page + 1)}
                disabled={!data.pagination.hasNext}
                className="p-2 rounded-lg bg-gray-700 text-white disabled:opacity-50"
              >
                <ChevronRight className="h-5 w-5" />
              </button>
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
    customer: { label: 'Customer', className: 'bg-blue-500/20 text-blue-400' },
    chef: { label: 'Chef', className: 'bg-purple-500/20 text-purple-400' },
    delivery: { label: 'Delivery', className: 'bg-cyan-500/20 text-cyan-400' },
    admin: { label: 'Admin', className: 'bg-orange-500/20 text-orange-400' },
    super_admin: { label: 'Super Admin', className: 'bg-red-500/20 text-red-400' },
  };

  const { label, className } = config[role] || { label: role, className: 'bg-gray-500/20 text-gray-400' };

  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}>
      {label}
    </span>
  );
}

function StatusBadge({ status }: { status: string }) {
  return status === 'active' ? (
    <span className="inline-flex items-center gap-1 rounded-full bg-green-500/20 px-2.5 py-0.5 text-xs font-medium text-green-400">
      <CheckCircle className="h-3 w-3" />
      Active
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 rounded-full bg-red-500/20 px-2.5 py-0.5 text-xs font-medium text-red-400">
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
      <div className="w-full max-w-md rounded-xl bg-gray-800 shadow-xl">
        <div className="p-6">
          <div className="flex items-start gap-4">
            <div className="h-16 w-16 rounded-full bg-gray-700 flex items-center justify-center">
              {user.avatar ? (
                <img src={user.avatar} alt="" className="h-full w-full rounded-full object-cover" />
              ) : (
                <span className="text-2xl font-medium text-gray-400">
                  {user.firstName?.charAt(0)}
                  {user.lastName?.charAt(0)}
                </span>
              )}
            </div>
            <div>
              <h2 className="text-xl font-semibold text-white">
                {user.firstName} {user.lastName}
              </h2>
              <div className="mt-1 flex items-center gap-2">
                <RoleBadge role={user.role} />
                <StatusBadge status={user.isActive ? 'active' : 'suspended'} />
              </div>
            </div>
          </div>

          <div className="mt-6 space-y-3">
            <div className="flex items-center gap-3 text-gray-300">
              <Mail className="h-4 w-4 text-gray-500" />
              {user.email}
            </div>
            {user.phone && (
              <div className="flex items-center gap-3 text-gray-300">
                <Phone className="h-4 w-4 text-gray-500" />
                {user.phone}
              </div>
            )}
            <div className="flex items-center gap-3 text-gray-300">
              <Calendar className="h-4 w-4 text-gray-500" />
              Joined {new Date(user.createdAt).toLocaleDateString()}
            </div>
          </div>
        </div>

        <div className="flex gap-3 border-t border-gray-700 p-4">
          <button onClick={onClose} className="flex-1 btn-outline border-gray-600 text-white">
            Close
          </button>
          <button
            onClick={onSuspend}
            disabled={isUpdating}
            className={`flex-1 ${
              user.isActive
                ? 'btn-base bg-red-600 text-white hover:bg-red-700'
                : 'btn-base bg-green-600 text-white hover:bg-green-700'
            }`}
          >
            {isUpdating ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : user.isActive ? (
              <>
                <Ban className="h-4 w-4" />
                Suspend User
              </>
            ) : (
              <>
                <CheckCircle className="h-4 w-4" />
                Activate User
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
