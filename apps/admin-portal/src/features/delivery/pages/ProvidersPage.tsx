import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  Filter,
  Loader2,
  Eye,
  Plus,
  Power,
  Trash2,
  Truck,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';

// ---------- Types ----------

interface DeliveryProvider {
  id: string;
  name: string;
  code: string;
  description: string;
  logoUrl: string;
  apiBaseUrl: string;
  statusMapping: Record<string, string>;
  supportedCities: string[];
  supportedCountries: string[];
  maxDistance: number;
  avgPickupTime: number;
  pricingModel: string;
  baseCost: number;
  perKmCost: number;
  currency: string;
  priority: number;
  isEnabled: boolean;
  isActive: boolean;
  maxConcurrentDeliveries: number;
  dailyLimit: number;
  totalDeliveries: number;
  successRate: number;
  avgDeliveryTime: number;
  lastUsedAt: string | null;
  contactName: string;
  contactEmail: string;
  contactPhone: string;
  notes: string;
  createdAt: string;
}

interface ProviderListResponse {
  data: DeliveryProvider[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
}

// ---------- Constants ----------

const PRICING_LABELS: Record<string, string> = {
  per_delivery: 'Per Delivery',
  per_km: 'Per KM',
  flat_rate: 'Flat Rate',
};

// ---------- Main Component ----------

export default function ProvidersPage() {
  const [search, setSearch] = useState('');
  const [enabledFilter, setEnabledFilter] = useState<string>('all');
  const [page, setPage] = useState(1);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-delivery-providers', search, enabledFilter, page],
    queryFn: () =>
      apiClient.get<ProviderListResponse>('/admin/delivery/providers', {
        search: search || undefined,
        enabled: enabledFilter !== 'all' ? enabledFilter : undefined,
        page,
        limit: 20,
      }),
  });

  const resp = data as unknown as ProviderListResponse | undefined;
  const providers = resp?.data ?? [];
  const pagination = resp?.pagination;

  const toggleMutation = useMutation({
    mutationFn: (id: string) =>
      apiClient.put(`/admin/delivery/providers/${id}/toggle`),
    onSuccess: () => {
      toast.success('Provider status updated');
      queryClient.invalidateQueries({ queryKey: ['admin-delivery-providers'] });
    },
    onError: () => toast.error('Failed to toggle provider'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) =>
      apiClient.delete(`/admin/delivery/providers/${id}`),
    onSuccess: () => {
      toast.success('Provider deleted');
      queryClient.invalidateQueries({ queryKey: ['admin-delivery-providers'] });
    },
    onError: () => toast.error('Failed to delete provider'),
  });

  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [confirmToggleId, setConfirmToggleId] = useState<string | null>(null);

  const handleToggle = (provider: DeliveryProvider) => {
    if (provider.isEnabled) {
      setConfirmToggleId(provider.id);
    } else {
      toggleMutation.mutate(provider.id);
    }
  };

  const handleDelete = (id: string) => {
    setConfirmDeleteId(id);
  };

  const confirmDelete = () => {
    if (confirmDeleteId) {
      deleteMutation.mutate(confirmDeleteId);
      setConfirmDeleteId(null);
    }
  };

  const confirmToggle = () => {
    if (confirmToggleId) {
      toggleMutation.mutate(confirmToggleId);
      setConfirmToggleId(null);
    }
  };

  function formatCost(provider: DeliveryProvider) {
    const currency = provider.currency || 'INR';
    const fmt = new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency,
      maximumFractionDigits: 0,
    });
    if (provider.pricingModel === 'per_km') {
      return `${fmt.format(provider.baseCost)} + ${fmt.format(provider.perKmCost)}/km`;
    }
    return fmt.format(provider.baseCost);
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="page-header">
          <h1 className="page-title">Delivery Providers</h1>
          <p className="page-description">
            {pagination ? `${pagination.total} provider${pagination.total !== 1 ? 's' : ''}` : 'Manage delivery provider integrations'}
          </p>
        </div>
        <button
          onClick={() => navigate('/delivery/providers/new')}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Add Provider
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search providers..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="h-10 w-full rounded-lg border border-input bg-card pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <select
            value={enabledFilter}
            onChange={(e) => { setEnabledFilter(e.target.value); setPage(1); }}
            className="h-10 rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="all">All Statuses</option>
            <option value="true">Enabled</option>
            <option value="false">Disabled</option>
          </select>
        </div>
      </div>

      {/* Providers Table */}
      <div className="rounded-xl border border-border bg-card shadow-card">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Provider</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">API Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Priority</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Pricing</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Cities</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Deliveries</th>
                  <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">Success Rate</th>
                  <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-muted-foreground">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {providers.map((provider) => (
                  <tr key={provider.id} className="hover:bg-secondary/30 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                          {provider.logoUrl ? (
                            <img
                              src={provider.logoUrl}
                              alt={provider.name}
                              className="h-8 w-8 rounded object-contain"
                            />
                          ) : (
                            <Truck className="h-5 w-5 text-primary" />
                          )}
                        </div>
                        <div>
                          <p className="font-medium text-foreground">{provider.name}</p>
                          <p className="text-xs text-muted-foreground">{provider.code}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      {provider.isEnabled ? (
                        <span className="inline-flex items-center rounded-full bg-success/10 px-2.5 py-0.5 text-xs font-medium text-success">
                          Enabled
                        </span>
                      ) : (
                        <span className="inline-flex items-center rounded-full bg-destructive/10 px-2.5 py-0.5 text-xs font-medium text-destructive">
                          Disabled
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-sm text-foreground">
                      {provider.priority}
                    </td>
                    <td className="px-6 py-4">
                      <div>
                        <p className="text-sm text-foreground">
                          {PRICING_LABELS[provider.pricingModel] || provider.pricingModel}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {formatCost(provider)}
                        </p>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-foreground">
                      {provider.supportedCities?.length ?? 0}
                    </td>
                    <td className="px-6 py-4 text-sm text-foreground">
                      {provider.totalDeliveries.toLocaleString()}
                    </td>
                    <td className="px-6 py-4">
                      <span className={`text-sm font-medium ${
                        provider.successRate >= 90
                          ? 'text-success'
                          : provider.successRate >= 70
                          ? 'text-warning'
                          : 'text-destructive'
                      }`}>
                        {provider.successRate.toFixed(1)}%
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => navigate(`/delivery/providers/${provider.id}`)}
                          title="View Details"
                          className="rounded-lg p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleToggle(provider)}
                          disabled={toggleMutation.isPending}
                          title={provider.isEnabled ? 'Disable' : 'Enable'}
                          className={`rounded-lg p-1.5 transition-colors ${
                            provider.isEnabled
                              ? 'text-muted-foreground hover:bg-warning/10 hover:text-warning'
                              : 'text-muted-foreground hover:bg-success/10 hover:text-success'
                          }`}
                        >
                          <Power className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(provider.id)}
                          disabled={deleteMutation.isPending}
                          title="Delete"
                          className="rounded-lg p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {providers.length === 0 && (
                  <tr>
                    <td colSpan={8} className="px-6 py-20 text-center text-muted-foreground">
                      No delivery providers found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {pagination && pagination.totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-border px-6 py-3">
            <p className="text-sm text-muted-foreground">
              Page {pagination.page} of {pagination.totalPages}
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={!pagination.hasPrev}
                className="rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Previous
              </button>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={!pagination.hasNext}
                className="rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      {confirmDeleteId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-foreground/50" onClick={() => setConfirmDeleteId(null)} />
          <div className="relative z-50 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl mx-4">
            <h3 className="text-lg font-semibold text-foreground mb-2">Delete Provider</h3>
            <p className="text-sm text-muted-foreground mb-6">
              Are you sure you want to delete this delivery provider? This action cannot be undone.
            </p>
            <div className="flex items-center justify-end gap-3">
              <button
                onClick={() => setConfirmDeleteId(null)}
                className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium hover:bg-secondary transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={confirmDelete}
                disabled={deleteMutation.isPending}
                className="inline-flex items-center gap-2 rounded-lg bg-destructive px-4 py-2.5 text-sm font-medium text-destructive-foreground hover:bg-destructive/90 transition-colors disabled:opacity-50"
              >
                {deleteMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Toggle Disable Confirmation Dialog */}
      {confirmToggleId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-foreground/50" onClick={() => setConfirmToggleId(null)} />
          <div className="relative z-50 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl mx-4">
            <h3 className="text-lg font-semibold text-foreground mb-2">Disable Provider</h3>
            <p className="text-sm text-muted-foreground mb-6">
              Are you sure you want to disable this delivery provider? No new deliveries will be assigned to it.
            </p>
            <div className="flex items-center justify-end gap-3">
              <button
                onClick={() => setConfirmToggleId(null)}
                className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium hover:bg-secondary transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={confirmToggle}
                disabled={toggleMutation.isPending}
                className="inline-flex items-center gap-2 rounded-lg bg-warning px-4 py-2.5 text-sm font-medium text-warning-foreground hover:bg-warning/90 transition-colors disabled:opacity-50"
              >
                {toggleMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                Disable
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
