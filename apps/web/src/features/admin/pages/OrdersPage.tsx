import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useFormatPrice } from '@/shared/utils/format-price';
import {
  Search,
  ShoppingBag,
  Eye,
  Loader2,
  ChevronLeft,
  ChevronRight,
  DollarSign,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import type { Order, OrderStatus, PaginatedResponse } from '@/shared/types';

const STATUS_CONFIG: Record<OrderStatus, { label: string; color: string }> = {
  pending: { label: 'Pending', color: 'bg-amber/20 text-amber' },
  accepted: { label: 'Accepted', color: 'bg-info/20 text-info' },
  preparing: { label: 'Preparing', color: 'bg-info/20 text-info' },
  ready: { label: 'Ready', color: 'bg-info/20 text-info' },
  picked_up: { label: 'Picked Up', color: 'bg-info/20 text-info' },
  delivering: { label: 'Delivering', color: 'bg-herb/20 text-herb' },
  delivered: { label: 'Delivered', color: 'bg-herb/20 text-herb' },
  cancelled: { label: 'Cancelled', color: 'bg-paprika/20 text-paprika' },
  refunded: { label: 'Refunded', color: 'bg-ink-muted/20 text-ink-muted' },
};

export default function AdminOrdersPage() {
  const fp = useFormatPrice();

  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [page, setPage] = useState(1);
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-orders', { status: statusFilter, search: searchQuery, page }],
    queryFn: () =>
      apiClient.get<PaginatedResponse<Order>>('/admin/orders', {
        status: statusFilter !== 'all' ? statusFilter : undefined,
        search: searchQuery || undefined,
        page,
        limit: 20,
      }),
  });

  const orders = data?.data || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-paper">Orders</h1>
          <p className="mt-1 text-ink-muted">{data?.pagination.total || 0} total orders</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-ink-muted" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search order number..."
            className="w-full rounded-lg bg-ink border-ink-soft pl-10 pr-4 py-2.5 text-paper placeholder:text-ink-muted"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded-lg bg-ink border-ink-soft text-paper"
        >
          <option value="all">All Status</option>
          {Object.entries(STATUS_CONFIG).map(([value, { label }]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </div>

      {/* Orders Table */}
      <div className="rounded-xl bg-ink overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-herb" />
          </div>
        ) : orders.length === 0 ? (
          <div className="p-12 text-center">
            <ShoppingBag className="mx-auto h-12 w-12 text-ink-soft" />
            <h3 className="mt-4 font-medium text-paper">No orders found</h3>
            <p className="mt-2 text-ink-muted">Try adjusting your filters</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-ink-soft/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Order
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Customer
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Chef
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Total
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-ink-muted uppercase">
                    Date
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-ink-muted uppercase">
                    Action
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                {orders.map((order) => {
                  const status = STATUS_CONFIG[order.status];
                  return (
                    <tr key={order.id} className="hover:bg-ink-soft/50">
                      <td className="px-6 py-4">
                        <p className="font-medium text-paper">#{order.orderNumber}</p>
                        <p className="text-sm text-ink-muted">{order.items.length} items</p>
                      </td>
                      <td className="px-6 py-4 text-ink-muted">
                        Customer #{order.customerId.slice(0, 8)}
                      </td>
                      <td className="px-6 py-4 text-ink-muted">
                        Chef #{order.chefId.slice(0, 8)}
                      </td>
                      <td className="px-6 py-4">
                        <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${status.color}`}>
                          {status.label}
                        </span>
                      </td>
                      <td className="px-6 py-4 font-medium text-paper">
                        {fp(order.total)}
                      </td>
                      <td className="px-6 py-4 text-sm text-ink-muted">
                        {new Date(order.createdAt).toLocaleDateString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => setSelectedOrder(order)}
                          className="p-2 text-ink-muted hover:text-paper"
                        >
                          <Eye className="h-5 w-5" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {data && data.pagination.totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-ink-soft px-6 py-4">
            <p className="text-sm text-ink-muted">
              Page {page} of {data.pagination.totalPages}
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(page - 1)}
                disabled={!data.pagination.hasPrev}
                className="p-2 rounded-lg bg-ink-soft text-paper disabled:opacity-50"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
              <button
                onClick={() => setPage(page + 1)}
                disabled={!data.pagination.hasNext}
                className="p-2 rounded-lg bg-ink-soft text-paper disabled:opacity-50"
              >
                <ChevronRight className="h-5 w-5" />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Order Detail Modal */}
      {selectedOrder && (
        <OrderDetailModal order={selectedOrder} onClose={() => setSelectedOrder(null)} />
      )}
    </div>
  );
}

function OrderDetailModal({ order, onClose }: { order: Order; onClose: () => void }) {
  const fp = useFormatPrice();
  const status = STATUS_CONFIG[order.status];

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-ink/70 p-4 pt-10">
      <div className="w-full max-w-2xl rounded-xl bg-ink shadow-xl">
        <div className="border-b border-ink-soft p-6">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-xl font-semibold text-paper">Order #{order.orderNumber}</h2>
              <p className="mt-1 text-sm text-ink-muted">
                {new Date(order.createdAt).toLocaleString()}
              </p>
            </div>
            <span className={`rounded-full px-3 py-1 text-sm font-medium ${status.color}`}>
              {status.label}
            </span>
          </div>
        </div>

        <div className="p-6 space-y-6">
          {/* Order Items */}
          <div>
            <h3 className="text-sm font-medium text-ink-muted mb-3">Items</h3>
            <div className="space-y-2">
              {order.items.map((item) => (
                <div key={item.id} className="flex justify-between text-ink-muted">
                  <span>
                    {item.quantity}x {item.name}
                  </span>
                  <span>{fp(item.subtotal)}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Payment Summary */}
          <div className="rounded-lg bg-ink-soft/50 p-4 space-y-2">
            <div className="flex justify-between text-ink-muted">
              <span>Subtotal</span>
              <span>{fp(order.subtotal)}</span>
            </div>
            <div className="flex justify-between text-ink-muted">
              <span>Delivery Fee</span>
              <span>{fp(order.deliveryFee)}</span>
            </div>
            <div className="flex justify-between text-ink-muted">
              <span>Service Fee</span>
              <span>{fp(order.serviceFee)}</span>
            </div>
            <div className="flex justify-between text-ink-muted">
              <span>Tax</span>
              <span>{fp(order.tax)}</span>
            </div>
            {order.discount > 0 && (
              <div className="flex justify-between text-herb">
                <span>Discount</span>
                <span>-{fp(order.discount)}</span>
              </div>
            )}
            <div className="flex justify-between border-t border-ink-soft pt-2 text-lg font-semibold text-paper">
              <span>Total</span>
              <span>{fp(order.total)}</span>
            </div>
          </div>

          {/* Delivery Address */}
          <div>
            <h3 className="text-sm font-medium text-ink-muted mb-2">Delivery Address</h3>
            <p className="text-ink-muted">
              {order.deliveryAddress.line1}
              {order.deliveryAddress.line2 && `, ${order.deliveryAddress.line2}`}
            </p>
            <p className="text-ink-muted">
              {order.deliveryAddress.city}, {order.deliveryAddress.state}{' '}
              {order.deliveryAddress.postalCode}
            </p>
          </div>

          {/* Payment Status */}
          <div className="flex items-center gap-2 text-ink-muted">
            <DollarSign className="h-4 w-4 text-ink-muted" />
            Payment: {order.paymentStatus} via {order.paymentMethod || 'Card'}
          </div>
        </div>

        <div className="border-t border-ink-soft p-4">
          <button onClick={onClose} className="w-full btn-outline border-ink-soft text-paper">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
