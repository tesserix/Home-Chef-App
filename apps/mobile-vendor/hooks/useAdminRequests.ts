import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
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

  // ── Reminders / escalation (#697) ──────────────────────────────────────────
  /** How many times the chef has bumped this. >= 3 means escalated. */
  reminderCount: number;
  lastRemindedAt?: string;
  /** Set once, when the 3rd bump escalated it. */
  escalatedAt?: string;
  /**
   * When the next bump unlocks. SERVER-computed: the cadence (24h for the first
   * three, 6h once escalated) is stated once in the API and rendered here, so it
   * cannot drift between clients or hinge on the device clock.
   */
  nextRemindAt?: string;
  /** The server's own verdict at response time — the source of truth. */
  canRemind: boolean;
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

/**
 * Bump an unattended request so an admin is notified (#697).
 *
 * The cooldown is enforced server-side per REQUEST (not per caller), so a 429
 * here is a normal outcome — a client whose countdown drifted — not an error to
 * panic about. It carries the real unlock time.
 */
export function useRemindAdminRequest() {
  const qc = useQueryClient();
  return useMutation<
    { escalated: boolean; data: AdminRequest },
    { response?: { status?: number; data?: { nextRemindAt?: string; error?: string } } },
    string
  >({
    mutationFn: (id: string) =>
      api.post(`/chef/admin-requests/${id}/remind`).then((r) => r.data),
    onSettled: () => {
      // Refetch either way: on success for the new count/cooldown, and on a 429
      // to resync a countdown that was evidently wrong.
      void qc.invalidateQueries({ queryKey: ['chef', 'admin-requests'] });
    },
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
