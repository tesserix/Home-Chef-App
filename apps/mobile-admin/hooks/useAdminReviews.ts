import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Paginated, ReviewRow } from '../lib/admin-types';

export interface ReviewsQuery {
  hidden?: boolean;
  page?: number;
  limit?: number;
}

export function useAdminReviews(q: ReviewsQuery = {}) {
  const { hidden, page = 1, limit = 20 } = q;
  return useQuery<Paginated<ReviewRow>>({
    queryKey: ['admin', 'reviews', { hidden: hidden ?? false, page, limit }],
    queryFn: () =>
      api
        .get<Paginated<ReviewRow>>('/admin/reviews', {
          params: { hidden: hidden ? 'true' : '', page, limit },
        })
        .then((r) => r.data),
  });
}

export function useHideReview() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.put(`/admin/reviews/${id}/hide`, { reason }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'reviews'] }),
  });
}

export function useUnhideReview() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.put(`/admin/reviews/${id}/unhide`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'reviews'] }),
  });
}
