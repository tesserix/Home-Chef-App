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
  relayedById?: string;
  relayedAt?: string;
  attachmentId?: string;
  filename?: string;
  contentType?: string;
  createdAt: string;
}

export interface Conversation {
  id: string;
  orderId: string;
  customerId: string;
  chefId: string;
  status: 'open' | 'closed';
  createdAt: string;
  lastMessageAt?: string;
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

// ── Communications audit (#312) ─────────────────────────────────────────────
// Admins (only) can browse EVERY conversation and read the COMPLETE transcript
// — including pending + blocked messages no participant ever sees — for audits.

export interface ConversationFilter {
  orderId?: string;
  customerId?: string;
  chefId?: string;
  status?: '' | 'open' | 'closed';
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

export interface ConversationListResult {
  data: Conversation[];
  total: number;
  limit: number;
  offset: number;
}

export interface ConversationTranscript {
  conversation: Conversation;
  messages: MediatedMessage[];
}

/** Audit list of all conversations, newest activity first. */
export function useConversations(filter: ConversationFilter) {
  return useQuery<ConversationListResult>({
    queryKey: ['admin-conversations', filter],
    queryFn: () => {
      const params: Record<string, string> = {};
      for (const [k, v] of Object.entries(filter)) {
        if (v !== undefined && v !== '') params[k] = String(v);
      }
      return apiClient.get<ConversationListResult>('/admin/conversations', params);
    },
  });
}

/** Full transcript (every message + status) of one conversation. */
export function useConversationTranscript(id: string | null) {
  return useQuery<ConversationTranscript>({
    queryKey: ['admin-conversation', id],
    enabled: !!id,
    queryFn: () => apiClient.get<ConversationTranscript>(`/admin/conversations/${id}`),
  });
}
