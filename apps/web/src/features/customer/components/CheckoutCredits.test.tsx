import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { CreditQuote } from '../hooks/useDeliveryQuote';
import { CheckoutCredits } from './CheckoutCredits';

// The web checkout could not spend wallet credit or loyalty points at all —
// a customer holding a refund balance was charged the full total. These pin the
// two properties that make the ported card safe to put in front of money:
// it renders what the server allocated (never its own arithmetic), and touching
// one rail pins the other instead of letting it silently expand.

function quote(over: Partial<CreditQuote> = {}): CreditQuote {
  return {
    redeemableCap: 270,
    nonRedeemable: 27.68,
    walletBalance: 500,
    walletApplied: 100,
    walletMax: 270,
    pointsBalance: 2000,
    pointsApplied: 400,
    pointsValue: 20,
    pointsMax: 3400,
    pointsBalanceValue: 100,
    pointsMaxValue: 170,
    payable: 177.68,
    loyaltyLimit: 'balance',
    walletEnabled: true,
    loyaltyEnabled: true,
    ...over,
  };
}

describe('CheckoutCredits', () => {
  it('renders the server allocation rather than recomputing it', () => {
    render(
      <CheckoutCredits
        quote={quote()}
        useWallet
        useLoyalty
        currency="INR"
        onChange={vi.fn()}
      />
    );

    // Wallet shows walletApplied, loyalty shows the server's RUPEE value of the
    // points — not points × some rate the client guessed at.
    expect(screen.getByText('−₹100.00')).toBeInTheDocument();
    expect(screen.getByText('−₹20.00')).toBeInTheDocument();
    expect(screen.getByText('−₹120.00')).toBeInTheDocument(); // credits applied
    expect(screen.getByText(/₹27.68/)).toBeInTheDocument(); // fees & taxes
  });

  it('pins the other rail when one is toggled off', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(
      <CheckoutCredits
        quote={quote()}
        useWallet
        useLoyalty
        currency="INR"
        onChange={onChange}
      />
    );

    await user.click(screen.getByRole('checkbox', { name: /wallet credit/i }));

    // Wallet goes off with its amount released to "auto"; loyalty is pinned to
    // exactly what the server had allocated, so it can't expand into the gap and
    // spend points the customer was preserving.
    expect(onChange).toHaveBeenCalledWith({
      useWallet: false,
      walletAmount: undefined,
      useLoyalty: true,
      loyaltyPoints: 400,
    });
  });

  it('renders nothing when neither rail has anything to spend', () => {
    const { container } = render(
      <CheckoutCredits
        quote={quote({ walletMax: 0, pointsMax: 0 })}
        useWallet
        useLoyalty
        currency="INR"
        onChange={vi.fn()}
      />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('hides a rail the SERVER has disabled, even with a balance on it', () => {
    render(
      <CheckoutCredits
        quote={quote({ loyaltyEnabled: false })}
        useWallet
        useLoyalty
        currency="INR"
        onChange={vi.fn()}
      />
    );
    expect(screen.getByRole('checkbox', { name: /wallet credit/i })).toBeInTheDocument();
    expect(screen.queryByRole('checkbox', { name: /loyalty points/i })).not.toBeInTheDocument();
  });
});
