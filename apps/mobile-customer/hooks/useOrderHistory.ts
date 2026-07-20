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
  id?: string;
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
}

interface ApiOrderChef {
  id: string;
  name: string;
  businessName?: string;
  ownerName?: string;
  imageUrl?: string;
  fssaiLicenseNumber?: string;
  gstin?: string;
  state?: string;
}

interface ApiOrder {
  id: string;
  orderNumber: string;
  status: Order['status'];
  paymentStatus?: Order['paymentStatus'];
  total?: number;
  fulfillmentType?: Order['fulfillmentType'];
  deliveryFee?: number;
  deliveryFeeFinal?: number;
  serviceFee?: number;
  tax?: number;
  taxRate?: number;
  discount?: number;
  readyPhotoUrl?: string;
  items?: ApiOrderItem[];
  deliveryAddress?: ApiAddress;
  chef?: ApiOrderChef;
  requestedFulfillmentAt?: string;
  confirmedFulfillmentAt?: string;
  fulfillmentTimeStatus?: Order['fulfillmentTimeStatus'];
  createdAt: string;
  payoutHoldStatus?: Order['payoutHoldStatus'];
  customerConfirmedAt?: string;
  cancelReason?: string;
  refundAmount?: number;
  refundedAt?: string;
}

interface ApiOrderListResponse {
  data?: ApiOrder[];
  pagination?: { total?: number; page?: number; limit?: number };
}

function mapOrderItem(i: ApiOrderItem): OrderItem {
  return {
    // The API sends a per-line `id`; carrying it through is what makes the
    // report-issue "which items?" checkboxes selectable (they're disabled
    // without an id).
    id: i.id,
    menuItemId: i.menuItemId,
    name: i.name,
    price: i.price,
    quantity: i.quantity,
  };
}

/**
 * Fetch a short-lived signed URL for the order's receipt PDF (#receipt parity).
 * The mobile app can't save an authenticated blob (no file-system module), so it
 * opens this self-authenticating URL in the in-app browser instead — where iOS
 * gives the customer save / share / print for free.
 */
export async function fetchInvoiceDownloadUrl(orderId: string): Promise<string> {
  const res = await api.get(`/v1/orders/${orderId}/invoice-link`);
  return (res.data as { url: string }).url;
}

function mapOrder(raw: ApiOrder): Order {
  return {
    id: raw.id,
    orderNumber: raw.orderNumber,
    status: raw.status,
    paymentStatus: raw.paymentStatus,
    cancelReason: raw.cancelReason,
    refundAmount: raw.refundAmount,
    refundedAt: raw.refundedAt,
    // Chef is now sent by the order API (OrderChefResponse: id/name/imageUrl).
    // Fill the rest of the Chef shape with neutral defaults — the order
    // list/detail only render name (and optionally image).
    chef: raw.chef
      ? {
          id: raw.chef.id,
          name: raw.chef.name,
          businessName: raw.chef.businessName,
          ownerName: raw.chef.ownerName,
          fssaiLicenseNumber: raw.chef.fssaiLicenseNumber,
          gstin: raw.chef.gstin,
          state: raw.chef.state,
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
    requestedFulfillmentAt: raw.requestedFulfillmentAt,
    confirmedFulfillmentAt: raw.confirmedFulfillmentAt,
    fulfillmentTimeStatus: raw.fulfillmentTimeStatus,
    deliveryFee: raw.deliveryFee,
    deliveryFeeFinal: raw.deliveryFeeFinal,
    serviceFee: raw.serviceFee,
    tax: raw.tax,
    taxRate: raw.taxRate,
    discount: raw.discount,
    readyPhotoUrl: raw.readyPhotoUrl,
    payoutHoldStatus: raw.payoutHoldStatus,
    customerConfirmedAt: raw.customerConfirmedAt,
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

export function useOrders(
  params: OrderListParams = {},
  opts: { refetchInterval?: number } = {},
) {
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
    // Opt-in live polling (the home active-order card) so a chef-driven status
    // change shows up without the customer leaving the screen. Off by default
    // for the paginated history list.
    refetchInterval: opts.refetchInterval ?? false,
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
      // Fallback cadence: the notification WebSocket (useOrderStatusWS) flips the
      // status within ~1s; this poll only covers a dropped socket, so 5s is a
      // safe floor without being real-time-dependent.
      return active ? 5000 : false;
    },
  });
}
