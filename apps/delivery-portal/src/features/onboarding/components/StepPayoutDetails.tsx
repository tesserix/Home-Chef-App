import { useState, useEffect } from 'react';
import { Banknote } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { toast } from 'sonner';
import { getCachedFormData, setCachedFormData } from '@/shared/utils/form-cache';

// UPI is not an accepted payout method (#767): Razorpay Route settles by
// NEFT/IMPS to a bank account and has no VPA destination, so a driver who
// nominated UPI could never be paid. Bank transfer is the only option.
interface PayoutData {
  payoutMethod: 'bank_transfer';
  bankAccountNumber: string;
  bankIFSC: string;
  bankAccountName: string;
}

interface StepPayoutDetailsProps {
  initialData?: Partial<PayoutData>;
  onComplete: () => void;
  onBack: () => void;
}

export function StepPayoutDetails({ initialData, onComplete, onBack }: StepPayoutDetailsProps) {
  const cached = getCachedFormData('payout');

  const [form, setForm] = useState<PayoutData>({
    payoutMethod: 'bank_transfer',
    bankAccountNumber: cached?.bankAccountNumber ?? initialData?.bankAccountNumber ?? '',
    bankIFSC: cached?.bankIFSC ?? initialData?.bankIFSC ?? '',
    bankAccountName: cached?.bankAccountName ?? initialData?.bankAccountName ?? '',
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
    if (!form.bankAccountName.trim() || !form.bankAccountNumber.trim() || !form.bankIFSC.trim()) {
      toast.error('Please fill in all bank details');
      return;
    }

    setSubmitting(true);
    try {
      await apiClient.post('/driver/onboarding/payout', {
        payoutMethod: 'bank_transfer',
        bankAccountNumber: form.bankAccountNumber,
        bankIFSC: form.bankIFSC,
        bankAccountName: form.bankAccountName,
      });
      // Keep the draft until the whole application is submitted so editing
      // this step from Review re-seeds the entered values instead of a blank
      // form.
      toast.success('Payout details saved');
      onComplete();
    } catch {
      toast.error('Failed to save. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const isValid =
    form.bankAccountName.trim() && form.bankAccountNumber.trim() && form.bankIFSC.trim();

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-foreground">Payout Details</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Earnings are paid to your bank account by NEFT/IMPS.
        </p>
      </div>

      <div className="space-y-4">
        <div className="flex items-center gap-3 rounded-xl border border-border bg-secondary/40 p-4">
          <Banknote aria-hidden="true" className="h-5 w-5 text-primary" />
          <span className="text-sm font-medium text-foreground">Bank Transfer</span>
        </div>

        <div>
          <label htmlFor="payout-bank-name" className="block text-sm font-medium text-foreground mb-1.5">
            Account Holder Name <span aria-hidden="true" className="text-muted-foreground">*</span>
          </label>
          <input
            id="payout-bank-name"
            type="text"
            autoComplete="name"
            required
            aria-required="true"
            value={form.bankAccountName}
            onChange={(e) => updateField('bankAccountName', e.target.value)}
            placeholder="Name as on bank account"
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <div>
          <label htmlFor="payout-bank-number" className="block text-sm font-medium text-foreground mb-1.5">
            Account Number <span aria-hidden="true" className="text-muted-foreground">*</span>
          </label>
          <input
            id="payout-bank-number"
            type="text"
            inputMode="numeric"
            required
            aria-required="true"
            value={form.bankAccountNumber}
            onChange={(e) => updateField('bankAccountNumber', e.target.value)}
            placeholder="Enter account number"
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <div>
          <label htmlFor="payout-bank-ifsc" className="block text-sm font-medium text-foreground mb-1.5">
            IFSC Code <span aria-hidden="true" className="text-muted-foreground">*</span>
          </label>
          <input
            id="payout-bank-ifsc"
            type="text"
            required
            aria-required="true"
            value={form.bankIFSC}
            onChange={(e) => updateField('bankIFSC', e.target.value)}
            placeholder="e.g., SBIN0001234"
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
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
