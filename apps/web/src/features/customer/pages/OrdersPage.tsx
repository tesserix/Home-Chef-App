import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Search,
  Clock,
  CheckCircle,
  XCircle,
  Truck,
  ChefHat,
  Package,
  Loader2,
  ChevronRight,
  RotateCcw,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { useFormatPrice } from '@/shared/utils/format-price';
import type { Order, PaginatedResponse, OrderStatus } from '@/shared/types';

// Status palette: amber = waiting, info = in transit, herb = success/active, paprika = failure.
const STATUS_CONFIG: Record<OrderStatus, { label: string; color: string; icon: typeof Clock }> = {
  pending: { label: 'Pending', color: 'bg-amber-tint text-amber', icon: Clock },
  accepted: { label: 'Accepted', color: 'bg-info/10 text-info', icon: CheckCircle },
  preparing: { label: 'Preparing', color: 'bg-info/10 text-info', icon: ChefHat },
  ready: { label: 'Ready', color: 'bg-herb-tint text-herb', icon: Package },
  picked_up: { label: 'Picked Up', color: 'bg-info/10 text-info', icon: Truck },
  delivering: { label: 'On the Way', color: 'bg-info/10 text-info', icon: Truck },
  delivered: { label: 'Delivered', color: 'bg-herb-tint text-herb', icon: CheckCircle },
  cancelled: { label: 'Cancelled', color: 'bg-paprika-tint text-paprika', icon: XCircle },
  refunded: { label: 'Refunded', color: 'bg-mist text-ink-soft', icon: RotateCcw },
};

const FILTER_OPTIONS = [
  { value: 'all', label: 'All Orders' },
  { value: 'active', label: 'Active' },
  { value: 'completed', label: 'Completed' },
  { value: 'cancelled', label: 'Cancelled' },
];

export default function OrdersPage() {
  const [filter, setFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['orders', filter],
    queryFn: () => {
      const statusFilter = filter === 'all'
        ? undefined
        : filter === 'active'
          ? ['pending', 'accepted', 'preparing', 'ready', 'picked_up', 'delivering']
          : filter === 'completed'
            ? ['delivered']
            : ['cancelled', 'refunded'];

      return apiClient.get<PaginatedResponse<Order>>('/orders', {
        status: statusFilter?.join(','),
      });
    },
  });

  const orders = data?.data || [];
  const filteredOrders = searchQuery
    ? orders.filter(
        (order) =>
          order.orderNumber.toLowerCase().includes(searchQuery.toLowerCase()) ||
          order.items.some((item) => item.name.toLowerCase().includes(searchQuery.toLowerCase()))
      )
    : orders;

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-4xl">
        <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">My Orders</h1>

        {/* Filters */}
        <div className="mt-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex gap-2 overflow-x-auto">
            {FILTER_OPTIONS.map((option) => (
              <button
                key={option.value}
                onClick={() => setFilter(option.value)}
                className={`rounded-full px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors ${
                  filter === option.value
                    ? 'bg-herb text-paper'
                    : 'bg-bone text-ink-soft hover:bg-mist'
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>

          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-ink-muted" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search orders..."
              className="input-base pl-10 w-full sm:w-64"
            />
          </div>
        </div>

        {/* Orders List */}
        <div className="mt-8 space-y-4">
          {filteredOrders.length === 0 ? (
            <div className="rounded-xl bg-bone p-12 text-center shadow-sm">
              <div className="mx-auto h-16 w-16 rounded-full bg-mist flex items-center justify-center">
                <Package className="h-8 w-8 text-ink-muted" />
              </div>
              <h3 className="mt-4 text-lg font-semibold text-ink">No orders found</h3>
              <p className="mt-2 text-ink-soft">
                {filter === 'all'
                  ? "You haven't placed any orders yet."
                  : `No ${filter} orders found.`}
              </p>
              <Link to="/chefs" className="btn-primary mt-6 inline-flex">
                Browse Chefs
              </Link>
            </div>
          ) : (
            filteredOrders.map((order) => <OrderCard key={order.id} order={order} />)
          )}
        </div>
      </div>
    </div>
  );
}

function OrderCard({ order }: { order: Order }) {
  const fp = useFormatPrice();
  const status = STATUS_CONFIG[order.status];
  const StatusIcon = status.icon;
  const isActive = !['delivered', 'cancelled', 'refunded'].includes(order.status);

  return (
    <Link
      to={`/orders/${order.id}`}
      className="block rounded-xl bg-bone shadow-sm hover:shadow-md transition-shadow"
    >
      <div className="p-4 sm:p-6">
        {/* Header */}
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <span className="font-semibold text-ink">Order #{order.orderNumber}</span>
              <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${status.color}`}>
                <StatusIcon className="h-3 w-3" />
                {status.label}
              </span>
            </div>
            <p className="mt-1 text-sm text-ink-muted">
              {new Date(order.createdAt).toLocaleDateString('en-US', {
                weekday: 'short',
                month: 'short',
                day: 'numeric',
                hour: 'numeric',
                minute: '2-digit',
              })}
            </p>
          </div>

          <div className="text-right">
            <p className="font-semibold text-ink">{fp(order.total)}</p>
            <p className="text-sm text-ink-muted">{order.items.length} item(s)</p>
          </div>
        </div>

        {/* Items Preview */}
        <div className="mt-4 flex items-center gap-3">
          {order.items.slice(0, 3).map((item, index) => (
            <div
              key={item.id}
              className="relative h-12 w-12 rounded-lg bg-mist overflow-hidden"
            >
              {item.imageUrl ? (
                <img
                  src={item.imageUrl}
                  alt={item.name}
                  className="h-full w-full object-cover"
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center text-xs text-ink-muted">
                  {item.name.charAt(0)}
                </div>
              )}
              {index === 2 && order.items.length > 3 && (
                <div className="absolute inset-0 flex items-center justify-center bg-ink/50 text-xs font-medium text-paper">
                  +{order.items.length - 3}
                </div>
              )}
            </div>
          ))}
          <div className="flex-1 min-w-0">
            <p className="text-sm text-ink-soft truncate">
              {order.items.map((item) => item.name).join(', ')}
            </p>
          </div>
        </div>

        {/* Progress for active orders */}
        {isActive && (
          <div className="mt-4 pt-4 border-t">
            <OrderProgress status={order.status} />
          </div>
        )}

        {/* Footer */}
        <div className="mt-4 flex items-center justify-between pt-4 border-t">
          <div className="flex items-center gap-2 text-sm text-ink-soft">
            {order.estimatedDeliveryAt && isActive && (
              <>
                <Clock className="h-4 w-4" />
                <span>
                  Est. delivery:{' '}
                  {new Date(order.estimatedDeliveryAt).toLocaleTimeString('en-US', {
                    hour: 'numeric',
                    minute: '2-digit',
                  })}
                </span>
              </>
            )}
          </div>
          <div className="flex items-center gap-1 text-herb">
            <span className="text-sm font-medium">View Details</span>
            <ChevronRight className="h-4 w-4" />
          </div>
        </div>
      </div>
    </Link>
  );
}

function OrderProgress({ status }: { status: OrderStatus }) {
  const steps = [
    { key: 'accepted', label: 'Confirmed' },
    { key: 'preparing', label: 'Preparing' },
    { key: 'ready', label: 'Ready' },
    { key: 'delivering', label: 'On the Way' },
    { key: 'delivered', label: 'Delivered' },
  ];

  const statusOrder = ['pending', 'accepted', 'preparing', 'ready', 'picked_up', 'delivering', 'delivered'];
  const currentIndex = statusOrder.indexOf(status);

  return (
    <div className="flex items-center gap-2">
      {steps.map((step, index) => {
        const stepIndex = statusOrder.indexOf(step.key);
        const isCompleted = currentIndex >= stepIndex;
        const isCurrent = status === step.key || (status === 'picked_up' && step.key === 'ready');

        return (
          <div key={step.key} className="flex flex-1 items-center">
            <div className="flex flex-col items-center flex-1">
              <div
                className={`h-2 w-2 rounded-full ${
                  isCompleted ? 'bg-herb' : 'bg-mist-strong'
                } ${isCurrent ? 'ring-4 ring-herb/30' : ''}`}
              />
              <span className={`mt-1 text-xs ${isCompleted ? 'text-herb' : 'text-ink-muted'}`}>
                {step.label}
              </span>
            </div>
            {index < steps.length - 1 && (
              <div
                className={`h-0.5 flex-1 ${
                  currentIndex > stepIndex ? 'bg-herb' : 'bg-mist-strong'
                }`}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}
