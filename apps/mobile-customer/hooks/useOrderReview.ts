import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Mirrors GET /v1/reviews/order/:orderId (handlers/reviews.go GetOrderReview) —
// the calling customer's own review of an order, or null if they have not
// reviewed it. `review: null` is a 200, not an error (#1046).
export interface OrderReviewDishRating {
  menuItemId: string;
  rating: number;
}

export interface OrderReview {
  id: string;
  orderId: string;
  overallRating: number;
  foodRating?: number;
  deliveryRating?: number;
  valueRating?: number;
  packagingRating?: number;
  hygieneRating?: number;
  title?: string;
  comment?: string;
  chefResponse?: string;
  createdAt?: string;
  dishRatings?: OrderReviewDishRating[];
}

export function useOrderReview(orderId: string) {
  return useQuery<OrderReview | null>({
    queryKey: ['order-review', orderId],
    enabled: Boolean(orderId),
    queryFn: async () => {
      const res = await api.get<{ review: OrderReview | null }>(
        `/v1/reviews/order/${orderId}`,
      );
      return res.data.review ?? null;
    },
  });
}
