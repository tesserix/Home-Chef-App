import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  ChefHat,
  Star,
  MapPin,
  CheckCircle,
  XCircle,
  Clock,
  Eye,
  Loader2,
  Shield,
  Ban,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import type { Chef, PaginatedResponse } from '@/shared/types';

const STATUS_TABS = [
  { value: 'all', label: 'All Chefs' },
  { value: 'pending', label: 'Pending Verification' },
  { value: 'verified', label: 'Verified' },
  { value: 'suspended', label: 'Suspended' },
];

export default function AdminChefsPage() {
  const [statusFilter, setStatusFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedChef, setSelectedChef] = useState<Chef | null>(null);
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-chefs', { status: statusFilter, search: searchQuery }],
    queryFn: () =>
      apiClient.get<PaginatedResponse<Chef>>('/admin/chefs', {
        status: statusFilter !== 'all' ? statusFilter : undefined,
        search: searchQuery || undefined,
      }),
  });

  const verifyMutation = useMutation({
    mutationFn: ({ chefId, action }: { chefId: string; action: 'verify' | 'reject' | 'suspend' }) =>
      apiClient.put(`/admin/chefs/${chefId}/${action}`),
    onSuccess: (_, { action }) => {
      queryClient.invalidateQueries({ queryKey: ['admin-chefs'] });
      toast.success(
        action === 'verify'
          ? 'Chef verified successfully'
          : action === 'reject'
          ? 'Chef application rejected'
          : 'Chef suspended'
      );
      setSelectedChef(null);
    },
  });

  const chefs = data?.data || [];
  const pendingCount = chefs.filter((c) => !c.verified && !c.verifiedAt).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="font-display text-2xl font-semibold text-paper">Chef Management</h1>
        <p className="mt-1 text-ink-muted">Verify and manage home chefs</p>
      </div>

      {/* Status Tabs */}
      <div className="flex gap-2 overflow-x-auto pb-2">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setStatusFilter(tab.value)}
            className={`relative rounded-full px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors ${
              statusFilter === tab.value
                ? 'bg-herb text-paper'
                : 'bg-ink-soft text-ink-muted hover:bg-ink-soft'
            }`}
          >
            {tab.label}
            {tab.value === 'pending' && pendingCount > 0 && (
              <span className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-paprika text-xs text-paper">
                {pendingCount}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-ink-muted" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search chefs..."
          className="w-full rounded-lg bg-ink border-ink-soft pl-10 pr-4 py-2.5 text-paper placeholder:text-ink-muted"
        />
      </div>

      {/* Chefs Grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-herb" />
        </div>
      ) : chefs.length === 0 ? (
        <div className="rounded-xl bg-ink p-12 text-center">
          <ChefHat className="mx-auto h-12 w-12 text-ink-soft" />
          <h3 className="mt-4 font-medium text-paper">No chefs found</h3>
          <p className="mt-2 text-ink-muted">
            {statusFilter === 'pending'
              ? 'No pending verification requests'
              : 'Try adjusting your search'}
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {chefs.map((chef) => (
            <ChefCard
              key={chef.id}
              chef={chef}
              onSelect={() => setSelectedChef(chef)}
            />
          ))}
        </div>
      )}

      {/* Chef Detail Modal */}
      {selectedChef && (
        <ChefDetailModal
          chef={selectedChef}
          onClose={() => setSelectedChef(null)}
          onVerify={() => verifyMutation.mutate({ chefId: selectedChef.id, action: 'verify' })}
          onReject={() => verifyMutation.mutate({ chefId: selectedChef.id, action: 'reject' })}
          onSuspend={() => verifyMutation.mutate({ chefId: selectedChef.id, action: 'suspend' })}
          isUpdating={verifyMutation.isPending}
        />
      )}
    </div>
  );
}

function ChefCard({ chef, onSelect }: { chef: Chef; onSelect: () => void }) {
  return (
    <div
      className={`rounded-xl bg-ink overflow-hidden ${
        !chef.verified && !chef.verifiedAt ? 'ring-2 ring-amber' : ''
      }`}
    >
      {/* Banner */}
      <div className="relative h-24">
        <img
          src={chef.bannerImage || chef.profileImage}
          alt={chef.businessName}
          className="h-full w-full object-cover"
        />
        <div className="absolute -bottom-6 left-4">
          <img
            src={chef.profileImage}
            alt={chef.businessName}
            className="h-12 w-12 rounded-xl border-4 border-ink object-cover"
          />
        </div>
        {/* Status Badge */}
        <div className="absolute top-2 right-2">
          {chef.verified ? (
            <span className="flex items-center gap-1 rounded-full bg-herb/20 px-2 py-0.5 text-xs font-medium text-herb">
              <Shield className="h-3 w-3" />
              Verified
            </span>
          ) : chef.verifiedAt === null ? (
            <span className="flex items-center gap-1 rounded-full bg-amber/20 px-2 py-0.5 text-xs font-medium text-amber">
              <Clock className="h-3 w-3" />
              Pending
            </span>
          ) : (
            <span className="flex items-center gap-1 rounded-full bg-paprika/20 px-2 py-0.5 text-xs font-medium text-paprika">
              <Ban className="h-3 w-3" />
              Suspended
            </span>
          )}
        </div>
      </div>

      <div className="p-4 pt-8">
        <h3 className="font-semibold text-paper truncate">{chef.businessName}</h3>
        <p className="mt-1 text-sm text-ink-muted truncate">
          {chef.cuisines.slice(0, 2).join(' • ')}
        </p>

        <div className="mt-4 flex items-center gap-4 text-sm text-ink-muted">
          <div className="flex items-center gap-1">
            <Star className="h-4 w-4 text-amber" />
            {chef.rating}
          </div>
          <div>{chef.totalOrders} orders</div>
        </div>

        <button
          onClick={onSelect}
          className="mt-4 w-full rounded-lg bg-ink-soft py-2 text-sm font-medium text-paper hover:bg-ink-soft"
        >
          <Eye className="mr-1 inline h-4 w-4" />
          View Details
        </button>
      </div>
    </div>
  );
}

function ChefDetailModal({
  chef,
  onClose,
  onVerify,
  onReject,
  onSuspend,
  isUpdating,
}: {
  chef: Chef;
  onClose: () => void;
  onVerify: () => void;
  onReject: () => void;
  onSuspend: () => void;
  isUpdating: boolean;
}) {
  const isPending = !chef.verified && !chef.verifiedAt;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-ink/70 p-4 pt-10">
      <div className="w-full max-w-2xl rounded-xl bg-ink shadow-3">
        {/* Banner */}
        <div className="relative h-32">
          <img
            src={chef.bannerImage || chef.profileImage}
            alt={chef.businessName}
            className="h-full w-full object-cover rounded-t-xl"
          />
          <div className="absolute -bottom-10 left-6">
            <img
              src={chef.profileImage}
              alt={chef.businessName}
              className="h-20 w-20 rounded-xl border-4 border-ink object-cover"
            />
          </div>
        </div>

        <div className="p-6 pt-12">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-xl font-semibold text-paper">{chef.businessName}</h2>
              <p className="mt-1 text-ink-muted">{chef.cuisines.join(' • ')}</p>
            </div>
            {chef.verified ? (
              <span className="flex items-center gap-1 rounded-full bg-herb/20 px-3 py-1 text-sm font-medium text-herb">
                <Shield className="h-4 w-4" />
                Verified
              </span>
            ) : isPending ? (
              <span className="flex items-center gap-1 rounded-full bg-amber/20 px-3 py-1 text-sm font-medium text-amber">
                <Clock className="h-4 w-4" />
                Pending Verification
              </span>
            ) : (
              <span className="flex items-center gap-1 rounded-full bg-paprika/20 px-3 py-1 text-sm font-medium text-paprika">
                <Ban className="h-4 w-4" />
                Suspended
              </span>
            )}
          </div>

          <p className="mt-4 text-ink-muted">{chef.description}</p>

          {/* Stats */}
          <div className="mt-6 grid grid-cols-3 gap-4">
            <div className="rounded-lg bg-ink-soft/50 p-4 text-center">
              <div className="flex items-center justify-center gap-1 font-display text-2xl font-semibold text-paper">
                <Star className="h-5 w-5 text-amber" />
                {chef.rating}
              </div>
              <p className="text-sm text-ink-muted">{chef.totalReviews} reviews</p>
            </div>
            <div className="rounded-lg bg-ink-soft/50 p-4 text-center">
              <p className="font-display text-2xl font-semibold text-paper">{chef.totalOrders}</p>
              <p className="text-sm text-ink-muted">Orders</p>
            </div>
            <div className="rounded-lg bg-ink-soft/50 p-4 text-center">
              <p className="font-display text-2xl font-semibold text-paper">{chef.serviceRadius} km</p>
              <p className="text-sm text-ink-muted">Service Radius</p>
            </div>
          </div>

          {/* Details */}
          <div className="mt-6 space-y-3">
            <div className="flex items-center gap-3 text-ink-muted">
              <MapPin className="h-4 w-4 text-ink-muted" />
              Service area: {chef.serviceRadius} km radius
            </div>
            <div className="flex items-center gap-3 text-ink-muted">
              <Clock className="h-4 w-4 text-ink-muted" />
              Prep time: {chef.prepTime}
            </div>
          </div>

          {/* Specialties */}
          {chef.specialties.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-medium text-ink-muted mb-2">Specialties</h3>
              <div className="flex flex-wrap gap-2">
                {chef.specialties.map((specialty) => (
                  <span
                    key={specialty}
                    className="rounded-full bg-herb/20 px-3 py-1 text-sm text-herb-soft"
                  >
                    {specialty}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex gap-3 border-t border-ink-soft p-4">
          <button onClick={onClose} className="flex-1 btn-outline border-ink-soft text-paper">
            Close
          </button>
          {isPending ? (
            <>
              <button
                onClick={onReject}
                disabled={isUpdating}
                className="flex-1 btn-base bg-paprika text-paper hover:bg-paprika"
              >
                {isUpdating ? <Loader2 className="h-4 w-4 animate-spin" /> : <XCircle className="h-4 w-4" />}
                Reject
              </button>
              <button
                onClick={onVerify}
                disabled={isUpdating}
                className="flex-1 btn-base bg-herb text-paper hover:bg-herb"
              >
                {isUpdating ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}
                Verify
              </button>
            </>
          ) : chef.verified ? (
            <button
              onClick={onSuspend}
              disabled={isUpdating}
              className="flex-1 btn-base bg-paprika text-paper hover:bg-paprika"
            >
              {isUpdating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ban className="h-4 w-4" />}
              Suspend Chef
            </button>
          ) : (
            <button
              onClick={onVerify}
              disabled={isUpdating}
              className="flex-1 btn-base bg-herb text-paper hover:bg-herb"
            >
              {isUpdating ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}
              Reactivate
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
