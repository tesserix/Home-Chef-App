import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// #1046: the review screen offered a blank form for an order the customer had
// already reviewed, then surfaced the create endpoint's raw 409 as
// "Request failed with status code 409". The screen needs to KNOW about the
// existing review before it renders, which is what this hook is for.

jest.mock('@tanstack/react-query', () => ({
  useQuery: (opts: { queryKey: unknown[]; queryFn: () => unknown; enabled?: boolean }) => opts,
}));

jest.mock('../lib/api', () => ({ api: { get: jest.fn() } }));

import { api } from '../lib/api';
import { useOrderReview, type OrderReview } from './useOrderReview';

type MockFn = ReturnType<typeof jest.fn>;
const mockApi = api as unknown as { get: MockFn };
const asQuery = (hook: unknown) =>
  hook as unknown as {
    queryKey: unknown[];
    enabled?: boolean;
    queryFn: () => Promise<OrderReview | null>;
  };

beforeEach(() => {
  mockApi.get.mockClear();
});

describe('useOrderReview', () => {
  it('reads the order-scoped review endpoint', async () => {
    mockApi.get.mockResolvedValueOnce({ data: { review: null } });
    await asQuery(useOrderReview('order-1')).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith('/v1/reviews/order/order-1');
  });

  it('unwraps the review envelope', async () => {
    mockApi.get.mockResolvedValueOnce({
      data: {
        review: {
          id: 'rev-1',
          orderId: 'order-1',
          overallRating: 5,
          foodRating: 4,
          comment: 'Superb',
          dishRatings: [{ menuItemId: 'dish-1', rating: 4 }],
        },
      },
    });

    const review = await asQuery(useOrderReview('order-1')).queryFn();

    expect(review?.id).toBe('rev-1');
    expect(review?.overallRating).toBe(5);
    expect(review?.dishRatings?.[0]).toMatchObject({ menuItemId: 'dish-1', rating: 4 });
  });

  it('returns null when the order has not been reviewed', async () => {
    mockApi.get.mockResolvedValueOnce({ data: { review: null } });
    expect(await asQuery(useOrderReview('order-1')).queryFn()).toBeNull();
  });

  it('stays disabled without an order id, so it cannot fetch /order/undefined', () => {
    expect(asQuery(useOrderReview('')).enabled).toBe(false);
    expect(asQuery(useOrderReview('order-1')).enabled).toBe(true);
  });

  it('keys the cache per order', () => {
    expect(asQuery(useOrderReview('order-1')).queryKey).toEqual(['order-review', 'order-1']);
  });
});
