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
import { Button } from '@/shared/components/ui/Button';

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
        <Button
          variant="primary"
          leftIcon={<Plus className="h-4 w-4" />}
          onClick={() => navigate('/delivery/providers/new')}
        >
          Add Provider
        </Button>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <label htmlFor="providers-search" className="sr-only">Search providers</label>
          <Search aria-hidden="true" className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            id="providers-search"
            type="search"
            placeholder="Search providers..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="h-10 w-full rounded-lg border border-input bg-card pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
          <label htmlFor="providers-status-filter" className="sr-only">Filter by status</label>
          <select
            id="providers-status-filter"
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
                              loading="lazy"
                              decoding="async"
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
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          title="View Details"
                          aria-label={`View ${provider.name} details`}
                          onClick={() => navigate(`/delivery/providers/${provider.id}`)}
                          className="text-muted-foreground hover:bg-secondary hover:text-foreground"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          disabled={toggleMutation.isPending}
                          title={provider.isEnabled ? 'Disable' : 'Enable'}
                          aria-label={`${provider.isEnabled ? 'Disable' : 'Enable'} ${provider.name}`}
                          onClick={() => handleToggle(provider)}
                          className={
                            provider.isEnabled
                              ? 'text-muted-foreground hover:bg-warning/10 hover:text-warning'
                              : 'text-muted-foreground hover:bg-success/10 hover:text-success'
                          }
                        >
                          <Power className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          disabled={deleteMutation.isPending}
                          title="Delete"
                          aria-label={`Delete ${provider.name}`}
                          onClick={() => handleDelete(provider.id)}
                          className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
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
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.hasPrev}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.hasNext}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      {confirmDeleteId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <button type="button" aria-label="Close" className="fixed inset-0 bg-foreground/50" onClick={() => setConfirmDeleteId(null)} />
          <div className="relative z-50 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-3 mx-4">
            <h3 className="text-lg font-semibold text-foreground mb-2">Delete Provider</h3>
            <p className="text-sm text-muted-foreground mb-6">
              Are you sure you want to delete this delivery provider? This action cannot be undone.
            </p>
            <div className="flex items-center justify-end gap-3">
              <Button variant="outline" onClick={() => setConfirmDeleteId(null)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                isLoading={deleteMutation.isPending}
                disabled={deleteMutation.isPending}
                onClick={confirmDelete}
              >
                Delete
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Toggle Disable Confirmation Dialog */}
      {confirmToggleId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <button type="button" aria-label="Close" className="fixed inset-0 bg-foreground/50" onClick={() => setConfirmToggleId(null)} />
          <div className="relative z-50 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-3 mx-4">
            <h3 className="text-lg font-semibold text-foreground mb-2">Disable Provider</h3>
            <p className="text-sm text-muted-foreground mb-6">
              Are you sure you want to disable this delivery provider? No new deliveries will be assigned to it.
            </p>
            <div className="flex items-center justify-end gap-3">
              <Button variant="outline" onClick={() => setConfirmToggleId(null)}>
                Cancel
              </Button>
              <Button
                isLoading={toggleMutation.isPending}
                disabled={toggleMutation.isPending}
                onClick={confirmToggle}
                className="bg-warning text-warning-foreground hover:bg-warning/90"
              >
                Disable
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
