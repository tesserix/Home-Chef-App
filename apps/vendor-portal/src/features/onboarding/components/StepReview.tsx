import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import {
  User,
  MapPin,
  ChefHat,
  Clock,
  FileText,
  Shield,
  CheckCircle2,
  AlertCircle,
  IndianRupee,
  Truck,
} from 'lucide-react';

function SectionCard({
  icon,
  title,
  stepNumber,
  onEdit,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  stepNumber: number;
  onEdit: (step: number) => void;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {icon}
          <h3 className="text-base font-semibold text-foreground">{title}</h3>
        </div>
        <button
          type="button"
          onClick={() => onEdit(stepNumber)}
          className="text-sm font-medium text-primary hover:text-primary/80 transition-colors"
        >
          Edit
        </button>
      </div>
      <div className="mt-4">{children}</div>
    </Card>
  );
}

function Field({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-medium text-foreground">{value || '—'}</p>
    </div>
  );
}

export function StepReview({ onEdit }: { onEdit: (step: number) => void }) {
  const { data } = useOnboardingStore();

  const requiredDocs = ['pan_card', 'aadhaar_card', 'kitchen_photo_1'];
  const uploadedTypes = data.documents.map((d) => d.type);
  const missingDocs = requiredDocs.filter((d) => !uploadedTypes.includes(d as never));

  const addr = data.kitchenAddress;
  const addressStr = [addr.line1, addr.line2, addr.city, addr.state, addr.postalCode]
    .filter(Boolean)
    .join(', ');

  const openDays = Object.entries(data.operatingHours)
    .filter(([, v]) => v)
    .map(([day, hours]) => `${day.charAt(0).toUpperCase() + day.slice(1)} (${hours!.open}–${hours!.close})`);

  return (
    <div className="space-y-6">
      {/* Warnings */}
      {missingDocs.length > 0 && (
        <div className="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4">
          <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <div>
            <p className="text-sm font-medium text-amber-900">Missing required documents</p>
            <p className="mt-1 text-xs text-amber-700">
              Please go back to Step 4 and upload: {missingDocs.map((d) => d.replace(/_/g, ' ')).join(', ')}
            </p>
          </div>
        </div>
      )}

      {!(data.acceptedTerms && data.acceptedHygienePolicy && data.acceptedCancellationPolicy) && (
        <div className="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4">
          <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <div>
            <p className="text-sm font-medium text-amber-900">Policies not accepted</p>
            <p className="mt-1 text-xs text-amber-700">
              Please go back to Step 5 and accept all required policies.
            </p>
          </div>
        </div>
      )}

      {/* Personal Info */}
      <SectionCard
        icon={<User className="h-5 w-5 text-muted-foreground" />}
        title="Personal Information"
        stepNumber={0}
        onEdit={onEdit}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Full Name" value={data.fullName} />
          <Field label="Phone" value={data.phone} />
          <Field label="Email" value={data.email} />
          <Field label="Kitchen Address" value={addressStr} />
        </div>
      </SectionCard>

      {/* Kitchen Details */}
      <SectionCard
        icon={<ChefHat className="h-5 w-5 text-muted-foreground" />}
        title="Kitchen Details"
        stepNumber={1}
        onEdit={onEdit}
      >
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Business Name" value={data.businessName} />
            <Field label="Kitchen Type" value={data.kitchenType.replace(/_/g, ' ')} />
            <Field label="Experience" value={data.yearsOfExperience} />
            <Field label="Daily Capacity" value={data.mealsPerDay} />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Description</p>
            <p className="mt-1 text-sm text-foreground">{data.description || '—'}</p>
          </div>
          <div>
            <p className="mb-2 text-xs text-muted-foreground">Cuisines</p>
            <div className="flex flex-wrap gap-1.5">
              {data.cuisines.map((c) => (
                <Badge key={c} variant="brand" size="sm">{c}</Badge>
              ))}
              {data.cuisines.length === 0 && <span className="text-sm text-muted-foreground">None selected</span>}
            </div>
          </div>
          {data.specialties.length > 0 && (
            <div>
              <p className="mb-2 text-xs text-muted-foreground">Specialties</p>
              <div className="flex flex-wrap gap-1.5">
                {data.specialties.map((s) => (
                  <Badge key={s} variant="outline" size="sm">{s}</Badge>
                ))}
              </div>
            </div>
          )}
        </div>
      </SectionCard>

      {/* Operations */}
      <SectionCard
        icon={<Clock className="h-5 w-5 text-muted-foreground" />}
        title="Operations & Pricing"
        stepNumber={2}
        onEdit={onEdit}
      >
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <Field label="Prep Time" value={data.prepTime} />
            </div>
            <div className="flex items-center gap-2">
              <MapPin className="h-4 w-4 text-muted-foreground" />
              <Field label="Delivery Radius" value={`${data.serviceRadius} km`} />
            </div>
            <div className="flex items-center gap-2">
              <IndianRupee className="h-4 w-4 text-muted-foreground" />
              <Field label="Min Order" value={`₹${data.minimumOrder}`} />
            </div>
            <div className="flex items-center gap-2">
              <Truck className="h-4 w-4 text-muted-foreground" />
              <Field label="Delivery Fee" value={`₹${data.deliveryFee}`} />
            </div>
          </div>
          <div>
            <p className="mb-1 text-xs text-muted-foreground">Operating Days</p>
            <p className="text-sm text-foreground">
              {openDays.length > 0 ? openDays.join(' · ') : 'No days set'}
            </p>
          </div>
        </div>
      </SectionCard>

      {/* Documents */}
      <SectionCard
        icon={<FileText className="h-5 w-5 text-muted-foreground" />}
        title="Documents"
        stepNumber={3}
        onEdit={onEdit}
      >
        <div className="space-y-2">
          {data.documents.length === 0 ? (
            <p className="text-sm text-muted-foreground">No documents uploaded</p>
          ) : (
            data.documents.map((doc) => (
              <div key={doc.type} className="flex items-center gap-3 rounded-lg bg-secondary/50 p-2.5">
                <CheckCircle2 className="h-4 w-4 text-green-600" />
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {doc.type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
                  </p>
                  <p className="text-xs text-muted-foreground">{doc.fileName}</p>
                </div>
              </div>
            ))
          )}
          {data.fssaiLicenseNumber && (
            <div className="flex items-center gap-2 text-sm text-foreground">
              <Shield className="h-4 w-4 text-primary" />
              FSSAI: {data.fssaiLicenseNumber}
            </div>
          )}
          {data.panNumber && (
            <div className="flex items-center gap-2 text-sm text-foreground">
              <FileText className="h-4 w-4 text-muted-foreground" />
              PAN: {data.panNumber}
            </div>
          )}
        </div>
      </SectionCard>

      {/* Policies */}
      <SectionCard
        icon={<Shield className="h-5 w-5 text-muted-foreground" />}
        title="Policies & Agreements"
        stepNumber={4}
        onEdit={onEdit}
      >
        <div className="space-y-2">
          {[
            { label: 'Hygiene & Food Safety', accepted: data.acceptedHygienePolicy },
            { label: 'Order & Cancellation Policy', accepted: data.acceptedCancellationPolicy },
            { label: 'Terms of Service', accepted: data.acceptedTerms },
          ].map((p) => (
            <div key={p.label} className="flex items-center gap-2">
              {p.accepted ? (
                <CheckCircle2 className="h-4 w-4 text-green-600" />
              ) : (
                <AlertCircle className="h-4 w-4 text-amber-500" />
              )}
              <span className="text-sm text-foreground">{p.label}</span>
            </div>
          ))}
        </div>
      </SectionCard>
    </div>
  );
}
