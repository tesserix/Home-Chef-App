import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Clock,
  MapPin,
  MessageCircle,
  Star,
  CheckCircle,
  XCircle,
  Truck,
  ChefHat,
  Package,
  Loader2,
  AlertCircle,
  RotateCcw,
  Copy,
  RefreshCw,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useFormatPrice } from '@/shared/utils/format-price';
import type { Order, OrderStatus } from '@/shared/types';

const STATUS_CONFIG: Record<OrderStatus, { label: string; color: string; bgColor: string; icon: typeof Clock }> = {
  pending: { label: 'Pending Confirmation', color: 'text-yellow-800', bgColor: 'bg-yellow-100', icon: Clock },
  accepted: { label: 'Order Confirmed', color: 'text-blue-800', bgColor: 'bg-blue-100', icon: CheckCircle },
  preparing: { label: 'Being Prepared', color: 'text-purple-800', bgColor: 'bg-purple-100', icon: ChefHat },
  ready: { label: 'Ready for Pickup', color: 'text-indigo-800', bgColor: 'bg-indigo-100', icon: Package },
  picked_up: { label: 'Picked Up', color: 'text-cyan-800', bgColor: 'bg-cyan-100', icon: Truck },
  delivering: { label: 'On the Way', color: 'text-orange-800', bgColor: 'bg-orange-100', icon: Truck },
  delivered: { label: 'Delivered', color: 'text-green-800', bgColor: 'bg-green-100', icon: CheckCircle },
  cancelled: { label: 'Cancelled', color: 'text-red-800', bgColor: 'bg-red-100', icon: XCircle },
  refunded: { label: 'Refunded', color: 'text-gray-800', bgColor: 'bg-gray-100', icon: RotateCcw },
};

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const fp = useFormatPrice();
  const queryClient = useQueryClient();
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [cancelReason, setCancelReason] = useState('');

  const { data: order, isLoading, error } = useQuery({
    queryKey: ['order', id],
    queryFn: () => apiClient.get<Order>(`/orders/${id}`),
    enabled: !!id,
    refetchInterval: (query) => {
      // Poll for updates on active orders
      const orderData = query.state.data;
      if (orderData && !['delivered', 'cancelled', 'refunded'].includes(orderData.status)) {
        return 30000; // Every 30 seconds
      }
      return false;
    },
  });

  const cancelMutation = useMutation({
    mutationFn: () =>
      apiClient.post<Order>(`/orders/${id}/cancel`, { reason: cancelReason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      toast.success('Order cancelled successfully');
      setShowCancelModal(false);
    },
    onError: () => {
      toast.error('Failed to cancel order');
    },
  });

  const handleCopyOrderNumber = () => {
    if (order) {
      navigator.clipboard.writeText(order.orderNumber);
      toast.success('Order number copied');
    }
  };

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <AlertCircle className="h-16 w-16 text-gray-400" />
        <h2 className="mt-4 text-xl font-semibold text-gray-900">Order not found</h2>
        <Link to="/orders" className="btn-primary mt-4">
          View All Orders
        </Link>
      </div>
    );
  }

  const status = STATUS_CONFIG[order.status];
  const StatusIcon = status.icon;
  const isActive = !['delivered', 'cancelled', 'refunded'].includes(order.status);
  const canCancel = ['pending', 'accepted'].includes(order.status);

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="container-app max-w-3xl">
        {/* Back Button */}
        <Link
          to="/orders"
          className="inline-flex items-center gap-2 text-gray-600 hover:text-gray-900"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Orders
        </Link>

        {/* Header */}
        <div className="mt-6 rounded-xl bg-white p-6 shadow-sm">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold text-gray-900">
                  Order #{order.orderNumber}
                </h1>
                <button
                  onClick={handleCopyOrderNumber}
                  className="p-1 text-gray-400 hover:text-gray-600"
                >
                  <Copy className="h-4 w-4" />
                </button>
              </div>
              <p className="mt-1 text-sm text-gray-500">
                Placed on{' '}
                {new Date(order.createdAt).toLocaleDateString('en-US', {
                  weekday: 'long',
                  month: 'long',
                  day: 'numeric',
                  year: 'numeric',
                  hour: 'numeric',
                  minute: '2-digit',
                })}
              </p>
            </div>

            <div className={`inline-flex items-center gap-2 rounded-full px-4 py-2 ${status.bgColor}`}>
              <StatusIcon className={`h-5 w-5 ${status.color}`} />
              <span className={`font-medium ${status.color}`}>{status.label}</span>
            </div>
          </div>

          {/* Progress */}
          {isActive && (
            <div className="mt-6">
              <OrderProgress status={order.status} />
              {order.estimatedDeliveryAt && (
                <div className="mt-4 flex items-center gap-2 text-sm text-gray-600">
                  <Clock className="h-4 w-4" />
                  <span>
                    Estimated delivery:{' '}
                    <span className="font-medium">
                      {new Date(order.estimatedDeliveryAt).toLocaleTimeString('en-US', {
                        hour: 'numeric',
                        minute: '2-digit',
                      })}
                    </span>
                  </span>
                </div>
              )}
            </div>
          )}

          {/* Cancelled Info */}
          {order.status === 'cancelled' && order.cancelReason && (
            <div className="mt-4 rounded-lg bg-red-50 p-4">
              <p className="text-sm text-red-800">
                <span className="font-medium">Cancellation reason:</span> {order.cancelReason}
              </p>
            </div>
          )}
        </div>

        {/* Order Items */}
        <div className="mt-6 rounded-xl bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900">Order Items</h2>

          <div className="mt-4 divide-y">
            {order.items.map((item) => (
              <div key={item.id} className="flex gap-4 py-4">
                {item.imageUrl && (
                  <img
                    src={item.imageUrl}
                    alt={item.name}
                    className="h-16 w-16 rounded-lg object-cover"
                  />
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <h4 className="font-medium text-gray-900">{item.name}</h4>
                      <p className="text-sm text-gray-500">Qty: {item.quantity}</p>
                      {item.notes && (
                        <p className="mt-1 text-sm text-gray-500 italic">Note: {item.notes}</p>
                      )}
                    </div>
                    <span className="font-medium text-gray-900">
                      {fp(item.subtotal)}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Delivery Address */}
        <div className="mt-6 rounded-xl bg-white p-6 shadow-sm">
          <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900">
            <MapPin className="h-5 w-5 text-brand-500" />
            Delivery Address
          </h2>
          <div className="mt-4">
            <p className="font-medium text-gray-900">{order.deliveryAddress.label}</p>
            <p className="mt-1 text-gray-600">
              {order.deliveryAddress.line1}
              {order.deliveryAddress.line2 && `, ${order.deliveryAddress.line2}`}
            </p>
            <p className="text-gray-600">
              {order.deliveryAddress.city}, {order.deliveryAddress.state}{' '}
              {order.deliveryAddress.postalCode}
            </p>
            {order.deliveryAddress.deliveryInstructions && (
              <p className="mt-2 text-sm text-gray-500 italic">
                {order.deliveryAddress.deliveryInstructions}
              </p>
            )}
          </div>
        </div>

        {/* Payment Summary */}
        <div className="mt-6 rounded-xl bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900">Payment Summary</h2>

          <div className="mt-4 space-y-3">
            <div className="flex justify-between text-gray-600">
              <span>Subtotal</span>
              <span>{fp(order.subtotal)}</span>
            </div>
            <div className="flex justify-between text-gray-600">
              <span>Delivery fee</span>
              <span>{fp(order.deliveryFee)}</span>
            </div>
            <div className="flex justify-between text-gray-600">
              <span>Service fee</span>
              <span>{fp(order.serviceFee)}</span>
            </div>
            <div className="flex justify-between text-gray-600">
              <span>Tax</span>
              <span>{fp(order.tax)}</span>
            </div>
            {order.discount > 0 && (
              <div className="flex justify-between text-green-600">
                <span>Discount</span>
                <span>-{fp(order.discount)}</span>
              </div>
            )}
            {order.tip > 0 && (
              <div className="flex justify-between text-gray-600">
                <span>Tip</span>
                <span>{fp(order.tip)}</span>
              </div>
            )}
          </div>

          <div className="mt-4 flex justify-between border-t pt-4 text-lg font-semibold">
            <span>Total</span>
            <span>{fp(order.total)}</span>
          </div>

          <div className="mt-4 flex items-center gap-2 text-sm text-gray-500">
            <CheckCircle className="h-4 w-4 text-green-500" />
            <span>
              Paid via {order.paymentMethod || 'Card'} • {order.paymentStatus}
            </span>
          </div>
        </div>

        {/* Special Instructions */}
        {order.specialInstructions && (
          <div className="mt-6 rounded-xl bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-gray-900">Special Instructions</h2>
            <p className="mt-2 text-gray-600">{order.specialInstructions}</p>
          </div>
        )}

        {/* Actions */}
        <div className="mt-6 flex flex-wrap gap-4">
          {order.status === 'delivered' && (
            <Link to={`/orders/${order.id}/review`} className="btn-primary">
              <Star className="h-4 w-4" />
              Leave a Review
            </Link>
          )}

          {isActive && (
            <>
              <button className="btn-outline">
                <MessageCircle className="h-4 w-4" />
                Contact Support
              </button>

              {canCancel && (
                <button
                  onClick={() => setShowCancelModal(true)}
                  className="btn-outline border-red-300 text-red-600 hover:bg-red-50"
                >
                  <XCircle className="h-4 w-4" />
                  Cancel Order
                </button>
              )}
            </>
          )}

          <Link to="/chefs" className="btn-outline">
            <RefreshCw className="h-4 w-4" />
            Reorder
          </Link>
        </div>

        {/* Cancel Modal */}
        {showCancelModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
            <div className="w-full max-w-md rounded-xl bg-white p-6">
              <h3 className="text-lg font-semibold text-gray-900">Cancel Order</h3>
              <p className="mt-2 text-gray-600">
                Are you sure you want to cancel this order? This action cannot be undone.
              </p>

              <div className="mt-4">
                <label className="block text-sm font-medium text-gray-700">
                  Reason for cancellation
                </label>
                <textarea
                  value={cancelReason}
                  onChange={(e) => setCancelReason(e.target.value)}
                  placeholder="Please let us know why you're cancelling..."
                  rows={3}
                  className="input-base mt-1"
                />
              </div>

              <div className="mt-6 flex justify-end gap-3">
                <button
                  onClick={() => setShowCancelModal(false)}
                  className="btn-outline"
                >
                  Keep Order
                </button>
                <button
                  onClick={() => cancelMutation.mutate()}
                  disabled={cancelMutation.isPending}
                  className="btn-base bg-red-600 text-white hover:bg-red-700"
                >
                  {cancelMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    'Cancel Order'
                  )}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function OrderProgress({ status }: { status: OrderStatus }) {
  const steps = [
    { key: 'accepted', label: 'Confirmed', icon: CheckCircle },
    { key: 'preparing', label: 'Preparing', icon: ChefHat },
    { key: 'ready', label: 'Ready', icon: Package },
    { key: 'delivering', label: 'On the Way', icon: Truck },
    { key: 'delivered', label: 'Delivered', icon: CheckCircle },
  ];

  const statusOrder = ['pending', 'accepted', 'preparing', 'ready', 'picked_up', 'delivering', 'delivered'];
  const currentIndex = statusOrder.indexOf(status);

  return (
    <div className="relative">
      {/* Progress Line */}
      <div className="absolute left-4 top-4 h-[calc(100%-32px)] w-0.5 bg-gray-200" />
      <div
        className="absolute left-4 top-4 w-0.5 bg-brand-500 transition-all"
        style={{
          height: `${Math.min((currentIndex / (steps.length)) * 100, 100)}%`,
        }}
      />

      {/* Steps */}
      <div className="space-y-6">
        {steps.map((step) => {
          const stepIndex = statusOrder.indexOf(step.key);
          const isCompleted = currentIndex >= stepIndex;
          const isCurrent = status === step.key || (status === 'picked_up' && step.key === 'ready');
          const StepIcon = step.icon;

          return (
            <div key={step.key} className="flex items-center gap-4">
              <div
                className={`relative z-10 flex h-8 w-8 items-center justify-center rounded-full ${
                  isCompleted
                    ? 'bg-brand-500 text-white'
                    : 'bg-gray-200 text-gray-400'
                } ${isCurrent ? 'ring-4 ring-brand-100' : ''}`}
              >
                <StepIcon className="h-4 w-4" />
              </div>
              <span
                className={`font-medium ${
                  isCompleted ? 'text-gray-900' : 'text-gray-400'
                }`}
              >
                {step.label}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
