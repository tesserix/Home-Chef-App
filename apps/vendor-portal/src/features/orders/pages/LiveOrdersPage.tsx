import { useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';
import {
  Clock,
  ChefHat,
  CheckCircle2,
  XCircle,
  Package,
  History,
  RefreshCw,
  Inbox,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import { Card } from '@/shared/components/ui/Card';
import { OrderStatusBadge } from '@/shared/components/ui/Badge';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import type { Order, OrderStatus } from '@/shared/types';

type LiveTab = 'all' | 'pending' | 'accepted' | 'preparing' | 'ready';

const LIVE_STATUSES = 'pending,accepted,preparing,ready';

const tabs: { key: LiveTab; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'pending', label: 'Pending' },
  { key: 'accepted', label: 'Accepted' },
  { key: 'preparing', label: 'Preparing' },
  { key: 'ready', label: 'Ready' },
];

function getNextStatusAction(status: OrderStatus): { label: string; nextStatus: OrderStatus; variant: 'default' | 'success' | 'destructive' } | null {
  switch (status) {
    case 'accepted':
      return { label: 'Start Preparing', nextStatus: 'preparing', variant: 'default' };
    case 'preparing':
      return { label: 'Mark Ready', nextStatus: 'ready', variant: 'success' };
    default:
      return null;
  }
}

export default function LiveOrdersPage() {
  const [activeTab, setActiveTab] = useState<LiveTab>('all');
  const queryClient = useQueryClient();

  const { data: orders = [], isLoading, isRefetching } = useQuery<Order[]>({
    queryKey: ['chef-orders', 'live'],
    queryFn: () => apiClient.get<Order[]>('/chef/orders', { status: LIVE_STATUSES }),
    refetchInterval: 30000,
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ orderId, status }: { orderId: string; status: OrderStatus }) =>
      apiClient.put(`/chef/orders/${orderId}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-orders'] });
      toast.success('Order status updated');
    },
    onError: () => {
      toast.error('Failed to update order status');
    },
  });

  const handleAccept = (orderId: string) => {
    updateStatusMutation.mutate({ orderId, status: 'accepted' });
  };

  const handleReject = (orderId: string) => {
    updateStatusMutation.mutate({ orderId, status: 'cancelled' });
  };

  const handleStatusAdvance = (orderId: string, nextStatus: OrderStatus) => {
    updateStatusMutation.mutate({ orderId, status: nextStatus });
  };

  const filteredOrders = activeTab === 'all'
    ? orders
    : orders.filter((order) => order.status === activeTab);

  const pendingCount = orders.filter((o) => o.status === 'pending').length;

  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Header */}
      <motion.div variants={fadeInUp} className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Live Orders</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {orders.length} active order{orders.length !== 1 ? 's' : ''}
            {pendingCount > 0 && (
              <span className="ml-2 font-medium text-warning">
                ({pendingCount} pending)
              </span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {isRefetching && (
            <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />
          )}
          <Link to="/orders/history">
            <Button variant="outline" size="sm" leftIcon={<History className="h-4 w-4" />}>
              Order History
            </Button>
          </Link>
        </div>
      </motion.div>

      {/* Tabs */}
      <motion.div variants={fadeInUp} className="flex gap-1 rounded-lg bg-secondary p-1">
        {tabs.map((tab) => {
          const count = tab.key === 'all'
            ? orders.length
            : orders.filter((o) => o.status === tab.key).length;

          return (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`flex-1 rounded-md px-3 py-2 text-sm font-medium transition-all duration-200 ${
                activeTab === tab.key
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {tab.label}
              {count > 0 && (
                <span
                  className={`ml-1.5 inline-flex h-5 w-5 items-center justify-center rounded-full text-xs ${
                    activeTab === tab.key
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-muted-foreground'
                  }`}
                >
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </motion.div>

      {/* Order Cards */}
      {isLoading ? (
        <motion.div variants={fadeInUp} className="space-y-4">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="animate-pulse">
              <div className="h-6 w-32 rounded bg-muted" />
              <div className="mt-3 h-4 w-48 rounded bg-muted" />
              <div className="mt-3 h-4 w-64 rounded bg-muted" />
              <div className="mt-4 flex gap-2">
                <div className="h-9 w-24 rounded-lg bg-muted" />
                <div className="h-9 w-24 rounded-lg bg-muted" />
              </div>
            </Card>
          ))}
        </motion.div>
      ) : filteredOrders.length === 0 ? (
        <motion.div variants={fadeInUp}>
          <Card className="flex flex-col items-center justify-center py-16">
            <Inbox className="h-12 w-12 text-muted-foreground/40" />
            <h3 className="mt-4 text-lg font-semibold text-foreground">
              No {activeTab === 'all' ? '' : activeTab} orders
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {activeTab === 'all'
                ? 'No active orders right now. New orders will appear here automatically.'
                : `No orders with "${activeTab}" status. Check other tabs or wait for new orders.`}
            </p>
          </Card>
        </motion.div>
      ) : (
        <motion.div variants={staggerContainer} className="space-y-4">
          {filteredOrders.map((order) => {
            const nextAction = getNextStatusAction(order.status);

            return (
              <motion.div key={order.id} variants={fadeInUp}>
                <Card className="space-y-4">
                  {/* Order header */}
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="text-base font-semibold text-foreground">
                          #{order.orderNumber}
                        </h3>
                        <OrderStatusBadge status={order.status} />
                      </div>
                      <p className="mt-0.5 text-sm text-muted-foreground">
                        Customer
                      </p>
                    </div>
                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                      <Clock className="h-3.5 w-3.5" />
                      {formatDistanceToNow(new Date(order.createdAt), { addSuffix: true })}
                    </div>
                  </div>

                  {/* Items */}
                  <div className="rounded-lg bg-secondary/50 p-3">
                    <div className="space-y-1.5">
                      {(order.items ?? []).map((item) => (
                        <div
                          key={item.id}
                          className="flex items-center justify-between text-sm"
                        >
                          <span className="text-foreground">
                            {item.quantity}x {item.name}
                          </span>
                          <span className="font-medium text-foreground">
                            ${(item.subtotal ?? 0).toFixed(2)}
                          </span>
                        </div>
                      ))}
                    </div>
                    {order.specialInstructions && (
                      <p className="mt-2 border-t border-border pt-2 text-xs italic text-muted-foreground">
                        Note: {order.specialInstructions}
                      </p>
                    )}
                  </div>

                  {/* Total */}
                  <div className="flex items-center justify-between border-t border-border pt-3">
                    <span className="text-sm font-medium text-muted-foreground">
                      {(order.items ?? []).length} item{(order.items ?? []).length !== 1 ? 's' : ''} total
                    </span>
                    <span className="text-lg font-bold text-foreground">
                      ${(order.total ?? 0).toFixed(2)}
                    </span>
                  </div>

                  {/* Actions */}
                  <div className="flex gap-2">
                    {order.status === 'pending' && (
                      <>
                        <Button
                          variant="success"
                          size="sm"
                          className="flex-1"
                          leftIcon={<CheckCircle2 className="h-4 w-4" />}
                          onClick={() => handleAccept(order.id)}
                          isLoading={
                            updateStatusMutation.isPending &&
                            updateStatusMutation.variables?.orderId === order.id &&
                            updateStatusMutation.variables?.status === 'accepted'
                          }
                        >
                          Accept
                        </Button>
                        <Button
                          variant="destructive"
                          size="sm"
                          className="flex-1"
                          leftIcon={<XCircle className="h-4 w-4" />}
                          onClick={() => handleReject(order.id)}
                          isLoading={
                            updateStatusMutation.isPending &&
                            updateStatusMutation.variables?.orderId === order.id &&
                            updateStatusMutation.variables?.status === 'cancelled'
                          }
                        >
                          Reject
                        </Button>
                      </>
                    )}
                    {nextAction && (
                      <Button
                        variant={nextAction.variant}
                        size="sm"
                        className="flex-1"
                        leftIcon={
                          nextAction.nextStatus === 'preparing' ? (
                            <ChefHat className="h-4 w-4" />
                          ) : (
                            <Package className="h-4 w-4" />
                          )
                        }
                        onClick={() => handleStatusAdvance(order.id, nextAction.nextStatus)}
                        isLoading={
                          updateStatusMutation.isPending &&
                          updateStatusMutation.variables?.orderId === order.id
                        }
                      >
                        {nextAction.label}
                      </Button>
                    )}
                    {order.status === 'ready' && (
                      <div className="flex flex-1 items-center justify-center rounded-lg border border-success/30 bg-success/5 px-3 py-2 text-sm font-medium text-success">
                        <CheckCircle2 className="mr-2 h-4 w-4" />
                        Waiting for pickup
                      </div>
                    )}
                  </div>
                </Card>
              </motion.div>
            );
          })}
        </motion.div>
      )}
    </motion.div>
  );
}
