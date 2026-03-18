import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { MapPin, Package, Clock } from 'lucide-react';
import type { AvailableDelivery } from '@/shared/types';
import { toast } from 'sonner';
import { PageLoader } from '@/shared/components/LoadingScreen';
import { useNavigate } from 'react-router-dom';
import { formatDistanceToNow } from 'date-fns';

export default function AvailableDeliveriesPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ['available-deliveries'],
    queryFn: () => apiClient.get<{ data: AvailableDelivery[] }>('/delivery/available'),
    refetchInterval: 10000,
  });

  const acceptDelivery = useMutation({
    mutationFn: (orderId: string) => apiClient.post(`/delivery/${orderId}/accept`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['available-deliveries'] });
      queryClient.invalidateQueries({ queryKey: ['delivery-stats'] });
      toast.success('Delivery accepted!');
      navigate('/active');
    },
    onError: (err: { error?: { message?: string } }) => {
      toast.error(err?.error?.message || 'Failed to accept delivery');
    },
  });

  if (isLoading) return <PageLoader />;

  const deliveries = (data as unknown as { data: AvailableDelivery[] })?.data ?? data ?? [];

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Available Deliveries</h1>
        <p className="page-description">{(deliveries as AvailableDelivery[]).length} orders waiting for pickup</p>
      </div>

      {(deliveries as AvailableDelivery[]).length === 0 ? (
        <div className="empty-state">
          <Package className="empty-state-icon" />
          <h2 className="empty-state-title">No Available Deliveries</h2>
          <p className="empty-state-description">New orders will appear here when they're ready for pickup.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {(deliveries as AvailableDelivery[]).map((d) => (
            <div key={d.orderId} className="rounded-xl border border-border bg-card p-5 transition-shadow hover:shadow-md">
              <div className="flex items-center justify-between mb-3">
                <span className="text-sm font-semibold text-foreground">#{d.orderNumber}</span>
                <span className="text-sm font-bold text-primary">${d.estimatedPayout.toFixed(2)}</span>
              </div>

              <div className="space-y-2 mb-4">
                <div className="flex items-start gap-2">
                  <MapPin className="h-4 w-4 text-success mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-xs text-success font-medium">PICKUP</p>
                    <p className="text-sm text-muted-foreground">{d.pickupAddress}</p>
                  </div>
                </div>
                <div className="flex items-start gap-2">
                  <MapPin className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-xs text-primary font-medium">DROPOFF</p>
                    <p className="text-sm text-muted-foreground">{d.dropoffAddress}</p>
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-4 text-xs text-muted-foreground mb-4">
                <span className="flex items-center gap-1">
                  <Package className="h-3.5 w-3.5" /> {d.itemCount} items
                </span>
                {d.distance > 0 && (
                  <span className="flex items-center gap-1">
                    <MapPin className="h-3.5 w-3.5" /> {d.distance.toFixed(1)} km
                  </span>
                )}
                <span className="flex items-center gap-1">
                  <Clock className="h-3.5 w-3.5" /> {formatDistanceToNow(new Date(d.createdAt), { addSuffix: true })}
                </span>
              </div>

              <button
                onClick={() => acceptDelivery.mutate(d.orderId)}
                disabled={acceptDelivery.isPending}
                className="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
              >
                {acceptDelivery.isPending ? 'Accepting...' : 'Accept Delivery'}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
