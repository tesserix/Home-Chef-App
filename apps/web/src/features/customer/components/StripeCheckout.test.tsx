import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { StripeCheckout } from './StripeCheckout';

afterEach(() => { delete window.Stripe; });

function gateway(error?: string) {
  const element = { mount: vi.fn(), destroy: vi.fn(), on: vi.fn((_event, cb) => cb()) };
  const elements = { create: vi.fn(() => element) };
  const confirmPayment = vi.fn().mockResolvedValue(error ? { error: { message: error } } : {});
  window.Stripe = vi.fn(() => ({ elements: () => elements, confirmPayment, retrievePaymentIntent: vi.fn() }));
  return { element, elements, confirmPayment };
}

const payment = { clientSecret: 'pi_fixture_secret_fixture', publishableKey: 'pk_test_fixture', stripePaymentIntentId: 'pi_fixture' };

describe('StripeCheckout', () => {
  it('collects payment details in Elements before confirmation and destroys it on exit', async () => {
    const sdk = gateway();
    const onConfirmed = vi.fn();
    const { unmount } = render(<StripeCheckout orderId="order-1" payment={payment} onConfirmed={onConfirmed} onCancel={vi.fn()} />);
    await waitFor(() => expect(sdk.element.mount).toHaveBeenCalled());
    await userEvent.click(screen.getByRole('button', { name: 'Pay securely' }));
    expect(sdk.confirmPayment).toHaveBeenCalledWith({ elements: sdk.elements, confirmParams: { return_url: `${window.location.origin}/orders/order-1?stripe_pi=pi_fixture` }, redirect: 'if_required' });
    expect(onConfirmed).toHaveBeenCalledOnce();
    unmount();
    expect(sdk.element.destroy).toHaveBeenCalledOnce();
  });

  it('keeps the form open on a decline and never reports confirmation', async () => {
    gateway('Your card was declined.');
    const onConfirmed = vi.fn();
    render(<StripeCheckout orderId="order-1" payment={payment} onConfirmed={onConfirmed} onCancel={vi.fn()} />);
    await waitFor(() => expect(screen.getByRole('button', { name: 'Pay securely' })).toBeEnabled());
    await userEvent.click(screen.getByRole('button', { name: 'Pay securely' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Your card was declined.');
    expect(onConfirmed).not.toHaveBeenCalled();
  });

  it('can leave an unpaid order without attempting confirmation', async () => {
    const sdk = gateway();
    const onCancel = vi.fn();
    render(<StripeCheckout orderId="order-1" payment={payment} onConfirmed={vi.fn()} onCancel={onCancel} />);
    await userEvent.click(screen.getByRole('button', { name: 'Return to order' }));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(sdk.confirmPayment).not.toHaveBeenCalled();
  });
});
