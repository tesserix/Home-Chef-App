import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Mirrors POST /v1/reviews (multipart form). Backend: handlers/reviews.go
// CreateReview — overallRating is required (1–5); the sub-scores and the
// dishRatings JSON array are optional (#35 Ratings 2.0 + #145 per-dish).
export interface DishRatingInput {
  menuItemId: string;
  rating: number;
}

export interface CreateReviewInput {
  orderId: string;
  overallRating: number;
  foodRating?: number;
  deliveryRating?: number;
  valueRating?: number;
  packagingRating?: number;
  hygieneRating?: number;
  title?: string;
  comment?: string;
  dishRatings?: DishRatingInput[];
}

export function useCreateReview() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, CreateReviewInput>({
    mutationFn: async (input) => {
      const fd = new FormData();
      fd.append('orderId', input.orderId);
      fd.append('overallRating', String(input.overallRating));
      if (input.foodRating) fd.append('foodRating', String(input.foodRating));
      if (input.deliveryRating) fd.append('deliveryRating', String(input.deliveryRating));
      if (input.valueRating) fd.append('valueRating', String(input.valueRating));
      if (input.packagingRating) fd.append('packagingRating', String(input.packagingRating));
      if (input.hygieneRating) fd.append('hygieneRating', String(input.hygieneRating));
      if (input.title?.trim()) fd.append('title', input.title.trim());
      if (input.comment?.trim()) fd.append('comment', input.comment.trim());
      const dishes = (input.dishRatings ?? []).filter((d) => d.rating >= 1 && d.rating <= 5);
      if (dishes.length) fd.append('dishRatings', JSON.stringify(dishes));
      await api.post('/v1/reviews', fd, { headers: { 'Content-Type': 'multipart/form-data' } });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['orders'] });
      void queryClient.invalidateQueries({ queryKey: ['chefs'] });
    },
  });
}
