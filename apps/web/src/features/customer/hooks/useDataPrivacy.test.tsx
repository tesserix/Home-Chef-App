import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import {
  useDeleteAccount,
  useDeletionEligibility,
} from '@/features/customer/hooks/useDataPrivacy';

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe('useDeletionEligibility', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('reads eligibility without the /v1 prefix', async () => {
    const get = vi.spyOn(apiClient, 'get').mockResolvedValue({
      deletable: false,
      blockers: [{ code: 'wallet_balance', label: 'Wallet credit left', amount: 250 }],
      retentionDays: 180,
    });

    const { result } = renderHook(() => useDeletionEligibility(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(get).toHaveBeenCalledWith('/customer/me/deletion-eligibility');
    expect(result.current.data?.blockers[0]?.code).toBe('wallet_balance');
  });
});

describe('useDeleteAccount', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('sends the typed confirmation email as the body', async () => {
    const post = vi.spyOn(apiClient, 'post').mockResolvedValue({
      status: 'deleted',
      deletedAt: '2026-07-25T00:00:00Z',
      purgeAfter: '2027-01-21T00:00:00Z',
    });

    const { result } = renderHook(() => useDeleteAccount(), { wrapper });
    result.current.mutate('customer@demo.com');
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(post).toHaveBeenCalledWith('/customer/me/delete', {
      confirmEmail: 'customer@demo.com',
    });
  });
});
