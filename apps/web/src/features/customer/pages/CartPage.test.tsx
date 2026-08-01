import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useCartStore } from '@/app/store/cart-store';
import CartPage from './CartPage';

// The cart page must render with items in it. It regressed to a blank
// error-boundary screen in production, and nothing caught it: typecheck passed
// and no test ever mounted this page with a populated cart.

vi.mock('@/app/providers/AuthProvider', () => ({
  useAuth: () => ({ isAuthenticated: true, user: { id: 'u1' } }),
}));

// The page reads the live platform-fee percent; the real client drags in
// Firebase auth, which rejects after teardown and fails the run.
vi.mock('@/shared/services/api-client', () => ({
  apiClient: {
    get: vi.fn().mockResolvedValue({ platformFeePercent: 4.99 }),
    post: vi.fn(),
  },
}));

function renderCart() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <CartPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('CartPage', () => {
  beforeEach(() => {
    useCartStore.getState().clearCart();
  });

  it('renders the empty state', () => {
    renderCart();
    expect(screen.getByText(/your cart is empty/i)).toBeInTheDocument();
  });

  it('renders a populated cart without crashing', () => {
    const chef = { id: 'chef-1', businessName: 'Saffron Home Kitchen' };
    useCartStore.getState().setChef(chef as never);
    useCartStore.getState().addItem(
      {
        id: 'item-1',
        chefId: 'chef-1',
        name: 'Mutton Rogan Josh',
        price: 420,
        description: '',
        imageUrl: '',
      } as never,
      2,
    );

    renderCart();

    expect(screen.getByText(/your cart/i)).toBeInTheDocument();
    expect(screen.getByText(/mutton rogan josh/i)).toBeInTheDocument();
    // The affordance that regressed.
    expect(screen.getByRole('button', { name: /clear cart/i })).toBeInTheDocument();
  });
});
