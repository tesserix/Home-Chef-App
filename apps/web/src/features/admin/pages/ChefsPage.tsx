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
        <h1 className="text-2xl font-bold text-white">Chef Management</h1>
        <p className="mt-1 text-gray-400">Verify and manage home chefs</p>
      </div>

      {/* Status Tabs */}
      <div className="flex gap-2 overflow-x-auto pb-2">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setStatusFilter(tab.value)}
            className={`relative rounded-full px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors ${
              statusFilter === tab.value
                ? 'bg-brand-500 text-white'
                : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
            }`}
          >
            {tab.label}
            {tab.value === 'pending' && pendingCount > 0 && (
              <span className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs text-white">
                {pendingCount}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-500" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search chefs..."
          className="w-full rounded-lg bg-gray-800 border-gray-700 pl-10 pr-4 py-2.5 text-white placeholder:text-gray-500"
        />
      </div>

      {/* Chefs Grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
        </div>
      ) : chefs.length === 0 ? (
        <div className="rounded-xl bg-gray-800 p-12 text-center">
          <ChefHat className="mx-auto h-12 w-12 text-gray-600" />
          <h3 className="mt-4 font-medium text-white">No chefs found</h3>
          <p className="mt-2 text-gray-400">
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
      className={`rounded-xl bg-gray-800 overflow-hidden ${
        !chef.verified && !chef.verifiedAt ? 'ring-2 ring-yellow-500' : ''
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
            className="h-12 w-12 rounded-xl border-4 border-gray-800 object-cover"
          />
        </div>
        {/* Status Badge */}
        <div className="absolute top-2 right-2">
          {chef.verified ? (
            <span className="flex items-center gap-1 rounded-full bg-green-500/20 px-2 py-0.5 text-xs font-medium text-green-400">
              <Shield className="h-3 w-3" />
              Verified
            </span>
          ) : chef.verifiedAt === null ? (
            <span className="flex items-center gap-1 rounded-full bg-yellow-500/20 px-2 py-0.5 text-xs font-medium text-yellow-400">
              <Clock className="h-3 w-3" />
              Pending
            </span>
          ) : (
            <span className="flex items-center gap-1 rounded-full bg-red-500/20 px-2 py-0.5 text-xs font-medium text-red-400">
              <Ban className="h-3 w-3" />
              Suspended
            </span>
          )}
        </div>
      </div>

      <div className="p-4 pt-8">
        <h3 className="font-semibold text-white truncate">{chef.businessName}</h3>
        <p className="mt-1 text-sm text-gray-400 truncate">
          {chef.cuisines.slice(0, 2).join(' • ')}
        </p>

        <div className="mt-4 flex items-center gap-4 text-sm text-gray-400">
          <div className="flex items-center gap-1">
            <Star className="h-4 w-4 text-yellow-400" />
            {chef.rating}
          </div>
          <div>{chef.totalOrders} orders</div>
        </div>

        <button
          onClick={onSelect}
          className="mt-4 w-full rounded-lg bg-gray-700 py-2 text-sm font-medium text-white hover:bg-gray-600"
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
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/70 p-4 pt-10">
      <div className="w-full max-w-2xl rounded-xl bg-gray-800 shadow-xl">
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
              className="h-20 w-20 rounded-xl border-4 border-gray-800 object-cover"
            />
          </div>
        </div>

        <div className="p-6 pt-12">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-xl font-semibold text-white">{chef.businessName}</h2>
              <p className="mt-1 text-gray-400">{chef.cuisines.join(' • ')}</p>
            </div>
            {chef.verified ? (
              <span className="flex items-center gap-1 rounded-full bg-green-500/20 px-3 py-1 text-sm font-medium text-green-400">
                <Shield className="h-4 w-4" />
                Verified
              </span>
            ) : isPending ? (
              <span className="flex items-center gap-1 rounded-full bg-yellow-500/20 px-3 py-1 text-sm font-medium text-yellow-400">
                <Clock className="h-4 w-4" />
                Pending Verification
              </span>
            ) : (
              <span className="flex items-center gap-1 rounded-full bg-red-500/20 px-3 py-1 text-sm font-medium text-red-400">
                <Ban className="h-4 w-4" />
                Suspended
              </span>
            )}
          </div>

          <p className="mt-4 text-gray-300">{chef.description}</p>

          {/* Stats */}
          <div className="mt-6 grid grid-cols-3 gap-4">
            <div className="rounded-lg bg-gray-700/50 p-4 text-center">
              <div className="flex items-center justify-center gap-1 text-2xl font-bold text-white">
                <Star className="h-5 w-5 text-yellow-400" />
                {chef.rating}
              </div>
              <p className="text-sm text-gray-400">{chef.totalReviews} reviews</p>
            </div>
            <div className="rounded-lg bg-gray-700/50 p-4 text-center">
              <p className="text-2xl font-bold text-white">{chef.totalOrders}</p>
              <p className="text-sm text-gray-400">Orders</p>
            </div>
            <div className="rounded-lg bg-gray-700/50 p-4 text-center">
              <p className="text-2xl font-bold text-white">{chef.serviceRadius} km</p>
              <p className="text-sm text-gray-400">Service Radius</p>
            </div>
          </div>

          {/* Details */}
          <div className="mt-6 space-y-3">
            <div className="flex items-center gap-3 text-gray-300">
              <MapPin className="h-4 w-4 text-gray-500" />
              Service area: {chef.serviceRadius} km radius
            </div>
            <div className="flex items-center gap-3 text-gray-300">
              <Clock className="h-4 w-4 text-gray-500" />
              Prep time: {chef.prepTime}
            </div>
          </div>

          {/* Specialties */}
          {chef.specialties.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-medium text-gray-400 mb-2">Specialties</h3>
              <div className="flex flex-wrap gap-2">
                {chef.specialties.map((specialty) => (
                  <span
                    key={specialty}
                    className="rounded-full bg-brand-500/20 px-3 py-1 text-sm text-brand-400"
                  >
                    {specialty}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex gap-3 border-t border-gray-700 p-4">
          <button onClick={onClose} className="flex-1 btn-outline border-gray-600 text-white">
            Close
          </button>
          {isPending ? (
            <>
              <button
                onClick={onReject}
                disabled={isUpdating}
                className="flex-1 btn-base bg-red-600 text-white hover:bg-red-700"
              >
                {isUpdating ? <Loader2 className="h-4 w-4 animate-spin" /> : <XCircle className="h-4 w-4" />}
                Reject
              </button>
              <button
                onClick={onVerify}
                disabled={isUpdating}
                className="flex-1 btn-base bg-green-600 text-white hover:bg-green-700"
              >
                {isUpdating ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}
                Verify
              </button>
            </>
          ) : chef.verified ? (
            <button
              onClick={onSuspend}
              disabled={isUpdating}
              className="flex-1 btn-base bg-red-600 text-white hover:bg-red-700"
            >
              {isUpdating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ban className="h-4 w-4" />}
              Suspend Chef
            </button>
          ) : (
            <button
              onClick={onVerify}
              disabled={isUpdating}
              className="flex-1 btn-base bg-green-600 text-white hover:bg-green-700"
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
