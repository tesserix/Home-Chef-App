import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  ShoppingBag,
  Search,
  Filter,
  MoreHorizontal,
  Loader2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import type { OrderStatus } from '@/shared/types';

interface Order {
  id: string;
  orderNumber: string;
  customerName: string;
  chefName: string;
  status: OrderStatus;
  total: number;
  itemCount: number;
  createdAt: string;
}

const STATUS_STYLES: Record<string, string> = {
  pending: 'bg-warning/10 text-warning',
  accepted: 'bg-info/10 text-info',
  preparing: 'bg-primary/10 text-primary',
  ready: 'bg-success/10 text-success',
  picked_up: 'bg-info/10 text-info',
  delivering: 'bg-info/10 text-info',
  delivered: 'bg-success/10 text-success',
  cancelled: 'bg-destructive/10 text-destructive',
  refunded: 'bg-muted text-muted-foreground',
};

export default function OrdersPage() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const { data: ordersData, isLoading } = useQuery({
    queryKey: ['admin-orders', search, statusFilter],
    queryFn: () =>
      apiClient.get<{ data: Order[]; pagination: unknown }>('/admin/orders', {
        search: search || undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
      }),
  });
  const rawOrders = ordersData as unknown;
  const orders: Order[] = Array.isArray(rawOrders)
    ? rawOrders
    : (rawOrders as { data: Order[] })?.data ?? [];

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Orders</h1>
        <p className="page-description">Monitor and manage all platform orders</p>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search by order number..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-10 w-full rounded-lg border border-input bg-card pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="h-10 rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="all">All Status</option>
            <option value="pending">Pending</option>
            <option value="accepted">Accepted</option>
            <option value="preparing">Preparing</option>
            <option value="ready">Ready</option>
            <option value="delivering">Delivering</option>
            <option value="delivered">Delivered</option>
            <option value="cancelled">Cancelled</option>
            <option value="refunded">Refunded</option>
          </select>
        </div>
      </div>

      {/* Orders Table */}
      <div className="rounded-xl border border-border bg-card shadow-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Order</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Customer</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Chef</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Total</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Date</th>
                  <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-muted-foreground">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {orders?.map((order) => (
                  <tr key={order.id} className="hover:bg-secondary/30 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
                          <ShoppingBag className="h-4 w-4 text-primary" />
                        </div>
                        <div>
                          <p className="font-medium text-foreground">#{order.orderNumber}</p>
                          <p className="text-xs text-muted-foreground">{order.itemCount} items</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-foreground">{order.customerName}</td>
                    <td className="px-6 py-4 text-sm text-foreground">{order.chefName}</td>
                    <td className="px-6 py-4">
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLES[order.status] || 'bg-muted text-muted-foreground'}`}>
                        {order.status.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-foreground">
                      ₹{order.total.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 text-sm text-muted-foreground">
                      {new Date(order.createdAt).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <button className="rounded-lg p-2 hover:bg-secondary transition-colors">
                        <MoreHorizontal className="h-4 w-4 text-muted-foreground" />
                      </button>
                    </td>
                  </tr>
                ))}
                {(!orders || orders.length === 0) && (
                  <tr>
                    <td colSpan={7} className="px-6 py-20 text-center text-muted-foreground">
                      No orders found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
