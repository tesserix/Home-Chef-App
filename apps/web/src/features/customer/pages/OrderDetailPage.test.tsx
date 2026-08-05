import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router';
import { toast } from 'sonner';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import type { Order } from '@/shared/types';
import OrderDetailPage from './OrderDetailPage';

// A 502 from POST /orders/:id/cancel-request means the cancellation was
// created and the order WAS cancelled — only the refund leg failed
// (apps/api/handlers/cancellation.go). Before this fix the page treated that
// the same as a hard failure: the modal stayed open with an armed "Request
// Cancellation" button over a now-stale order, and clicking it again would
// retry straight into a 409. This covers the success-with-warning path: the
// modal must close and the order/cancel-request queries must be invalidated,
// exactly like the plain-success path already does.

const ORDER_ID = 'order-1';

function baseOrder(): Order {
  return {
    id: ORDER_ID,
    orderNumber: 'HC-1001',
    customerId: 'customer-1',
    chefId: 'chef-1',
    status: 'pending',
    items: [
      {
        id: 'item-1',
        menuItemId: 'menu-1',
        name: 'Butter Chicken',
        price: 250,
        quantity: 1,
        subtotal: 250,
      },
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
  };
}

function renderOrderDetail() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  const utils = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orders/${ORDER_ID}`]}>
        <Routes>
          <Route path="/orders/:id" element={<OrderDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { queryClient, ...utils };
}

describe('OrderDetailPage — cancellation 502 (cancelled, refund follow-up needed)', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('closes the modal and does not leave a retry armed on a 502', async () => {
    const get = vi.spyOn(apiClient, 'get').mockResolvedValue(baseOrder());
    vi.spyOn(apiClient, 'post').mockRejectedValue({
      error: 'Cancelled, but the refund could not be issued — support will follow up',
      status: 502,
    });
    const toastWarning = vi.spyOn(toast, 'warning');
    const toastError = vi.spyOn(toast, 'error');
    const user = userEvent.setup();

    renderOrderDetail();

    await user.click(await screen.findByRole('button', { name: 'Request Cancellation' }));
    expect(screen.getByText('Cancel Order')).toBeInTheDocument();

    const getCallsBeforeSubmit = get.mock.calls.length;

    // The modal's submit button shares its accessible name with the trigger
    // button, so scope to the modal heading's container.
    const submitButton = screen.getAllByRole('button', { name: 'Request Cancellation' })[1];
    expect(submitButton).toBeDefined();
    await user.click(submitButton!);

    // Success-with-warning: modal closes, no armed retry button left behind.
    await waitFor(() => expect(screen.queryByText('Cancel Order')).not.toBeInTheDocument());
    expect(screen.queryByText('Keep Order')).not.toBeInTheDocument();

    // The server's message is surfaced as a warning, not a flat failure.
    expect(toastWarning).toHaveBeenCalledWith(
      'Cancelled, but the refund could not be issued — support will follow up',
    );
    expect(toastError).not.toHaveBeenCalled();

    // The order query was invalidated — with an active observer mounted,
    // that triggers a refetch so the page re-reads as cancelled.
    await waitFor(() => expect(get.mock.calls.length).toBeGreaterThan(getCallsBeforeSubmit));
  });
});
