import { useState, useEffect } from 'react';
import { Crown, Check, Loader2, Heart, Shield, Sparkles, IndianRupee } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { toast } from 'sonner';

interface Plan {
  interval: string;
  amount: number;
  currency: string;
  savingsPercent?: number;
}

interface PlansResponse {
  plans: Plan[];
  trialDays: number;
  currency: string;
  paymentGateway: string;
  minEarningsThreshold: number;
}

interface StepSubscriptionPlanProps {
  onComplete: () => void;
  onBack: () => void;
}

const INTERVAL_LABELS: Record<string, string> = {
  monthly: 'Monthly',
  quarterly: 'Quarterly',
  yearly: 'Yearly',
};

const INTERVAL_DESCRIPTIONS: Record<string, string> = {
  monthly: 'Billed every month',
  quarterly: 'Billed every 3 months',
  yearly: 'Billed once a year',
};

const GATEWAY_LABELS: Record<string, string> = {
  razorpay: 'Razorpay',
  stripe: 'Stripe',
  esewa: 'eSewa',
  sslcommerz: 'SSLCommerz',
  jazzcash: 'JazzCash',
  payhere: 'PayHere',
};

function formatCurrency(amount: number, currency: string) {
  try {
    const locale = currency === 'AUD' ? 'en-AU' : currency === 'INR' ? 'en-IN' : 'en-US';
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency,
      maximumFractionDigits: 0,
    }).format(amount);
  } catch {
    return `${currency} ${amount}`;
  }
}

export function StepSubscriptionPlan({ onComplete, onBack }: StepSubscriptionPlanProps) {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [trialDays, setTrialDays] = useState(0);
  const [threshold, setThreshold] = useState(0);
  const [currency, setCurrency] = useState('INR');
  const [gateway, setGateway] = useState('razorpay');
  const [loading, setLoading] = useState(true);
  const [selectedInterval, setSelectedInterval] = useState('monthly');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const fetchPlans = async () => {
      try {
        const data = await apiClient.get<PlansResponse>('/driver/subscription/plans');
        setPlans(data.plans ?? []);
        setTrialDays(data.trialDays ?? 0);
        setThreshold(data.minEarningsThreshold ?? 0);
        setCurrency(data.currency ?? 'INR');
        setGateway(data.paymentGateway ?? 'razorpay');
      } catch {
        toast.error('Failed to load subscription plans');
      } finally {
        setLoading(false);
      }
    };
    fetchPlans();
  }, []);

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      await apiClient.post('/driver/subscription/plan', {
        interval: selectedInterval,
      });
      toast.success('Plan selected successfully');
      onComplete();
    } catch {
      toast.error('Failed to select plan. Please try again.');
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

  const selectedPlan = plans.find((p) => p.interval === selectedInterval);
  const thresholdFormatted = formatCurrency(threshold, currency);
  const gatewayLabel = GATEWAY_LABELS[gateway] || gateway;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-foreground">Choose Your Plan</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          A small subscription to keep Fe3dr running — you keep every rupee you earn.
        </p>
      </div>

      {/* Zero Commission Promise */}
      <div className="rounded-xl bg-success/5 border border-success/20 p-5">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-success/10 shrink-0">
            <Heart className="h-5 w-5 text-success" />
          </div>
          <div className="space-y-2">
            <p className="text-sm font-semibold text-foreground">
              We don't take any commission from your earnings
            </p>
            <p className="text-sm text-muted-foreground">
              Every delivery fee and every tip goes directly to your account — 100%.
              We only charge a small subscription fee to keep the platform running and
              help you get more orders. We're here to help you succeed, not to make
              profit from your hard work.
            </p>
          </div>
        </div>
      </div>

      {/* Trial + Threshold Info */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {trialDays > 0 && (
          <div className="rounded-xl bg-primary/5 border border-primary/20 p-4">
            <div className="flex items-center gap-3">
              <Crown className="h-5 w-5 text-primary shrink-0" />
              <div>
                <p className="text-sm font-semibold text-foreground">
                  {trialDays}-day free trial
                </p>
                <p className="text-xs text-muted-foreground">
                  Try everything free. No card needed.
                </p>
              </div>
            </div>
          </div>
        )}

        {threshold > 0 && (
          <div className="rounded-xl bg-warning/5 border border-warning/20 p-4">
            <div className="flex items-center gap-3">
              <IndianRupee className="h-5 w-5 text-warning shrink-0" />
              <div>
                <p className="text-sm font-semibold text-foreground">
                  No charge until you earn {thresholdFormatted}
                </p>
                <p className="text-xs text-muted-foreground">
                  We only bill after you start earning well.
                </p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* How it works */}
      <div className="rounded-xl border border-border bg-muted/30 p-4">
        <p className="text-xs font-medium text-foreground mb-2">How billing works:</p>
        <ol className="space-y-1.5 text-xs text-muted-foreground list-decimal list-inside">
          <li>Start with a <span className="font-medium text-foreground">{trialDays}-day free trial</span> — deliver and earn with no charges</li>
          <li>After the trial, your subscription activates but <span className="font-medium text-foreground">we won't charge you</span> until your earnings cross <span className="font-medium text-foreground">{thresholdFormatted}</span></li>
          <li>Once you cross the threshold, your subscription fee is deducted — and you <span className="font-medium text-foreground">keep 100% of everything you earn</span> after that</li>
          <li>Cancel anytime — no lock-in, no hidden fees</li>
        </ol>
      </div>

      {/* Plan Cards */}
      <div className="space-y-3">
        {plans.map((plan) => (
          <button
            key={plan.interval}
            type="button"
            onClick={() => setSelectedInterval(plan.interval)}
            className={`w-full rounded-xl border-2 p-4 text-left transition-all ${
              selectedInterval === plan.interval
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-primary/30'
            }`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className={`flex h-5 w-5 items-center justify-center rounded-full border-2 ${
                    selectedInterval === plan.interval
                      ? 'border-primary bg-primary'
                      : 'border-border'
                  }`}
                >
                  {selectedInterval === plan.interval && (
                    <Check className="h-3 w-3 text-primary-foreground" />
                  )}
                </div>
                <div>
                  <p className="text-sm font-semibold text-foreground">
                    {INTERVAL_LABELS[plan.interval] || plan.interval}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {INTERVAL_DESCRIPTIONS[plan.interval] || ''}
                  </p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-lg font-bold text-foreground">
                  {formatCurrency(plan.amount, plan.currency)}
                </p>
                {plan.savingsPercent && plan.savingsPercent > 0 && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-success/10 px-2 py-0.5 text-xs font-medium text-success">
                    <Sparkles className="h-3 w-3" />
                    Save {Math.round(plan.savingsPercent)}%
                  </span>
                )}
              </div>
            </div>
          </button>
        ))}
      </div>

      {/* Secure Payment */}
      <div className="rounded-xl border border-border bg-muted/30 p-4">
        <div className="flex items-start gap-3">
          <Shield className="h-5 w-5 text-muted-foreground mt-0.5 shrink-0" />
          <p className="text-xs text-muted-foreground">
            Payments are securely processed via <span className="font-medium text-foreground">{gatewayLabel}</span>.
            You can change your plan or cancel anytime from your dashboard.
          </p>
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
          disabled={submitting || !selectedPlan}
          className="flex-1 rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {submitting ? 'Saving...' : `Start with ${INTERVAL_LABELS[selectedInterval] || selectedInterval} Plan`}
        </button>
      </div>
    </div>
  );
}
