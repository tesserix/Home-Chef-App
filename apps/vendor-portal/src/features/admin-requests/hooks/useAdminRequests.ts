import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Admin verification / info requests (#697). The web twin of
// apps/mobile-vendor/hooks/useAdminRequests.ts — SAME endpoints, SAME shape.
//
// Web had no admin-requests screen at all: the /admin-requests route was wired
// to the notifications page, so a chef on the portal could never see that an
// admin had asked them for information, let alone answer or bump it.
//
// Mirrors GET /chef/admin-requests in apps/api/handlers/approval.go. Field names
// are the camelCase JSON keys the gin handler emits, not the Go struct names.

export type AdminRequestStatus =
  | 'pending'
  | 'approved'
  | 'rejected'
  | 'info_requested'
  | 'cancelled';

export interface AdminRequest {
  id: string;
  type: string;
  status: AdminRequestStatus;
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
  pagination?: { page: number; limit: number; total: number };
}

export function useAdminRequests() {
  return useQuery({
    queryKey: ['chef', 'admin-requests'],
    queryFn: () =>
      apiClient
        .get<AdminRequestsResponse>('/chef/admin-requests')
        .then((r) => r?.data ?? []),
    staleTime: 30_000,
  });
}

interface RemindError {
  response?: { status?: number; data?: { nextRemindAt?: string; error?: string } };
  status?: number;
}

/**
 * Bump an unattended request so an admin is notified (#697).
 *
 * The cooldown is enforced server-side per REQUEST (not per caller), so a 429
 * here is a normal outcome — a client whose countdown drifted — not an error to
 * panic about. Either way we refetch, so the countdown resyncs to the server.
 */
export function useRemindAdminRequest() {
  const qc = useQueryClient();
  return useMutation<{ escalated: boolean; data: AdminRequest }, RemindError, string>({
    mutationFn: (id: string) =>
      apiClient.post<{ escalated: boolean; data: AdminRequest }>(
        `/chef/admin-requests/${id}/remind`,
      ),
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'admin-requests'] });
    },
  });
}

/**
 * The subset that needs the CHEF's attention, for the dashboard badge.
 *
 * Deliberately `info_requested` only: a `pending` request is admin-side work and
 * nothing the chef can act on, so badging it would train them to ignore the badge.
 */
export function useActionRequiredAdminRequests() {
  const q = useAdminRequests();
  return {
    ...q,
    data: (q.data ?? []).filter((r) => r.status === 'info_requested'),
  };
}

/**
 * How long until the chef can bump again, phrased the way a person waits.
 * Rounds UP so the label never claims "0h" while the button is still locked.
 */
export function untilLabel(iso: string, now: number): string {
  const ms = new Date(iso).getTime() - now;
  if (ms <= 0) return '';
  const mins = Math.ceil(ms / 60_000);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.ceil(mins / 60);
  if (hrs < 24) return `${hrs}h`;
  return `${Math.ceil(hrs / 24)}d`;
}
