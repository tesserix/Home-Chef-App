import { useState, useEffect } from 'react';
import {
  User,
  Truck,
  FileText,
  Wallet,
  CheckCircle,
  Pencil,
  Loader2,
  AlertCircle,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { toast } from 'sonner';

interface OnboardingStatus {
  step: number;
  status: string;
  profile?: {
    city?: string;
    emergencyContact?: string;
    emergencyPhone?: string;
    dateOfBirth?: string;
    vehicleType?: string;
    vehicleMake?: string;
    vehicleModel?: string;
    vehicleYear?: number;
    vehicleColor?: string;
    vehicleNumber?: string;
    licenseNumber?: string;
    hasDeliveryBoxSpace?: boolean;
    payoutMethod?: string;
    bankAccountName?: string;
    bankAccountNumber?: string;
    bankIFSC?: string;
    upiId?: string;
  };
  documentCount?: number;
  payoutMethodSet?: boolean;
}

interface UploadedDoc {
  id: string;
  type: string;
  fileName: string;
  status: string;
}

interface StepReviewProps {
  onComplete: () => void;
  onBack: () => void;
  onGoToStep: (step: number) => void;
}

export function StepReview({ onComplete, onBack, onGoToStep }: StepReviewProps) {
  const [status, setStatus] = useState<OnboardingStatus | null>(null);
  const [docs, setDocs] = useState<UploadedDoc[]>([]);
  const [loading, setLoading] = useState(true);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statusData, docsData] = await Promise.all([
          apiClient.get<OnboardingStatus>('/driver/onboarding/status'),
          apiClient.get<{ data: UploadedDoc[] } | UploadedDoc[]>('/driver/onboarding/documents').catch(() => [] as UploadedDoc[]),
        ]);
        setStatus(statusData);
        const docsArray = Array.isArray(docsData) ? docsData : ((docsData as { data: UploadedDoc[] })?.data ?? []);
        setDocs(docsArray);
      } catch {
        toast.error('Failed to load review data');
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const handleSubmit = async () => {
    if (!termsAccepted) {
      toast.error('Please accept the terms and conditions');
      return;
    }

    setSubmitting(true);
    try {
      await apiClient.post('/driver/onboarding/submit', { termsAccepted: true });
      toast.success('Application submitted successfully!');
      onComplete();
    } catch {
      toast.error('Failed to submit. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  const profile = status?.profile;

  const docTypeLabels: Record<string, string> = {
    driving_license: 'Driving License',
    vehicle_rc: 'Vehicle RC',
    insurance: 'Insurance',
    aadhaar: 'Aadhaar Card',
    pan_card: 'PAN Card',
    photo: 'Profile Photo',
    police_verification: 'Police Verification',
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-foreground">Review & Submit</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Review your information before submitting
        </p>
      </div>

      {/* Personal Info Section */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <User className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-semibold text-foreground">Personal Information</h3>
          </div>
          <button
            type="button"
            onClick={() => onGoToStep(1)}
            className="flex items-center gap-1 text-xs font-medium text-primary hover:text-primary/80"
          >
            <Pencil className="h-3 w-3" />
            Edit
          </button>
        </div>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <p className="text-muted-foreground">City</p>
            <p className="font-medium text-foreground">{profile?.city || '-'}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Vehicle Type</p>
            <p className="font-medium text-foreground capitalize">{profile?.vehicleType || '-'}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Emergency Contact</p>
            <p className="font-medium text-foreground">{profile?.emergencyContact || '-'}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Emergency Phone</p>
            <p className="font-medium text-foreground">{profile?.emergencyPhone || '-'}</p>
          </div>
          {profile?.dateOfBirth && (
            <div>
              <p className="text-muted-foreground">Date of Birth</p>
              <p className="font-medium text-foreground">{profile.dateOfBirth}</p>
            </div>
          )}
        </div>
      </div>

      {/* Vehicle Details Section */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Truck className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-semibold text-foreground">Vehicle Details</h3>
          </div>
          <button
            type="button"
            onClick={() => onGoToStep(2)}
            className="flex items-center gap-1 text-xs font-medium text-primary hover:text-primary/80"
          >
            <Pencil className="h-3 w-3" />
            Edit
          </button>
        </div>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <p className="text-muted-foreground">{profile?.vehicleType === 'bicycle' ? 'Brand & Model' : 'Make & Model'}</p>
            <p className="font-medium text-foreground">
              {[profile?.vehicleMake, profile?.vehicleModel].filter(Boolean).join(' ') || '-'}
            </p>
          </div>
          {profile?.vehicleType === 'bicycle' ? (
            <div>
              <p className="text-muted-foreground">Delivery Box Space</p>
              <p className="font-medium text-foreground">
                {profile?.hasDeliveryBoxSpace ? 'Yes' : 'No'}
              </p>
            </div>
          ) : (
            <>
              <div>
                <p className="text-muted-foreground">Year & Color</p>
                <p className="font-medium text-foreground">
                  {[profile?.vehicleYear, profile?.vehicleColor].filter(Boolean).join(', ') || '-'}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">Registration Number</p>
                <p className="font-medium text-foreground">{profile?.vehicleNumber || '-'}</p>
              </div>
              <div>
                <p className="text-muted-foreground">License Number</p>
                <p className="font-medium text-foreground">{profile?.licenseNumber || '-'}</p>
              </div>
            </>
          )}
          {profile?.vehicleColor && profile?.vehicleType === 'bicycle' && (
            <div>
              <p className="text-muted-foreground">Color</p>
              <p className="font-medium text-foreground">{profile.vehicleColor}</p>
            </div>
          )}
        </div>
      </div>

      {/* Documents Section */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <FileText className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-semibold text-foreground">
              Documents ({docs.length} uploaded)
            </h3>
          </div>
          <button
            type="button"
            onClick={() => onGoToStep(3)}
            className="flex items-center gap-1 text-xs font-medium text-primary hover:text-primary/80"
          >
            <Pencil className="h-3 w-3" />
            Edit
          </button>
        </div>
        <div className="space-y-2">
          {docs.map((doc) => (
            <div key={doc.id} className="flex items-center justify-between text-sm">
              <span className="text-foreground">{docTypeLabels[doc.type] || doc.type}</span>
              <span className="flex items-center gap-1 text-green-600">
                <CheckCircle className="h-3 w-3" />
                {doc.status === 'verified' ? 'Verified' : 'Uploaded'}
              </span>
            </div>
          ))}
          {docs.length === 0 && (
            <div className="flex items-center gap-2 text-sm text-destructive">
              <AlertCircle className="h-4 w-4" />
              No documents uploaded
            </div>
          )}
        </div>
      </div>

      {/* Subscription Plan Section */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Wallet className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-semibold text-foreground">Subscription Plan</h3>
          </div>
          <button
            type="button"
            onClick={() => onGoToStep(4)}
            className="flex items-center gap-1 text-xs font-medium text-primary hover:text-primary/80"
          >
            <Pencil className="h-3 w-3" />
            Change
          </button>
        </div>
        <div className="text-sm">
          <p className="text-muted-foreground">
            Plan selected. Payments handled securely via Razorpay.
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            Billing starts only after you reach the minimum earnings threshold.
          </p>
        </div>
      </div>

      {/* Terms */}
      <div className="rounded-xl border border-border bg-card p-5">
        <label className="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={termsAccepted}
            onChange={(e) => setTermsAccepted(e.target.checked)}
            className="mt-0.5 h-4 w-4 rounded border-border text-primary focus:ring-primary"
          />
          <span className="text-sm text-foreground">
            I agree to the{' '}
            <span className="text-primary font-medium">Terms & Conditions</span>,{' '}
            <span className="text-primary font-medium">Privacy Policy</span>, and confirm that
            all information provided is accurate and up to date.
          </span>
        </label>
      </div>

      <div className="flex gap-3">
        <button
          type="button"
          onClick={onBack}
          className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium text-foreground hover:bg-secondary"
        >
          Back
        </button>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={submitting || !termsAccepted}
          className="flex-1 rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {submitting ? 'Submitting...' : 'Submit Application'}
        </button>
      </div>
    </div>
  );
}
