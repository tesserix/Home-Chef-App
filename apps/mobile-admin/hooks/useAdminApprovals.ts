import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type {
  ApprovalCounts,
  ApprovalRequest,
  Paginated,
} from '../lib/admin-types';

export interface ApprovalsQuery {
  status?: string;
  type?: string;
  priority?: string;
  search?: string;
  page?: number;
  limit?: number;
}

export function useAdminApprovals(q: ApprovalsQuery = {}) {
  const { status = 'pending', type = '', priority = '', search = '', page = 1, limit = 20 } = q;
  return useQuery<Paginated<ApprovalRequest>>({
    queryKey: ['admin', 'approvals', { status, type, priority, search, page, limit }],
    queryFn: () =>
      api
        .get<Paginated<ApprovalRequest>>('/admin/approvals', {
          params: { status, type, priority, search, page, limit },
        })
        .then((r) => r.data),
  });
}

export function useApprovalCounts() {
  return useQuery<ApprovalCounts>({
    queryKey: ['admin', 'approvals', 'counts'],
    queryFn: () => api.get<ApprovalCounts>('/admin/approvals/counts').then((r) => r.data),
    refetchInterval: 60_000,
  });
}

export function useApprovalDetail(id: string) {
  return useQuery<ApprovalRequest>({
    queryKey: ['admin', 'approval', id],
    queryFn: () => api.get<ApprovalRequest>(`/admin/approvals/${id}`).then((r) => r.data),
    enabled: !!id,
  });
}

function invalidateApprovals(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ['admin', 'approvals'] });
  qc.invalidateQueries({ queryKey: ['admin', 'approval'] });
  qc.invalidateQueries({ queryKey: ['admin', 'chefs'] });
  qc.invalidateQueries({ queryKey: ['admin', 'stats'] });
}

export function useApproveRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, notes }: { id: string; notes?: string }) =>
      api.put(`/admin/approvals/${id}/approve`, { notes: notes ?? '' }),
    onSuccess: () => invalidateApprovals(qc),
  });
}

export function useRejectRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, notes }: { id: string; notes: string }) =>
      api.put(`/admin/approvals/${id}/reject`, { notes }),
    onSuccess: () => invalidateApprovals(qc),
  });
}

export function useRequestMoreInfo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, notes }: { id: string; notes: string }) =>
      api.put(`/admin/approvals/${id}/request-info`, { notes }),
    onSuccess: () => invalidateApprovals(qc),
  });
}
