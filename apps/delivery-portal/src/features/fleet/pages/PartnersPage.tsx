import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { useNavigate } from 'react-router-dom';
import {
  Search,
  Filter,
  Star,
  ShieldCheck,
  Wifi,
  ChevronLeft,
  ChevronRight,
  Truck,
} from 'lucide-react';
import { PageLoader } from '@/shared/components/LoadingScreen';

interface Partner {
  id: string;
  name: string;
  email: string;
  phone: string;
  vehicleType: string;
  vehicleNumber: string;
  isVerified: boolean;
  isOnline: boolean;
  rating: number;
  totalDeliveries: number;
  status: string;
}

interface PartnersResponse {
  data: Partner[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
}

const statusOptions = [
  { value: '', label: 'All' },
  { value: 'verified', label: 'Verified' },
  { value: 'pending', label: 'Pending' },
  { value: 'online', label: 'Online' },
];

export default function PartnersPage() {
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const limit = 12;

  const { data, isLoading } = useQuery({
    queryKey: ['fleet-partners', page, search, status],
    queryFn: () =>
      apiClient.get<PartnersResponse>('/delivery/staff/fleet/partners', {
        page,
        limit,
        search: search || undefined,
        status: status || undefined,
      }),
  });

  if (isLoading && page === 1) return <PageLoader />;

  const partners = data?.data ?? [];
  const total = data?.pagination?.total ?? 0;
  const totalPages = data?.pagination?.totalPages ?? 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="page-title">Delivery Partners</h1>
        <p className="page-description">
          {total} partner{total !== 1 ? 's' : ''} registered
        </p>
      </div>

      {/* Search & Filters */}
      <div className="flex flex-col gap-3 sm:flex-row">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search by name, email, or vehicle..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            className="w-full rounded-lg border border-border bg-background pl-9 pr-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          {statusOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => {
                setStatus(opt.value);
                setPage(1);
              }}
              className={`rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                status === opt.value
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Partner Cards */}
      {partners.length === 0 ? (
        <div className="rounded-xl border border-border bg-card p-12 text-center">
          <Truck className="mx-auto h-10 w-10 text-muted-foreground mb-3" />
          <p className="text-sm text-muted-foreground">
            No partners found matching your criteria.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {partners.map((partner) => (
            <button
              key={partner.id}
              onClick={() => navigate(`/fleet/partners/${partner.id}`)}
              className="rounded-xl border border-border bg-card p-5 text-left transition-colors hover:bg-secondary"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                    <span className="text-sm font-semibold text-primary">
                      {(partner.name || 'P').charAt(0).toUpperCase()}
                    </span>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {partner.name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {partner.email}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-1.5">
                  {partner.isVerified && (
                    <ShieldCheck className="h-4 w-4 text-success" />
                  )}
                  {partner.isOnline && (
                    <Wifi className="h-4 w-4 text-success" />
                  )}
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground">Vehicle</span>
                  <span className="font-medium text-foreground">
                    {partner.vehicleType}
                    {partner.vehicleNumber ? ` - ${partner.vehicleNumber}` : ''}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground">Rating</span>
                  <span className="flex items-center gap-1 font-medium text-foreground">
                    <Star className="h-3.5 w-3.5 text-warning fill-warning" />
                    {(partner.rating ?? 0).toFixed(1)}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground">Deliveries</span>
                  <span className="font-medium text-foreground">
                    {partner.totalDeliveries}
                  </span>
                </div>
              </div>

              {!partner.isVerified && (
                <div className="mt-3 rounded-lg bg-warning/10 px-2.5 py-1 text-center">
                  <span className="text-xs font-medium text-warning">
                    Pending Verification
                  </span>
                </div>
              )}
            </button>
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Page {page} of {totalPages}
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="flex items-center gap-1 rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground hover:bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="h-4 w-4" />
              Previous
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="flex items-center gap-1 rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground hover:bg-secondary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
