import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import {
  orderCancellable,
  useCancellationRequest,
  useRequestCancellation,
} from '@/features/customer/hooks/useCancellation';

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe('useRequestCancellation', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('posts to the cancel-request endpoint, never the legacy direct cancel', async () => {
    const post = vi.spyOn(apiClient, 'post').mockResolvedValue({});

    const { result } = renderHook(() => useRequestCancellation(), { wrapper });
    result.current.mutate({ orderId: 'order-1', reason: 'changed my mind' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(post).toHaveBeenCalledWith('/orders/order-1/cancel-request', {
      reason: 'changed my mind',
    });
    // The legacy endpoint bypasses the refund engine — it must never be hit.
    expect(post).not.toHaveBeenCalledWith(
      '/orders/order-1/cancel',
      expect.anything(),
    );
  });
});

describe('useCancellationRequest', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('treats a 404 as "no request filed yet" and resolves to null', async () => {
    vi.spyOn(apiClient, 'get').mockRejectedValue({
      success: false,
      error: { code: 'NOT_FOUND', message: 'not found' },
      status: 404,
    });

    const { result } = renderHook(() => useCancellationRequest('order-1'), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBeNull();
  });

  it('does not swallow a non-404 failure into "no request" — it must propagate', async () => {
    vi.spyOn(apiClient, 'get').mockRejectedValue({
      success: false,
      error: { code: 'INTERNAL_ERROR', message: 'server error' },
      status: 500,
    });

    const { result } = renderHook(() => useCancellationRequest('order-1'), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.data).toBeUndefined();
  });
});

describe('orderCancellable', () => {
  it('allows the stages the API still accepts a request for', () => {
    expect(orderCancellable('pending')).toBe(true);
    expect(orderCancellable('accepted')).toBe(true);
    expect(orderCancellable('preparing')).toBe(true);
  });

  it('rejects terminal and in-transit stages', () => {
    expect(orderCancellable('delivered')).toBe(false);
    expect(orderCancellable('cancelled')).toBe(false);
    expect(orderCancellable('out_for_delivery')).toBe(false);
  });
});
