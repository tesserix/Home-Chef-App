import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Order, OrderItem } from '../types/customer';

export interface OrderListParams {
  page?: number;
  limit?: number;
  status?: string;
}

export interface OrderListResponse {
  data: Order[];
  meta: { total: number; page: number; limit: number };
}

// ─── Wire shapes (apps/api models/order.go OrderResponse / ToResponse) ────────
// The order API does NOT match the customer `Order` type 1:1, so map at this
// hook boundary (same pattern as useChefs / useAddresses). Without the map:
//   • deliveryAddress renders blank (API line1/postalCode vs addressLine1/pincode)
//   • totalAmount is NaN (API field is `total`)
//   • the list's total count is 0 (API key is `pagination`, not `meta`)
// The API carries no chef object/id, so Order.chef stays undefined (the screens
// render it defensively).

interface ApiAddress {
  line1?: string;
  line2?: string;
  city?: string;
  state?: string;
  postalCode?: string;
}

interface ApiOrderItem {
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
}

interface ApiOrderChef {
  id: string;
  name: string;
  imageUrl?: string;
}

interface ApiOrder {
  id: string;
  orderNumber: string;
  status: Order['status'];
  paymentStatus?: Order['paymentStatus'];
  total?: number;
  fulfillmentType?: Order['fulfillmentType'];
  deliveryFee?: number;
  serviceFee?: number;
  tax?: number;
  discount?: number;
  items?: ApiOrderItem[];
  deliveryAddress?: ApiAddress;
  chef?: ApiOrderChef;
  createdAt: string;
}

interface ApiOrderListResponse {
  data?: ApiOrder[];
  pagination?: { total?: number; page?: number; limit?: number };
}

function mapOrderItem(i: ApiOrderItem): OrderItem {
  return {
    menuItemId: i.menuItemId,
    name: i.name,
    price: i.price,
    quantity: i.quantity,
  };
}

function mapOrder(raw: ApiOrder): Order {
  return {
    id: raw.id,
    orderNumber: raw.orderNumber,
    status: raw.status,
    paymentStatus: raw.paymentStatus,
    // Chef is now sent by the order API (OrderChefResponse: id/name/imageUrl).
    // Fill the rest of the Chef shape with neutral defaults — the order
    // list/detail only render name (and optionally image).
    chef: raw.chef
      ? {
          id: raw.chef.id,
          name: raw.chef.name,
          imageUrl: raw.chef.imageUrl,
          cuisine: '',
          rating: 0,
          reviewCount: 0,
          isOpen: false,
        }
      : undefined,
    items: (raw.items ?? []).map(mapOrderItem),
    totalAmount: raw.total ?? 0,
    fulfillmentType: raw.fulfillmentType,
    deliveryFee: raw.deliveryFee,
    serviceFee: raw.serviceFee,
    tax: raw.tax,
    discount: raw.discount,
    deliveryAddress: {
      addressLine1: raw.deliveryAddress?.line1 ?? '',
      addressLine2: raw.deliveryAddress?.line2 || undefined,
      city: raw.deliveryAddress?.city ?? '',
      state: raw.deliveryAddress?.state ?? '',
      pincode: raw.deliveryAddress?.postalCode ?? '',
    },
    createdAt: raw.createdAt,
  };
}

export function useOrders(params: OrderListParams = {}) {
  return useQuery<OrderListResponse>({
    queryKey: ['orders', params],
    queryFn: () =>
      api.get<ApiOrderListResponse>('/v1/orders', { params }).then((r) => {
        const body = r.data ?? {};
        const p = body.pagination ?? {};
        return {
          data: (body.data ?? []).map(mapOrder),
          meta: {
            total: p.total ?? 0,
            page: p.page ?? params.page ?? 1,
            limit: p.limit ?? params.limit ?? 20,
          },
        };
      }),
    staleTime: 1000 * 30, // 30 seconds — orders change frequently
  });
}

// `pollUntilPaid` keeps refetching every 2s while paymentStatus is still
// pending — used by the payment-result screen to pick up the server-side
// confirmation (webhook or verify) even if the client-side verify failed.
export function useOrder(id: string, opts: { pollUntilPaid?: boolean } = {}) {
  return useQuery<{ data: Order }>({
    queryKey: ['order', id],
    queryFn: () =>
      api.get<ApiOrder>(`/v1/orders/${id}`).then((r) => ({ data: mapOrder(r.data) })),
    enabled: !!id,
    // Keep the order live: fast-poll while waiting for payment, then keep the
    // status fresh while the order is in flight (#50 — drives the cooking
    // animation), and stop once it's terminal.
    refetchInterval: (query) => {
      const o = query.state.data?.data;
      if (!o) return false;
      if (opts.pollUntilPaid && o.paymentStatus === 'pending') return 2000;
      const active = ['pending', 'accepted', 'preparing', 'ready', 'picked_up', 'delivering'].includes(
        o.status,
      );
      return active ? 15000 : false;
    },
  });
}
