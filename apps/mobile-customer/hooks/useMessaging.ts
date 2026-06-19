import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// In-app messaging (#53). Admin-mediated, order-scoped: the customer's messages
// are reviewed by support before reaching the chef, and replies arrive relayed.
// Mirrors GET/POST /v1/customer/orders/:id/messages.

export interface Message {
  id: string;
  senderRole: 'customer' | 'chef' | 'admin';
  recipientRole: string;
  content: string;
  relayStatus: 'pending' | 'relayed' | 'blocked';
  piiDetected: boolean;
  createdAt: string;
}

export function useOrderMessages(orderId: string) {
  return useQuery<Message[]>({
    queryKey: ['order-messages', orderId],
    queryFn: async () => {
      const r = await api.get(`/v1/customer/orders/${orderId}/messages`);
      return (r.data?.data ?? []) as Message[];
    },
    enabled: !!orderId,
    refetchInterval: 15000, // poll so relayed replies show without a manual refresh
  });
}

export function useSendMessage(orderId: string) {
  const qc = useQueryClient();
  return useMutation<{ piiDetected?: boolean }, Error, string>({
    mutationFn: async (content: string) => {
      const r = await api.post(`/v1/customer/orders/${orderId}/messages`, { content });
      return { piiDetected: r.data?.data?.piiDetected };
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['order-messages', orderId] }),
  });
}
