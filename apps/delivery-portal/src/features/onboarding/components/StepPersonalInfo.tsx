import { useState, useEffect } from 'react';
import { Bike, Car, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { toast } from 'sonner';
import { getCachedFormData, setCachedFormData, clearStepCache } from '@/shared/utils/form-cache';

const vehicleTypes = [
  { value: 'bicycle', label: 'Bicycle', icon: Bike },
  { value: 'scooter', label: 'Scooter/Motorcycle', icon: Bike },
  { value: 'car', label: 'Car', icon: Car },
];

interface PersonalInfoData {
  city: string;
  emergencyContact: string;
  emergencyPhone: string;
  dateOfBirth: string;
  vehicleType: string;
  referralCode: string;
}

interface StepPersonalInfoProps {
  initialData?: Partial<PersonalInfoData>;
  onComplete: () => void;
}

export function StepPersonalInfo({ initialData, onComplete }: StepPersonalInfoProps) {
  const cached = getCachedFormData('personal');

  const [form, setForm] = useState<PersonalInfoData>({
    city: cached?.city ?? initialData?.city ?? '',
    emergencyContact: cached?.emergencyContact ?? initialData?.emergencyContact ?? '',
    emergencyPhone: cached?.emergencyPhone ?? initialData?.emergencyPhone ?? '',
    dateOfBirth: cached?.dateOfBirth ?? initialData?.dateOfBirth ?? '',
    vehicleType: cached?.vehicleType ?? initialData?.vehicleType ?? '',
    referralCode: cached?.referralCode ?? initialData?.referralCode ?? '',
  });
  const [submitting, setSubmitting] = useState(false);
  const [referralStatus, setReferralStatus] = useState<'idle' | 'validating' | 'valid' | 'invalid'>('idle');
  const [referrerName, setReferrerName] = useState('');

  // Cache form data on every change
  useEffect(() => {
    setCachedFormData('personal', form as unknown as Record<string, string>);
  }, [form]);

  const updateField = (field: keyof PersonalInfoData, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    if (field === 'referralCode') {
      setReferralStatus('idle');
      setReferrerName('');
    }
  };

  const validateReferral = async () => {
    if (!form.referralCode.trim()) return;
    setReferralStatus('validating');
    try {
      const result = await apiClient.post<{ valid: boolean; referrerName: string }>(
        '/driver/referral/validate',
        { code: form.referralCode }
      );
      if (result.valid) {
        setReferralStatus('valid');
        setReferrerName(result.referrerName);
      } else {
        setReferralStatus('invalid');
      }
    } catch {
      setReferralStatus('invalid');
    }
  };

  const handleSubmit = async () => {
    if (!form.city.trim() || !form.emergencyContact.trim() || !form.emergencyPhone.trim() || !form.vehicleType) {
      toast.error('Please fill in all required fields');
      return;
    }

    setSubmitting(true);
    try {
      await apiClient.post('/driver/onboarding/personal', {
        city: form.city,
        emergencyContact: form.emergencyContact,
        emergencyPhone: form.emergencyPhone,
        dateOfBirth: form.dateOfBirth || undefined,
        vehicleType: form.vehicleType,
        referralCode: form.referralCode || undefined,
      });
      clearStepCache('personal');
      toast.success('Personal info saved');
      onComplete();
    } catch {
      toast.error('Failed to save. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const isValid = form.city.trim() && form.emergencyContact.trim() && form.emergencyPhone.trim() && form.vehicleType;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-foreground">Personal Information</h2>
        <p className="mt-1 text-sm text-muted-foreground">Tell us about yourself to get started</p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            City <span className="text-destructive">*</span>
          </label>
          <input
            type="text"
            value={form.city}
            onChange={(e) => updateField('city', e.target.value)}
            placeholder="e.g., Bangalore"
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            Emergency Contact Name <span className="text-destructive">*</span>
          </label>
          <input
            type="text"
            value={form.emergencyContact}
            onChange={(e) => updateField('emergencyContact', e.target.value)}
            placeholder="Full name"
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            Emergency Phone <span className="text-destructive">*</span>
          </label>
          <input
            type="tel"
            value={form.emergencyPhone}
            onChange={(e) => updateField('emergencyPhone', e.target.value)}
            placeholder="+91 9876543210"
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">Date of Birth</label>
          <input
            type="date"
            value={form.dateOfBirth}
            onChange={(e) => updateField('dateOfBirth', e.target.value)}
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-3">
            Vehicle Type <span className="text-destructive">*</span>
          </label>
          <div className="grid grid-cols-3 gap-3">
            {vehicleTypes.map((type) => {
              const Icon = type.icon;
              return (
                <button
                  key={type.value}
                  type="button"
                  onClick={() => updateField('vehicleType', type.value)}
                  className={`flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all ${
                    form.vehicleType === type.value
                      ? 'border-primary bg-primary/5 text-primary'
                      : 'border-border hover:border-primary/30'
                  }`}
                >
                  <Icon className="h-6 w-6" />
                  <span className="text-xs font-medium">{type.label}</span>
                </button>
              );
            })}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">Referral Code</label>
          <div className="flex gap-2">
            <input
              type="text"
              value={form.referralCode}
              onChange={(e) => updateField('referralCode', e.target.value)}
              placeholder="Enter referral code (optional)"
              className="flex-1 rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <button
              type="button"
              onClick={validateReferral}
              disabled={!form.referralCode.trim() || referralStatus === 'validating'}
              className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium text-foreground hover:bg-secondary disabled:opacity-50"
            >
              {referralStatus === 'validating' ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                'Validate'
              )}
            </button>
          </div>
          {referralStatus === 'valid' && (
            <p className="mt-1.5 flex items-center gap-1 text-sm text-green-600">
              <CheckCircle className="h-4 w-4" />
              Referred by {referrerName}
            </p>
          )}
          {referralStatus === 'invalid' && (
            <p className="mt-1.5 flex items-center gap-1 text-sm text-destructive">
              <AlertCircle className="h-4 w-4" />
              Invalid referral code
            </p>
          )}
        </div>
      </div>

      <button
        type="button"
        onClick={handleSubmit}
        disabled={submitting || !isValid}
        className="w-full rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {submitting ? 'Saving...' : 'Continue'}
      </button>
    </div>
  );
}
