import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// #1047: a customer's own review offered only "report" and "block yourself".
// Correcting or withdrawing it needs the author-only PATCH/DELETE routes, and
// every cache that renders a review has to drop when they succeed.

const mockInvalidateQueries = jest.fn();
jest.mock('@tanstack/react-query', () => ({
  useMutation: (opts: unknown) => opts,
  useQueryClient: () => ({ invalidateQueries: mockInvalidateQueries }),
}));
jest.mock('../lib/api', () => ({ api: { patch: jest.fn(), delete: jest.fn() } }));

import { api } from '../lib/api';
import { useUpdateReview, useDeleteReview } from './useReviewMutations';

type MockFn = ReturnType<typeof jest.fn>;
const mockApi = api as unknown as { patch: MockFn; delete: MockFn };
const asMutation = (hook: unknown) =>
  hook as unknown as {
    mutationFn: (v: Record<string, unknown>) => Promise<unknown>;
    onSuccess: (data: unknown, vars: Record<string, unknown>) => void;
  };

beforeEach(() => {
  mockApi.patch.mockClear();
  mockApi.delete.mockClear();
  mockInvalidateQueries.mockClear();
});

describe('useUpdateReview', () => {
  it('patches only the fields the customer changed', async () => {
    mockApi.patch.mockResolvedValueOnce({ data: {} });

    await asMutation(useUpdateReview()).mutationFn({
      reviewId: 'review-1',
      orderId: 'order-1',
      overallRating: 3,
      title: 'Good, not great',
    });

    expect(mockApi.patch).toHaveBeenCalledWith('/v1/reviews/review-1', {
      overallRating: 3,
      title: 'Good, not great',
    });
  });

  it('refreshes the order review and the chef review list', () => {
    asMutation(useUpdateReview()).onSuccess(undefined, { reviewId: 'r1', orderId: 'order-1' });

    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['order-review', 'order-1'] });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['chef-reviews'] });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['chefs'] });
  });
});

describe('useDeleteReview', () => {
  it('deletes the review by id', async () => {
    mockApi.delete.mockResolvedValueOnce({ data: { success: true } });

    await asMutation(useDeleteReview()).mutationFn({ reviewId: 'review-1', orderId: 'order-1' });

    expect(mockApi.delete).toHaveBeenCalledWith('/v1/reviews/review-1');
  });

  // A withdrawn review must vanish from the chef's list and free the order to be
  // reviewed again, so the order-review cache has to drop too.
  it('refreshes every surface that renders the review', () => {
    asMutation(useDeleteReview()).onSuccess(undefined, { reviewId: 'r1', orderId: 'order-1' });

    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['order-review', 'order-1'] });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['chef-reviews'] });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['orders'] });
  });

  it('still refreshes the chef list when the order id is unknown', () => {
    asMutation(useDeleteReview()).onSuccess(undefined, { reviewId: 'r1' });

    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['chef-reviews'] });
    expect(mockInvalidateQueries).not.toHaveBeenCalledWith({ queryKey: ['order-review', undefined] });
  });
});
