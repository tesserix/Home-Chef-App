import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { FulfillmentType, OrderStatus } from '@/shared/types';

// GET /orders/:id/track — the only endpoint that will tell a customer where the
// kitchen actually is.
//
// The chef's address is deliberately not on the order payload: for delivery and
// chef-delivery it stays fuzzed (a home kitchen's address is private), and only a
// PICKUP order reveals the exact street address and pin, because the customer has
// to go there. See chefTrackCoords in apps/api/handlers/orders.go — `address` is
// present on this response only when the order is pickup.

export interface OrderTrackingChef {
  name: string;
  latitude: number;
  longitude: number;
  /** Exact street address — present for pickup orders only. */
  address?: string;
}

export interface OrderTracking {
  orderId: string;
  orderNumber: string;
  status: OrderStatus;
  fulfillmentType: FulfillmentType;
  readyPhotoUrl?: string;
  chef: OrderTrackingChef;
}

/**
 * Live tracking for one order. Enabled only when asked for, so a delivery order
 * (which has nothing extra to reveal here) never pays for the request.
 */
export function useOrderTracking(orderId: string | undefined, enabled: boolean) {
  return useQuery<OrderTracking>({
    queryKey: ['order-tracking', orderId],
    queryFn: () => apiClient.get<OrderTracking>(`/orders/${orderId}/track`),
    enabled: Boolean(orderId) && enabled,
    // The kitchen doesn't move; the status might, and the detail page already
    // polls the order itself.
    staleTime: 60_000,
  });
}
