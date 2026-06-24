import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// ---- API contract types -------------------------------------------------------
// Shape returned by GET /chef/orders/:orderId.

export interface OrderDetailItem {
  id: string;
  name: string;
  quantity: number;
  unitPrice: number;
  lineTotal: number;
  isVeg?: boolean;
  specialInstructions?: string;
  // Per-line cancellation state — populated by the backend after a
  // chef triggers POST /chef/orders/:id/items/:itemId/cancel. The
  // detail screen renders cancelled lines with strikethrough + a
  // "Refunded ₹X" badge so the chef sees what's still in prep.
  isCancelled?: boolean;
  cancelledReason?: string;
  cancelledAt?: string | null;
  refundAmount?: number;
}

export interface OrderDetailAddress {
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postalCode: string;
}

export interface OrderDetailTiming {
  orderedAt: string;
  acceptedAt?: string | null;
  preparedAt?: string | null;
  pickedUpAt?: string | null;
  deliveredAt?: string | null;
}

export interface OrderDetailPricing {
  subtotal: number;
  deliveryFee: number;
  serviceFee: number;
  tax: number;
  chefTip: number;
  total: number;
}

export type OrderDetailStatus =
  | 'pending'
  | 'accepted'
  | 'preparing'
  | 'ready'
  | 'picked_up'
  | 'delivered'
  | 'cancelled'
  | 'rejected';

// How the order reaches the customer (backend OrderResponse.fulfillmentType).
// 'pickup' → customer collects from the chef; the chef confirms handover.
export type FulfillmentType = 'delivery' | 'chef_delivery' | 'pickup';

export interface OrderDetail {
  id: string;
  orderNumber: string;
  status: OrderDetailStatus;
  fulfillmentType: FulfillmentType;
  customerName: string;
  customerPhone?: string;
  items: OrderDetailItem[];
  deliveryAddress: OrderDetailAddress | string;
  specialInstructions?: string;
  deliveryInstructions?: string;
  timing: OrderDetailTiming;
  pricing: OrderDetailPricing;
  // Authoritative self-delivery CAPABILITY (the chef's "I deliver myself"
  // toggle). The Mark-Ready carrier choice is gated on THIS — not on the
  // distance fields below, which are 0 when the chef set no radius / coords are
  // missing and so can't distinguish "can't self-deliver" from "no radius set".
  offersSelfDelivery: boolean;
  // Chef self-delivery distance gate (chef_delivery only). distanceKm is the
  // chef→drop straight-line distance; maxDistanceKm is the chef's configured
  // comfort radius. The screen shows a soft "beyond your range" warning when
  // distanceKm > maxDistanceKm > 0. Both 0 for other modes / unknown coords.
  selfDeliveryDistanceKm: number;
  selfDeliveryMaxDistanceKm: number;
}

// ---- Hook --------------------------------------------------------------------

// Backend ships a FLAT ChefOrderDetailResponse (timing + pricing inline at
// the top level, items with `price`/`subtotal` field names). The screen
// consumes the nested shape above. Adapt in the queryFn so the screen
// stays untouched.
interface RawOrderItemResponse {
  id: string;
  name: string;
  quantity: number;
  price: number;
  subtotal: number;
  isVeg?: boolean;
  specialInstructions?: string;
  notes?: string;
  isCancelled?: boolean;
  cancelledReason?: string;
  cancelledAt?: string | null;
  refundAmount?: number;
}

interface RawChefOrderDetailResponse {
  id: string;
  orderNumber: string;
  status: OrderDetailStatus;
  fulfillmentType?: FulfillmentType;
  customerName?: string;
  customerPhone?: string;
  items?: RawOrderItemResponse[];
  deliveryAddress?: OrderDetailAddress | string;
  specialInstructions?: string;
  deliveryInstructions?: string;
  // Flat timing
  createdAt: string;
  acceptedAt?: string | null;
  preparedAt?: string | null;
  pickedUpAt?: string | null;
  deliveredAt?: string | null;
  // Flat pricing
  subtotal?: number;
  deliveryFee?: number;
  serviceFee?: number;
  tax?: number;
  chefTip?: number;
  total?: number;
  // Self-delivery capability + distance gate
  offersSelfDelivery?: boolean;
  selfDeliveryDistanceKm?: number;
  selfDeliveryMaxDistanceKm?: number;
}

function adaptOrderDetail(raw: RawChefOrderDetailResponse): OrderDetail {
  return {
    id: raw.id,
    orderNumber: raw.orderNumber,
    status: raw.status,
    // Legacy orders predate the field — default to delivery (3PL).
    fulfillmentType: raw.fulfillmentType ?? 'delivery',
    customerName: raw.customerName ?? '',
    customerPhone: raw.customerPhone,
    items: (raw.items ?? []).map((i) => ({
      id: i.id,
      name: i.name,
      quantity: i.quantity,
      unitPrice: i.price,
      lineTotal: i.subtotal,
      isVeg: i.isVeg,
      specialInstructions: i.specialInstructions || i.notes,
      isCancelled: i.isCancelled,
      cancelledReason: i.cancelledReason,
      cancelledAt: i.cancelledAt,
      refundAmount: i.refundAmount,
    })),
    deliveryAddress: raw.deliveryAddress ?? '',
    specialInstructions: raw.specialInstructions,
    deliveryInstructions: raw.deliveryInstructions,
    timing: {
      orderedAt: raw.createdAt,
      acceptedAt: raw.acceptedAt ?? null,
      preparedAt: raw.preparedAt ?? null,
      pickedUpAt: raw.pickedUpAt ?? null,
      deliveredAt: raw.deliveredAt ?? null,
    },
    pricing: {
      subtotal: raw.subtotal ?? 0,
      deliveryFee: raw.deliveryFee ?? 0,
      serviceFee: raw.serviceFee ?? 0,
      tax: raw.tax ?? 0,
      chefTip: raw.chefTip ?? 0,
      total: raw.total ?? 0,
    },
    offersSelfDelivery: raw.offersSelfDelivery ?? false,
    selfDeliveryDistanceKm: raw.selfDeliveryDistanceKm ?? 0,
    selfDeliveryMaxDistanceKm: raw.selfDeliveryMaxDistanceKm ?? 0,
  };
}

/**
 * Fetch a single order by ID from the dedicated endpoint.
 * The backend ships GET /chef/orders/:orderId with full detail.
 */
export function useOrderDetail(orderId: string | null | undefined) {
  return useQuery<OrderDetail>({
    queryKey: ['chef', 'orders', 'detail-v2', orderId],
    enabled: !!orderId,
    queryFn: () => {
      if (!orderId) throw new Error('orderId required');
      return api
        .get<RawChefOrderDetailResponse>(`/chef/orders/${orderId}`)
        .then((r) => adaptOrderDetail(r.data));
    },
    staleTime: 15_000,
  });
}
