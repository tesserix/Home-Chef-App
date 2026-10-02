import { describe, it, expect, jest, beforeEach } from '@jest/globals';
jest.mock('expo-router', () => ({
  router: { replace: jest.fn(), push: jest.fn() },
}));
jest.mock('react-native', () => ({
  Linking: { addEventListener: jest.fn(() => ({ remove: jest.fn() })) },
}));
jest.mock('./api', () => ({ api: { post: jest.fn() } }));
jest.mock(
  '@stripe/stripe-react-native',
  () => ({
    initStripe: jest.fn(),
    initPaymentSheet: jest.fn(),
    presentPaymentSheet: jest.fn(),
  }),
  { virtual: true },
);
import { initStripe, initPaymentSheet, presentPaymentSheet } from '@stripe/stripe-react-native';
import { router } from 'expo-router';
import { api } from './api';
import { launchGateway, startOrderPayment } from './payment';

beforeEach(() => {
  jest.clearAllMocks();
});

describe('payment session recovery', () => {
  it('shows a known preflight rejection without losing the existing order', async () => {
    jest.mocked(api.post).mockRejectedValueOnce({
      response: {
        status: 503,
        data: {
          error: 'Stripe publishable key is not configured for this environment',
        },
      },
    });
    await startOrderPayment('existing-order', undefined, { holdSeconds: 10 });
    expect(router.replace).toHaveBeenCalledWith(
      '/payment/result?order_id=existing-order&not_started=1',
    );
    expect(presentPaymentSheet).not.toHaveBeenCalled();
  });
  it('does not treat an unknown service outage as proof payment never started', async () => {
    jest.mocked(api.post).mockRejectedValueOnce({
      response: { status: 503, data: { error: 'upstream unavailable' } },
    });
    await startOrderPayment('existing-order', undefined, { holdSeconds: 10 });
    expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=existing-order');
  });

  it('keeps the created order reachable when its initial payment session fails', async () => {
    jest.mocked(api.post).mockRejectedValueOnce(new Error('gateway timeout'));
    await expect(
      startOrderPayment(
        'existing-order',
        { useWallet: false, useLoyalty: false },
        { holdSeconds: 10 },
      ),
    ).resolves.toBeUndefined();
    expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=existing-order');
    expect(api.post).toHaveBeenCalledTimes(1);
  });
  it('surfaces retry errors on the existing order without navigating to another checkout', async () => {
    jest.mocked(api.post).mockRejectedValueOnce(new Error('gateway timeout'));
    await expect(startOrderPayment('existing-order')).rejects.toThrow('gateway timeout');
    expect(router.replace).not.toHaveBeenCalled();
  });
});

describe('Stripe checkout', () => {
  const payment = {
    provider: 'stripe',
    amount: 5000,
    currency: 'AUD',
    stripePaymentIntentId: 'pi_fixture',
    clientSecret: 'pi_fixture_secret_test',
    publishableKey: 'pk_test_fixture',
  };

  beforeEach(() => {
    jest.mocked(initStripe).mockResolvedValue(undefined);
    jest.mocked(initPaymentSheet).mockResolvedValue({});
    jest.mocked(presentPaymentSheet).mockResolvedValue({});
    jest.mocked(api.post).mockResolvedValue({ data: { status: 'completed' } });
  });

  it('opens PaymentSheet then verifies through the existing order API', async () => {
    await launchGateway('order-au', payment);
    expect(initStripe).toHaveBeenCalledWith(
      expect.objectContaining({ publishableKey: 'pk_test_fixture' }),
    );
    expect(initPaymentSheet).toHaveBeenCalledWith(
      expect.objectContaining({
        paymentIntentClientSecret: payment.clientSecret,
        merchantDisplayName: 'Fe3dr',
      }),
    );
    expect(presentPaymentSheet).toHaveBeenCalledTimes(1);
    expect(api.post).toHaveBeenCalledWith('/v1/payments/order/order-au/verify', {
      stripePaymentIntentId: 'pi_fixture',
    });
    expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=order-au');
    expect(JSON.stringify(jest.mocked(router.replace).mock.calls)).not.toContain('secret');
  });

  it('does not verify when the customer cancels', async () => {
    jest.mocked(presentPaymentSheet).mockResolvedValue({
      error: { code: 'Canceled', message: 'Canceled' },
    } as never);
    await launchGateway('order-au', payment);
    expect(api.post).not.toHaveBeenCalled();
    expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=order-au');
  });

  it('keeps a captured order reachable if verification fails', async () => {
    jest.mocked(api.post).mockRejectedValueOnce(new Error('network offline'));
    await launchGateway('order-au', payment);
    expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=order-au');
    expect(api.post).toHaveBeenCalledTimes(1);
  });
});
