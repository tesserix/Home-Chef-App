import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useFormatPrice } from '@/shared/utils/format-price';
import {
  Package,
  CheckCircle,
  XCircle,
  MapPin,
  Calendar,
  Search,
  ChevronRight,
  DollarSign,
  Star,
  Navigation,
  Loader2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';

interface DeliveryOrder {
  id: string;
  orderNumber: string;
  status: 'completed' | 'cancelled';
  pickupAddress: {
    line1: string;
    city: string;
  };
  deliveryAddress: {
    line1: string;
    city: string;
  };
  chefName: string;
  customerName: string;
  distance: number;
  deliveryFee: number;
  tip: number;
  completedAt: string;
  rating?: number;
  feedback?: string;
}

const STATUS_TABS = [
  { value: 'all', label: 'All Deliveries' },
  { value: 'completed', label: 'Completed' },
  { value: 'cancelled', label: 'Cancelled' },
];

export default function DeliveryOrdersPage() {
  const fp = useFormatPrice();
  const [statusFilter, setStatusFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [dateRange, setDateRange] = useState('30d');
  const [selectedOrder, setSelectedOrder] = useState<DeliveryOrder | null>(null);

  const { data: orders, isLoading } = useQuery({
    queryKey: ['delivery-orders', { status: statusFilter, search: searchQuery, dateRange }],
    queryFn: () =>
      apiClient.get<DeliveryOrder[]>('/delivery/orders', {
        status: statusFilter !== 'all' ? statusFilter : undefined,
        search: searchQuery || undefined,
        dateRange,
      }),
  });

  const completedOrders = orders?.filter((o) => o.status === 'completed') || [];
  const totalEarnings = completedOrders.reduce((sum, o) => sum + o.deliveryFee + o.tip, 0);
  const averageRating =
    completedOrders.filter((o) => o.rating).reduce((sum, o) => sum + (o.rating || 0), 0) /
      (completedOrders.filter((o) => o.rating).length || 1) || 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Delivery History</h1>
        <p className="mt-1 text-gray-500">View your past deliveries and earnings</p>
      </div>

      {/* Summary Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl bg-white border border-gray-200 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100">
              <Package className="h-5 w-5 text-blue-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{orders?.length || 0}</p>
              <p className="text-sm text-gray-500">Total Deliveries</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl bg-white border border-gray-200 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100">
              <DollarSign className="h-5 w-5 text-green-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{fp(totalEarnings)}</p>
              <p className="text-sm text-gray-500">Total Earnings</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl bg-white border border-gray-200 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-yellow-100">
              <Star className="h-5 w-5 text-yellow-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{averageRating.toFixed(1)}</p>
              <p className="text-sm text-gray-500">Average Rating</p>
            </div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex gap-2 overflow-x-auto pb-2 sm:pb-0">
          {STATUS_TABS.map((tab) => (
            <button
              key={tab.value}
              onClick={() => setStatusFilter(tab.value)}
              className={`rounded-full px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors ${
                statusFilter === tab.value
                  ? 'bg-brand-500 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="flex gap-3">
          <div className="relative flex-1 sm:flex-initial">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search orders..."
              className="w-full sm:w-64 rounded-lg border-gray-300 pl-9 pr-4 py-2 text-sm"
            />
          </div>
          <select
            value={dateRange}
            onChange={(e) => setDateRange(e.target.value)}
            className="rounded-lg border-gray-300 text-sm"
          >
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
            <option value="90d">Last 90 days</option>
            <option value="all">All time</option>
          </select>
        </div>
      </div>

      {/* Orders List */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
        </div>
      ) : orders && orders.length > 0 ? (
        <div className="space-y-3">
          {orders.map((order) => (
            <div
              key={order.id}
              onClick={() => setSelectedOrder(order)}
              className="rounded-xl bg-white border border-gray-200 p-4 cursor-pointer hover:border-brand-300 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <span className="font-medium text-gray-900">#{order.orderNumber}</span>
                    {order.status === 'completed' ? (
                      <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">
                        <CheckCircle className="h-3 w-3" />
                        Completed
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">
                        <XCircle className="h-3 w-3" />
                        Cancelled
                      </span>
                    )}
                    {order.rating && (
                      <span className="inline-flex items-center gap-1 text-sm text-yellow-600">
                        <Star className="h-4 w-4 fill-yellow-400" />
                        {order.rating}
                      </span>
                    )}
                  </div>

                  <div className="mt-2 flex items-center gap-4 text-sm text-gray-500">
                    <span className="flex items-center gap-1">
                      <Calendar className="h-4 w-4" />
                      {new Date(order.completedAt).toLocaleDateString()}
                    </span>
                    <span className="flex items-center gap-1">
                      <Navigation className="h-4 w-4" />
                      {order.distance} km
                    </span>
                  </div>

                  <div className="mt-3 space-y-1 text-sm">
                    <p className="text-gray-600">
                      <span className="text-gray-400">From:</span> {order.chefName}
                    </p>
                    <p className="text-gray-600">
                      <span className="text-gray-400">To:</span> {order.deliveryAddress.line1},{' '}
                      {order.deliveryAddress.city}
                    </p>
                  </div>
                </div>

                <div className="text-right">
                  <p className="text-lg font-semibold text-gray-900">
                    {fp(order.deliveryFee + order.tip)}
                  </p>
                  {order.tip > 0 && (
                    <p className="text-sm text-green-600">+{fp(order.tip)} tip</p>
                  )}
                  <ChevronRight className="ml-auto mt-2 h-5 w-5 text-gray-400" />
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="rounded-xl bg-gray-50 p-12 text-center">
          <Package className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-4 font-medium text-gray-900">No deliveries found</h3>
          <p className="mt-2 text-gray-500">
            {statusFilter !== 'all'
              ? `No ${statusFilter} deliveries in this period`
              : 'Your delivery history will appear here'}
          </p>
        </div>
      )}

      {/* Order Detail Modal */}
      {selectedOrder && (
        <OrderDetailModal order={selectedOrder} onClose={() => setSelectedOrder(null)} />
      )}
    </div>
  );
}

function OrderDetailModal({
  order,
  onClose,
}: {
  order: DeliveryOrder;
  onClose: () => void;
}) {
  const fp = useFormatPrice();
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-md rounded-xl bg-white shadow-xl">
        <div className="p-6">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-xl font-semibold text-gray-900">Order #{order.orderNumber}</h2>
              <p className="mt-1 text-sm text-gray-500">
                {new Date(order.completedAt).toLocaleString()}
              </p>
            </div>
            {order.status === 'completed' ? (
              <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-3 py-1 text-sm font-medium text-green-700">
                <CheckCircle className="h-4 w-4" />
                Completed
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-3 py-1 text-sm font-medium text-red-700">
                <XCircle className="h-4 w-4" />
                Cancelled
              </span>
            )}
          </div>

          {/* Route */}
          <div className="mt-6 space-y-4">
            <div>
              <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase">
                <div className="h-4 w-4 rounded-full bg-orange-100 flex items-center justify-center">
                  <div className="h-1.5 w-1.5 rounded-full bg-orange-500" />
                </div>
                Pickup
              </div>
              <div className="mt-1 pl-6">
                <p className="font-medium text-gray-900">{order.chefName}</p>
                <p className="text-sm text-gray-600">
                  {order.pickupAddress.line1}, {order.pickupAddress.city}
                </p>
              </div>
            </div>

            <div className="ml-2 h-6 border-l-2 border-dashed border-gray-300" />

            <div>
              <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase">
                <MapPin className="h-4 w-4 text-green-600" />
                Drop-off
              </div>
              <div className="mt-1 pl-6">
                <p className="font-medium text-gray-900">{order.customerName}</p>
                <p className="text-sm text-gray-600">
                  {order.deliveryAddress.line1}, {order.deliveryAddress.city}
                </p>
              </div>
            </div>
          </div>

          {/* Trip Details */}
          <div className="mt-6 rounded-lg bg-gray-50 p-4">
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">Distance</span>
              <span className="font-medium text-gray-900">{order.distance} km</span>
            </div>
          </div>

          {/* Earnings */}
          <div className="mt-4 space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">Delivery Fee</span>
              <span className="text-gray-900">{fp(order.deliveryFee)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">Tip</span>
              <span className="text-green-600">+{fp(order.tip)}</span>
            </div>
            <div className="flex justify-between border-t border-gray-200 pt-2 text-base font-semibold">
              <span className="text-gray-900">Total Earned</span>
              <span className="text-gray-900">{fp(order.deliveryFee + order.tip)}</span>
            </div>
          </div>

          {/* Rating & Feedback */}
          {order.rating && (
            <div className="mt-6 rounded-lg bg-yellow-50 p-4">
              <div className="flex items-center gap-2">
                <div className="flex">
                  {[1, 2, 3, 4, 5].map((star) => (
                    <Star
                      key={star}
                      className={`h-5 w-5 ${
                        star <= order.rating!
                          ? 'fill-yellow-400 text-yellow-400'
                          : 'text-gray-300'
                      }`}
                    />
                  ))}
                </div>
                <span className="font-medium text-gray-900">{order.rating}/5</span>
              </div>
              {order.feedback && (
                <p className="mt-2 text-sm text-gray-600 italic">"{order.feedback}"</p>
              )}
            </div>
          )}
        </div>

        <div className="border-t border-gray-200 p-4">
          <button onClick={onClose} className="w-full btn-outline">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
