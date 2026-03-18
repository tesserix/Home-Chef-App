import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Loader2,
  Power,
  Zap,
  Truck,
  CheckCircle,
  Clock,
  Activity,
  Globe,
  MapPin,
  IndianRupee,
  Mail,
  Phone,
  User,
  Trash2,
  Pencil,
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
  activeDeliveries?: number;
}

interface ProviderStats {
  totalDeliveries: number;
  completedDeliveries: number;
  failedDeliveries: number;
  cancelledDeliveries: number;
  activeDeliveries: number;
  avgCost: number;
  totalCost: number;
  avgDeliveryTime: number;
  dailyStats: DailyStat[];
}

interface DailyStat {
  date: string;
  deliveries: number;
  completed: number;
  cost: number;
}

interface TestResult {
  success: boolean;
  responseTimeMs: number;
  error?: string;
}

// ---------- Constants ----------

const PRICING_LABELS: Record<string, string> = {
  per_delivery: 'Per Delivery',
  per_km: 'Per KM',
  flat_rate: 'Flat Rate',
};

// ---------- Main Component ----------

export default function ProviderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [confirmAction, setConfirmAction] = useState<'delete' | 'toggle' | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-delivery-provider', id],
    queryFn: () => apiClient.get<DeliveryProvider>(`/admin/delivery/providers/${id}`),
    enabled: !!id,
  });

  const { data: statsData } = useQuery({
    queryKey: ['admin-delivery-provider-stats', id],
    queryFn: () => apiClient.get<ProviderStats>(`/admin/delivery/providers/${id}/stats`),
    enabled: !!id,
  });

  const provider = data as unknown as DeliveryProvider | undefined;
  const stats = statsData as unknown as ProviderStats | undefined;

  const toggleMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/delivery/providers/${id}/toggle`),
    onSuccess: () => {
      toast.success('Provider status updated');
      queryClient.invalidateQueries({ queryKey: ['admin-delivery-provider', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-delivery-providers'] });
      setConfirmAction(null);
    },
    onError: () => toast.error('Failed to toggle provider'),
  });

  const deleteMutation = useMutation({
    mutationFn: () => apiClient.delete(`/admin/delivery/providers/${id}`),
    onSuccess: () => {
      toast.success('Provider deleted');
      navigate('/delivery/providers');
    },
    onError: () => toast.error('Failed to delete provider'),
  });

  const testMutation = useMutation({
    mutationFn: () =>
      apiClient.post<TestResult>(`/admin/delivery/providers/${id}/test`),
    onSuccess: (data) => {
      const result = data as unknown as TestResult;
      if (result.success) {
        toast.success(`Connection successful (${result.responseTimeMs}ms)`);
      } else {
        toast.error(`Connection failed: ${result.error || 'Unknown error'}`);
      }
    },
    onError: () => toast.error('Failed to test connection'),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!provider) {
    return (
      <div className="py-20 text-center">
        <p className="text-muted-foreground">Provider not found</p>
        <button
          onClick={() => navigate('/delivery/providers')}
          className="mt-4 text-sm text-primary hover:underline"
        >
          Back to Providers
        </button>
      </div>
    );
  }

  const currency = provider.currency || 'INR';
  const fmt = new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency,
    maximumFractionDigits: 0,
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/delivery/providers')}
          className="rounded-lg p-2 hover:bg-secondary transition-colors"
        >
          <ArrowLeft className="h-5 w-5 text-muted-foreground" />
        </button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-foreground">{provider.name}</h1>
            <span className="text-sm text-muted-foreground font-mono">{provider.code}</span>
          </div>
          {provider.description && (
            <p className="text-sm text-muted-foreground mt-0.5">{provider.description}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => testMutation.mutate()}
            disabled={testMutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium hover:bg-secondary transition-colors disabled:opacity-50"
          >
            {testMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Zap className="h-4 w-4" />
            )}
            Test Connection
          </button>
          <button
            onClick={() => setConfirmAction('toggle')}
            className={`inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${
              provider.isEnabled
                ? 'border-warning/30 text-warning hover:bg-warning/10'
                : 'border-success/30 text-success hover:bg-success/10'
            }`}
          >
            <Power className="h-4 w-4" />
            {provider.isEnabled ? 'Disable' : 'Enable'}
          </button>
        </div>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Truck className="h-5 w-5 text-primary" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">
            {(stats?.totalDeliveries ?? provider.totalDeliveries).toLocaleString()}
          </p>
          <p className="text-xs text-muted-foreground">Total Deliveries</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-success/10">
              <CheckCircle className="h-5 w-5 text-success" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">
            {provider.successRate.toFixed(1)}%
          </p>
          <p className="text-xs text-muted-foreground">Success Rate</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-50">
              <Clock className="h-5 w-5 text-amber-600" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">
            {(stats?.avgDeliveryTime ?? provider.avgDeliveryTime).toFixed(0)} min
          </p>
          <p className="text-xs text-muted-foreground">Avg Delivery Time</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-5 shadow-card">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info/10">
              <Activity className="h-5 w-5 text-info" />
            </div>
          </div>
          <p className="text-2xl font-bold text-foreground">
            {(stats?.activeDeliveries ?? provider.activeDeliveries ?? 0).toLocaleString()}
          </p>
          <p className="text-xs text-muted-foreground">Active Deliveries</p>
        </div>
      </div>

      {/* Main Content */}
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          {/* API Configuration */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">API Configuration</h2>
            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">API Base URL</span>
                <code className="text-sm font-mono text-foreground">{provider.apiBaseUrl || '--'}</code>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">API Key</span>
                <span className="text-sm text-foreground font-mono">****</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">Webhook Secret</span>
                <span className="text-sm text-foreground font-mono">****</span>
              </div>
            </div>

            {/* Status Mapping */}
            {provider.statusMapping && Object.keys(provider.statusMapping).length > 0 && (
              <div className="mt-6">
                <h3 className="text-sm font-medium text-foreground mb-3">Status Mapping</h3>
                <div className="rounded-lg border border-border overflow-hidden">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-border bg-muted/30">
                        <th className="px-4 py-2 text-left text-xs font-medium uppercase text-muted-foreground">Provider Status</th>
                        <th className="px-4 py-2 text-left text-xs font-medium uppercase text-muted-foreground">Fe3dr Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {Object.entries(provider.statusMapping).map(([key, value]) => (
                        <tr key={key}>
                          <td className="px-4 py-2 text-sm text-foreground font-mono">{key}</td>
                          <td className="px-4 py-2 text-sm text-foreground">{value}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>

          {/* Pricing */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Pricing</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <InfoRow icon={IndianRupee} label="Pricing Model" value={PRICING_LABELS[provider.pricingModel] || provider.pricingModel} />
              <InfoRow icon={IndianRupee} label="Base Cost" value={fmt.format(provider.baseCost)} />
              <InfoRow icon={IndianRupee} label="Per KM Cost" value={fmt.format(provider.perKmCost)} />
              <InfoRow icon={IndianRupee} label="Currency" value={currency} />
              <InfoRow icon={MapPin} label="Max Distance" value={`${provider.maxDistance} km`} />
            </div>
          </div>

          {/* Coverage */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Coverage</h2>
            {provider.supportedCountries && provider.supportedCountries.length > 0 && (
              <div className="mb-4">
                <p className="text-sm font-medium text-foreground mb-2">Supported Countries</p>
                <div className="flex flex-wrap gap-2">
                  {provider.supportedCountries.map((country) => (
                    <span
                      key={country}
                      className="inline-flex items-center rounded-full border border-border bg-secondary/30 px-3 py-1 text-xs font-medium text-foreground"
                    >
                      {country}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {provider.supportedCities && provider.supportedCities.length > 0 && (
              <div className="mb-4">
                <p className="text-sm font-medium text-foreground mb-2">Supported Cities</p>
                <div className="flex flex-wrap gap-2">
                  {provider.supportedCities.map((city) => (
                    <span
                      key={city}
                      className="inline-flex items-center rounded-full border border-border bg-secondary/30 px-3 py-1 text-xs font-medium text-foreground"
                    >
                      {city}
                    </span>
                  ))}
                </div>
              </div>
            )}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
              <InfoRow icon={Truck} label="Max Concurrent Deliveries" value={String(provider.maxConcurrentDeliveries)} />
              <InfoRow icon={Truck} label="Daily Limit" value={provider.dailyLimit === 0 ? 'Unlimited' : String(provider.dailyLimit)} />
            </div>
          </div>

          {/* Rate Limits & Config */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Rate Limits & Config</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <InfoRow icon={Activity} label="Max Concurrent" value={String(provider.maxConcurrentDeliveries)} />
              <InfoRow icon={Activity} label="Daily Limit" value={provider.dailyLimit === 0 ? 'Unlimited' : String(provider.dailyLimit)} />
              <InfoRow icon={Clock} label="Avg Pickup Time" value={`${provider.avgPickupTime} min`} />
              <InfoRow icon={Activity} label="Priority" value={String(provider.priority)} />
            </div>
          </div>

          {/* Performance (last 30 days) */}
          {stats?.dailyStats && stats.dailyStats.length > 0 && (
            <div className="rounded-xl border border-border bg-card p-6 shadow-card">
              <h2 className="text-lg font-semibold text-foreground mb-4">Performance (Last 30 Days)</h2>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
                <div className="rounded-lg bg-muted/50 p-3">
                  <p className="text-xs text-muted-foreground">Completed</p>
                  <p className="text-lg font-bold text-foreground">{stats.completedDeliveries}</p>
                </div>
                <div className="rounded-lg bg-muted/50 p-3">
                  <p className="text-xs text-muted-foreground">Failed</p>
                  <p className="text-lg font-bold text-destructive">{stats.failedDeliveries}</p>
                </div>
                <div className="rounded-lg bg-muted/50 p-3">
                  <p className="text-xs text-muted-foreground">Cancelled</p>
                  <p className="text-lg font-bold text-warning">{stats.cancelledDeliveries}</p>
                </div>
                <div className="rounded-lg bg-muted/50 p-3">
                  <p className="text-xs text-muted-foreground">Total Cost</p>
                  <p className="text-lg font-bold text-foreground">{fmt.format(stats.totalCost)}</p>
                </div>
              </div>

              {/* Daily Stats Table */}
              <div className="rounded-lg border border-border overflow-hidden">
                <div className="overflow-x-auto max-h-80">
                  <table className="w-full">
                    <thead className="sticky top-0">
                      <tr className="border-b border-border bg-muted/30">
                        <th className="px-4 py-2 text-left text-xs font-medium uppercase text-muted-foreground">Date</th>
                        <th className="px-4 py-2 text-right text-xs font-medium uppercase text-muted-foreground">Deliveries</th>
                        <th className="px-4 py-2 text-right text-xs font-medium uppercase text-muted-foreground">Completed</th>
                        <th className="px-4 py-2 text-right text-xs font-medium uppercase text-muted-foreground">Cost</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {stats.dailyStats.map((day) => (
                        <tr key={day.date}>
                          <td className="px-4 py-2 text-sm text-foreground">
                            {new Date(day.date).toLocaleDateString('en-IN', {
                              day: 'numeric',
                              month: 'short',
                            })}
                          </td>
                          <td className="px-4 py-2 text-sm text-foreground text-right">{day.deliveries}</td>
                          <td className="px-4 py-2 text-sm text-foreground text-right">{day.completed}</td>
                          <td className="px-4 py-2 text-sm text-foreground text-right">{fmt.format(day.cost)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Right Column */}
        <div className="space-y-6">
          {/* Status Card */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Status</h2>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground">Enabled</span>
                {provider.isEnabled ? (
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-success/10 px-2.5 py-0.5 text-xs font-medium text-success">
                    <span className="h-1.5 w-1.5 rounded-full bg-success" />
                    Yes
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-destructive/10 px-2.5 py-0.5 text-xs font-medium text-destructive">
                    <span className="h-1.5 w-1.5 rounded-full bg-destructive" />
                    No
                  </span>
                )}
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground">Active</span>
                {provider.isActive ? (
                  <span className="inline-flex items-center rounded-full bg-success/10 px-2.5 py-0.5 text-xs font-medium text-success">
                    Active
                  </span>
                ) : (
                  <span className="inline-flex items-center rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
                    Inactive
                  </span>
                )}
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground">Last Used</span>
                <span className="text-sm text-muted-foreground">
                  {provider.lastUsedAt
                    ? new Date(provider.lastUsedAt).toLocaleDateString('en-IN', {
                        day: 'numeric',
                        month: 'short',
                        year: 'numeric',
                      })
                    : 'Never'}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-foreground">Created</span>
                <span className="text-sm text-muted-foreground">
                  {new Date(provider.createdAt).toLocaleDateString('en-IN', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                  })}
                </span>
              </div>
            </div>
          </div>

          {/* Contact Info */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Contact Info</h2>
            <div className="space-y-3">
              <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/20 p-3">
                <User className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-muted-foreground">Name</p>
                  <p className="text-sm font-medium text-foreground truncate">{provider.contactName || 'Not provided'}</p>
                </div>
              </div>
              <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/20 p-3">
                <Mail className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-muted-foreground">Email</p>
                  <p className="text-sm font-medium text-foreground truncate">{provider.contactEmail || 'Not provided'}</p>
                </div>
              </div>
              <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/20 p-3">
                <Phone className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-muted-foreground">Phone</p>
                  <p className="text-sm font-medium text-foreground truncate">{provider.contactPhone || 'Not provided'}</p>
                </div>
              </div>
              {provider.notes && (
                <div className="rounded-lg border border-border bg-secondary/20 p-3">
                  <p className="text-xs text-muted-foreground mb-1">Notes</p>
                  <p className="text-sm text-foreground whitespace-pre-wrap">{provider.notes}</p>
                </div>
              )}
            </div>
          </div>

          {/* Actions */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Actions</h2>
            <div className="space-y-3">
              <button
                onClick={() => navigate(`/delivery/providers/${id}/edit`)}
                className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-border px-4 py-2.5 text-sm font-medium hover:bg-secondary transition-colors"
              >
                <Pencil className="h-4 w-4" />
                Edit Provider
              </button>
              <button
                onClick={() => setConfirmAction('delete')}
                className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-destructive/30 px-4 py-2.5 text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
              >
                <Trash2 className="h-4 w-4" />
                Delete Provider
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Confirmation Dialog */}
      {confirmAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-foreground/50" onClick={() => setConfirmAction(null)} />
          <div className="relative z-50 w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl mx-4">
            <h3 className="text-lg font-semibold text-foreground mb-2">
              {confirmAction === 'delete' ? 'Delete Provider' : provider.isEnabled ? 'Disable Provider' : 'Enable Provider'}
            </h3>
            <p className="text-sm text-muted-foreground mb-6">
              {confirmAction === 'delete'
                ? 'Are you sure you want to delete this delivery provider? This action cannot be undone.'
                : provider.isEnabled
                ? 'Are you sure you want to disable this provider? No new deliveries will be assigned to it.'
                : 'Are you sure you want to enable this provider? It will start receiving delivery assignments.'}
            </p>
            <div className="flex items-center justify-end gap-3">
              <button
                onClick={() => setConfirmAction(null)}
                className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium hover:bg-secondary transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  if (confirmAction === 'delete') {
                    deleteMutation.mutate();
                  } else {
                    toggleMutation.mutate();
                  }
                }}
                disabled={deleteMutation.isPending || toggleMutation.isPending}
                className={`inline-flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors disabled:opacity-50 ${
                  confirmAction === 'delete'
                    ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
                    : 'bg-primary text-primary-foreground hover:bg-primary/90'
                }`}
              >
                {(deleteMutation.isPending || toggleMutation.isPending) && (
                  <Loader2 className="h-4 w-4 animate-spin" />
                )}
                {confirmAction === 'delete' ? 'Delete' : provider.isEnabled ? 'Disable' : 'Enable'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ---------- Shared Components ----------

function InfoRow({ icon: Icon, label, value }: { icon: typeof Globe; label: string; value: string }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/20 p-3">
      <Icon className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm font-medium text-foreground truncate">{value}</p>
      </div>
    </div>
  );
}
