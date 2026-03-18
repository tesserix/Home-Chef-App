import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { User, Truck, Star, Award, Shield } from 'lucide-react';
import type { DeliveryPartner } from '@/shared/types';
import { PageLoader } from '@/shared/components/LoadingScreen';

export default function ProfilePage() {
  const { data: profile, isLoading } = useQuery({
    queryKey: ['delivery-profile'],
    queryFn: () => apiClient.get<DeliveryPartner>('/delivery/profile'),
  });

  if (isLoading) return <PageLoader />;

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Profile</h1>
      </div>

      {/* Profile Card */}
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
            <User className="h-8 w-8 text-primary" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-foreground">{profile?.name || 'Delivery Partner'}</h2>
            <p className="text-sm text-muted-foreground">{profile?.email}</p>
            <div className="mt-1 flex items-center gap-2">
              {profile?.verified ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-success/10 px-2 py-0.5 text-xs font-medium text-success">
                  <Shield className="h-3 w-3" /> Verified
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 rounded-full bg-warning/10 px-2 py-0.5 text-xs font-medium text-warning">
                  Pending Verification
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Vehicle Info */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h3 className="font-semibold text-foreground mb-4 flex items-center gap-2">
          <Truck className="h-5 w-5" /> Vehicle Details
        </h3>
        <div className="space-y-3">
          <div className="flex justify-between">
            <span className="text-sm text-muted-foreground">Vehicle Type</span>
            <span className="text-sm font-medium text-foreground capitalize">{profile?.vehicleType}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-sm text-muted-foreground">Vehicle Number</span>
            <span className="text-sm font-medium text-foreground">{profile?.vehicleNumber}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-sm text-muted-foreground">License Number</span>
            <span className="text-sm font-medium text-foreground">{profile?.licenseNumber}</span>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h3 className="font-semibold text-foreground mb-4 flex items-center gap-2">
          <Award className="h-5 w-5" /> Performance
        </h3>
        <div className="grid grid-cols-3 gap-4 text-center">
          <div>
            <p className="text-2xl font-bold text-foreground">{profile?.totalDeliveries ?? 0}</p>
            <p className="text-xs text-muted-foreground">Deliveries</p>
          </div>
          <div>
            <div className="flex items-center justify-center gap-1">
              <Star className="h-4 w-4 text-warning fill-warning" />
              <p className="text-2xl font-bold text-foreground">{(profile?.rating ?? 0).toFixed(1)}</p>
            </div>
            <p className="text-xs text-muted-foreground">Rating</p>
          </div>
          <div>
            <p className="text-2xl font-bold text-foreground">{profile?.totalReviews ?? 0}</p>
            <p className="text-xs text-muted-foreground">Reviews</p>
          </div>
        </div>
      </div>
    </div>
  );
}
