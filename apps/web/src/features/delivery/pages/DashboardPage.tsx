import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useFormatPrice } from '@/shared/utils/format-price';
import {
  Package,
  Navigation,
  DollarSign,
  Clock,
  MapPin,
  Phone,
  CheckCircle,
  Play,
  Loader2,
  Bike,
  Star,
  AlertCircle,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import type { Order } from '@/shared/types';

interface DeliveryStats {
  todayDeliveries: number;
  todayEarnings: number;
  weekDeliveries: number;
  weekEarnings: number;
  averageRating: number;
  totalReviews: number;
  onTimeRate: number;
}

interface AvailableDelivery {
  id: string;
  orderNumber: string;
  pickupAddress: {
    line1: string;
    city: string;
  };
  deliveryAddress: {
    line1: string;
    city: string;
  };
  chefName: string;
  distance: number;
  estimatedPayout: number;
  estimatedTime: number;
  itemCount: number;
}

export default function DeliveryDashboardPage() {
  const fp = useFormatPrice();
  const [isOnline, setIsOnline] = useState(true);
  const queryClient = useQueryClient();

  const { data: stats } = useQuery({
    queryKey: ['delivery-stats'],
    queryFn: () => apiClient.get<DeliveryStats>('/delivery/stats'),
  });

  const { data: currentDelivery } = useQuery({
    queryKey: ['current-delivery'],
    queryFn: () => apiClient.get<Order | null>('/delivery/current'),
  });

  const { data: availableDeliveries } = useQuery({
    queryKey: ['available-deliveries'],
    queryFn: () => apiClient.get<AvailableDelivery[]>('/delivery/available'),
    enabled: !currentDelivery && isOnline,
    refetchInterval: 30000,
  });

  const acceptMutation = useMutation({
    mutationFn: (deliveryId: string) => apiClient.post(`/delivery/${deliveryId}/accept`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['current-delivery'] });
      queryClient.invalidateQueries({ queryKey: ['available-deliveries'] });
      toast.success('Delivery accepted!');
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ orderId, status }: { orderId: string; status: string }) =>
      apiClient.put(`/delivery/${orderId}/status`, { status }),
    onSuccess: (_, { status }) => {
      queryClient.invalidateQueries({ queryKey: ['current-delivery'] });
      if (status === 'delivered') {
        queryClient.invalidateQueries({ queryKey: ['delivery-stats'] });
        toast.success('Delivery completed! Great job!');
      } else {
        toast.success('Status updated');
      }
    },
  });

  const toggleOnline = () => {
    setIsOnline(!isOnline);
    toast.success(isOnline ? 'You are now offline' : 'You are now online');
  };

  return (
    <div className="space-y-6">
      {/* Header with Online Toggle */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
          <p className="mt-1 text-gray-500">
            {isOnline ? 'Ready for deliveries' : 'Currently offline'}
          </p>
        </div>
        <button
          onClick={toggleOnline}
          className={`relative inline-flex h-12 w-24 items-center justify-center rounded-full font-medium transition-colors ${
            isOnline
              ? 'bg-green-500 text-white hover:bg-green-600'
              : 'bg-gray-300 text-gray-700 hover:bg-gray-400'
          }`}
        >
          {isOnline ? 'Online' : 'Offline'}
        </button>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Today's Deliveries"
          value={stats?.todayDeliveries || 0}
          icon={Package}
          color="bg-blue-500"
        />
        <StatCard
          title="Today's Earnings"
          value={fp(stats?.todayEarnings || 0)}
          icon={DollarSign}
          color="bg-green-500"
        />
        <StatCard
          title="On-Time Rate"
          value={`${stats?.onTimeRate || 0}%`}
          icon={Clock}
          color="bg-purple-500"
        />
        <StatCard
          title="Rating"
          value={stats?.averageRating?.toFixed(1) || '0.0'}
          subtitle={`${stats?.totalReviews || 0} reviews`}
          icon={Star}
          color="bg-yellow-500"
        />
      </div>

      {/* Current Delivery */}
      {currentDelivery && (
        <div className="rounded-xl bg-brand-50 border border-brand-200 p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">Current Delivery</h2>
            <span className="rounded-full bg-brand-500 px-3 py-1 text-sm font-medium text-white">
              In Progress
            </span>
          </div>

          <div className="grid gap-6 md:grid-cols-2">
            {/* Pickup */}
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-sm font-medium text-gray-500">
                <div className="h-6 w-6 rounded-full bg-orange-100 flex items-center justify-center">
                  <div className="h-2 w-2 rounded-full bg-orange-500" />
                </div>
                PICKUP
              </div>
              <div className="pl-8">
                <p className="font-medium text-gray-900">Chef's Kitchen</p>
                <p className="text-gray-600">{currentDelivery.deliveryAddress.line1}</p>
                <button className="mt-2 text-sm text-brand-600 font-medium flex items-center gap-1">
                  <Navigation className="h-4 w-4" />
                  Navigate
                </button>
              </div>
            </div>

            {/* Dropoff */}
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-sm font-medium text-gray-500">
                <div className="h-6 w-6 rounded-full bg-green-100 flex items-center justify-center">
                  <MapPin className="h-3 w-3 text-green-600" />
                </div>
                DROP-OFF
              </div>
              <div className="pl-8">
                <p className="font-medium text-gray-900">Customer</p>
                <p className="text-gray-600">{currentDelivery.deliveryAddress.line1}</p>
                <p className="text-gray-500 text-sm">
                  {currentDelivery.deliveryAddress.city}, {currentDelivery.deliveryAddress.state}
                </p>
              </div>
            </div>
          </div>

          {/* Order Details */}
          <div className="mt-6 flex items-center justify-between rounded-lg bg-white p-4">
            <div>
              <p className="font-medium text-gray-900">Order #{currentDelivery.orderNumber}</p>
              <p className="text-sm text-gray-500">{currentDelivery.items.length} items</p>
            </div>
            <div className="text-right">
              <p className="font-semibold text-gray-900">{fp(currentDelivery.deliveryFee)}</p>
              <p className="text-sm text-gray-500">+ tip</p>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="mt-6 flex gap-3">
            {currentDelivery.status === 'ready' && (
              <button
                onClick={() =>
                  updateStatusMutation.mutate({ orderId: currentDelivery.id, status: 'picked_up' })
                }
                disabled={updateStatusMutation.isPending}
                className="flex-1 btn-primary"
              >
                {updateStatusMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Package className="h-4 w-4" />
                )}
                Mark as Picked Up
              </button>
            )}
            {currentDelivery.status === 'picked_up' && (
              <button
                onClick={() =>
                  updateStatusMutation.mutate({ orderId: currentDelivery.id, status: 'delivering' })
                }
                disabled={updateStatusMutation.isPending}
                className="flex-1 btn-primary"
              >
                {updateStatusMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Bike className="h-4 w-4" />
                )}
                Start Delivery
              </button>
            )}
            {currentDelivery.status === 'delivering' && (
              <button
                onClick={() =>
                  updateStatusMutation.mutate({ orderId: currentDelivery.id, status: 'delivered' })
                }
                disabled={updateStatusMutation.isPending}
                className="flex-1 btn-primary bg-green-600 hover:bg-green-700"
              >
                {updateStatusMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <CheckCircle className="h-4 w-4" />
                )}
                Complete Delivery
              </button>
            )}
            <button className="btn-outline flex items-center gap-2">
              <Phone className="h-4 w-4" />
              Contact
            </button>
          </div>
        </div>
      )}

      {/* Available Deliveries */}
      {!currentDelivery && isOnline && (
        <div>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Available Deliveries</h2>
          {availableDeliveries && availableDeliveries.length > 0 ? (
            <div className="space-y-4">
              {availableDeliveries.map((delivery) => (
                <div
                  key={delivery.id}
                  className="rounded-xl border border-gray-200 bg-white p-4 hover:border-brand-300 transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-900">
                          #{delivery.orderNumber}
                        </span>
                        <span className="text-gray-400">•</span>
                        <span className="text-gray-600">{delivery.chefName}</span>
                      </div>

                      <div className="mt-3 space-y-2">
                        <div className="flex items-center gap-2 text-sm text-gray-600">
                          <div className="h-2 w-2 rounded-full bg-orange-500" />
                          {delivery.pickupAddress.line1}, {delivery.pickupAddress.city}
                        </div>
                        <div className="flex items-center gap-2 text-sm text-gray-600">
                          <MapPin className="h-3 w-3 text-green-600" />
                          {delivery.deliveryAddress.line1}, {delivery.deliveryAddress.city}
                        </div>
                      </div>

                      <div className="mt-3 flex items-center gap-4 text-sm text-gray-500">
                        <span className="flex items-center gap-1">
                          <Navigation className="h-4 w-4" />
                          {delivery.distance} km
                        </span>
                        <span className="flex items-center gap-1">
                          <Clock className="h-4 w-4" />
                          ~{delivery.estimatedTime} min
                        </span>
                        <span className="flex items-center gap-1">
                          <Package className="h-4 w-4" />
                          {delivery.itemCount} items
                        </span>
                      </div>
                    </div>

                    <div className="text-right">
                      <p className="text-xl font-bold text-green-600">
                        {fp(delivery.estimatedPayout)}
                      </p>
                      <button
                        onClick={() => acceptMutation.mutate(delivery.id)}
                        disabled={acceptMutation.isPending}
                        className="mt-2 btn-primary py-2 px-4"
                      >
                        {acceptMutation.isPending ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Play className="h-4 w-4" />
                        )}
                        Accept
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="rounded-xl bg-gray-50 p-12 text-center">
              <Bike className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-4 font-medium text-gray-900">No deliveries available</h3>
              <p className="mt-2 text-gray-500">
                New delivery requests will appear here. Stay online!
              </p>
            </div>
          )}
        </div>
      )}

      {/* Offline Message */}
      {!isOnline && (
        <div className="rounded-xl bg-gray-100 p-12 text-center">
          <AlertCircle className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-4 font-medium text-gray-900">You're currently offline</h3>
          <p className="mt-2 text-gray-500">
            Go online to start receiving delivery requests
          </p>
          <button onClick={toggleOnline} className="mt-4 btn-primary">
            Go Online
          </button>
        </div>
      )}

      {/* Weekly Summary */}
      <div className="rounded-xl bg-white border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900">This Week</h2>
        <div className="mt-4 grid gap-6 sm:grid-cols-2">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-100">
              <Package className="h-6 w-6 text-blue-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{stats?.weekDeliveries || 0}</p>
              <p className="text-sm text-gray-500">Deliveries completed</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-green-100">
              <DollarSign className="h-6 w-6 text-green-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">
                {fp(stats?.weekEarnings || 0)}
              </p>
              <p className="text-sm text-gray-500">Total earnings</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({
  title,
  value,
  subtitle,
  icon: Icon,
  color,
}: {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: typeof Package;
  color: string;
}) {
  return (
    <div className="rounded-xl bg-white border border-gray-200 p-4">
      <div className="flex items-center justify-between">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${color}`}>
          <Icon className="h-5 w-5 text-white" />
        </div>
      </div>
      <p className="mt-3 text-2xl font-bold text-gray-900">{value}</p>
      <p className="text-sm text-gray-500">{title}</p>
      {subtitle && <p className="text-xs text-gray-400">{subtitle}</p>}
    </div>
  );
}
