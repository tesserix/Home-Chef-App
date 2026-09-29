import { describe, it, expect, jest, beforeEach } from '@jest/globals';
jest.mock('expo-router', () => ({ router: { replace: jest.fn(), push: jest.fn() } }));
jest.mock('./api', () => ({ api: { post: jest.fn() } }));
import { router } from 'expo-router';
import { api } from './api';
import { startOrderPayment } from './payment';

beforeEach(() => { jest.clearAllMocks(); });

describe('payment session recovery', () => {
  it('keeps the created order reachable when its initial payment session fails', async () => {
    jest.mocked(api.post).mockRejectedValueOnce(new Error('gateway timeout'));
    await expect(startOrderPayment('existing-order', { useWallet: false, useLoyalty: false }, { holdSeconds: 10 })).resolves.toBeUndefined();
    expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=existing-order');
    expect(api.post).toHaveBeenCalledTimes(1);
  });
  it('surfaces retry errors on the existing order without navigating to another checkout', async () => {
    jest.mocked(api.post).mockRejectedValueOnce(new Error('gateway timeout'));
    await expect(startOrderPayment('existing-order')).rejects.toThrow('gateway timeout');
    expect(router.replace).not.toHaveBeenCalled();
  });
});
