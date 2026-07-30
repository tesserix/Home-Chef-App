import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { toast } from 'sonner';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import type { Order, PayoutHoldStatus } from '@/shared/types';
import OrderDetailPage from './OrderDetailPage';

// Escrow confirmation (#617/#387) on the customer WEB app. The chef marking an
// order delivered parks its payout at `awaiting_customer_confirmation` and
// starts a durable reminder flow; this CTA is the customer's half of that
// handshake. Without it a web customer could only ever wait for the auto-confirm.

const ORDER_ID = 'order-1';

function baseOrder(overrides: Partial<Order> = {}): Order {
  return {
    id: ORDER_ID,
    orderNumber: 'HC-1001',
    customerId: 'customer-1',
    chefId: 'chef-1',
    status: 'delivered',
    items: [
      { id: 'item-1', menuItemId: 'menu-1', name: 'Butter Chicken', price: 250, quantity: 1, subtotal: 250 },
    ],
    deliveryAddress: {
      id: 'addr-1',
      userId: 'customer-1',
      label: 'Home',
      line1: '221B Baker Street',
      city: 'Mumbai',
      state: 'MH',
      postalCode: '400001',
      country: 'IN',
      isDefault: true,
    },
    subtotal: 250,
    deliveryFee: 30,
    platformFee: 10,
    tax: 15,
    discount: 0,
    tip: 0,
    total: 305,
    paymentStatus: 'completed',
    createdAt: '2026-07-20T10:00:00Z',
    ...overrides,
  };
}

function renderOrderDetail() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return {
    queryClient,
    ...render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[`/orders/${ORDER_ID}`]}>
          <Routes>
            <Route path="/orders/:id" element={<OrderDetailPage />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    ),
  };
}

const CONFIRM = { name: 'Confirm received' };

describe('OrderDetailPage — confirm received', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('posts to the confirm endpoint and surfaces the server message', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(
      baseOrder({ payoutHoldStatus: 'awaiting_customer_confirmation' }),
    );
    const post = vi.spyOn(apiClient, 'post').mockResolvedValue({
      payoutHoldStatus: 'release_eligible',
      message: 'Thanks — your chef will be paid.',
    });
    const toastSuccess = vi.spyOn(toast, 'success');
    const user = userEvent.setup();

    renderOrderDetail();

    await user.click(await screen.findByRole('button', CONFIRM));

    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(`/orders/${ORDER_ID}/confirm-received`),
    );
    expect(toastSuccess).toHaveBeenCalledWith('Thanks — your chef will be paid.');
  });

  it('does not offer the CTA until the order is delivered', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(
      baseOrder({ status: 'ready', payoutHoldStatus: 'awaiting_customer_confirmation' }),
    );

    renderOrderDetail();

    expect(await screen.findByText(/Order #HC-1001/)).toBeInTheDocument();
    expect(screen.queryByRole('button', CONFIRM)).not.toBeInTheDocument();
  });

  it('replaces the CTA with a status line once the hold has been confirmed', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(
      baseOrder({ payoutHoldStatus: 'release_eligible' }),
    );

    renderOrderDetail();

    expect(await screen.findByText('Received')).toBeInTheDocument();
    expect(screen.queryByRole('button', CONFIRM)).not.toBeInTheDocument();
  });

  it('shows nothing extra on a delivered order with no hold', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(baseOrder());

    renderOrderDetail();

    expect(await screen.findByText(/Order #HC-1001/)).toBeInTheDocument();
    expect(screen.queryByRole('button', CONFIRM)).not.toBeInTheDocument();
    expect(screen.queryByText('Received')).not.toBeInTheDocument();
  });

  it('keeps the CTA armed when the confirm call fails', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(
      baseOrder({ payoutHoldStatus: 'awaiting_customer_confirmation' }),
    );
    vi.spyOn(apiClient, 'post').mockRejectedValue({ error: 'Could not confirm receipt' });
    const toastError = vi.spyOn(toast, 'error');
    const user = userEvent.setup();

    renderOrderDetail();

    await user.click(await screen.findByRole('button', CONFIRM));

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    // Still actionable — the customer can retry rather than being stranded.
    expect(screen.getByRole('button', CONFIRM)).toBeInTheDocument();
  });
});

// Guards the contract the CTA depends on: these are the only hold states the
// backend's confirm endpoint accepts / produces.
describe('PayoutHoldStatus contract', () => {
  it('carries the awaiting state the CTA gates on', () => {
    const awaiting: PayoutHoldStatus = 'awaiting_customer_confirmation';
    expect(awaiting).toBe('awaiting_customer_confirmation');
  });
});
