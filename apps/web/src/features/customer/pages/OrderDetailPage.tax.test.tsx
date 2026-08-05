import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import type { Order } from '@/shared/types';
import OrderDetailPage from './OrderDetailPage';

// The payment summary must print the tax rows the SERVER split, and nothing it
// worked out for itself. This page used to show a single "Tax" line while the
// mobile receipt and the invoice PDF for the same order showed CGST+SGST — three
// documents, one order, three answers.
//
// These tests pin the contract rather than the arithmetic: given taxLines, render
// exactly those labels and amounts, and let the rows add up to the total.

const ORDER_ID = 'order-tax-1';

// A ₹500 order under the live Indian configuration: food 5% exclusive, platform
// fee 18% quoted all-in (so ₹24.95 shows as ₹21.14 + ₹3.81 of GST).
function optionBOrder(overrides: Partial<Order> = {}): Order {
  return {
    id: ORDER_ID,
    orderNumber: 'HC-2001',
    customerId: 'customer-1',
    chefId: 'chef-1',
    status: 'delivered',
    items: [
      { id: 'item-1', menuItemId: 'menu-1', name: 'Home-style Veg Thali', price: 500, quantity: 1, subtotal: 500 },
    ],
    deliveryAddress: {
      id: 'addr-1',
      userId: 'customer-1',
      label: 'Home',
      line1: '221B Baker Street',
      city: 'Bhubaneswar',
      state: 'OR',
      postalCode: '751001',
      country: 'IN',
      isDefault: true,
    },
    subtotal: 500,
    deliveryFee: 0,
    platformFee: 21.14,
    tax: 28.81,
    taxLines: [
      { code: 'cgst', label: 'CGST (2.5%) on food', rate: 2.5, amount: 12.5 },
      { code: 'sgst', label: 'SGST (2.5%) on food', rate: 2.5, amount: 12.5 },
      { code: 'cgst', label: 'CGST (9%) on platform fee', rate: 9, amount: 1.91 },
      { code: 'sgst', label: 'SGST (9%) on platform fee', rate: 9, amount: 1.9 },
    ],
    discount: 0,
    tip: 0,
    total: 549.95,
    paymentStatus: 'completed',
    createdAt: '2026-08-03T10:00:00Z',
    ...overrides,
  };
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orders/${ORDER_ID}`]}>
        <Routes>
          <Route path="/orders/:id" element={<OrderDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('OrderDetailPage — tax rows come from the server', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders every tax line the server split, per rate', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(optionBOrder());
    renderPage();

    // Both rates appear, each naming what it is charged on — which is what an
    // invoice must show once one bill carries two.
    expect(await screen.findByText('CGST (2.5%) on food')).toBeInTheDocument();
    expect(screen.getByText('SGST (2.5%) on food')).toBeInTheDocument();
    expect(screen.getByText('CGST (9%) on platform fee')).toBeInTheDocument();
    expect(screen.getByText('SGST (9%) on platform fee')).toBeInTheDocument();

    // And never a generic one it derived itself.
    expect(screen.queryByText('Tax')).not.toBeInTheDocument();
  });

  it('shows the platform fee net of its own GST, so the rows reach the total', async () => {
    const order = optionBOrder();
    vi.spyOn(apiClient, 'get').mockResolvedValue(order);
    renderPage();

    expect(await screen.findByText('₹21.14')).toBeInTheDocument();

    const rows =
      order.subtotal +
      order.deliveryFee +
      order.platformFee -
      order.discount +
      order.tip +
      (order.taxLines ?? []).reduce((s, l) => s + l.amount, 0);
    expect(Math.round(rows * 100) / 100).toBe(order.total);
  });

  it('prints the rounding row only when an older order needs one to add up', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(
      optionBOrder({
        platformFee: 11.98,
        subtotal: 240,
        tax: 12.6,
        taxLines: [
          { code: 'cgst', label: 'CGST (2.5%)', rate: 2.5, amount: 6.3 },
          { code: 'sgst', label: 'SGST (2.5%)', rate: 2.5, amount: 6.3 },
        ],
        rounding: -0.01,
        total: 264.57,
      }),
    );
    renderPage();

    // The formatter puts the symbol first, so a negative reads "₹-0.01".
    expect(await screen.findByText('Rounding')).toBeInTheDocument();
    expect(screen.getByText('₹-0.01')).toBeInTheDocument();
  });

  it('omits the rounding row entirely on an order that already adds up', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(optionBOrder());
    renderPage();

    await screen.findByText('CGST (2.5%) on food');
    expect(screen.queryByText('Rounding')).not.toBeInTheDocument();
  });
});
