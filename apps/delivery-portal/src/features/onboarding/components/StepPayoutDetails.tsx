import { useState, useEffect } from 'react';
import { Banknote, Smartphone } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { toast } from 'sonner';
import { getCachedFormData, setCachedFormData, clearStepCache } from '@/shared/utils/form-cache';

type PayoutMethod = 'bank_transfer' | 'upi';

interface PayoutData {
  payoutMethod: PayoutMethod;
  bankAccountNumber: string;
  bankIFSC: string;
  bankAccountName: string;
  upiId: string;
}

interface StepPayoutDetailsProps {
  initialData?: Partial<PayoutData>;
  onComplete: () => void;
  onBack: () => void;
}

export function StepPayoutDetails({ initialData, onComplete, onBack }: StepPayoutDetailsProps) {
  const cached = getCachedFormData('payout');

  const [form, setForm] = useState<PayoutData>({
    payoutMethod: (cached?.payoutMethod ?? initialData?.payoutMethod ?? 'bank_transfer') as PayoutMethod,
    bankAccountNumber: cached?.bankAccountNumber ?? initialData?.bankAccountNumber ?? '',
    bankIFSC: cached?.bankIFSC ?? initialData?.bankIFSC ?? '',
    bankAccountName: cached?.bankAccountName ?? initialData?.bankAccountName ?? '',
    upiId: cached?.upiId ?? initialData?.upiId ?? '',
  });
  const [submitting, setSubmitting] = useState(false);

  // Cache form data on every change
  useEffect(() => {
    setCachedFormData('payout', form as unknown as Record<string, string>);
  }, [form]);

  const updateField = (field: keyof PayoutData, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async () => {
    if (form.payoutMethod === 'bank_transfer') {
      if (!form.bankAccountName.trim() || !form.bankAccountNumber.trim() || !form.bankIFSC.trim()) {
        toast.error('Please fill in all bank details');
        return;
      }
    } else {
      if (!form.upiId.trim()) {
        toast.error('Please enter your UPI ID');
        return;
      }
    }

    setSubmitting(true);
    try {
      await apiClient.post('/driver/onboarding/payout', {
        payoutMethod: form.payoutMethod,
        bankAccountNumber: form.payoutMethod === 'bank_transfer' ? form.bankAccountNumber : undefined,
        bankIFSC: form.payoutMethod === 'bank_transfer' ? form.bankIFSC : undefined,
        bankAccountName: form.payoutMethod === 'bank_transfer' ? form.bankAccountName : undefined,
        upiId: form.payoutMethod === 'upi' ? form.upiId : undefined,
      });
      clearStepCache('payout');
      toast.success('Payout details saved');
      onComplete();
    } catch {
      toast.error('Failed to save. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const isValid =
    form.payoutMethod === 'bank_transfer'
      ? form.bankAccountName.trim() && form.bankAccountNumber.trim() && form.bankIFSC.trim()
      : form.upiId.trim();

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-foreground">Payout Details</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          How would you like to receive your earnings?
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-foreground mb-3">Payout Method</label>
          <div className="grid grid-cols-2 gap-3">
            <button
              type="button"
              onClick={() => updateField('payoutMethod', 'bank_transfer')}
              className={`flex items-center gap-3 rounded-xl border-2 p-4 transition-all ${
                form.payoutMethod === 'bank_transfer'
                  ? 'border-primary bg-primary/5 text-primary'
                  : 'border-border hover:border-primary/30'
              }`}
            >
              <Banknote className="h-5 w-5" />
              <span className="text-sm font-medium">Bank Transfer</span>
            </button>
            <button
              type="button"
              onClick={() => updateField('payoutMethod', 'upi')}
              className={`flex items-center gap-3 rounded-xl border-2 p-4 transition-all ${
                form.payoutMethod === 'upi'
                  ? 'border-primary bg-primary/5 text-primary'
                  : 'border-border hover:border-primary/30'
              }`}
            >
              <Smartphone className="h-5 w-5" />
              <span className="text-sm font-medium">UPI</span>
            </button>
          </div>
        </div>

        {form.payoutMethod === 'bank_transfer' ? (
          <>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                Account Holder Name <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={form.bankAccountName}
                onChange={(e) => updateField('bankAccountName', e.target.value)}
                placeholder="Name as on bank account"
                className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                Account Number <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={form.bankAccountNumber}
                onChange={(e) => updateField('bankAccountNumber', e.target.value)}
                placeholder="Enter account number"
                className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                IFSC Code <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                value={form.bankIFSC}
                onChange={(e) => updateField('bankIFSC', e.target.value)}
                placeholder="e.g., SBIN0001234"
                className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </>
        ) : (
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">
              UPI ID <span className="text-destructive">*</span>
            </label>
            <input
              type="text"
              value={form.upiId}
              onChange={(e) => updateField('upiId', e.target.value)}
              placeholder="e.g., name@upi"
              className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
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
