import { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ChevronLeft } from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
import { openRazorpayCheckout } from '@/shared/utils/razorpay';

// Post-delivery tip (#45) — web parity. 100% pass-through to the chef and/or
// rider via the shared Razorpay web checkout.
const PRESETS = [20, 50, 100];

interface CreateTipResponse {
  tipId: string;
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number;
  currency: string;
}

export default function TipPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [chef, setChef] = useState(50);
  const [rider, setRider] = useState(0);
  const [busy, setBusy] = useState(false);
  const total = chef + rider;

  async function sendTip() {
    if (!id || total < 1) return;
    setBusy(true);
    try {
      const data = await apiClient.post<CreateTipResponse>(
        `/payments/order/${id}/tip`,
        { chefAmount: chef, riderAmount: rider }
      );
      openRazorpayCheckout({
        data,
        description: 'Tip for your chef / rider',
        onVerified: async (resp) => {
          await apiClient.post(`/payments/tip/${data.tipId}/verify`, {
            razorpayPaymentId: resp.razorpay_payment_id,
            razorpayOrderId: resp.razorpay_order_id,
          });
          toast.success('Tip sent — thank you!');
          navigate(`/orders/${id}`);
        },
        onDismiss: () => toast.error('Tip cancelled'),
      });
    } catch {
      toast.error('Could not start the tip. Please try again.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-6">
      <Link to={`/orders/${id}`} className="mb-4 inline-flex items-center text-sm text-ink-muted hover:text-ink">
        <ChevronLeft className="h-4 w-4" aria-hidden="true" /> Back to order
      </Link>
      <h1 className="text-2xl font-semibold text-ink">Add a tip</h1>
      <p className="mt-1 text-ink-soft">
        Say thanks — 100% goes straight to your chef and rider, with no platform cut.
      </p>

      <AmountPicker label="Your chef" amount={chef} onChange={setChef} />
      <AmountPicker
        label="Your rider"
        caption="Only if a rider delivered your order"
        amount={rider}
        onChange={setRider}
      />

      <div className="mt-6 flex items-center justify-between border-t border-mist pt-4">
        <span className="text-ink-soft">Total tip</span>
        <span className="text-xl font-semibold text-ink tabular-nums">₹{total}</span>
      </div>
      <Button
        variant="primary"
        fullWidth
        className="mt-4"
        isLoading={busy}
        disabled={total < 1}
        onClick={sendTip}
      >
        {total < 1 ? 'Choose an amount' : `Tip ₹${total}`}
      </Button>
    </div>
  );
}

function AmountPicker({
  label,
  caption,
  amount,
  onChange,
}: {
  label: string;
  caption?: string;
  amount: number;
  onChange: (n: number) => void;
}) {
  const isCustom = amount > 0 && !PRESETS.includes(amount);
  return (
    <div className="mt-6 rounded-xl border border-mist bg-paper p-4">
      <p className="font-medium text-ink">{label}</p>
      {caption && <p className="text-xs text-ink-muted">{caption}</p>}
      <div className="mt-3 flex flex-wrap gap-2">
        <Chip label="None" active={amount === 0} onClick={() => onChange(0)} />
        {PRESETS.map((p) => (
          <Chip key={p} label={`₹${p}`} active={amount === p} onClick={() => onChange(p)} />
        ))}
      </div>
      <input
        type="number"
        min={0}
        inputMode="numeric"
        placeholder="Custom amount (₹)"
        value={isCustom ? String(amount) : ''}
        onChange={(e) => onChange(Number(e.target.value.replace(/[^0-9]/g, '')) || 0)}
        className="mt-3 w-full rounded-lg border border-mist-strong px-3 py-2 text-sm text-ink focus:border-herb focus:outline-none"
      />
    </div>
  );
}

function Chip({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full border px-4 py-2 text-sm font-medium transition-colors ${
        active ? 'border-herb bg-herb-tint text-herb' : 'border-mist text-ink-soft hover:border-herb'
      }`}
    >
      {label}
    </button>
  );
}
