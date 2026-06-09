import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Mirrors the response shape from GET /chef/admin-requests in
// apps/api/handlers/approval.go:631. Field names are the camelCase
// JSON keys emitted by the gin handler — not the underlying Go
// struct names — so they stay aligned with what mobile sees on the
// wire.
export interface AdminRequest {
  id: string;
  type: string;
  status: 'pending' | 'approved' | 'rejected' | 'info_requested' | 'cancelled';
  priority: string;
  title: string;
  description: string;
  adminNotes?: string;
  entityType?: string;
  entityId?: string;
  createdAt: string;
  updatedAt: string;
}

interface AdminRequestsResponse {
  data: AdminRequest[];
  pagination?: {
    page: number;
    limit: number;
    total: number;
  };
}

export function useAdminRequests() {
  return useQuery({
    queryKey: ['chef', 'admin-requests'],
    queryFn: () =>
      api
        .get<AdminRequestsResponse>('/chef/admin-requests')
        .then((r) => r.data?.data ?? []),
    staleTime: 30_000,
  });
}

// Subset that needs the chef's attention — used by the dashboard
// "ACTION REQUIRED" badge. We intentionally include `info_requested`
// only; pending requests are admin-side work and don't block the chef.
export function useActionRequiredAdminRequests() {
  const q = useAdminRequests();
  return {
    ...q,
    data: (q.data ?? []).filter((r) => r.status === 'info_requested'),
  };
}
