import { useEffect, useState } from 'react';
import { useParams, Link, useSearchParams } from 'react-router-dom';
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
  FileText,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useFormatPrice } from '@/shared/utils/format-price';
import { formatDateTime, formatTime } from '@/shared/utils/format-date';
import { Button } from '@/shared/components/ui';
import type { Order, OrderStatus } from '@/shared/types';

// Status palette: amber = waiting, info = in transit, herb = success/active, paprika = failure.
const STATUS_CONFIG: Record<OrderStatus, { label: string; color: string; bgColor: string; icon: typeof Clock }> = {
  pending: { label: 'Pending Confirmation', color: 'text-amber', bgColor: 'bg-amber-tint', icon: Clock },
  accepted: { label: 'Order Confirmed', color: 'text-info', bgColor: 'bg-info/10', icon: CheckCircle },
  preparing: { label: 'Being Prepared', color: 'text-info', bgColor: 'bg-info/10', icon: ChefHat },
  ready: { label: 'Ready for Pickup', color: 'text-herb', bgColor: 'bg-herb-tint', icon: Package },
  picked_up: { label: 'Picked Up', color: 'text-info', bgColor: 'bg-info/10', icon: Truck },
  delivering: { label: 'On the Way', color: 'text-info', bgColor: 'bg-info/10', icon: Truck },
  delivered: { label: 'Delivered', color: 'text-herb', bgColor: 'bg-herb-tint', icon: CheckCircle },
  cancelled: { label: 'Cancelled', color: 'text-paprika', bgColor: 'bg-paprika-tint', icon: XCircle },
  refunded: { label: 'Refunded', color: 'text-ink-soft', bgColor: 'bg-mist', icon: RotateCcw },
};

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const fp = useFormatPrice();
  const queryClient = useQueryClient();
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [cancelReason, setCancelReason] = useState('');
  const [searchParams, setSearchParams] = useSearchParams();

  // Stripe redirects here after Checkout confirmation with ?stripe_pi=...
  // Turn that into a VerifyPayment call so the order flips to "paid" even
  // when the browser skipped the inline verify step. Strips the query on
  // completion so a refresh doesn't re-verify.
  useEffect(() => {
    const pi = searchParams.get('stripe_pi');
    if (!id || !pi) return;
    let cancelled = false;
    (async () => {
      try {
        await apiClient.post(`/payments/order/${id}/verify`, { stripePaymentIntentId: pi });
        if (cancelled) return;
        toast.success('Payment confirmed');
        queryClient.invalidateQueries({ queryKey: ['order', id] });
      } catch {
        if (!cancelled) toast.error('Payment verification failed — please contact support');
      } finally {
        if (!cancelled) {
          const next = new URLSearchParams(searchParams);
          next.delete('stripe_pi');
          setSearchParams(next, { replace: true });
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id, searchParams, queryClient, setSearchParams]);

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

  // CW-01e: customers are legally entitled to a GST invoice under CGST Act
  // 2017 §31 once an order is fulfilled or refunded. We surface the existing
  // /orders/:id/invoice endpoint here; backend still owes the PDF rendering
  // and full CGST §31/Rule 46 particulars (HSN/SAC, GST breakup, supplier).
  // TODO(CW-01e-backend): replace the JSON fetch with a signed PDF download
  // URL once the invoice-generation service ships.
  const [isInvoiceLoading, setIsInvoiceLoading] = useState(false);
  const handleDownloadInvoice = async () => {
    if (!order || isInvoiceLoading) return;
    setIsInvoiceLoading(true);
    try {
      await apiClient.get(`/orders/${order.id}/invoice`);
      toast.success('Invoice generated — PDF download coming soon');
    } catch {
      toast.error('Invoice is not yet available — please contact support');
    } finally {
      setIsInvoiceLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb"  aria-hidden="true" />
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <AlertCircle className="h-16 w-16 text-ink-muted"  aria-hidden="true" />
        <h2 className="mt-4 text-xl font-semibold text-ink">Order not found</h2>
        <Button asChild variant="primary" className="mt-4">
          <Link to="/orders">View All Orders</Link>
        </Button>
      </div>
    );
  }

  const status = STATUS_CONFIG[order.status];
  const StatusIcon = status.icon;
  const isActive = !['delivered', 'cancelled', 'refunded'].includes(order.status);
  const canCancel = ['pending', 'accepted'].includes(order.status);

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-3xl">
        {/* Back Button */}
        <Link
          to="/orders"
          className="inline-flex items-center gap-2 text-ink-soft hover:text-ink"
        >
          <ArrowLeft className="h-4 w-4"  aria-hidden="true" />
          Back to Orders
        </Link>

        {/* Header */}
        <div className="mt-6 rounded-xl bg-bone p-6 shadow-1">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-semibold text-ink">
                  Order #{order.orderNumber}
                </h1>
                <button type="button"
                  onClick={handleCopyOrderNumber}
                  className="p-1 text-ink-muted hover:text-ink-soft"
                >
                  <Copy className="h-4 w-4"  aria-hidden="true" />
                </button>
              </div>
              <p className="mt-1 text-sm text-ink-muted">
                Placed on{' '}
                {formatDateTime(order.createdAt, {
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
                <div className="mt-4 flex items-center gap-2 text-sm text-ink-soft">
                  <Clock className="h-4 w-4"  aria-hidden="true" />
                  <span>
                    Estimated delivery:{' '}
                    <span className="font-medium">
                      {formatTime(order.estimatedDeliveryAt)}
                    </span>
                  </span>
                </div>
              )}
            </div>
          )}

          {/* Cancelled Info */}
          {order.status === 'cancelled' && order.cancelReason && (
            <div className="mt-4 rounded-lg bg-paprika-tint p-4">
              <p className="text-sm text-paprika">
                <span className="font-medium">Cancellation reason:</span> {order.cancelReason}
              </p>
            </div>
          )}
        </div>

        {/* Order Items */}
        <div className="mt-6 rounded-xl bg-bone p-6 shadow-1">
          <h2 className="text-lg font-semibold text-ink">Order Items</h2>

          <div className="mt-4 divide-y">
            {order.items.map((item) => (
              <div key={item.id} className="flex gap-4 py-4">
                {item.imageUrl && (
                  <img
                    src={item.imageUrl}
                    alt={item.name}
                    className="h-16 w-16 rounded-lg object-cover"
                   loading="lazy" decoding="async"/>
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <h4 className="font-medium text-ink">{item.name}</h4>
                      <p className="text-sm text-ink-muted">Qty: {item.quantity}</p>
                      {item.notes && (
                        <p className="mt-1 text-sm text-ink-muted italic">Note: {item.notes}</p>
                      )}
                    </div>
                    <span className="font-medium text-ink">
                      {fp(item.subtotal)}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Delivery Address */}
        <div className="mt-6 rounded-xl bg-bone p-6 shadow-1">
          <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
            <MapPin className="h-5 w-5 text-herb"  aria-hidden="true" />
            Delivery Address
          </h2>
          <div className="mt-4">
            <p className="font-medium text-ink">{order.deliveryAddress.label}</p>
            <p className="mt-1 text-ink-soft">
              {order.deliveryAddress.line1}
              {order.deliveryAddress.line2 && `, ${order.deliveryAddress.line2}`}
            </p>
            <p className="text-ink-soft">
              {order.deliveryAddress.city}, {order.deliveryAddress.state}{' '}
              {order.deliveryAddress.postalCode}
            </p>
            {order.deliveryAddress.deliveryInstructions && (
              <p className="mt-2 text-sm text-ink-muted italic">
                {order.deliveryAddress.deliveryInstructions}
              </p>
            )}
          </div>
        </div>

        {/* Payment Summary */}
        <div className="mt-6 rounded-xl bg-bone p-6 shadow-1">
          <h2 className="text-lg font-semibold text-ink">Payment Summary</h2>

          <div className="mt-4 space-y-3">
            <div className="flex justify-between text-ink-soft">
              <span>Subtotal</span>
              <span>{fp(order.subtotal)}</span>
            </div>
            <div className="flex justify-between text-ink-soft">
              <span>Delivery fee</span>
              <span>{fp(order.deliveryFee)}</span>
            </div>
            <div className="flex justify-between text-ink-soft">
              <span>Service fee</span>
              <span>{fp(order.serviceFee)}</span>
            </div>
            <div className="flex justify-between text-ink-soft">
              <span>Tax</span>
              <span>{fp(order.tax)}</span>
            </div>
            {order.discount > 0 && (
              <div className="flex justify-between text-herb">
                <span>Discount</span>
                <span>-{fp(order.discount)}</span>
              </div>
            )}
            {order.tip > 0 && (
              <div className="flex justify-between text-ink-soft">
                <span>Tip</span>
                <span>{fp(order.tip)}</span>
              </div>
            )}
          </div>

          <div className="mt-4 flex justify-between border-t pt-4 text-lg font-semibold">
            <span>Total</span>
            <span>{fp(order.total)}</span>
          </div>

          <div className="mt-4 flex items-center gap-2 text-sm text-ink-muted">
            <CheckCircle className="h-4 w-4 text-herb"  aria-hidden="true" />
            <span>
              Paid via {order.paymentMethod || 'Card'} • {order.paymentStatus}
            </span>
          </div>
        </div>

        {/* Special Instructions */}
        {order.specialInstructions && (
          <div className="mt-6 rounded-xl bg-bone p-6 shadow-1">
            <h2 className="text-lg font-semibold text-ink">Special Instructions</h2>
            <p className="mt-2 text-ink-soft">{order.specialInstructions}</p>
          </div>
        )}

        {/* Actions */}
        <div className="mt-6 flex flex-wrap gap-4">
          {order.status === 'delivered' && (
            <Button asChild variant="primary" leftIcon={<Star aria-hidden="true" className="h-4 w-4" />}>
              <Link to={`/orders/${order.id}/review`}>Leave a Review</Link>
            </Button>
          )}

          {/* CW-01e: GST invoice download — shown only once the order is in a
              terminal state with payment captured (delivered or refunded). */}
          {(order.status === 'delivered' || order.status === 'refunded') && (
            <Button
              variant="outline"
              onClick={handleDownloadInvoice}
              disabled={isInvoiceLoading}
              isLoading={isInvoiceLoading}
              leftIcon={<FileText aria-hidden="true" className="h-4 w-4" />}
            >
              Download invoice
            </Button>
          )}

          {isActive && (
            <>
              <Button variant="outline" leftIcon={<MessageCircle aria-hidden="true" className="h-4 w-4" />}>
                Contact Support
              </Button>

              {canCancel && (
                <Button
                  variant="outline"
                  onClick={() => setShowCancelModal(true)}
                  leftIcon={<XCircle aria-hidden="true" className="h-4 w-4" />}
                  className="border-paprika/30 text-paprika hover:bg-paprika-tint"
                >
                  Cancel Order
                </Button>
              )}
            </>
          )}

          <Button asChild variant="outline" leftIcon={<RefreshCw aria-hidden="true" className="h-4 w-4" />}>
            <Link to="/chefs">Reorder</Link>
          </Button>
        </div>

        {/* Cancel Modal */}
        {showCancelModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/50 p-4">
            <div className="w-full max-w-md rounded-xl bg-bone p-6">
              <h3 className="text-lg font-semibold text-ink">Cancel Order</h3>
              <p className="mt-2 text-ink-soft">
                Are you sure you want to cancel this order? This action cannot be undone.
              </p>

              <div className="mt-4">
                <label htmlFor="order-cancel-reason" className="block text-sm font-medium text-ink-soft">
                  Reason for cancellation
                </label>
                <textarea
                  id="order-cancel-reason"
                  value={cancelReason}
                  onChange={(e) => setCancelReason(e.target.value)}
                  placeholder="Please let us know why you're cancelling..."
                  rows={3}
                  className="input-base mt-1"
                />
              </div>

              <div className="mt-6 flex justify-end gap-3">
                <Button variant="outline" onClick={() => setShowCancelModal(false)}>
                  Keep Order
                </Button>
                <Button
                  variant="destructive"
                  isLoading={cancelMutation.isPending}
                  disabled={cancelMutation.isPending}
                  onClick={() => cancelMutation.mutate()}
                >
                  {cancelMutation.isPending ? 'Cancelling…' : 'Cancel Order'}
                </Button>
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
      <div className="absolute left-4 top-4 h-[calc(100%-32px)] w-0.5 bg-mist" />
      <div
        className="absolute left-4 top-4 w-0.5 bg-herb transition-all"
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
                    ? 'bg-herb text-paper'
                    : 'bg-mist text-ink-muted'
                } ${isCurrent ? 'ring-4 ring-herb/30' : ''}`}
              >
                <StepIcon className="h-4 w-4" />
              </div>
              <span
                className={`font-medium ${
                  isCompleted ? 'text-ink' : 'text-ink-muted'
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
