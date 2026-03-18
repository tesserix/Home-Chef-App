import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Truck,
  Users,
  MapPin,
  Clock,
  IndianRupee,
  Loader2,
  Package,
  Settings2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';

interface DeliveryStats {
  totalPartners: number;
  verifiedPartners: number;
  onlinePartners: number;
  totalDeliveries: number;
  activeDeliveries: number;
  todayDeliveries: number;
  todayEarnings: number;
}

interface DeliveryRecord {
  id: string;
  orderId: string;
  orderNumber?: string;
  orderTotal?: number;
  orderStatus?: string;
  status: string;
  driverName?: string;
  driverPhone?: string;
  vehicleType?: string;
  chefName?: string;
  pickup: { address: string; city: string };
  dropoff: { address: string; city: string };
  distance: number;
  deliveryFee: number;
  tip: number;
  totalPayout: number;
  assignedAt: string;
  pickedUpAt?: string;
  deliveredAt?: string;
  cancelledAt?: string;
  cancelReason?: string;
}

const STATUS_STYLES: Record<string, string> = {
  pending: 'bg-amber-50 text-amber-700',
  assigned: 'bg-blue-50 text-blue-700',
  at_pickup: 'bg-indigo-50 text-indigo-700',
  picked_up: 'bg-purple-50 text-purple-700',
  in_transit: 'bg-cyan-50 text-cyan-700',
  at_dropoff: 'bg-teal-50 text-teal-700',
  delivered: 'bg-green-50 text-green-700',
  failed: 'bg-red-50 text-red-700',
  returned: 'bg-orange-50 text-orange-700',
  cancelled: 'bg-gray-100 text-gray-600',
};

const STATUS_LABELS: Record<string, string> = {
  pending: 'Pending',
  assigned: 'Assigned',
  at_pickup: 'At Pickup',
  picked_up: 'Picked Up',
  in_transit: 'In Transit',
  at_dropoff: 'At Dropoff',
  delivered: 'Delivered',
  failed: 'Failed',
  returned: 'Returned',
  cancelled: 'Cancelled',
};

function formatCurrency(amount: number) {
  return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amount);
}

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

export default function DeliveryPage() {
  const [statusFilter, setStatusFilter] = useState('all');
  const [page, setPage] = useState(1);
  const navigate = useNavigate();

  const { data: stats } = useQuery({
    queryKey: ['admin-delivery-stats'],
    queryFn: () => apiClient.get<DeliveryStats>('/admin/delivery/stats'),
  });

  const { data: deliveriesData, isLoading } = useQuery({
    queryKey: ['admin-deliveries', statusFilter, page],
    queryFn: () =>
      apiClient.get<{ data: DeliveryRecord[]; pagination: { total: number; totalPages: number; hasNext: boolean; hasPrev: boolean } }>(
        '/admin/delivery/list',
        {
          status: statusFilter !== 'all' ? statusFilter : undefined,
          page,
          limit: 20,
        }
      ),
  });

  const deliveries: DeliveryRecord[] = (deliveriesData as { data: DeliveryRecord[] })?.data ?? [];
  const pagination = (deliveriesData as { pagination: { total: number; totalPages: number; hasNext: boolean; hasPrev: boolean } })?.pagination;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="page-header">
          <h1 className="page-title">Delivery Management</h1>
          <p className="page-description">Track deliveries, partners, and payouts</p>
        </div>
        <button
          onClick={() => navigate('/delivery/providers')}
          className="inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2.5 text-sm font-medium hover:bg-secondary transition-colors"
        >
          <Settings2 className="h-4 w-4" />
          Manage Providers
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Users className="h-5 w-5 text-primary" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">{stats?.totalPartners ?? 0}</p>
          <p className="text-xs text-muted-foreground">{stats?.onlinePartners ?? 0} online now</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-50">
              <Truck className="h-5 w-5 text-amber-600" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">{stats?.activeDeliveries ?? 0}</p>
          <p className="text-xs text-muted-foreground">Active deliveries</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-50">
              <Package className="h-5 w-5 text-green-600" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">{stats?.todayDeliveries ?? 0}</p>
          <p className="text-xs text-muted-foreground">Delivered today</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-50">
              <IndianRupee className="h-5 w-5 text-emerald-600" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">{formatCurrency(stats?.todayEarnings ?? 0)}</p>
          <p className="text-xs text-muted-foreground">Today's payouts</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
          className="rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground"
        >
          <option value="all">All Statuses</option>
          {Object.entries(STATUS_LABELS).map(([val, label]) => (
            <option key={val} value={val}>{label}</option>
          ))}
        </select>
        {pagination && (
          <span className="text-sm text-muted-foreground ml-auto">
            {pagination.total} total deliveries
          </span>
        )}
      </div>

      {/* Deliveries Table */}
      <div className="rounded-xl border border-border bg-card shadow-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Order</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Driver</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">From / To</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Payout</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-muted-foreground">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center">
                    <Loader2 className="h-6 w-6 animate-spin text-primary mx-auto" />
                  </td>
                </tr>
              ) : deliveries.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-sm text-muted-foreground">
                    No deliveries found
                  </td>
                </tr>
              ) : (
                deliveries.map((d) => (
                  <tr key={d.id} className="hover:bg-muted/20 transition-colors">
                    <td className="px-4 py-3">
                      <div>
                        <p className="text-sm font-medium text-foreground">
                          #{d.orderNumber || d.orderId.slice(0, 8)}
                        </p>
                        {d.chefName && (
                          <p className="text-xs text-muted-foreground">{d.chefName}</p>
                        )}
                        {d.orderTotal != null && (
                          <p className="text-xs text-muted-foreground">{formatCurrency(d.orderTotal)}</p>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div>
                        <p className="text-sm font-medium text-foreground">{d.driverName || 'Unassigned'}</p>
                        {d.driverPhone && (
                          <p className="text-xs text-muted-foreground">{d.driverPhone}</p>
                        )}
                        {d.vehicleType && (
                          <p className="text-xs text-muted-foreground capitalize">{d.vehicleType}</p>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="space-y-1">
                        <div className="flex items-start gap-1.5">
                          <MapPin className="h-3 w-3 text-green-500 mt-0.5 shrink-0" />
                          <p className="text-xs text-foreground truncate max-w-[180px]">
                            {d.pickup?.address || d.pickup?.city || '-'}
                          </p>
                        </div>
                        <div className="flex items-start gap-1.5">
                          <MapPin className="h-3 w-3 text-red-500 mt-0.5 shrink-0" />
                          <p className="text-xs text-foreground truncate max-w-[180px]">
                            {d.dropoff?.address || d.dropoff?.city || '-'}
                          </p>
                        </div>
                        {d.distance > 0 && (
                          <p className="text-xs text-muted-foreground">{d.distance.toFixed(1)} km</p>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_STYLES[d.status] || 'bg-muted text-muted-foreground'}`}>
                        {STATUS_LABELS[d.status] || d.status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div>
                        <p className="text-sm font-medium text-foreground">{formatCurrency(d.totalPayout)}</p>
                        <div className="flex gap-2 text-xs text-muted-foreground">
                          <span>Fee: {formatCurrency(d.deliveryFee)}</span>
                          {d.tip > 0 && <span>Tip: {formatCurrency(d.tip)}</span>}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        {timeAgo(d.deliveredAt || d.assignedAt)}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {pagination && pagination.totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-border px-4 py-3">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={!pagination.hasPrev}
              className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary disabled:opacity-50"
            >
              Previous
            </button>
            <span className="text-xs text-muted-foreground">
              Page {page} of {pagination.totalPages}
            </span>
            <button
              onClick={() => setPage((p) => p + 1)}
              disabled={!pagination.hasNext}
              className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
