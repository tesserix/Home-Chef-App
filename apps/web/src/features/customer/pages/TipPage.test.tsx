import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import { openCashfreeCheckout } from '@/shared/utils/cashfree';
import TipPage from './TipPage';

// A tip is minted on Cashfree whatever gateway the order was stamped with
// (#1086). This page opened the Razorpay sheet with a Cashfree payload, so the
// whole web tip surface was dead the moment the server's tip leg moved.

vi.mock('@/shared/services/api-client', () => ({
  apiClient: { post: vi.fn(), get: vi.fn() },
}));
vi.mock('@/shared/utils/cashfree', () => ({
  openCashfreeCheckout: vi.fn(async (opts: { onSettled: () => Promise<void> }) => {
    await opts.onSettled();
  }),
}));

const post = vi.mocked(apiClient.post);

function renderTipPage() {
  return render(
    <MemoryRouter initialEntries={['/orders/order-1/tip']}>
      <Routes>
        <Route path="/orders/:id/tip" element={<TipPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('TipPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    post.mockResolvedValue({
      tipId: 'tip-1',
      provider: 'cashfree',
      cashfreeOrderId: 'tip-abc',
      cashfreePaymentSessionId: 'sess_tip',
      cashfreeEnv: 'SANDBOX',
      amount: 5000,
      currency: 'INR',
    } as never);
  });

  it('opens the Cashfree sheet for the created tip', async () => {
    renderTipPage();
    await userEvent.click(screen.getByRole('button', { name: /tip ₹50/i }));

    await waitFor(() => expect(openCashfreeCheckout).toHaveBeenCalled());
    const opts = vi.mocked(openCashfreeCheckout).mock.calls[0]![0];
    expect(opts.data.cashfreePaymentSessionId).toBe('sess_tip');
    expect(opts.data.cashfreeEnv).toBe('SANDBOX');
  });

  // Cashfree hands back no client signature, so the server's own fetch is the
  // only authority — the client must not post Razorpay-shaped fields at it.
  it('verifies against the server without client payment fields', async () => {
    renderTipPage();
    await userEvent.click(screen.getByRole('button', { name: /tip ₹50/i }));

    await waitFor(() => expect(post).toHaveBeenCalledTimes(2));
    expect(post.mock.calls[1]![0]).toBe('/payments/tip/tip-1/verify');
    expect(post.mock.calls[1]![1]).toBeUndefined();
  });
});
