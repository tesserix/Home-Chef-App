import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Truck,
  Clock,
  CheckCircle,
  XCircle,
  RefreshCw,
  Mail,
  Loader2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';

interface OnboardingStatusResponse {
  step: number;
  status: string;
  profile?: Record<string, unknown>;
  rejectionReason?: string;
  documentCount?: number;
  payoutMethodSet?: boolean;
}

export default function OnboardingStatusPage() {
  const navigate = useNavigate();

  const { data: status, isLoading } = useQuery({
    queryKey: ['onboarding-status'],
    queryFn: () => apiClient.get<OnboardingStatusResponse>('/driver/onboarding/status'),
    refetchInterval: 30000,
  });

  useEffect(() => {
    if (!status) return;

    if (status.status === 'approved') {
      navigate('/dashboard', { replace: true });
      return;
    }

    if (status.status === 'not_started' || status.status === 'in_progress') {
      navigate('/onboarding', { replace: true });
    }
  }, [status, navigate]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  const isRejected = status?.status === 'rejected';
  const isSubmitted = status?.status === 'submitted';
  const isInReview = status?.status === 'in_review';

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Header */}
      <header className="border-b border-border bg-card px-4 py-4">
        <div className="mx-auto flex max-w-lg items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary">
            <Truck className="h-5 w-5 text-primary-foreground" />
          </div>
          <div>
            <h1 className="text-lg font-bold text-foreground">Application Status</h1>
            <p className="text-xs text-muted-foreground">Delivery Partner Onboarding</p>
          </div>
        </div>
      </header>

      {/* Content */}
      <div className="mx-auto w-full max-w-lg flex-1 px-4 py-8">
        {/* Status Card */}
        <div className="rounded-2xl border border-border bg-card p-8 text-center">
          {/* Status Icon */}
          <div className="mx-auto mb-4">
            {isRejected ? (
              <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-destructive/10">
                <XCircle className="h-10 w-10 text-destructive" />
              </div>
            ) : isInReview ? (
              <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-amber-50">
                <RefreshCw className="h-10 w-10 text-amber-600 animate-spin" style={{ animationDuration: '3s' }} />
              </div>
            ) : (
              <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-primary/10">
                <CheckCircle className="h-10 w-10 text-primary" />
              </div>
            )}
          </div>

          {/* Status Title */}
          <h2 className="text-xl font-bold text-foreground">
            {isRejected
              ? 'Application Needs Changes'
              : isInReview
                ? 'Under Review'
                : 'Application Submitted'}
          </h2>

          {/* Status Description */}
          <p className="mt-2 text-sm text-muted-foreground">
            {isRejected
              ? 'Your application requires some changes before it can be approved.'
              : isInReview
                ? 'Our team is currently reviewing your application. This usually takes 1-2 business days.'
                : 'Your application has been received and will be reviewed shortly.'}
          </p>

          {/* Status Badge */}
          <div className="mt-4 inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium"
            style={{
              backgroundColor: isRejected
                ? 'rgb(254 226 226)'
                : isInReview
                  ? 'rgb(254 243 199)'
                  : 'rgb(220 252 231)',
              color: isRejected
                ? 'rgb(185 28 28)'
                : isInReview
                  ? 'rgb(161 98 7)'
                  : 'rgb(21 128 61)',
            }}
          >
            <Clock className="h-3 w-3" />
            {isRejected ? 'Rejected' : isInReview ? 'In Review' : 'Submitted'}
          </div>

          {/* Rejection Reason */}
          {isRejected && status?.rejectionReason && (
            <div className="mt-6 rounded-xl border border-destructive/20 bg-destructive/5 p-4 text-left">
              <p className="text-sm font-medium text-destructive mb-1">Reason for rejection:</p>
              <p className="text-sm text-foreground">{status.rejectionReason}</p>
            </div>
          )}

          {/* Actions */}
          <div className="mt-6 space-y-3">
            {isRejected && (
              <button
                type="button"
                onClick={() => navigate('/onboarding', { replace: true })}
                className="w-full rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground hover:bg-primary/90"
              >
                Fix & Resubmit
              </button>
            )}

            {(isSubmitted || isInReview) && (
              <div className="rounded-xl border border-border bg-muted/30 p-4">
                <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                  <RefreshCw className="h-4 w-4" />
                  Auto-refreshing every 30 seconds
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Contact Support */}
        <div className="mt-6 rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
              <Mail className="h-5 w-5 text-muted-foreground" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">Need Help?</p>
              <p className="text-xs text-muted-foreground">
                Contact us at{' '}
                <a
                  href="mailto:support@fe3dr.com"
                  className="text-primary hover:underline"
                >
                  support@fe3dr.com
                </a>
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
