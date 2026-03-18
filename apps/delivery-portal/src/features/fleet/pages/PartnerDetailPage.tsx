import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ArrowLeft,
  ShieldCheck,
  Star,
  Truck,
  Phone,
  Mail,
  Clock,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  FileText,
  TrendingUp,
  UserX,
  UserCheck,
} from 'lucide-react';
import { toast } from 'sonner';
import { PageLoader } from '@/shared/components/LoadingScreen';

interface Document {
  id: string;
  fileName: string;
  type: string;
  status: string;
  createdAt: string;
  rejectionReason?: string;
}

interface PartnerData {
  id: string;
  name: string;
  email: string;
  phone: string;
  avatar?: string;
  vehicleType: string;
  vehicleNumber: string;
  verified: boolean;
  isActive: boolean;
  isOnline: boolean;
  verificationStatus: string;
  rating: number;
  totalDeliveries: number;
  totalReviews: number;
  acceptanceRate: number;
  onTimeRate: number;
  csatScore: number;
  offeredCount: number;
  acceptedCount: number;
  completedOnTime: number;
  agentType: string;
  rejectionReason?: string;
  createdAt: string;
  documents?: Document[];
}

interface PartnerDetailResponse {
  partner: PartnerData;
  activeDeliveries: number;
}

export default function PartnerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: detailData, isLoading } = useQuery({
    queryKey: ['fleet-partner', id],
    queryFn: () =>
      apiClient.get<PartnerDetailResponse>(`/delivery/staff/fleet/partners/${id}`),
    enabled: !!id,
  });

  // Fetch current user's staff profile for permission checks
  const { data: myProfile } = useQuery({
    queryKey: ['delivery-staff-me'],
    queryFn: () => apiClient.get<{ permissions: string[] }>('/delivery/staff/me'),
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  const canVerify = myProfile?.permissions?.includes('delivery_partners:verify') ?? false;
  const canManage = myProfile?.permissions?.includes('delivery_partners:manage') ?? false;

  const partner = detailData?.partner;
  const activeDeliveries = detailData?.activeDeliveries ?? 0;

  const verifyPartner = useMutation({
    mutationFn: () =>
      apiClient.put(`/delivery/staff/fleet/partners/${id}/verify`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fleet-partner', id] });
      queryClient.invalidateQueries({ queryKey: ['fleet-overview'] });
      toast.success('Partner verified successfully');
    },
    onError: () => toast.error('Failed to verify partner'),
  });

  const suspendPartner = useMutation({
    mutationFn: () =>
      apiClient.put(`/delivery/staff/fleet/partners/${id}/suspend`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fleet-partner', id] });
      toast.success(
        !partner?.isActive
          ? 'Partner reactivated'
          : 'Partner suspended'
      );
    },
    onError: () => toast.error('Failed to update partner status'),
  });

  if (isLoading) return <PageLoader />;

  if (!partner) {
    return (
      <div className="space-y-6">
        <button
          onClick={() => navigate('/fleet/partners')}
          className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Partners
        </button>
        <div className="rounded-xl border border-border bg-card p-12 text-center">
          <p className="text-sm text-muted-foreground">Partner not found.</p>
        </div>
      </div>
    );
  }

  const isPending = partner.verificationStatus === 'pending';
  const isSuspended = !partner.isActive;

  const docStatusIcon = (status: string) => {
    switch (status) {
      case 'verified':
        return <CheckCircle2 className="h-4 w-4 text-success" />;
      case 'rejected':
        return <XCircle className="h-4 w-4 text-destructive" />;
      default:
        return <Clock className="h-4 w-4 text-warning" />;
    }
  };

  const performanceItems = [
    {
      label: 'Acceptance Rate',
      value: `${(partner.acceptanceRate ?? 0).toFixed(1)}%`,
      icon: TrendingUp,
    },
    {
      label: 'On-Time Rate',
      value: `${(partner.onTimeRate ?? 0).toFixed(1)}%`,
      icon: Clock,
    },
    {
      label: 'CSAT Score',
      value: (partner.csatScore ?? 0).toFixed(1),
      icon: Star,
    },
    {
      label: 'Total Deliveries',
      value: partner.totalDeliveries ?? 0,
      icon: Truck,
    },
  ];

  return (
    <div className="space-y-6">
      {/* Back Button */}
      <button
        onClick={() => navigate('/fleet/partners')}
        className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Partners
      </button>

      {/* Profile Card */}
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex items-center gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
              <span className="text-xl font-bold text-primary">
                {(partner.name || 'P').charAt(0).toUpperCase()}
              </span>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-xl font-semibold text-foreground">
                  {partner.name}
                </h2>
                {partner.verified && (
                  <ShieldCheck className="h-5 w-5 text-success" />
                )}
                {isSuspended && (
                  <span className="rounded-full bg-destructive/10 px-2 py-0.5 text-xs font-medium text-destructive">
                    Suspended
                  </span>
                )}
              </div>
              <div className="mt-1 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
                <span className="flex items-center gap-1">
                  <Mail className="h-3.5 w-3.5" />
                  {partner.email}
                </span>
                {partner.phone && (
                  <span className="flex items-center gap-1">
                    <Phone className="h-3.5 w-3.5" />
                    {partner.phone}
                  </span>
                )}
              </div>
              <div className="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
                <Truck className="h-3.5 w-3.5" />
                <span>
                  {partner.vehicleType}
                  {partner.vehicleNumber ? ` - ${partner.vehicleNumber}` : ''}
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {partner.isOnline && (
              <span className="flex items-center gap-1.5 rounded-full bg-success/10 px-3 py-1 text-xs font-medium text-success">
                <span className="h-2 w-2 rounded-full bg-success" />
                Online
              </span>
            )}
            <div className="flex items-center gap-1 rounded-full bg-warning/10 px-3 py-1">
              <Star className="h-3.5 w-3.5 text-warning fill-warning" />
              <span className="text-xs font-medium text-foreground">
                {(partner.rating ?? 0).toFixed(1)}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Verification Section */}
      {isPending && canVerify && (
        <div className="rounded-xl border-2 border-warning bg-warning/5 p-6">
          <div className="flex items-center gap-2 mb-3">
            <AlertTriangle className="h-5 w-5 text-warning" />
            <h3 className="text-lg font-semibold text-foreground">
              Pending Verification
            </h3>
          </div>
          <p className="text-sm text-muted-foreground mb-4">
            This partner is awaiting verification. Review their documents and
            approve or reject their application.
          </p>
          <div className="flex gap-3">
            <button
              onClick={() => verifyPartner.mutate()}
              disabled={verifyPartner.isPending}
              className="flex items-center gap-2 rounded-lg bg-success px-4 py-2 text-sm font-medium text-success-foreground hover:bg-success/90 disabled:opacity-50"
            >
              <CheckCircle2 className="h-4 w-4" />
              {verifyPartner.isPending ? 'Verifying...' : 'Approve'}
            </button>
            <button
              onClick={() => suspendPartner.mutate()}
              disabled={suspendPartner.isPending}
              className="flex items-center gap-2 rounded-lg border border-destructive/30 px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive/5 disabled:opacity-50"
            >
              <XCircle className="h-4 w-4" />
              Reject
            </button>
          </div>
        </div>
      )}

      {/* Read-only verification notice for users without verify permission */}
      {isPending && !canVerify && (
        <div className="rounded-xl border border-warning/30 bg-warning/5 p-4">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-warning" />
            <p className="text-sm text-muted-foreground">
              This partner is pending verification. Contact a fleet manager to approve.
            </p>
          </div>
        </div>
      )}

      {/* Documents */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h3 className="text-lg font-semibold text-foreground mb-4">Documents</h3>
        {(partner.documents ?? []).length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-6">
            No documents uploaded.
          </p>
        ) : (
          <div className="space-y-3">
            {(partner.documents ?? []).map((doc) => (
              <div
                key={doc.id}
                className="flex items-center justify-between rounded-lg border border-border p-4"
              >
                <div className="flex items-center gap-3">
                  <FileText className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {doc.fileName}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {doc.type.split('_').join(' ')} &middot; Uploaded{' '}
                      {new Date(doc.createdAt).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {docStatusIcon(doc.status)}
                  <span
                    className={`text-xs font-medium capitalize ${
                      doc.status === 'verified'
                        ? 'text-success'
                        : doc.status === 'rejected'
                          ? 'text-destructive'
                          : 'text-warning'
                    }`}
                  >
                    {doc.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Performance Metrics */}
      <div className="rounded-xl border border-border bg-card p-6">
        <h3 className="text-lg font-semibold text-foreground mb-4">
          Performance Metrics
        </h3>
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {performanceItems.map((item) => {
            const Icon = item.icon;
            return (
              <div
                key={item.label}
                className="rounded-lg border border-border p-4 text-center"
              >
                <Icon className="mx-auto h-5 w-5 text-muted-foreground mb-2" />
                <p className="text-2xl font-bold text-foreground">
                  {item.value}
                </p>
                <p className="text-xs text-muted-foreground">{item.label}</p>
              </div>
            );
          })}
        </div>
        <div className="mt-4 rounded-lg bg-primary/5 p-4 text-center">
          <p className="text-sm text-muted-foreground">Active Deliveries</p>
          <p className="text-3xl font-bold text-primary">
            {activeDeliveries}
          </p>
        </div>
      </div>

      {/* Actions - only shown to users with manage permission */}
      {canManage && (
        <div className="rounded-xl border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Actions</h3>
          <div className="flex flex-wrap gap-3">
            {isSuspended ? (
              <button
                onClick={() => suspendPartner.mutate()}
                disabled={suspendPartner.isPending}
                className="flex items-center gap-2 rounded-lg bg-success px-4 py-2 text-sm font-medium text-success-foreground hover:bg-success/90 disabled:opacity-50"
              >
                <UserCheck className="h-4 w-4" />
                {suspendPartner.isPending ? 'Processing...' : 'Reactivate Partner'}
              </button>
            ) : (
              <button
                onClick={() => suspendPartner.mutate()}
                disabled={suspendPartner.isPending}
                className="flex items-center gap-2 rounded-lg border border-destructive/30 px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive/5 disabled:opacity-50"
              >
                <UserX className="h-4 w-4" />
                {suspendPartner.isPending ? 'Processing...' : 'Suspend Partner'}
              </button>
            )}
            {!partner.verified && canVerify && (
              <button
                onClick={() => verifyPartner.mutate()}
                disabled={verifyPartner.isPending}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                <ShieldCheck className="h-4 w-4" />
                {verifyPartner.isPending ? 'Verifying...' : 'Verify Partner'}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
