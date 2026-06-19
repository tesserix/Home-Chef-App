import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Admin mediation inbox (#53). Customer↔chef messages are held pending until an
// admin relays (or blocks) them — there is no direct chef↔customer channel.
// Mirrors the Go admin endpoints under /admin/messages + /admin/conversations.

export interface MediatedMessage {
  id: string;
  conversationId: string;
  orderId: string;
  senderId: string;
  senderRole: 'customer' | 'chef' | 'admin';
  recipientRole: 'customer' | 'chef' | 'admin';
  content: string;
  piiDetected: boolean;
  relayStatus: 'pending' | 'relayed' | 'blocked';
  createdAt: string;
}

/** The pending-relay queue. Polls so new inbound messages surface. */
export function useMediationInbox() {
  return useQuery<MediatedMessage[]>({
    queryKey: ['admin-mediation-inbox'],
    queryFn: async () => {
      const r = await apiClient.get<{ data: MediatedMessage[] }>('/admin/messages/inbox');
      return (r as unknown as { data: MediatedMessage[] }).data ?? [];
    },
    refetchInterval: 15000,
  });
}

export function useRelayMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiClient.post(`/admin/messages/${id}/relay`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-mediation-inbox'] }),
  });
}

export function useBlockMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiClient.post(`/admin/messages/${id}/block`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-mediation-inbox'] }),
  });
}

export function useAdminSend() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ conversationId, recipientRole, content }: { conversationId: string; recipientRole: 'customer' | 'chef'; content: string }) =>
      apiClient.post(`/admin/conversations/${conversationId}/send`, { recipientRole, content }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-mediation-inbox'] }),
  });
}
