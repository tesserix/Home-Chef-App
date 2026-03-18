import { useState } from 'react';
import { toast } from 'sonner';
import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Input } from '@/shared/components/ui/Input';
import { Card } from '@/shared/components/ui/Card';
import { FileUpload } from '@tesserix/web';
import {
  FileText,
  Camera,
  ShieldCheck,
  Info,
  CheckCircle2,
  Loader2,
} from 'lucide-react';
import { uploadDocument } from '@/shared/services/upload-service';
import type { DocumentType } from '@/shared/types';

interface Props {
  errors: Record<string, string>;
}

interface DocSection {
  type: DocumentType;
  label: string;
  description: string;
  accept: string;
  required: boolean;
  icon: React.ReactNode;
}

const REQUIRED_DOCS: DocSection[] = [
  {
    type: 'pan_card',
    label: 'PAN Card',
    description: 'Required for tax purposes and payouts',
    accept: '.jpg,.jpeg,.png,.pdf',
    required: true,
    icon: <FileText className="h-5 w-5" />,
  },
  {
    type: 'aadhaar_card',
    label: 'Aadhaar Card',
    description: 'For identity verification',
    accept: '.jpg,.jpeg,.png,.pdf',
    required: true,
    icon: <ShieldCheck className="h-5 w-5" />,
  },
];

const KITCHEN_PHOTOS: DocSection[] = [
  {
    type: 'kitchen_photo_1',
    label: 'Kitchen Photo — Cooking Area',
    description: 'Show your main cooking space',
    accept: '.jpg,.jpeg,.png,.webp',
    required: true,
    icon: <Camera className="h-5 w-5" />,
  },
  {
    type: 'kitchen_photo_2',
    label: 'Kitchen Photo — Preparation Area',
    description: 'Where you prep ingredients',
    accept: '.jpg,.jpeg,.png,.webp',
    required: false,
    icon: <Camera className="h-5 w-5" />,
  },
  {
    type: 'kitchen_photo_3',
    label: 'Kitchen Photo — Storage / Packaging',
    description: 'How you store and package food',
    accept: '.jpg,.jpeg,.png,.webp',
    required: false,
    icon: <Camera className="h-5 w-5" />,
  },
];

const OPTIONAL_DOCS: DocSection[] = [
  {
    type: 'fssai_license',
    label: 'FSSAI License',
    description: 'If you have one — gives your profile a verified badge. You can add this later.',
    accept: '.jpg,.jpeg,.png,.pdf',
    required: false,
    icon: <ShieldCheck className="h-5 w-5" />,
  },
  {
    type: 'food_safety_cert',
    label: 'Food Safety Training Certificate',
    description: 'Any food handling or safety certification (optional)',
    accept: '.jpg,.jpeg,.png,.pdf',
    required: false,
    icon: <FileText className="h-5 w-5" />,
  },
  {
    type: 'cancelled_cheque',
    label: 'Cancelled Cheque / Bank Proof',
    description: 'For setting up direct payouts to your bank account (optional, can add later)',
    accept: '.jpg,.jpeg,.png,.pdf',
    required: false,
    icon: <FileText className="h-5 w-5" />,
  },
];

function DocUploadCard({ section }: { section: DocSection }) {
  const { data, addDocument, removeDocument } = useOnboardingStore();
  const [files, setFiles] = useState<File[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const existing = data.documents.find((d) => d.type === section.type);

  const handleChange = async (newFiles: File[]) => {
    if (newFiles.length > 0) {
      const file = newFiles[0]!;
      setFiles(newFiles);
      setIsUploading(true);
      try {
        const result = await uploadDocument(file, section.type);
        addDocument({
          type: section.type,
          fileName: file.name,
          fileUrl: result.fileUrl,
          status: 'pending',
        });
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Upload failed');
        setFiles([]);
        removeDocument(section.type);
      } finally {
        setIsUploading(false);
      }
    } else {
      setFiles([]);
      removeDocument(section.type);
    }
  };

  return (
    <div className="rounded-xl border border-border p-4">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary text-muted-foreground">
          {section.icon}
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-semibold text-foreground">{section.label}</p>
            {section.required && (
              <span className="rounded-full bg-destructive/10 px-2 py-0.5 text-xs font-medium text-destructive">
                Required
              </span>
            )}
            {isUploading && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
            {existing && !isUploading && (
              <CheckCircle2 className="h-4 w-4 text-green-600" />
            )}
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">{section.description}</p>
        </div>
      </div>
      <div className="mt-3">
        <FileUpload
          value={files}
          onValueChange={handleChange}
          accept={section.accept}
          multiple={false}
          maxFiles={1}
          maxSizeBytes={5 * 1024 * 1024}
          helperText="Drop file here or click to browse. Max 5 MB."
        />
      </div>
    </div>
  );
}

export function StepDocuments({ errors }: Props) {
  const { data, updateData } = useOnboardingStore();

  return (
    <div className="space-y-6">
      {/* Info banner */}
      <div className="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50 p-4">
        <Info className="mt-0.5 h-5 w-5 shrink-0 text-blue-600" />
        <div>
          <p className="text-sm font-medium text-blue-900">
            Documents help us verify your kitchen and enable payouts
          </p>
          <p className="mt-1 text-xs text-blue-700">
            Your documents are encrypted and stored securely. They are only used for verification.
            FSSAI license is optional — many home chefs start without one and add it later.
          </p>
        </div>
      </div>

      {/* Identity Documents */}
      <Card>
        <h3 className="text-lg font-semibold text-foreground">Identity Documents</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          These are required to verify your identity and set up payouts.
        </p>

        <div className="mt-4 space-y-4">
          <Input
            label="PAN Number"
            placeholder="ABCDE1234F"
            value={data.panNumber || ''}
            onChange={(e) => updateData({ panNumber: e.target.value.toUpperCase() })}
            error={errors.panNumber}
            hint="10-character alphanumeric PAN"
          />
          {REQUIRED_DOCS.map((doc) => (
            <DocUploadCard key={doc.type} section={doc} />
          ))}
        </div>
      </Card>

      {/* Kitchen Photos */}
      <Card>
        <h3 className="text-lg font-semibold text-foreground">Kitchen Photos</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Photos of your kitchen help build trust. At least one photo of your cooking area is required.
        </p>

        <div className="mt-4 space-y-4">
          {KITCHEN_PHOTOS.map((doc) => (
            <DocUploadCard key={doc.type} section={doc} />
          ))}
        </div>
      </Card>

      {/* Optional Documents */}
      <Card>
        <h3 className="text-lg font-semibold text-foreground">
          Additional Documents
          <span className="ml-2 text-sm font-normal text-muted-foreground">(Optional)</span>
        </h3>
        <p className="mt-1 text-sm text-muted-foreground">
          These are optional but help speed up verification and build customer trust.
        </p>

        <div className="mt-4 space-y-4">
          <Input
            label="FSSAI License Number (Optional)"
            placeholder="e.g. 12345678901234"
            value={data.fssaiLicenseNumber || ''}
            onChange={(e) => updateData({ fssaiLicenseNumber: e.target.value })}
            hint="14-digit FSSAI number. You can add this later from your profile settings."
          />
          {OPTIONAL_DOCS.map((doc) => (
            <DocUploadCard key={doc.type} section={doc} />
          ))}
        </div>
      </Card>
    </div>
  );
}
