import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Chef-side in-app messaging (#53). Admin-mediated and order-scoped: what the
// chef sends is reviewed by support before it reaches the customer, and the
// customer's messages arrive here already relayed. Mirrors GET/POST
// /chef/orders/:orderId/messages — the same pair the vendor portal uses
// (apps/vendor-portal/src/features/orders/hooks/useOrderMessaging.ts) and the
// chef-side counterpart of the customer app's hooks/useMessaging.ts.

export interface OrderMessage {
  id: string;
  senderRole: 'customer' | 'chef' | 'admin';
  recipientRole?: string;
  content: string;
  relayStatus: 'pending' | 'relayed' | 'blocked';
  piiDetected: boolean;
  createdAt: string;
}

/**
 * The order's mediated thread.
 *
 * `enabled` is a parameter rather than always-on: the thread sits behind a
 * collapsed row on the order screen, and polling every open order's messages
 * would put a request every 15s behind a card nobody has looked at.
 */
export function useOrderMessages(orderId: string, enabled = true) {
  return useQuery<OrderMessage[]>({
    queryKey: ['chef', 'order-messages', orderId],
    queryFn: () =>
      api
        .get<{ data: OrderMessage[] }>(`/chef/orders/${orderId}/messages`)
        .then((r) => r.data?.data ?? []),
    enabled: enabled && !!orderId,
    // Relay happens out-of-band (support approves), so nothing pushes the
    // reply back to this screen — poll while the thread is open.
    refetchInterval: 15_000,
  });
}

/**
 * Send a message on this order.
 *
 * Resolves with `piiDetected` so the caller can tell the chef their phone
 * number or address was redacted, rather than letting them believe it went
 * through intact.
 */
export function useSendOrderMessage(orderId: string) {
  const queryClient = useQueryClient();
  return useMutation<{ piiDetected: boolean }, Error, string>({
    mutationFn: (content: string) =>
      api
        .post<{ data: OrderMessage }>(`/chef/orders/${orderId}/messages`, { content })
        .then((r) => ({ piiDetected: r.data?.data?.piiDetected ?? false })),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['chef', 'order-messages', orderId] }),
  });
}
