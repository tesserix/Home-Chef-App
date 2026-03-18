import { useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useQuery } from '@tanstack/react-query';
import { formatDistanceToNow, format } from 'date-fns';
import {
  ArrowLeft,
  Calendar,
  Inbox,
  Receipt,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import { Card } from '@/shared/components/ui/Card';
import { OrderStatusBadge } from '@/shared/components/ui/Badge';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import type { Order } from '@/shared/types';

type DateRange = 'today' | '7days' | '30days' | 'all';

const HISTORY_STATUSES = 'delivered,cancelled';

const dateRangeOptions: { value: DateRange; label: string }[] = [
  { value: 'today', label: 'Today' },
  { value: '7days', label: 'Last 7 days' },
  { value: '30days', label: 'Last 30 days' },
  { value: 'all', label: 'All time' },
];

function getDateRangeFilter(range: DateRange): string | undefined {
  const now = new Date();
  switch (range) {
    case 'today': {
      const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      return start.toISOString();
    }
    case '7days': {
      const start = new Date(now);
      start.setDate(start.getDate() - 7);
      return start.toISOString();
    }
    case '30days': {
      const start = new Date(now);
      start.setDate(start.getDate() - 30);
      return start.toISOString();
    }
    case 'all':
    default:
      return undefined;
  }
}

export default function OrderHistoryPage() {
  const [dateRange, setDateRange] = useState<DateRange>('7days');

  const fromDate = getDateRangeFilter(dateRange);

  const { data: orders = [], isLoading } = useQuery<Order[]>({
    queryKey: ['chef-orders', 'history', dateRange],
    queryFn: () =>
      apiClient.get<Order[]>('/chef/orders', {
        status: HISTORY_STATUSES,
        ...(fromDate ? { from: fromDate } : {}),
      }),
  });

  const deliveredCount = orders.filter((o) => o.status === 'delivered').length;
  const cancelledCount = orders.filter((o) => o.status === 'cancelled').length;
  const totalRevenue = orders
    .filter((o) => o.status === 'delivered')
    .reduce((sum, o) => sum + (o.total ?? 0), 0);

  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Header */}
      <motion.div variants={fadeInUp} className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/orders">
            <Button variant="ghost" size="icon-sm">
              <ArrowLeft className="h-5 w-5" />
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-foreground">Order History</h1>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {orders.length} order{orders.length !== 1 ? 's' : ''} found
            </p>
          </div>
        </div>
        <Link to="/orders">
          <Button variant="outline" size="sm" leftIcon={<Receipt className="h-4 w-4" />}>
            Live Orders
          </Button>
        </Link>
      </motion.div>

      {/* Date Range Filter */}
      <motion.div variants={fadeInUp} className="flex items-center gap-2">
        <Calendar className="h-4 w-4 text-muted-foreground" />
        <div className="flex gap-1 rounded-lg bg-secondary p-1">
          {dateRangeOptions.map((option) => (
            <button
              key={option.value}
              onClick={() => setDateRange(option.value)}
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition-all duration-200 ${
                dateRange === option.value
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      </motion.div>

      {/* Summary Cards */}
      <motion.div variants={fadeInUp} className="grid grid-cols-3 gap-4">
        <Card padding="sm" className="text-center">
          <p className="text-xs font-medium text-muted-foreground">Delivered</p>
          <p className="mt-1 text-xl font-bold text-success">{deliveredCount}</p>
        </Card>
        <Card padding="sm" className="text-center">
          <p className="text-xs font-medium text-muted-foreground">Cancelled</p>
          <p className="mt-1 text-xl font-bold text-destructive">{cancelledCount}</p>
        </Card>
        <Card padding="sm" className="text-center">
          <p className="text-xs font-medium text-muted-foreground">Revenue</p>
          <p className="mt-1 text-xl font-bold text-foreground">
            ${totalRevenue.toFixed(2)}
          </p>
        </Card>
      </motion.div>

      {/* Orders List */}
      {isLoading ? (
        <motion.div variants={fadeInUp} className="space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i} padding="sm" className="animate-pulse">
              <div className="flex items-center justify-between">
                <div className="space-y-2">
                  <div className="h-5 w-28 rounded bg-muted" />
                  <div className="h-4 w-40 rounded bg-muted" />
                </div>
                <div className="h-6 w-20 rounded-full bg-muted" />
              </div>
            </Card>
          ))}
        </motion.div>
      ) : orders.length === 0 ? (
        <motion.div variants={fadeInUp}>
          <Card className="flex flex-col items-center justify-center py-16">
            <Inbox className="h-12 w-12 text-muted-foreground/40" />
            <h3 className="mt-4 text-lg font-semibold text-foreground">
              No orders found
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {dateRange === 'all'
                ? 'You have no completed or cancelled orders yet.'
                : 'No orders found for the selected time period. Try a wider date range.'}
            </p>
          </Card>
        </motion.div>
      ) : (
        <motion.div variants={staggerContainer} className="space-y-3">
          {orders.map((order) => (
            <motion.div key={order.id} variants={fadeInUp}>
              <Card padding="sm" className="space-y-3">
                {/* Top row */}
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-sm font-semibold text-foreground">
                        #{order.orderNumber}
                      </h3>
                      <OrderStatusBadge status={order.status} size="sm" />
                    </div>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {format(new Date(order.createdAt), 'MMM d, yyyy h:mm a')}
                      {' -- '}
                      {formatDistanceToNow(new Date(order.createdAt), { addSuffix: true })}
                    </p>
                  </div>
                  <span className="text-base font-bold text-foreground">
                    ${(order.total ?? 0).toFixed(2)}
                  </span>
                </div>

                {/* Items summary */}
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>
                    {(order.items ?? []).length} item{(order.items ?? []).length !== 1 ? 's' : ''}
                    {' -- '}
                    {(order.items ?? []).map((item) => `${item.quantity}x ${item.name}`).join(', ')}
                  </span>
                </div>

                {/* Cancelled reason */}
                {order.status === 'cancelled' && order.cancelReason && (
                  <p className="rounded-md bg-destructive/5 px-2.5 py-1.5 text-xs text-destructive">
                    Reason: {order.cancelReason}
                  </p>
                )}
              </Card>
            </motion.div>
          ))}
        </motion.div>
      )}
    </motion.div>
  );
}
