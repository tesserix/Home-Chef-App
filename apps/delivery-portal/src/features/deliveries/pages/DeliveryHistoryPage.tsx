import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { History, CheckCircle2, XCircle } from 'lucide-react';
import { PageLoader } from '@/shared/components/LoadingScreen';
import { format } from 'date-fns';

export default function DeliveryHistoryPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['delivery-history'],
    queryFn: () => apiClient.get<{ data: Record<string, unknown>[]; pagination: Record<string, unknown> }>('/delivery/orders'),
  });

  if (isLoading) return <PageLoader />;

  const deliveries = (data as unknown as { data: Record<string, unknown>[] })?.data ?? data ?? [];

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Delivery History</h1>
        <p className="page-description">Your past deliveries</p>
      </div>

      {(deliveries as Record<string, unknown>[]).length === 0 ? (
        <div className="empty-state">
          <History className="empty-state-icon" />
          <h2 className="empty-state-title">No Deliveries Yet</h2>
          <p className="empty-state-description">Your completed deliveries will appear here.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {(deliveries as Record<string, unknown>[]).map((d: Record<string, unknown>) => {
            const order = d.order as Record<string, unknown> | undefined;
            return (
              <div key={d.id as string} className="rounded-xl border border-border bg-card p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-semibold text-foreground">
                    #{order?.orderNumber as string}
                  </span>
                  <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                    d.status === 'delivered' ? 'bg-success/10 text-success' : 'bg-destructive/10 text-destructive'
                  }`}>
                    {d.status === 'delivered' ? <CheckCircle2 className="h-3 w-3" /> : <XCircle className="h-3 w-3" />}
                    {(d.status as string).replace('_', ' ')}
                  </span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">
                    {d.deliveredAt ? format(new Date(d.deliveredAt as string), 'MMM d, yyyy h:mm a') :
                     format(new Date(d.assignedAt as string), 'MMM d, yyyy h:mm a')}
                  </span>
                  <span className="font-medium text-foreground">${(d.totalPayout as number)?.toFixed(2)}</span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
