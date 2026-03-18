import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Card } from '@/shared/components/ui/Card';
import {
  Shield,
  Banknote,
  CheckCircle2,
  AlertTriangle,
  Heart,
  Sparkles,
  BookOpen,
} from 'lucide-react';

interface Props {
  errors: Record<string, string>;
}

interface PolicyItem {
  key: 'acceptedTerms' | 'acceptedHygienePolicy' | 'acceptedCancellationPolicy';
  title: string;
  description: string;
  icon: React.ReactNode;
}

const POLICIES: PolicyItem[] = [
  {
    key: 'acceptedHygienePolicy',
    title: 'Kitchen Hygiene & Food Safety Commitment',
    description:
      'I commit to maintaining a clean and hygienic kitchen at all times. I will use fresh ingredients, follow proper food handling practices, store food at safe temperatures, and ensure all cooking utensils and surfaces are sanitised regularly. I understand Fe3dr may conduct periodic kitchen checks.',
    icon: <Heart className="h-5 w-5 text-pink-500" />,
  },
  {
    key: 'acceptedCancellationPolicy',
    title: 'Order & Cancellation Policy',
    description:
      'I agree to accept or reject orders within 5 minutes. Once accepted, I will prepare the food within the estimated time. If I cannot fulfill an order, I will cancel immediately so the customer can be notified. Repeated late cancellations may affect my kitchen rating and visibility.',
    icon: <AlertTriangle className="h-5 w-5 text-amber-500" />,
  },
  {
    key: 'acceptedTerms',
    title: 'Fe3dr Platform Terms of Service',
    description:
      'I have read and agree to the Fe3dr Terms of Service, Privacy Policy, and Vendor Agreement. I understand that Fe3dr charges a platform commission on each order and that my payouts will be processed weekly to my registered bank account.',
    icon: <BookOpen className="h-5 w-5 text-blue-500" />,
  },
];

const COMPLIANCE_ITEMS = [
  'My kitchen has access to clean running water',
  'I use separate cutting boards for vegetarian and non-vegetarian items',
  'I store raw and cooked food separately',
  'I wear clean attire and maintain personal hygiene while cooking',
  'I have a designated waste disposal area away from cooking space',
  'I ensure all food is freshly prepared on the day of delivery',
  'I will label allergens and ingredients for each dish',
];

export function StepPolicies({ errors }: Props) {
  const { data, updateData } = useOnboardingStore();

  const allPoliciesAccepted =
    data.acceptedTerms && data.acceptedHygienePolicy && data.acceptedCancellationPolicy;

  return (
    <div className="space-y-6">
      {/* Compliance Checklist */}
      <Card>
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold text-foreground">Kitchen Compliance Checklist</h3>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          Please confirm that your kitchen meets these basic food safety requirements.
          This ensures all home kitchens on Fe3dr maintain quality standards.
        </p>

        <div className="mt-4 space-y-2">
          {COMPLIANCE_ITEMS.map((item, i) => (
            <div key={i} className="flex items-start gap-3 rounded-lg bg-secondary/50 p-3">
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-green-600" />
              <span className="text-sm text-foreground">{item}</span>
            </div>
          ))}
        </div>
        <p className="mt-3 text-xs text-muted-foreground">
          By proceeding, you confirm that all the above conditions are met in your kitchen.
        </p>
      </Card>

      {/* Policies */}
      <Card>
        <div className="flex items-center gap-2">
          <Sparkles className="h-5 w-5 text-primary" />
          <h3 className="text-lg font-semibold text-foreground">Agreements & Policies</h3>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          Please read and accept each policy to complete your registration.
        </p>

        <div className="mt-4 space-y-4">
          {POLICIES.map((policy) => (
            <label
              key={policy.key}
              className={`flex cursor-pointer items-start gap-4 rounded-xl border-2 p-4 transition-all ${
                data[policy.key]
                  ? 'border-primary bg-primary/5'
                  : 'border-border hover:border-primary/30'
              }`}
            >
              <input
                type="checkbox"
                checked={data[policy.key]}
                onChange={(e) => updateData({ [policy.key]: e.target.checked })}
                className="mt-1 h-5 w-5 rounded border-input text-primary focus:ring-primary"
              />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  {policy.icon}
                  <p className="text-sm font-semibold text-foreground">{policy.title}</p>
                </div>
                <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                  {policy.description}
                </p>
              </div>
            </label>
          ))}
        </div>
        {errors.policies && (
          <p className="mt-2 text-sm text-destructive">{errors.policies}</p>
        )}
      </Card>

      {/* Payout Info */}
      <Card>
        <div className="flex items-center gap-2">
          <Banknote className="h-5 w-5 text-muted-foreground" />
          <h3 className="text-lg font-semibold text-foreground">Payout Details</h3>
        </div>
        <p className="mt-2 text-sm text-muted-foreground">
          You can set up your bank account or UPI details for receiving payouts from{' '}
          <span className="font-medium text-foreground">Settings</span> after your kitchen is approved.
        </p>
      </Card>

      {/* Summary */}
      {allPoliciesAccepted && (
        <div className="flex items-start gap-3 rounded-xl border border-green-200 bg-green-50 p-4">
          <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-green-600" />
          <div>
            <p className="text-sm font-medium text-green-900">
              You're all set to submit your application!
            </p>
            <p className="mt-1 text-xs text-green-700">
              Our team will review your details and you'll be notified within 24-48 hours.
              You can start setting up your menu in the meantime.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
