import { useId, useState } from 'react';
import { Wallet, Award } from 'lucide-react';
import { useFormatPrice } from '@/shared/utils/format-price';
import type { CreditIntent, CreditQuote, LoyaltyLimit } from '../hooks/useDeliveryQuote';

// CheckoutCredits — "Pay with your credits" on web.
//
// The web checkout had no way to spend either balance: a customer could hold
// wallet credit from a refund and loyalty points from past orders, see both on
// /wallet and /loyalty, and still be charged the full total here. This is the
// port of the mobile CreditsCard, driven by the same server allocation.
//
// Every figure comes from the server quote. The component renders; it does not
// calculate. That is deliberate — the mobile app previously computed a payable
// from a cached balance and posted it, so any drift showed one number and
// charged another.

export interface CheckoutCreditsProps {
  quote: CreditQuote;
  useWallet: boolean;
  useLoyalty: boolean;
  currency: string;
  onChange: (next: CreditIntent) => void;
}

/** The loyalty caption explains WHY the row is capped, using the server's reason
 *  rather than re-deriving the cap and risking a different answer. */
function loyaltyCaption(
  quote: CreditQuote,
  limit: LoyaltyLimit,
  money: (n: number) => string
): string {
  switch (limit) {
    case 'per_order_cap':
      return `worth ${money(quote.pointsBalanceValue)} · up to ${money(quote.pointsMaxValue)} here`;
    case 'monthly_cap':
      return `${money(quote.pointsMaxValue)} of your monthly limit left`;
    case 'order_covered':
      return 'your wallet already covers this order';
    default:
      return `worth ${money(quote.pointsMaxValue)}`;
  }
}

interface RailProps {
  icon: React.ReactNode;
  title: string;
  caption: string;
  on: boolean;
  applied: number;
  /** Slider position + its ceiling, in the rail's own unit (rupees or points). */
  value: number;
  max: number;
  /** Unit label for the numeric field's accessible name. */
  unit: string;
  money: (n: number) => string;
  onToggle: () => void;
  onValue: (v: number) => void;
}

function CreditRail({
  icon,
  title,
  caption,
  on,
  applied,
  value,
  max,
  unit,
  money,
  onToggle,
  onValue,
}: RailProps) {
  const id = useId();
  // Held while typing so a half-typed "1" doesn't immediately clamp to the max
  // and fight the customer's keystrokes. Committed on blur.
  const [draft, setDraft] = useState<string | null>(null);

  const commit = (raw: string) => {
    setDraft(null);
    const n = Number(raw);
    if (!Number.isNaN(n)) onValue(Math.max(0, Math.min(max, Math.round(n))));
  };

  return (
    <div className="border-t border-mist py-4 first:border-t-0 first:pt-0">
      <div className="flex items-start gap-3">
        <input
          id={id}
          type="checkbox"
          checked={on}
          onChange={onToggle}
          className="mt-1 h-4 w-4 flex-shrink-0 text-herb focus-visible:ring-herb"
        />
        <label htmlFor={id} className="flex flex-1 cursor-pointer items-start justify-between gap-3">
          <span className="flex items-start gap-2">
            <span className="mt-0.5 text-herb" aria-hidden="true">
              {icon}
            </span>
            <span>
              <span className="block text-sm font-medium text-ink">{title}</span>
              <span className="block text-xs tabular-nums text-ink-muted">{caption}</span>
            </span>
          </span>
          <span
            className={`shrink-0 text-sm font-semibold tabular-nums ${
              on && applied > 0 ? 'text-herb' : 'text-ink-muted'
            }`}
          >
            {on && applied > 0 ? `−${money(applied)}` : money(0)}
          </span>
        </label>
      </div>

      {on && max > 0 && (
        <div className="mt-3 flex items-center gap-3 pl-7">
          <input
            type="range"
            min={0}
            max={max}
            step={1}
            value={Math.min(value, max)}
            onChange={(e) => onValue(Number(e.target.value))}
            aria-label={`${title} to apply`}
            className="h-1 flex-1 cursor-pointer accent-herb"
          />
          <input
            type="text"
            inputMode="numeric"
            value={draft ?? String(Math.round(value))}
            onChange={(e) => setDraft(e.target.value.replace(/[^0-9]/g, ''))}
            onBlur={(e) => commit(e.target.value)}
            aria-label={`${title} amount in ${unit}`}
            className="w-24 rounded-lg border border-mist bg-paper px-3 py-2 text-right text-sm tabular-nums text-ink"
          />
        </div>
      )}
    </div>
  );
}

export function CheckoutCredits({
  quote,
  useWallet,
  useLoyalty,
  currency,
  onChange,
}: CheckoutCreditsProps) {
  const fp = useFormatPrice();
  const money = (n: number) => fp(n, { currency });

  const walletUsable = quote.walletEnabled && quote.walletMax > 0;
  const loyaltyUsable = quote.loyaltyEnabled && quote.pointsMax > 0;
  if (!walletUsable && !loyaltyUsable) return null;

  const walletShown = useWallet ? quote.walletApplied : 0;
  const pointsShown = useLoyalty ? quote.pointsApplied : 0;
  const creditApplied = walletShown + (useLoyalty ? quote.pointsValue : 0);

  // Touching EITHER control pins BOTH rails to explicit values. Without this,
  // dragging the wallet down would let loyalty silently expand into the gap and
  // spend points the customer was deliberately preserving.
  const pin = (over: Partial<CreditIntent>) =>
    onChange({
      useWallet,
      useLoyalty,
      walletAmount: quote.walletApplied,
      loyaltyPoints: quote.pointsApplied,
      ...over,
    });

  return (
    <section className="rounded-xl bg-bone p-6 shadow-1">
      <h2 className="text-lg font-semibold text-ink">Pay with your credits</h2>

      <div className="mt-4">
        {walletUsable && (
          <CreditRail
            icon={<Wallet className="h-4 w-4" />}
            title="Wallet credit"
            caption={`Balance ${money(quote.walletBalance)}`}
            on={useWallet}
            applied={walletShown}
            value={walletShown}
            max={quote.walletMax}
            unit="rupees"
            money={money}
            onToggle={() => pin({ useWallet: !useWallet, walletAmount: undefined })}
            onValue={(v) => pin({ walletAmount: v })}
          />
        )}

        {loyaltyUsable && (
          <CreditRail
            icon={<Award className="h-4 w-4" />}
            title="Loyalty points"
            caption={`${Math.round(quote.pointsBalance).toLocaleString('en-IN')} pts · ${loyaltyCaption(
              quote,
              quote.loyaltyLimit,
              money
            )}`}
            on={useLoyalty}
            applied={useLoyalty ? quote.pointsValue : 0}
            value={pointsShown}
            max={quote.pointsMax}
            unit="points"
            money={money}
            onToggle={() => pin({ useLoyalty: !useLoyalty, loyaltyPoints: undefined })}
            onValue={(v) => pin({ loyaltyPoints: v })}
          />
        )}
      </div>

      <div className="mt-4 border-t border-mist pt-4">
        <div className="flex items-center justify-between">
          <span className="text-sm text-ink">Credits applied</span>
          <span className="text-sm font-semibold tabular-nums text-herb">
            −{money(creditApplied)}
          </span>
        </div>
        <p className="mt-1 text-xs text-ink-muted">
          Fees &amp; taxes are always paid separately ({money(quote.nonRedeemable)}).
        </p>
      </div>
    </section>
  );
}
