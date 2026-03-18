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
  pending: { label: 'Pending', color: 'bg-yellow-500/20 text-yellow-400' },
  accepted: { label: 'Accepted', color: 'bg-blue-500/20 text-blue-400' },
  preparing: { label: 'Preparing', color: 'bg-purple-500/20 text-purple-400' },
  ready: { label: 'Ready', color: 'bg-indigo-500/20 text-indigo-400' },
  picked_up: { label: 'Picked Up', color: 'bg-cyan-500/20 text-cyan-400' },
  delivering: { label: 'Delivering', color: 'bg-orange-500/20 text-orange-400' },
  delivered: { label: 'Delivered', color: 'bg-green-500/20 text-green-400' },
  cancelled: { label: 'Cancelled', color: 'bg-red-500/20 text-red-400' },
  refunded: { label: 'Refunded', color: 'bg-gray-500/20 text-gray-400' },
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
          <h1 className="text-2xl font-bold text-white">Orders</h1>
          <p className="mt-1 text-gray-400">{data?.pagination.total || 0} total orders</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search order number..."
            className="w-full rounded-lg bg-gray-800 border-gray-700 pl-10 pr-4 py-2.5 text-white placeholder:text-gray-500"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded-lg bg-gray-800 border-gray-700 text-white"
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
      <div className="rounded-xl bg-gray-800 overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
          </div>
        ) : orders.length === 0 ? (
          <div className="p-12 text-center">
            <ShoppingBag className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 font-medium text-white">No orders found</h3>
            <p className="mt-2 text-gray-400">Try adjusting your filters</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-700/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Order
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Customer
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Chef
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Total
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">
                    Date
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase">
                    Action
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                {orders.map((order) => {
                  const status = STATUS_CONFIG[order.status];
                  return (
                    <tr key={order.id} className="hover:bg-gray-700/50">
                      <td className="px-6 py-4">
                        <p className="font-medium text-white">#{order.orderNumber}</p>
                        <p className="text-sm text-gray-400">{order.items.length} items</p>
                      </td>
                      <td className="px-6 py-4 text-gray-300">
                        Customer #{order.customerId.slice(0, 8)}
                      </td>
                      <td className="px-6 py-4 text-gray-300">
                        Chef #{order.chefId.slice(0, 8)}
                      </td>
                      <td className="px-6 py-4">
                        <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${status.color}`}>
                          {status.label}
                        </span>
                      </td>
                      <td className="px-6 py-4 font-medium text-white">
                        {fp(order.total)}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-400">
                        {new Date(order.createdAt).toLocaleDateString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <button
                          onClick={() => setSelectedOrder(order)}
                          className="p-2 text-gray-400 hover:text-white"
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
          <div className="flex items-center justify-between border-t border-gray-700 px-6 py-4">
            <p className="text-sm text-gray-400">
              Page {page} of {data.pagination.totalPages}
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(page - 1)}
                disabled={!data.pagination.hasPrev}
                className="p-2 rounded-lg bg-gray-700 text-white disabled:opacity-50"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
              <button
                onClick={() => setPage(page + 1)}
                disabled={!data.pagination.hasNext}
                className="p-2 rounded-lg bg-gray-700 text-white disabled:opacity-50"
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
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/70 p-4 pt-10">
      <div className="w-full max-w-2xl rounded-xl bg-gray-800 shadow-xl">
        <div className="border-b border-gray-700 p-6">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-xl font-semibold text-white">Order #{order.orderNumber}</h2>
              <p className="mt-1 text-sm text-gray-400">
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
            <h3 className="text-sm font-medium text-gray-400 mb-3">Items</h3>
            <div className="space-y-2">
              {order.items.map((item) => (
                <div key={item.id} className="flex justify-between text-gray-300">
                  <span>
                    {item.quantity}x {item.name}
                  </span>
                  <span>{fp(item.subtotal)}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Payment Summary */}
          <div className="rounded-lg bg-gray-700/50 p-4 space-y-2">
            <div className="flex justify-between text-gray-400">
              <span>Subtotal</span>
              <span>{fp(order.subtotal)}</span>
            </div>
            <div className="flex justify-between text-gray-400">
              <span>Delivery Fee</span>
              <span>{fp(order.deliveryFee)}</span>
            </div>
            <div className="flex justify-between text-gray-400">
              <span>Service Fee</span>
              <span>{fp(order.serviceFee)}</span>
            </div>
            <div className="flex justify-between text-gray-400">
              <span>Tax</span>
              <span>{fp(order.tax)}</span>
            </div>
            {order.discount > 0 && (
              <div className="flex justify-between text-green-400">
                <span>Discount</span>
                <span>-{fp(order.discount)}</span>
              </div>
            )}
            <div className="flex justify-between border-t border-gray-600 pt-2 text-lg font-semibold text-white">
              <span>Total</span>
              <span>{fp(order.total)}</span>
            </div>
          </div>

          {/* Delivery Address */}
          <div>
            <h3 className="text-sm font-medium text-gray-400 mb-2">Delivery Address</h3>
            <p className="text-gray-300">
              {order.deliveryAddress.line1}
              {order.deliveryAddress.line2 && `, ${order.deliveryAddress.line2}`}
            </p>
            <p className="text-gray-300">
              {order.deliveryAddress.city}, {order.deliveryAddress.state}{' '}
              {order.deliveryAddress.postalCode}
            </p>
          </div>

          {/* Payment Status */}
          <div className="flex items-center gap-2 text-gray-300">
            <DollarSign className="h-4 w-4 text-gray-500" />
            Payment: {order.paymentStatus} via {order.paymentMethod || 'Card'}
          </div>
        </div>

        <div className="border-t border-gray-700 p-4">
          <button onClick={onClose} className="w-full btn-outline border-gray-600 text-white">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
