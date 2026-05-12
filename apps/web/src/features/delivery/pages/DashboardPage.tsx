import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useFormatPrice } from '@/shared/utils/format-price';
import {
  Package,
  Navigation,
  Clock,
  MapPin,
  Phone,
  CheckCircle,
  Play,

  Bike,
  AlertCircle,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
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
          <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
            Dashboard
          </h1>
          <p className="mt-1 flex items-center gap-2 text-sm text-ink-soft">
            <span
              className={`inline-block h-2 w-2 rounded-full ${isOnline ? 'bg-herb' : 'bg-ink-muted'}`}
              aria-hidden
            />
            {isOnline ? 'Ready for deliveries' : 'Currently offline'}
          </p>
        </div>
        <button
          type="button"
          onClick={toggleOnline}
          aria-pressed={isOnline}
          aria-label={isOnline ? 'Go offline' : 'Go online'}
          className={`inline-flex h-11 items-center justify-center rounded-full px-6 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2 ${
            isOnline
              ? 'bg-foreground text-background hover:bg-ink-soft'
              : 'border border-mist bg-bone text-foreground hover:bg-mist'
          }`}
        >
          {isOnline ? 'Go offline' : 'Go online'}
        </button>
      </div>

      {/* Lead block — Today's earnings (dominant) */}
      <section>
        <p className="text-sm text-ink-soft">Today's earnings</p>
        <p className="mt-1 text-5xl font-semibold tabular-nums tracking-tight text-foreground sm:text-6xl">
          {fp(stats?.todayEarnings || 0)}
        </p>
        <p className="mt-2 text-sm text-ink-soft tabular-nums">
          From {stats?.todayDeliveries ?? 0}{' '}
          {stats?.todayDeliveries === 1 ? 'delivery' : 'deliveries'} today
        </p>
      </section>

      {/* Stats — hairline-divided */}
      <section
        aria-label="Performance at a glance"
        className="grid grid-cols-2 divide-y divide-mist border-y border-mist sm:grid-cols-4 sm:divide-x sm:divide-y-0"
      >
        <StatRow label="On-time rate" value={`${stats?.onTimeRate ?? 0}%`} />
        <StatRow
          label="Rating"
          value={stats?.averageRating !== undefined ? stats.averageRating.toFixed(1) : '—'}
          subtitle={stats?.totalReviews ? `${stats.totalReviews} reviews` : 'No reviews yet'}
        />
        <StatRow
          label="This week"
          value={stats?.weekDeliveries ?? 0}
          subtitle={stats?.weekDeliveries === 1 ? 'delivery' : 'deliveries'}
        />
        <StatRow label="Week earnings" value={fp(stats?.weekEarnings || 0)} />
      </section>

      {/* Current Delivery */}
      {currentDelivery && (
        <div className="rounded-xl bg-herb-tint border border-herb-tint p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-ink">Current Delivery</h2>
            <span className="rounded-full bg-herb px-3 py-1 text-sm font-medium text-paper">
              In Progress
            </span>
          </div>

          <div className="grid gap-6 md:grid-cols-2">
            {/* Pickup */}
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-sm font-medium text-ink-muted">
                <div className="h-6 w-6 rounded-full bg-herb-tint flex items-center justify-center">
                  <div className="h-2 w-2 rounded-full bg-herb" />
                </div>
                PICKUP
              </div>
              <div className="pl-8">
                <p className="font-medium text-ink">Chef's Kitchen</p>
                <p className="text-ink-soft">{currentDelivery.deliveryAddress.line1}</p>
                <button className="mt-2 text-sm text-herb font-medium flex items-center gap-1">
                  <Navigation className="h-4 w-4" />
                  Navigate
                </button>
              </div>
            </div>

            {/* Dropoff */}
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-sm font-medium text-ink-muted">
                <div className="h-6 w-6 rounded-full bg-herb-tint flex items-center justify-center">
                  <MapPin className="h-3 w-3 text-herb" />
                </div>
                DROP-OFF
              </div>
              <div className="pl-8">
                <p className="font-medium text-ink">Customer</p>
                <p className="text-ink-soft">{currentDelivery.deliveryAddress.line1}</p>
                <p className="text-ink-muted text-sm">
                  {currentDelivery.deliveryAddress.city}, {currentDelivery.deliveryAddress.state}
                </p>
              </div>
            </div>
          </div>

          {/* Order Details */}
          <div className="mt-6 flex items-center justify-between rounded-lg bg-bone p-4">
            <div>
              <p className="font-medium text-ink">Order #{currentDelivery.orderNumber}</p>
              <p className="text-sm text-ink-muted">{currentDelivery.items.length} items</p>
            </div>
            <div className="text-right">
              <p className="font-semibold text-ink">{fp(currentDelivery.deliveryFee)}</p>
              <p className="text-sm text-ink-muted">+ tip</p>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="mt-6 flex gap-3">
            {currentDelivery.status === 'ready' && (
              <Button
                variant="primary"
                fullWidth
                isLoading={updateStatusMutation.isPending}
                disabled={updateStatusMutation.isPending}
                onClick={() =>
                  updateStatusMutation.mutate({ orderId: currentDelivery.id, status: 'picked_up' })
                }
                leftIcon={!updateStatusMutation.isPending ? <Package aria-hidden="true" className="h-4 w-4" /> : undefined}
                className="flex-1"
              >
                Mark as Picked Up
              </Button>
            )}
            {currentDelivery.status === 'picked_up' && (
              <Button
                variant="primary"
                fullWidth
                isLoading={updateStatusMutation.isPending}
                disabled={updateStatusMutation.isPending}
                onClick={() =>
                  updateStatusMutation.mutate({ orderId: currentDelivery.id, status: 'delivering' })
                }
                leftIcon={!updateStatusMutation.isPending ? <Bike aria-hidden="true" className="h-4 w-4" /> : undefined}
                className="flex-1"
              >
                Start Delivery
              </Button>
            )}
            {currentDelivery.status === 'delivering' && (
              <Button
                variant="success"
                fullWidth
                isLoading={updateStatusMutation.isPending}
                disabled={updateStatusMutation.isPending}
                onClick={() =>
                  updateStatusMutation.mutate({ orderId: currentDelivery.id, status: 'delivered' })
                }
                leftIcon={!updateStatusMutation.isPending ? <CheckCircle aria-hidden="true" className="h-4 w-4" /> : undefined}
                className="flex-1"
              >
                Complete Delivery
              </Button>
            )}
            <Button variant="outline" leftIcon={<Phone aria-hidden="true" className="h-4 w-4" />}>
              Contact
            </Button>
          </div>
        </div>
      )}

      {/* Available Deliveries */}
      {!currentDelivery && isOnline && (
        <div>
          <h2 className="text-lg font-semibold text-ink mb-4">Available Deliveries</h2>
          {availableDeliveries && availableDeliveries.length > 0 ? (
            <div className="space-y-4">
              {availableDeliveries.map((delivery) => (
                <div
                  key={delivery.id}
                  className="rounded-xl border border-mist bg-bone p-4 hover:border-herb-tint transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-ink">
                          #{delivery.orderNumber}
                        </span>
                        <span className="text-ink-muted">•</span>
                        <span className="text-ink-soft">{delivery.chefName}</span>
                      </div>

                      <div className="mt-3 space-y-2">
                        <div className="flex items-center gap-2 text-sm text-ink-soft">
                          <div className="h-2 w-2 rounded-full bg-herb" />
                          {delivery.pickupAddress.line1}, {delivery.pickupAddress.city}
                        </div>
                        <div className="flex items-center gap-2 text-sm text-ink-soft">
                          <MapPin className="h-3 w-3 text-herb" />
                          {delivery.deliveryAddress.line1}, {delivery.deliveryAddress.city}
                        </div>
                      </div>

                      <div className="mt-3 flex items-center gap-4 text-sm text-ink-muted">
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
                      <p className="text-xl font-semibold text-herb">
                        {fp(delivery.estimatedPayout)}
                      </p>
                      <Button
                        variant="primary"
                        size="sm"
                        isLoading={acceptMutation.isPending}
                        disabled={acceptMutation.isPending}
                        onClick={() => acceptMutation.mutate(delivery.id)}
                        leftIcon={!acceptMutation.isPending ? <Play aria-hidden="true" className="h-4 w-4" /> : undefined}
                        className="mt-2"
                      >
                        Accept
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="rounded-xl bg-paper p-12 text-center">
              <Bike className="mx-auto h-12 w-12 text-ink-muted" />
              <h3 className="mt-4 font-medium text-ink">No deliveries available</h3>
              <p className="mt-2 text-ink-muted">
                New delivery requests will appear here. Stay online!
              </p>
            </div>
          )}
        </div>
      )}

      {/* Offline Message */}
      {!isOnline && (
        <div className="rounded-xl bg-mist p-12 text-center">
          <AlertCircle className="mx-auto h-12 w-12 text-ink-muted" />
          <h3 className="mt-4 font-medium text-ink">You're currently offline</h3>
          <p className="mt-2 text-ink-muted">
            Go online to start receiving delivery requests
          </p>
          <Button variant="primary" onClick={toggleOnline} className="mt-4">
            Go Online
          </Button>
        </div>
      )}

    </div>
  );
}

function StatRow({
  label,
  value,
  subtitle,
}: {
  label: string;
  value: string | number;
  subtitle?: string;
}) {
  return (
    <div className="px-4 py-4 sm:px-5">
      <p className="text-sm text-ink-soft">{label}</p>
      <p className="mt-1.5 text-2xl font-semibold tabular-nums tracking-tight text-foreground">
        {value}
      </p>
      {subtitle && <p className="mt-0.5 text-xs text-ink-soft tabular-nums">{subtitle}</p>}
    </div>
  );
}
