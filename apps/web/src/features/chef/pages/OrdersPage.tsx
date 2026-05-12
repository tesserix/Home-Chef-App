import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Search,
  CheckCircle,
  XCircle,
  Package,
  Loader2,
  MapPin,
  RefreshCw,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button, Card, Input, Badge, SimpleDialog } from '@/shared/components/ui';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';
import { useFormatPrice } from '@/shared/utils/format-price';
import type { Order, OrderStatus, PaginatedResponse } from '@/shared/types';

const STATUS_TABS = [
  { value: 'all', label: 'All Orders' },
  { value: 'pending', label: 'New' },
  { value: 'accepted', label: 'Accepted' },
  { value: 'preparing', label: 'Preparing' },
  { value: 'ready', label: 'Ready' },
  { value: 'completed', label: 'Completed' },
];

const STATUS_CONFIG: Record<string, { label: string; variant: 'warning' | 'info' | 'success' | 'default' | 'error'; nextAction?: string; nextStatus?: OrderStatus }> = {
  pending: { label: 'New Order', variant: 'warning', nextAction: 'Accept', nextStatus: 'accepted' },
  accepted: { label: 'Accepted', variant: 'info', nextAction: 'Start Preparing', nextStatus: 'preparing' },
  preparing: { label: 'Preparing', variant: 'info', nextAction: 'Mark Ready', nextStatus: 'ready' },
  ready: { label: 'Ready for Pickup', variant: 'success' },
  picked_up: { label: 'Picked Up', variant: 'info' },
  delivering: { label: 'Delivering', variant: 'warning' },
  delivered: { label: 'Delivered', variant: 'default' },
  cancelled: { label: 'Cancelled', variant: 'error' },
};

export default function ChefOrdersPage() {
  const [statusFilter, setStatusFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const queryClient = useQueryClient();

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['chef-orders', statusFilter],
    queryFn: () => {
      const params: Record<string, string> = {};
      if (statusFilter !== 'all') {
        if (statusFilter === 'completed') {
          params.status = 'delivered,cancelled';
        } else {
          params.status = statusFilter;
        }
      }
      return apiClient.get<PaginatedResponse<Order>>('/chef/orders', params);
    },
    refetchInterval: 30000,
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ orderId, status }: { orderId: string; status: OrderStatus }) =>
      apiClient.put(`/chef/orders/${orderId}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-orders'] });
      toast.success('Order status updated');
      setSelectedOrder(null);
    },
    onError: () => {
      toast.error('Failed to update order status');
    },
  });

  const orders = data?.data || [];
  const filteredOrders = searchQuery
    ? orders.filter((order) =>
        order.orderNumber.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : orders;

  const pendingCount = orders.filter((o) => o.status === 'pending').length;

  return (
    <motion.div
      initial="hidden"
      animate="visible"
      variants={staggerContainer}
      className="space-y-6"
    >
      {/* Header */}
      <motion.div variants={fadeInUp} className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-display-xs text-ink">Orders</h1>
          <p className="mt-1 text-ink-soft">
            Manage and track your incoming orders
          </p>
        </div>
        <Button
          variant="outline"
          leftIcon={<RefreshCw className="h-4 w-4" />}
          onClick={() => refetch()}
        >
          Refresh
        </Button>
      </motion.div>

      {/* Status Tabs */}
      <motion.div variants={fadeInUp} className="flex gap-2 overflow-x-auto pb-2">
        {STATUS_TABS.map((tab) => (
          <Button
            key={tab.value}
            variant={statusFilter === tab.value ? 'primary' : 'outline'}
            size="sm"
            onClick={() => setStatusFilter(tab.value)}
            className="relative whitespace-nowrap"
          >
            {tab.label}
            {tab.value === 'pending' && pendingCount > 0 && (
              <span className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-paprika text-xs text-paper">
                {pendingCount}
              </span>
            )}
          </Button>
        ))}
      </motion.div>

      {/* Search */}
      <motion.div variants={fadeInUp} className="max-w-md">
        <Input
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search by order number..."
          leftIcon={<Search className="h-5 w-5" />}
        />
      </motion.div>

      {/* Orders List */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-herb" />
        </div>
      ) : filteredOrders.length === 0 ? (
        <motion.div variants={fadeInUp}>
          <Card variant="filled" padding="lg" className="text-center">
            <Package className="mx-auto h-12 w-12 text-ink-muted" />
            <h3 className="mt-4 font-medium text-ink">No orders found</h3>
            <p className="mt-2 text-ink-soft">
              {statusFilter === 'all'
                ? "You don't have any orders yet"
                : `No ${statusFilter} orders`}
            </p>
          </Card>
        </motion.div>
      ) : (
        <motion.div variants={staggerContainer} className="space-y-4">
          {filteredOrders.map((order) => (
            <motion.div key={order.id} variants={fadeInUp}>
              <OrderCard
                order={order}
                onSelect={() => setSelectedOrder(order)}
                onUpdateStatus={(status) =>
                  updateStatusMutation.mutate({ orderId: order.id, status })
                }
                isUpdating={updateStatusMutation.isPending}
              />
            </motion.div>
          ))}
        </motion.div>
      )}

      {/* Order Detail Dialog */}
      <SimpleDialog
        open={!!selectedOrder}
        onOpenChange={(open) => !open && setSelectedOrder(null)}
        title={`Order #${selectedOrder?.orderNumber}`}
        size="lg"
      >
        {selectedOrder && (
          <OrderDetailContent
            order={selectedOrder}
            onClose={() => setSelectedOrder(null)}
            onUpdateStatus={(status) =>
              updateStatusMutation.mutate({ orderId: selectedOrder.id, status })
            }
            isUpdating={updateStatusMutation.isPending}
          />
        )}
      </SimpleDialog>
    </motion.div>
  );
}

function OrderCard({
  order,
  onSelect,
  onUpdateStatus,
  isUpdating,
}: {
  order: Order;
  onSelect: () => void;
  onUpdateStatus: (status: OrderStatus) => void;
  isUpdating: boolean;
}) {
  const fp = useFormatPrice();
  const status = STATUS_CONFIG[order.status] ?? { label: order.status, variant: 'default' as const };
  const isNew = order.status === 'pending';

  return (
    <Card
      variant="default"
      padding="lg"
      className={isNew ? 'ring-2 ring-amber' : ''}
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        {/* Order Info */}
        <div className="flex items-start gap-4">
          {isNew && (
            <span className="mt-1 flex h-3 w-3 rounded-full bg-amber animate-pulse" />
          )}
          <div>
            <div className="flex items-center gap-3">
              <button
                onClick={onSelect}
                className="text-lg font-semibold text-ink hover:text-herb transition-colors"
              >
                #{order.orderNumber}
              </button>
              <Badge variant={status.variant} size="sm">
                {status.label}
              </Badge>
            </div>
            <p className="mt-1 text-sm text-ink-muted">
              {new Date(order.createdAt).toLocaleString('en-US', {
                month: 'short',
                day: 'numeric',
                hour: 'numeric',
                minute: '2-digit',
              })}
            </p>
          </div>
        </div>

        {/* Total */}
        <div className="text-right">
          <p className="text-xl font-semibold text-ink">{fp(order.total)}</p>
          <p className="text-sm text-ink-muted">{order.items.length} item(s)</p>
        </div>
      </div>

      {/* Items Preview */}
      <div className="mt-4 rounded-xl bg-paper p-3">
        <ul className="space-y-1 text-sm">
          {order.items.slice(0, 3).map((item) => (
            <li key={item.id} className="flex justify-between text-ink-soft">
              <span>
                {item.quantity}x {item.name}
              </span>
              <span>{fp(item.subtotal)}</span>
            </li>
          ))}
          {order.items.length > 3 && (
            <li className="text-ink-muted">+{order.items.length - 3} more items</li>
          )}
        </ul>
      </div>

      {/* Special Instructions */}
      {order.specialInstructions && (
        <div className="mt-3 rounded-xl bg-amber-tint p-3 text-sm text-amber">
          <span className="font-medium">Note:</span> {order.specialInstructions}
        </div>
      )}

      {/* Actions */}
      <div className="mt-4 flex flex-wrap gap-3">
        {status.nextAction && status.nextStatus && (
          <Button
            variant="primary"
            onClick={() => onUpdateStatus(status.nextStatus!)}
            isLoading={isUpdating}
          >
            {status.nextAction}
          </Button>
        )}
        {order.status === 'pending' && (
          <Button
            variant="outline"
            onClick={() => onUpdateStatus('cancelled')}
            disabled={isUpdating}
            leftIcon={<XCircle className="h-4 w-4" />}
            className="text-paprika border-paprika/30 hover:bg-paprika-tint"
          >
            Reject
          </Button>
        )}
        <Button variant="outline" onClick={onSelect}>
          View Details
        </Button>
      </div>
    </Card>
  );
}

function OrderDetailContent({
  order,
  onClose,
  onUpdateStatus,
  isUpdating,
}: {
  order: Order;
  onClose: () => void;
  onUpdateStatus: (status: OrderStatus) => void;
  isUpdating: boolean;
}) {
  const fp = useFormatPrice();
  const status = STATUS_CONFIG[order.status] ?? { label: order.status, variant: 'default' as const };

  return (
    <div className="space-y-6">
      {/* Status Badge */}
      <div className="flex justify-between items-center">
        <p className="text-sm text-ink-muted">
          {new Date(order.createdAt).toLocaleString()}
        </p>
        <Badge variant={status.variant} size="md">
          {status.label}
        </Badge>
      </div>

      {/* Customer Info - Anonymized per RBAC */}
      <Card variant="filled" padding="md">
        <h3 className="font-medium text-ink">Delivery Information</h3>
        <div className="mt-3 space-y-2 text-sm text-ink-soft">
          <div className="flex items-center gap-2">
            <MapPin className="h-4 w-4 text-ink-muted" />
            <span>
              {order.deliveryAddress.line1}, {order.deliveryAddress.city},{' '}
              {order.deliveryAddress.state} {order.deliveryAddress.postalCode}
            </span>
          </div>
          {order.deliveryAddress.deliveryInstructions && (
            <p className="text-ink-muted italic">
              {order.deliveryAddress.deliveryInstructions}
            </p>
          )}
        </div>
      </Card>

      {/* Order Items */}
      <div>
        <h3 className="font-medium text-ink">Order Items</h3>
        <div className="mt-3 divide-y rounded-xl border">
          {order.items.map((item) => (
            <div key={item.id} className="flex items-center gap-4 p-3">
              {item.imageUrl && (
                <img
                  src={item.imageUrl}
                  alt={item.name}
                  className="h-12 w-12 rounded-lg object-cover"
                />
              )}
              <div className="flex-1">
                <p className="font-medium text-ink">{item.name}</p>
                {item.notes && (
                  <p className="text-sm text-ink-muted">Note: {item.notes}</p>
                )}
              </div>
              <div className="text-right">
                <p className="font-medium text-ink">x{item.quantity}</p>
                <p className="text-sm text-ink-muted">{fp(item.subtotal)}</p>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Special Instructions */}
      {order.specialInstructions && (
        <Card variant="filled" padding="md" className="bg-amber-tint">
          <h3 className="font-medium text-amber">Special Instructions</h3>
          <p className="mt-2 text-sm text-amber">{order.specialInstructions}</p>
        </Card>
      )}

      {/* Payment Summary */}
      <Card variant="filled" padding="md">
        <h3 className="font-medium text-ink">Payment Summary</h3>
        <div className="mt-3 space-y-2 text-sm">
          <div className="flex justify-between text-ink-soft">
            <span>Subtotal</span>
            <span>{fp(order.subtotal)}</span>
          </div>
          {order.discount > 0 && (
            <div className="flex justify-between text-fresh-600">
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
          <div className="flex justify-between border-t pt-2 font-semibold text-ink">
            <span>Your Earnings</span>
            <span>{fp(order.subtotal - order.discount + order.tip)}</span>
          </div>
        </div>
      </Card>

      {/* Actions */}
      <div className="flex justify-between gap-3 border-t pt-6">
        <Button variant="outline" onClick={onClose}>
          Close
        </Button>
        <div className="flex gap-3">
          {order.status === 'pending' && (
            <Button
              variant="outline"
              onClick={() => onUpdateStatus('cancelled')}
              disabled={isUpdating}
              className="text-paprika border-paprika/30 hover:bg-paprika-tint"
            >
              Reject Order
            </Button>
          )}
          {status.nextAction && status.nextStatus && (
            <Button
              variant="primary"
              onClick={() => onUpdateStatus(status.nextStatus!)}
              isLoading={isUpdating}
              leftIcon={<CheckCircle className="h-4 w-4" />}
            >
              {status.nextAction}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
