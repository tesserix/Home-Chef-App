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
        <h1 className="font-display text-2xl font-semibold text-ink">Delivery History</h1>
        <p className="mt-1 text-ink-muted">View your past deliveries and earnings</p>
      </div>

      {/* Summary Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl bg-bone border border-mist p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info/10">
              <Package className="h-5 w-5 text-info" />
            </div>
            <div>
              <p className="font-display text-2xl font-semibold text-ink">{orders?.length || 0}</p>
              <p className="text-sm text-ink-muted">Total Deliveries</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl bg-bone border border-mist p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-herb-tint">
              <DollarSign className="h-5 w-5 text-herb" />
            </div>
            <div>
              <p className="font-display text-2xl font-semibold text-ink">{fp(totalEarnings)}</p>
              <p className="text-sm text-ink-muted">Total Earnings</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl bg-bone border border-mist p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-tint">
              <Star className="h-5 w-5 text-amber" />
            </div>
            <div>
              <p className="font-display text-2xl font-semibold text-ink">{averageRating.toFixed(1)}</p>
              <p className="text-sm text-ink-muted">Average Rating</p>
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
                  ? 'bg-herb text-paper'
                  : 'bg-mist text-ink-soft hover:bg-mist'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="flex gap-3">
          <div className="relative flex-1 sm:flex-initial">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-muted" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search orders..."
              className="w-full sm:w-64 rounded-lg border-mist-strong pl-9 pr-4 py-2 text-sm"
            />
          </div>
          <select
            value={dateRange}
            onChange={(e) => setDateRange(e.target.value)}
            className="rounded-lg border-mist-strong text-sm"
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
          <Loader2 className="h-8 w-8 animate-spin text-herb" />
        </div>
      ) : orders && orders.length > 0 ? (
        <div className="space-y-3">
          {orders.map((order) => (
            <div
              key={order.id}
              onClick={() => setSelectedOrder(order)}
              className="rounded-xl bg-bone border border-mist p-4 cursor-pointer hover:border-herb-tint transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <span className="font-medium text-ink">#{order.orderNumber}</span>
                    {order.status === 'completed' ? (
                      <span className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-2 py-0.5 text-xs font-medium text-herb">
                        <CheckCircle className="h-3 w-3" />
                        Completed
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 rounded-full bg-paprika-tint px-2 py-0.5 text-xs font-medium text-paprika">
                        <XCircle className="h-3 w-3" />
                        Cancelled
                      </span>
                    )}
                    {order.rating && (
                      <span className="inline-flex items-center gap-1 text-sm text-amber">
                        <Star className="h-4 w-4 fill-amber" />
                        {order.rating}
                      </span>
                    )}
                  </div>

                  <div className="mt-2 flex items-center gap-4 text-sm text-ink-muted">
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
                    <p className="text-ink-soft">
                      <span className="text-ink-muted">From:</span> {order.chefName}
                    </p>
                    <p className="text-ink-soft">
                      <span className="text-ink-muted">To:</span> {order.deliveryAddress.line1},{' '}
                      {order.deliveryAddress.city}
                    </p>
                  </div>
                </div>

                <div className="text-right">
                  <p className="text-lg font-semibold text-ink">
                    {fp(order.deliveryFee + order.tip)}
                  </p>
                  {order.tip > 0 && (
                    <p className="text-sm text-herb">+{fp(order.tip)} tip</p>
                  )}
                  <ChevronRight className="ml-auto mt-2 h-5 w-5 text-ink-muted" />
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="rounded-xl bg-paper p-12 text-center">
          <Package className="mx-auto h-12 w-12 text-ink-muted" />
          <h3 className="mt-4 font-medium text-ink">No deliveries found</h3>
          <p className="mt-2 text-ink-muted">
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/50 p-4">
      <div className="w-full max-w-md rounded-xl bg-bone shadow-xl">
        <div className="p-6">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-xl font-semibold text-ink">Order #{order.orderNumber}</h2>
              <p className="mt-1 text-sm text-ink-muted">
                {new Date(order.completedAt).toLocaleString()}
              </p>
            </div>
            {order.status === 'completed' ? (
              <span className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-3 py-1 text-sm font-medium text-herb">
                <CheckCircle className="h-4 w-4" />
                Completed
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 rounded-full bg-paprika-tint px-3 py-1 text-sm font-medium text-paprika">
                <XCircle className="h-4 w-4" />
                Cancelled
              </span>
            )}
          </div>

          {/* Route */}
          <div className="mt-6 space-y-4">
            <div>
              <div className="flex items-center gap-2 text-xs font-medium text-ink-muted uppercase">
                <div className="h-4 w-4 rounded-full bg-herb-tint flex items-center justify-center">
                  <div className="h-1.5 w-1.5 rounded-full bg-herb" />
                </div>
                Pickup
              </div>
              <div className="mt-1 pl-6">
                <p className="font-medium text-ink">{order.chefName}</p>
                <p className="text-sm text-ink-soft">
                  {order.pickupAddress.line1}, {order.pickupAddress.city}
                </p>
              </div>
            </div>

            <div className="ml-2 h-6 border-l-2 border-dashed border-mist-strong" />

            <div>
              <div className="flex items-center gap-2 text-xs font-medium text-ink-muted uppercase">
                <MapPin className="h-4 w-4 text-herb" />
                Drop-off
              </div>
              <div className="mt-1 pl-6">
                <p className="font-medium text-ink">{order.customerName}</p>
                <p className="text-sm text-ink-soft">
                  {order.deliveryAddress.line1}, {order.deliveryAddress.city}
                </p>
              </div>
            </div>
          </div>

          {/* Trip Details */}
          <div className="mt-6 rounded-lg bg-paper p-4">
            <div className="flex justify-between text-sm">
              <span className="text-ink-muted">Distance</span>
              <span className="font-medium text-ink">{order.distance} km</span>
            </div>
          </div>

          {/* Earnings */}
          <div className="mt-4 space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-ink-muted">Delivery Fee</span>
              <span className="text-ink">{fp(order.deliveryFee)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-ink-muted">Tip</span>
              <span className="text-herb">+{fp(order.tip)}</span>
            </div>
            <div className="flex justify-between border-t border-mist pt-2 text-base font-semibold">
              <span className="text-ink">Total Earned</span>
              <span className="text-ink">{fp(order.deliveryFee + order.tip)}</span>
            </div>
          </div>

          {/* Rating & Feedback */}
          {order.rating && (
            <div className="mt-6 rounded-lg bg-amber-tint p-4">
              <div className="flex items-center gap-2">
                <div className="flex">
                  {[1, 2, 3, 4, 5].map((star) => (
                    <Star
                      key={star}
                      className={`h-5 w-5 ${
                        star <= order.rating!
                          ? 'fill-amber text-amber'
                          : 'text-ink-muted'
                      }`}
                    />
                  ))}
                </div>
                <span className="font-medium text-ink">{order.rating}/5</span>
              </div>
              {order.feedback && (
                <p className="mt-2 text-sm text-ink-soft italic">"{order.feedback}"</p>
              )}
            </div>
          )}
        </div>

        <div className="border-t border-mist p-4">
          <button onClick={onClose} className="w-full btn-outline">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
