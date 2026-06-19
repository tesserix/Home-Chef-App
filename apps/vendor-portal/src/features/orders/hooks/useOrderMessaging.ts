import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Chef-side in-app messaging (#53). Admin-mediated: the chef's messages are
// reviewed by support before reaching the customer (no direct channel), and the
// customer's relayed messages arrive here. Mirrors GET/POST
// /chef/orders/:orderId/messages.

export interface Message {
  id: string;
  senderRole: 'customer' | 'chef' | 'admin';
  content: string;
  relayStatus: 'pending' | 'relayed' | 'blocked';
  piiDetected: boolean;
  createdAt: string;
}

export function useOrderMessages(orderId: string, enabled: boolean) {
  return useQuery<Message[]>({
    queryKey: ['vendor-order-messages', orderId],
    queryFn: async () => {
      const r = await apiClient.get<{ data: Message[] }>(`/chef/orders/${orderId}/messages`);
      return (r as unknown as { data: Message[] }).data ?? [];
    },
    enabled: enabled && !!orderId,
    refetchInterval: 15000,
  });
}

export function useSendOrderMessage(orderId: string) {
  const qc = useQueryClient();
  return useMutation<unknown, Error, string>({
    mutationFn: (content: string) => apiClient.post(`/chef/orders/${orderId}/messages`, { content }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-order-messages', orderId] }),
  });
}
