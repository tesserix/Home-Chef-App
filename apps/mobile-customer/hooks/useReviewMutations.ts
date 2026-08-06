import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Author-only edit and withdraw for a review the customer wrote (#1047).
// Mirrors PATCH/DELETE /v1/reviews/:id.

export interface UpdateReviewInput {
  reviewId: string;
  /** Kept out of the payload — it keys the order-review cache to drop. */
  orderId?: string;
  overallRating?: number;
  foodRating?: number;
  deliveryRating?: number;
  valueRating?: number;
  packagingRating?: number;
  hygieneRating?: number;
  title?: string;
  comment?: string;
}

export interface DeleteReviewInput {
  reviewId: string;
  orderId?: string;
}

// Every surface that renders a review: the order's own copy, the chef's public
// list, and the chef card that carries the aggregate rating.
function invalidateReviewCaches(
  queryClient: ReturnType<typeof useQueryClient>,
  orderId: string | undefined,
  extraKeys: string[][] = [],
) {
  if (orderId) {
    void queryClient.invalidateQueries({ queryKey: ['order-review', orderId] });
  }
  void queryClient.invalidateQueries({ queryKey: ['chef-reviews'] });
  void queryClient.invalidateQueries({ queryKey: ['chefs'] });
  for (const key of extraKeys) {
    void queryClient.invalidateQueries({ queryKey: key });
  }
}

export function useUpdateReview() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ reviewId, orderId: _orderId, ...fields }: UpdateReviewInput) => {
      const res = await api.patch(`/v1/reviews/${reviewId}`, fields);
      return res.data;
    },
    onSuccess: (_data, input: UpdateReviewInput) => {
      invalidateReviewCaches(queryClient, input.orderId);
    },
  });
}

export function useDeleteReview() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ reviewId }: DeleteReviewInput) => {
      const res = await api.delete(`/v1/reviews/${reviewId}`);
      return res.data;
    },
    onSuccess: (_data, input: DeleteReviewInput) => {
      // The order becomes reviewable again, so its history entry has to refresh.
      invalidateReviewCaches(queryClient, input.orderId, [['orders']]);
    },
  });
}
