import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Paginated, StaffMember } from '../lib/admin-types';

export function useAdminStaff(q: { search?: string; page?: number } = {}) {
  const { search = '', page = 1 } = q;
  return useQuery<Paginated<StaffMember>>({
    queryKey: ['admin', 'staff', { search, page }],
    queryFn: () =>
      api
        .get<Paginated<StaffMember>>('/admin/staff', {
          params: { search, page, limit: 20 },
        })
        .then((r) => r.data),
  });
}

export function useMyStaffProfile() {
  return useQuery<StaffMember>({
    queryKey: ['admin', 'staff', 'me'],
    queryFn: () => api.get<StaffMember>('/admin/staff/me').then((r) => r.data),
    retry: false,
  });
}

export interface StaffRoleOption {
  key: string;
  label?: string;
  permissions?: string[];
}

export function useStaffRoles() {
  return useQuery<StaffRoleOption[]>({
    queryKey: ['admin', 'staff', 'roles'],
    queryFn: () =>
      api
        .get<{ roles?: StaffRoleOption[] } | StaffRoleOption[]>('/admin/staff/roles')
        .then((r) => (Array.isArray(r.data) ? r.data : (r.data.roles ?? []))),
  });
}

function invalidateStaff(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ['admin', 'staff'] });
}

export function useInviteStaff() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      email: string;
      staffRole: string;
      department?: string;
      title?: string;
      message?: string;
    }) => api.post('/admin/staff/invitations', body),
    onSuccess: () => invalidateStaff(qc),
  });
}

export function useDeactivateStaff() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.put(`/admin/staff/${id}/deactivate`),
    onSuccess: () => invalidateStaff(qc),
  });
}

export function useReactivateStaff() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.put(`/admin/staff/${id}/reactivate`),
    onSuccess: () => invalidateStaff(qc),
  });
}
