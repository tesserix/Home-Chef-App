import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { MapPin, Package, CheckCircle2, Navigation, XCircle } from 'lucide-react';
import type { Delivery } from '@/shared/types';
import { toast } from 'sonner';
import { PageLoader } from '@/shared/components/LoadingScreen';

export default function ActiveDeliveryPage() {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['active-delivery'],
    queryFn: () => apiClient.get<{ delivery: Delivery | null }>('/delivery/current'),
    refetchInterval: 10000,
  });

  const updateStatus = useMutation({
    mutationFn: ({ id, status, cancelReason }: { id: string; status: string; cancelReason?: string }) =>
      apiClient.put(`/delivery/${id}/status`, { status, cancelReason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-delivery'] });
      queryClient.invalidateQueries({ queryKey: ['delivery-stats'] });
      toast.success('Status updated');
    },
    onError: () => toast.error('Failed to update status'),
  });

  if (isLoading) return <PageLoader />;

  const delivery = data?.delivery;

  if (!delivery) {
    return (
      <div className="empty-state">
        <Navigation className="empty-state-icon" />
        <h2 className="empty-state-title">No Active Delivery</h2>
        <p className="empty-state-description">
          Check available deliveries to pick up a new order.
        </p>
      </div>
    );
  }

  const statusActions: Record<string, { label: string; nextStatus: string; icon: typeof CheckCircle2 }[]> = {
    assigned: [
      { label: 'Picked Up', nextStatus: 'picked_up', icon: Package },
    ],
    picked_up: [
      { label: 'In Transit', nextStatus: 'in_transit', icon: Navigation },
    ],
    in_transit: [
      { label: 'Delivered', nextStatus: 'delivered', icon: CheckCircle2 },
    ],
  };

  const actions = statusActions[delivery.status] || [];

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Active Delivery</h1>
        <div className="flex items-center gap-2">
          <span className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-medium ${
            delivery.status === 'assigned' ? 'bg-info/10 text-info' :
            delivery.status === 'picked_up' ? 'bg-warning/10 text-warning' :
            delivery.status === 'in_transit' ? 'bg-primary/10 text-primary' :
            'bg-success/10 text-success'
          }`}>
            {delivery.status.replace('_', ' ').toUpperCase()}
          </span>
        </div>
      </div>

      {/* Order Info */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold text-foreground">Order #{delivery.order?.orderNumber}</h3>
          <span className="text-sm font-medium text-primary">${delivery.totalPayout.toFixed(2)}</span>
        </div>
        {delivery.order?.items?.map((item, i) => (
          <div key={i} className="flex justify-between py-1 text-sm">
            <span className="text-muted-foreground">{item.quantity}x {item.name}</span>
            <span className="text-foreground">${item.subtotal.toFixed(2)}</span>
          </div>
        ))}
        {delivery.order?.specialInstructions && (
          <p className="mt-3 rounded-lg bg-muted p-3 text-xs text-muted-foreground">
            Note: {delivery.order.specialInstructions}
          </p>
        )}
      </div>

      {/* Pickup & Dropoff */}
      <div className="space-y-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-start gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-success/10">
              <MapPin className="h-4 w-4 text-success" />
            </div>
            <div>
              <p className="text-xs font-medium text-success uppercase">Pickup from</p>
              <p className="text-sm font-medium text-foreground">{delivery.chef?.name}</p>
              <p className="text-sm text-muted-foreground">{delivery.pickup?.address}, {delivery.pickup?.city}</p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-start gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
              <MapPin className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-xs font-medium text-primary uppercase">Deliver to</p>
              <p className="text-sm text-muted-foreground">{delivery.dropoff?.address}, {delivery.dropoff?.city}</p>
              {delivery.order?.deliveryInstructions && (
                <p className="mt-1 text-xs text-muted-foreground italic">{delivery.order.deliveryInstructions}</p>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Distance & Time */}
      <div className="flex gap-4">
        <div className="flex-1 rounded-xl border border-border bg-card p-4 text-center">
          <p className="text-lg font-bold text-foreground">{delivery.distance?.toFixed(1)} km</p>
          <p className="text-xs text-muted-foreground">Distance</p>
        </div>
        <div className="flex-1 rounded-xl border border-border bg-card p-4 text-center">
          <p className="text-lg font-bold text-foreground">{delivery.estimatedDuration} min</p>
          <p className="text-xs text-muted-foreground">Est. Time</p>
        </div>
        <div className="flex-1 rounded-xl border border-border bg-card p-4 text-center">
          <p className="text-lg font-bold text-foreground">${delivery.totalPayout.toFixed(2)}</p>
          <p className="text-xs text-muted-foreground">Payout</p>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="space-y-3">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <button
              key={action.nextStatus}
              onClick={() => updateStatus.mutate({ id: delivery.id, status: action.nextStatus })}
              disabled={updateStatus.isPending}
              className="flex w-full items-center justify-center gap-2 rounded-xl bg-primary px-4 py-3.5 text-sm font-medium text-primary-foreground shadow-sm transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              <Icon className="h-5 w-5" />
              Mark as {action.label}
            </button>
          );
        })}

        {delivery.status !== 'delivered' && (
          <button
            onClick={() => {
              const reason = prompt('Reason for cancellation?');
              if (reason) {
                updateStatus.mutate({ id: delivery.id, status: 'cancelled', cancelReason: reason });
              }
            }}
            className="flex w-full items-center justify-center gap-2 rounded-xl border border-destructive/30 px-4 py-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/5"
          >
            <XCircle className="h-5 w-5" />
            Cancel Delivery
          </button>
        )}
      </div>
    </div>
  );
}
