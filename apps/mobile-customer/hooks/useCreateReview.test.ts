import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// #1046: after submitting, the screen navigates back to an order detail that
// still believes the order is un-reviewed unless the order-review cache is
// dropped — so the CTA kept saying "Leave a Review" and re-opened a form that
// could only 409.

const mockInvalidateQueries = jest.fn();
jest.mock('@tanstack/react-query', () => ({
  useMutation: (opts: unknown) => opts,
  useQueryClient: () => ({ invalidateQueries: mockInvalidateQueries }),
}));
jest.mock('../lib/api', () => ({ api: { post: jest.fn() } }));

import { api } from '../lib/api';
import { useCreateReview } from './useCreateReview';

type MockFn = ReturnType<typeof jest.fn>;
const mockApi = api as unknown as { post: MockFn };
const asMutation = (hook: unknown) =>
  hook as unknown as {
    mutationFn: (v: { orderId: string; overallRating: number }) => Promise<void>;
    onSuccess: (data: unknown, vars: { orderId: string }) => void;
  };

beforeEach(() => {
  mockApi.post.mockClear();
  mockInvalidateQueries.mockClear();
});

describe('useCreateReview', () => {
  it('invalidates the reviewed order so the read state replaces the form', () => {
    asMutation(useCreateReview()).onSuccess(undefined, { orderId: 'order-1' });

    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['order-review', 'order-1'] });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['orders'] });
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ['chefs'] });
  });

  it('posts the review as multipart form data', async () => {
    mockApi.post.mockResolvedValueOnce({ data: {} });

    await asMutation(useCreateReview()).mutationFn({ orderId: 'order-1', overallRating: 5 });

    expect(mockApi.post).toHaveBeenCalledWith(
      '/v1/reviews',
      expect.anything(),
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
  });
});
