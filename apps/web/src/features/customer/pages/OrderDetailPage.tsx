import { useEffect, useState } from 'react';
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom';
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
import { useCartStore } from '@/app/store/cart-store';
import type { Order, OrderStatus, MenuItem, SelectedModifier } from '@/shared/types';

// Reorder preview shape returned by POST /orders/:id/reorder (#238).
interface ReorderResponseItem {
  menuItemId: string;
  name: string;
  quantity: number;
  notes?: string;
  imageUrl?: string;
  modifiers: SelectedModifier[];
  unitPrice: number;
  available: boolean;
  reason?: string;
  needsReview?: boolean;
}
interface ReorderResponse {
  chefId: string;
  chefName: string;
  chefAccepting: boolean;
  items: ReorderResponseItem[];
}

// Report-an-issue (#37). Reasons mirror the API IssueReason enum
// (models/order_issue.go); refund amount is decided server-side.
type IssueReason = 'missing_item' | 'wrong_item' | 'damaged' | 'quality_issue' | 'other';
const ISSUE_REASONS: { value: IssueReason; label: string }[] = [
  { value: 'missing_item', label: 'Missing item' },
  { value: 'wrong_item', label: 'Wrong item' },
  { value: 'damaged', label: 'Damaged / spilled' },
  { value: 'quality_issue', label: 'Quality issue' },
  { value: 'other', label: 'Something else' },
];
interface ReportIssueResponse {
  issueId: string;
  status: 'pending' | 'auto_refunded' | 'resolved' | 'rejected';
  refundAmount: number;
  message: string;
}

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
  const navigate = useNavigate();
  const cart = useCartStore();
  const queryClient = useQueryClient();
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [cancelReason, setCancelReason] = useState('');
  // Report-an-issue flow (#37): reason + optional affected items / description /
  // photo → instant or assisted partial refund to the wallet.
  const [showReportModal, setShowReportModal] = useState(false);
  const [reportReason, setReportReason] = useState<IssueReason | ''>('');
  const [reportDescription, setReportDescription] = useState('');
  const [reportItems, setReportItems] = useState<Set<string>>(new Set());
  const [reportPhoto, setReportPhoto] = useState<File | null>(null);
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

  // Reorder (#238) — re-add a past order's still-available items to the cart,
  // resolving current add-on option IDs, then send the customer to the chef to
  // review and check out.
  const reorderMutation = useMutation({
    mutationFn: () => apiClient.post<ReorderResponse>(`/orders/${id}/reorder`),
    onSuccess: (res) => {
      const available = res.items.filter((i) => i.available);
      if (available.length === 0) {
        toast.error('None of these items are available right now');
        return;
      }
      // Cross-chef: clear first so addItem won't reject with DIFFERENT_CHEF.
      if (cart.chefId && cart.chefId !== res.chefId) {
        cart.clearCart();
      }
      for (const it of available) {
        const mods = it.modifiers ?? [];
        const base = it.unitPrice - mods.reduce((s, m) => s + m.priceDelta, 0);
        // Construct the minimal MenuItem the cart needs; addItem re-applies the
        // modifier deltas, so we pass the base price.
        const menuItem = {
          id: it.menuItemId,
          chefId: res.chefId,
          name: it.name,
          price: base,
          imageUrl: it.imageUrl,
          dietaryTags: [],
          allergens: [],
          prepTime: 0,
          isAvailable: true,
          isFeatured: false,
          serves: 1,
        } as MenuItem;
        cart.addItem(menuItem, it.quantity, it.notes, mods.length ? mods : undefined);
      }
      const dropped = res.items.length - available.length;
      if (dropped > 0) toast.warning(`${dropped} item${dropped > 1 ? 's are' : ' is'} no longer available`);
      if (available.some((i) => i.needsReview)) {
        toast.message('Some add-ons changed — please review your cart');
      }
      navigate(`/chefs/${res.chefId}`);
    },
    onError: () => toast.error('Could not reorder right now. Please try again.'),
  });

  // Report an issue (#37) → instant/assisted partial refund to the wallet. The
  // server decides the amount; we just surface the outcome. Multipart so an
  // optional photo rides along (mirrors the review upload).
  const reportMutation = useMutation({
    mutationFn: () => {
      const fd = new FormData();
      fd.append('reason', reportReason);
      if (reportDescription.trim()) fd.append('description', reportDescription.trim());
      reportItems.forEach((itemId) => fd.append('affectedItemIds', itemId));
      if (reportPhoto) fd.append('photo', reportPhoto);
      return apiClient.upload<ReportIssueResponse>(`/orders/${id}/report-issue`, fd);
    },
    onSuccess: (res) => {
      setShowReportModal(false);
      setReportReason('');
      setReportDescription('');
      setReportItems(new Set());
      setReportPhoto(null);
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      queryClient.invalidateQueries({ queryKey: ['wallet'] });
      queryClient.invalidateQueries({ queryKey: ['wallet-transactions'] });
      if (res.status === 'auto_refunded' && res.refundAmount > 0) {
        toast.success(`${fp(res.refundAmount)} refunded to your Fe3dr wallet`);
      } else {
        toast.success(res.message || 'Thanks — our team will review this shortly');
      }
    },
    onError: (e: unknown) => {
      toast.error(e instanceof Error ? e.message : 'Could not submit your report');
    },
  });

  const toggleReportItem = (itemId: string) => {
    setReportItems((prev) => {
      const next = new Set(prev);
      if (next.has(itemId)) next.delete(itemId);
      else next.add(itemId);
      return next;
    });
  };

  const handleCopyOrderNumber = () => {
    if (order) {
      navigator.clipboard.writeText(order.orderNumber);
      toast.success('Order number copied');
    }
  };

  // CW-01e: customers are legally entitled to a GST invoice under CGST Act
  // 2017 §31 once an order is fulfilled or refunded. This now downloads the real
  // PDF the backend serves — a TAX INVOICE for a delivered sale, a PAYMENT
  // RECEIPT for a paid-then-cancelled/refunded order — the same document and the
  // same availability rule the mobile app uses (/orders/:id/invoice.pdf, any
  // paid order). Kept consistent so a customer sees the same receipt on web and
  // mobile.
  const [isInvoiceLoading, setIsInvoiceLoading] = useState(false);
  // Money was captured → a receipt exists. Matches the mobile gate + the backend.
  const hasReceipt =
    order?.paymentStatus === 'completed' || order?.paymentStatus === 'refunded';
  const handleDownloadInvoice = async () => {
    if (!order || isInvoiceLoading) return;
    setIsInvoiceLoading(true);
    try {
      const { blob, filename } = await apiClient.getBlob(
        `/orders/${order.id}/invoice.pdf`,
      );
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {
      toast.error('Receipt is not available yet — please contact support');
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
  // Report-an-issue eligibility mirrors the API guard: paid order, not cancelled (#37).
  const canReport = order.paymentStatus === 'completed' && order.status !== 'cancelled';

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
              {/* Live cooking animation while the chef is preparing (#50) */}
              <StatusIcon
                className={`h-5 w-5 ${status.color} ${order.status === 'preparing' ? 'animate-bounce' : ''}`}
              />
              <span className={`font-medium ${status.color}`}>
                {order.status === 'preparing' ? 'Cooking now' : status.label}
              </span>
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

          {/* Post-delivery tip (#45) — 100% pass-through to chef/rider. */}
          {order.status === 'delivered' && (
            <Button asChild variant="outline">
              <Link to={`/orders/${order.id}/tip`}>Tip your chef / rider</Link>
            </Button>
          )}

          {/* CW-01e: receipt / GST invoice download — for ANY order the customer
              paid for, so a paid-then-cancelled order (status cancelled,
              payment refunded) is covered, not just delivered. This gate was
              status-based (delivered || refunded) and missed exactly that case;
              it now matches the mobile app + the backend (money captured). */}
          {hasReceipt && (
            <Button
              variant="outline"
              onClick={handleDownloadInvoice}
              disabled={isInvoiceLoading}
              isLoading={isInvoiceLoading}
              leftIcon={<FileText aria-hidden="true" className="h-4 w-4" />}
            >
              Download receipt
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

          <Button
            variant="outline"
            leftIcon={<RefreshCw aria-hidden="true" className="h-4 w-4" />}
            isLoading={reorderMutation.isPending}
            disabled={reorderMutation.isPending}
            onClick={() => reorderMutation.mutate()}
          >
            Reorder
          </Button>

          {canReport && (
            <Button
              variant="ghost"
              leftIcon={<AlertCircle aria-hidden="true" className="h-4 w-4" />}
              onClick={() => setShowReportModal(true)}
            >
              Report an issue
            </Button>
          )}
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

        {/* Report-an-issue Modal (#37) */}
        {showReportModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/50 p-4">
            <div className="max-h-[90vh] w-full max-w-md overflow-y-auto rounded-xl bg-bone p-6">
              <h3 className="text-lg font-semibold text-ink">Report an issue</h3>
              <p className="mt-1 text-sm text-ink-soft">
                Tell us what went wrong. Eligible issues are refunded to your Fe3dr wallet right away.
              </p>

              <fieldset className="mt-4">
                <legend className="text-sm font-medium text-ink-soft">What went wrong?</legend>
                <div className="mt-2 flex flex-wrap gap-2">
                  {ISSUE_REASONS.map((r) => {
                    const active = reportReason === r.value;
                    return (
                      <button
                        key={r.value}
                        type="button"
                        aria-pressed={active}
                        onClick={() => setReportReason(r.value)}
                        className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                          active
                            ? 'border-herb bg-herb-tint text-herb'
                            : 'border-mist bg-paper text-ink-soft hover:border-ink-soft'
                        }`}
                      >
                        {r.label}
                      </button>
                    );
                  })}
                </div>
              </fieldset>

              {order.items.length > 0 && (
                <fieldset className="mt-4">
                  <legend className="text-sm font-medium text-ink-soft">Which items? (optional)</legend>
                  <div className="mt-2 divide-y divide-mist rounded-lg border border-mist">
                    {order.items.map((item) => {
                      const checked = reportItems.has(item.id);
                      return (
                        <label
                          key={item.id}
                          className="flex cursor-pointer items-center gap-3 px-3 py-2.5 text-sm"
                        >
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => toggleReportItem(item.id)}
                            className="h-4 w-4 accent-herb"
                          />
                          <span className="text-ink">
                            {item.quantity}× {item.name}
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </fieldset>
              )}

              <div className="mt-4">
                <label htmlFor="order-issue-description" className="block text-sm font-medium text-ink-soft">
                  Tell us more (optional)
                </label>
                <textarea
                  id="order-issue-description"
                  value={reportDescription}
                  onChange={(e) => setReportDescription(e.target.value)}
                  placeholder="Describe the problem…"
                  rows={3}
                  maxLength={500}
                  className="input-base mt-1"
                />
              </div>

              <div className="mt-4">
                <label htmlFor="order-issue-photo" className="block text-sm font-medium text-ink-soft">
                  Add a photo (optional)
                </label>
                <input
                  id="order-issue-photo"
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  onChange={(e) => setReportPhoto(e.target.files?.[0] ?? null)}
                  className="mt-1 block w-full text-sm text-ink-soft file:mr-3 file:rounded-md file:border-0 file:bg-mist file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-ink hover:file:bg-mist/80"
                />
              </div>

              <div className="mt-6 flex justify-end gap-3">
                <Button variant="outline" onClick={() => setShowReportModal(false)}>
                  Cancel
                </Button>
                <Button
                  isLoading={reportMutation.isPending}
                  disabled={!reportReason || reportMutation.isPending}
                  onClick={() => reportMutation.mutate()}
                >
                  {reportMutation.isPending ? 'Submitting…' : 'Submit report'}
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
