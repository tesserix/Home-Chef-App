import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  ChefHat,
  Search,
  Filter,
  Star,
  ShoppingBag,
  CheckCircle,
  Clock,
  XCircle,
  Loader2,
  MapPin,
  UtensilsCrossed,
  IndianRupee,
  FileText,
  Mail,
  Phone,
  Circle,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';

interface Chef {
  id: string;
  userId: string;
  businessName: string;
  description: string;
  cuisines: string[];
  specialties: string[];
  profileImage: string;
  rating: number;
  totalReviews: number;
  totalOrders: number;
  totalRevenue: number;
  menuItemCount: number;
  documentCount: number;
  verified: boolean;
  isActive: boolean;
  acceptingOrders: boolean;
  onlineStatus: string;
  ownerName: string;
  ownerEmail: string;
  ownerPhone: string;
  addressLine1: string;
  city: string;
  state: string;
  postalCode: string;
  serviceRadius: number;
  prepTime: string;
  minimumOrder: number;
  createdAt: string;
}

interface ChefsResponse {
  data: Chef[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
}

export default function ChefsPage() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();

  const verifyMutation = useMutation({
    mutationFn: (id: string) => apiClient.put(`/admin/chefs/${id}/verify`),
    onSuccess: () => { toast.success('Chef verified'); queryClient.invalidateQueries({ queryKey: ['admin-chefs'] }); },
    onError: () => toast.error('Failed to verify chef'),
  });

  const rejectMutation = useMutation({
    mutationFn: (id: string) => apiClient.put(`/admin/chefs/${id}/reject`, { reason: 'Rejected by admin' }),
    onSuccess: () => { toast.success('Chef rejected'); queryClient.invalidateQueries({ queryKey: ['admin-chefs'] }); },
    onError: () => toast.error('Failed to reject chef'),
  });

  const suspendMutation = useMutation({
    mutationFn: (id: string) => apiClient.put(`/admin/chefs/${id}/suspend`),
    onSuccess: () => { toast.success('Chef suspended'); queryClient.invalidateQueries({ queryKey: ['admin-chefs'] }); },
    onError: () => toast.error('Failed to suspend chef'),
  });

  const { data, isLoading } = useQuery({
    queryKey: ['admin-chefs', search, statusFilter, page],
    queryFn: () =>
      apiClient.get<ChefsResponse>('/admin/chefs', {
        search: search || undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        page,
        limit: 20,
      }),
  });

  const resp = data as unknown as ChefsResponse | undefined;
  const chefs = resp?.data ?? [];
  const pagination = resp?.pagination;

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Chefs / Kitchens</h1>
        <p className="page-description">
          Manage home chef registrations and kitchen details
          {pagination && <span className="ml-1 font-medium">({pagination.total} total)</span>}
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search by kitchen name or cuisine..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="h-10 w-full rounded-lg border border-input bg-card pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <select
            value={statusFilter}
            onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
            className="h-10 rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="all">All Status</option>
            <option value="pending">Pending Verification</option>
            <option value="approved">Verified</option>
          </select>
        </div>
      </div>

      {/* Chefs List */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : chefs.length === 0 ? (
        <div className="rounded-xl border border-border bg-card p-20 text-center">
          <ChefHat className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <p className="mt-4 text-muted-foreground">No chefs found</p>
        </div>
      ) : (
        <div className="space-y-4">
          {chefs.map((chef) => (
            <div key={chef.id} className="rounded-xl border border-border bg-card shadow-card overflow-hidden">
              {/* Header */}
              <div className="flex items-start justify-between p-6 pb-4">
                <div className="flex items-center gap-4">
                  <div className="relative flex h-14 w-14 items-center justify-center rounded-xl bg-primary/10">
                    {chef.profileImage ? (
                      <img src={chef.profileImage} alt={chef.businessName} className="h-14 w-14 rounded-xl object-cover" />
                    ) : (
                      <ChefHat className="h-7 w-7 text-primary" />
                    )}
                    <OnlineIndicator status={chef.onlineStatus} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-lg font-semibold text-foreground">{chef.businessName || 'Unnamed Kitchen'}</h3>
                      <VerificationBadge verified={chef.verified} />
                    </div>
                    <p className="text-sm text-muted-foreground">
                      by {chef.ownerName} &middot; {chef.ownerEmail}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-1">
                  {!chef.verified && (
                    <button onClick={() => verifyMutation.mutate(chef.id)} title="Verify Kitchen"
                      className="rounded-lg p-1.5 text-muted-foreground hover:bg-success/10 hover:text-success transition-colors">
                      <CheckCircle className="h-4 w-4" />
                    </button>
                  )}
                  {!chef.verified && (
                    <button onClick={() => rejectMutation.mutate(chef.id)} title="Reject Application"
                      className="rounded-lg p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors">
                      <XCircle className="h-4 w-4" />
                    </button>
                  )}
                  {chef.isActive && chef.verified && (
                    <button onClick={() => suspendMutation.mutate(chef.id)} title="Suspend Kitchen"
                      className="rounded-lg p-1.5 text-muted-foreground hover:bg-warning/10 hover:text-warning transition-colors">
                      <XCircle className="h-4 w-4" />
                    </button>
                  )}
                </div>
              </div>

              {/* Stats Row */}
              <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 gap-4 px-6 pb-4">
                <Stat icon={Star} label="Rating" value={chef.rating > 0 ? `${chef.rating.toFixed(1)} (${chef.totalReviews})` : 'No reviews'} />
                <Stat icon={ShoppingBag} label="Orders" value={String(chef.totalOrders)} />
                <Stat icon={IndianRupee} label="Revenue" value={`₹${(chef.totalRevenue || 0).toLocaleString('en-IN', { maximumFractionDigits: 0 })}`} />
                <Stat icon={UtensilsCrossed} label="Menu Items" value={String(chef.menuItemCount)} />
                <Stat icon={FileText} label="Documents" value={String(chef.documentCount)} />
                <Stat icon={Clock} label="Prep Time" value={chef.prepTime || '--'} />
              </div>

              {/* Kitchen Details */}
              <div className="border-t border-border bg-muted/30 px-6 py-4">
                <div className="flex flex-wrap gap-x-8 gap-y-2 text-sm">
                  {chef.city && (
                    <div className="flex items-center gap-1.5 text-muted-foreground">
                      <MapPin className="h-3.5 w-3.5" />
                      {[chef.addressLine1, chef.city, chef.state].filter(Boolean).join(', ')}
                      {chef.postalCode && ` - ${chef.postalCode}`}
                    </div>
                  )}
                  {chef.ownerPhone && (
                    <div className="flex items-center gap-1.5 text-muted-foreground">
                      <Phone className="h-3.5 w-3.5" />
                      {chef.ownerPhone}
                    </div>
                  )}
                  <div className="flex items-center gap-1.5 text-muted-foreground">
                    <Mail className="h-3.5 w-3.5" />
                    {chef.ownerEmail}
                  </div>
                </div>

                {/* Cuisines & Specialties */}
                {(chef.cuisines?.length > 0 || chef.specialties?.length > 0) && (
                  <div className="mt-3 flex flex-wrap gap-1.5">
                    {chef.cuisines?.map((c) => (
                      <span key={c} className="rounded-md bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">{c}</span>
                    ))}
                    {chef.specialties?.map((s) => (
                      <span key={s} className="rounded-md bg-secondary px-2 py-0.5 text-xs text-secondary-foreground">{s}</span>
                    ))}
                  </div>
                )}

                <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
                  <span>Min order: ₹{chef.minimumOrder || 0}</span>
                  <span>Service radius: {chef.serviceRadius || 0} km</span>
                  <span>Joined: {new Date(chef.createdAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {pagination && pagination.totalPages > 1 && (
        <div className="flex items-center justify-between rounded-xl border border-border bg-card px-6 py-3">
          <p className="text-sm text-muted-foreground">
            Page {pagination.page} of {pagination.totalPages} ({pagination.total} kitchens)
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={!pagination.hasPrev}
              className="rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>
            <button
              onClick={() => setPage(p => p + 1)}
              disabled={!pagination.hasNext}
              className="rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function Stat({ icon: Icon, label, value }: { icon: typeof Star; label: string; value: string }) {
  return (
    <div>
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <p className="mt-0.5 text-sm font-medium text-foreground">{value}</p>
    </div>
  );
}

function VerificationBadge({ verified }: { verified: boolean }) {
  if (verified) {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-success/10 px-2 py-0.5 text-xs font-medium text-success">
        <CheckCircle className="h-3 w-3" />
        Verified
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-warning/10 px-2 py-0.5 text-xs font-medium text-warning">
      <XCircle className="h-3 w-3" />
      Pending
    </span>
  );
}

function OnlineIndicator({ status }: { status: string }) {
  const colors: Record<string, string> = {
    online: 'text-success',
    away: 'text-warning',
    offline: 'text-muted-foreground',
  };
  return (
    <Circle className={`absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 fill-current ${colors[status] || colors.offline}`} />
  );
}

