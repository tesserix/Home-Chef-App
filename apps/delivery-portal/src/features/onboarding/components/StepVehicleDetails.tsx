import { useState, useEffect } from 'react';
import { apiClient } from '@/shared/services/api-client';
import { toast } from 'sonner';
import { getCachedFormData, setCachedFormData, clearStepCache } from '@/shared/utils/form-cache';

interface VehicleData {
  vehicleType: string;
  vehicleMake: string;
  vehicleModel: string;
  vehicleYear: string;
  vehicleColor: string;
  vehicleNumber: string;
  licenseNumber: string;
  hasDeliveryBoxSpace: string;
}

interface StepVehicleDetailsProps {
  initialData?: Partial<VehicleData>;
  onComplete: () => void;
  onBack: () => void;
}

function toBoxSpaceString(val: unknown): string {
  if (val === 'yes' || val === 'no') return val;
  if (val === true || val === 'true') return 'yes';
  if (val === false || val === 'false') return 'no';
  return '';
}

export function StepVehicleDetails({ initialData, onComplete, onBack }: StepVehicleDetailsProps) {
  const cached = getCachedFormData('vehicle');
  const vehicleType = cached?.vehicleType ?? initialData?.vehicleType ?? '';
  const isBicycle = vehicleType === 'bicycle';

  const [form, setForm] = useState<VehicleData>({
    vehicleType,
    vehicleMake: cached?.vehicleMake ?? initialData?.vehicleMake ?? '',
    vehicleModel: cached?.vehicleModel ?? initialData?.vehicleModel ?? '',
    vehicleYear: cached?.vehicleYear ?? initialData?.vehicleYear ?? '',
    vehicleColor: cached?.vehicleColor ?? initialData?.vehicleColor ?? '',
    vehicleNumber: cached?.vehicleNumber ?? initialData?.vehicleNumber ?? '',
    licenseNumber: cached?.licenseNumber ?? initialData?.licenseNumber ?? '',
    hasDeliveryBoxSpace: cached?.hasDeliveryBoxSpace ?? toBoxSpaceString(initialData?.hasDeliveryBoxSpace),
  });
  const [submitting, setSubmitting] = useState(false);

  // Cache form data on every change
  useEffect(() => {
    setCachedFormData('vehicle', form as unknown as Record<string, string>);
  }, [form]);

  const updateField = (field: keyof VehicleData, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async () => {
    if (isBicycle) {
      if (!form.hasDeliveryBoxSpace) {
        toast.error('Please indicate if your bicycle can carry a delivery box');
        return;
      }
    } else {
      if (!form.vehicleNumber.trim() || !form.licenseNumber.trim()) {
        toast.error('Please fill in all required fields');
        return;
      }
    }

    setSubmitting(true);
    try {
      await apiClient.post('/driver/onboarding/vehicle', {
        vehicleType: form.vehicleType || undefined,
        vehicleMake: form.vehicleMake || undefined,
        vehicleModel: form.vehicleModel || undefined,
        vehicleYear: form.vehicleYear ? Number(form.vehicleYear) : undefined,
        vehicleColor: form.vehicleColor || undefined,
        vehicleNumber: isBicycle ? undefined : form.vehicleNumber,
        licenseNumber: isBicycle ? undefined : form.licenseNumber,
        hasDeliveryBoxSpace: isBicycle ? form.hasDeliveryBoxSpace === 'yes' : undefined,
      });
      clearStepCache('vehicle');
      toast.success('Vehicle details saved');
      onComplete();
    } catch {
      toast.error('Failed to save. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const isValid = isBicycle
    ? !!form.hasDeliveryBoxSpace
    : form.vehicleNumber.trim() && form.licenseNumber.trim();

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-foreground">
          {isBicycle ? 'Bicycle Details' : 'Vehicle Details'}
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          {isBicycle ? 'Tell us about your bicycle' : 'Tell us about your vehicle'}
        </p>
      </div>

      <div className="space-y-4">
        {isBicycle ? (
          <>
            {/* Bicycle-specific fields */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">Bicycle Brand</label>
                <input
                  type="text"
                  value={form.vehicleMake}
                  onChange={(e) => updateField('vehicleMake', e.target.value)}
                  placeholder="e.g., Hero, Atlas, Firefox"
                  className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">Bicycle Model</label>
                <input
                  type="text"
                  value={form.vehicleModel}
                  onChange={(e) => updateField('vehicleModel', e.target.value)}
                  placeholder="e.g., Sprint Pro"
                  className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Bicycle Color</label>
              <input
                type="text"
                value={form.vehicleColor}
                onChange={(e) => updateField('vehicleColor', e.target.value)}
                placeholder="e.g., Black"
                className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-3">
                Can your bicycle carry a delivery box / bag? <span className="text-destructive">*</span>
              </label>
              <p className="mb-3 text-xs text-muted-foreground">
                You'll need a carrier rack or basket to hold the delivery bag securely.
              </p>
              <div className="grid grid-cols-2 gap-3">
                <button
                  type="button"
                  onClick={() => updateField('hasDeliveryBoxSpace', 'yes')}
                  className={`flex items-center justify-center gap-2 rounded-xl border-2 p-4 transition-all ${
                    form.hasDeliveryBoxSpace === 'yes'
                      ? 'border-primary bg-primary/5 text-primary'
                      : 'border-border hover:border-primary/30'
                  }`}
                >
                  <span className="text-sm font-medium">Yes, it has a carrier/rack</span>
                </button>
                <button
                  type="button"
                  onClick={() => updateField('hasDeliveryBoxSpace', 'no')}
                  className={`flex items-center justify-center gap-2 rounded-xl border-2 p-4 transition-all ${
                    form.hasDeliveryBoxSpace === 'no'
                      ? 'border-primary bg-primary/5 text-primary'
                      : 'border-border hover:border-primary/30'
                  }`}
                >
                  <span className="text-sm font-medium">No, not yet</span>
                </button>
              </div>
              {form.hasDeliveryBoxSpace === 'no' && (
                <p className="mt-2 rounded-lg bg-amber-50 p-3 text-xs text-amber-700">
                  You can still proceed, but you'll need to attach a carrier or use a delivery backpack before starting deliveries.
                </p>
              )}
            </div>
          </>
        ) : (
          <>
            {/* Motor vehicle fields */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">Vehicle Make</label>
                <input
                  type="text"
                  value={form.vehicleMake}
                  onChange={(e) => updateField('vehicleMake', e.target.value)}
                  placeholder="e.g., Honda, Bajaj"
                  className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">Vehicle Model</label>
                <input
                  type="text"
                  value={form.vehicleModel}
                  onChange={(e) => updateField('vehicleModel', e.target.value)}
                  placeholder="e.g., Activa 6G"
                  className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">Vehicle Year</label>
                <input
                  type="number"
                  value={form.vehicleYear}
                  onChange={(e) => updateField('vehicleYear', e.target.value)}
                  placeholder="e.g., 2022"
                  min="1990"
                  max="2030"
                  className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1.5">Vehicle Color</label>
                <input
                  type="text"
                  value={form.vehicleColor}
                  onChange={(e) => updateField('vehicleColor', e.target.value)}
                  placeholder="e.g., Black"
                  className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                Vehicle Registration Number <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={form.vehicleNumber}
                onChange={(e) => updateField('vehicleNumber', e.target.value)}
                placeholder="e.g., KA01AB1234"
                className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                Driving License Number <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={form.licenseNumber}
                onChange={(e) => updateField('licenseNumber', e.target.value)}
                placeholder="e.g., DL1234567890"
                className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </>
        )}
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
          disabled={submitting || !isValid}
          className="flex-1 rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {submitting ? 'Saving...' : 'Continue'}
        </button>
      </div>
    </div>
  );
}
