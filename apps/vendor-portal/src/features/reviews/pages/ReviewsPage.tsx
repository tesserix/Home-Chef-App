import { useState } from 'react';
import { motion } from 'framer-motion';
import { Star, MessageSquare, Filter } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import type { Review } from '@/shared/types';
import { format } from 'date-fns';

export default function ReviewsPage() {
  const [filterStars, setFilterStars] = useState<number | null>(null);
  const [replyingTo, setReplyingTo] = useState<string | null>(null);
  const [replyText, setReplyText] = useState('');
  const queryClient = useQueryClient();

  const { data: reviews = [], isLoading } = useQuery({
    queryKey: ['chef-reviews'],
    queryFn: () => apiClient.get<Review[]>('/chef/reviews'),
  });

  const replyMutation = useMutation({
    mutationFn: ({ reviewId, response }: { reviewId: string; response: string }) =>
      apiClient.post(`/chef/reviews/${reviewId}/reply`, { response }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-reviews'] });
      setReplyingTo(null);
      setReplyText('');
      toast.success('Reply posted successfully');
    },
  });

  const filtered = filterStars
    ? reviews.filter((r) => r.overallRating === filterStars)
    : reviews;

  const avgRating =
    reviews.length > 0
      ? reviews.reduce((sum, r) => sum + r.overallRating, 0) / reviews.length
      : 0;

  const ratingDistribution = [5, 4, 3, 2, 1].map((stars) => ({
    stars,
    count: reviews.filter((r) => r.overallRating === stars).length,
    percentage:
      reviews.length > 0
        ? (reviews.filter((r) => r.overallRating === stars).length / reviews.length) * 100
        : 0,
  }));

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-500 border-t-transparent" />
      </div>
    );
  }

  return (
    <motion.div variants={staggerContainer} initial="hidden" animate="visible" className="space-y-6">
      {/* Header */}
      <motion.div variants={fadeInUp} className="page-header">
        <h1 className="page-title">Reviews</h1>
        <p className="page-description">See what your customers are saying</p>
      </motion.div>

      {/* Rating Summary */}
      <motion.div variants={fadeInUp} className="grid gap-6 lg:grid-cols-3">
        {/* Overall Rating */}
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <div className="flex items-center gap-4">
            <div className="text-4xl font-bold text-gray-900">{avgRating.toFixed(1)}</div>
            <div>
              <div className="flex items-center gap-0.5">
                {[1, 2, 3, 4, 5].map((s) => (
                  <Star
                    key={s}
                    className={`h-5 w-5 ${s <= Math.round(avgRating) ? 'fill-yellow-400 text-yellow-400' : 'text-gray-300'}`}
                  />
                ))}
              </div>
              <p className="mt-1 text-sm text-gray-500">{reviews.length} reviews</p>
            </div>
          </div>
        </div>

        {/* Rating Distribution */}
        <div className="col-span-2 rounded-xl border border-gray-200 bg-white p-6">
          <h3 className="mb-4 text-sm font-medium text-gray-700">Rating Distribution</h3>
          <div className="space-y-2">
            {ratingDistribution.map((item) => (
              <button
                key={item.stars}
                onClick={() => setFilterStars(filterStars === item.stars ? null : item.stars)}
                className={`flex w-full items-center gap-3 rounded-lg px-2 py-1 text-sm transition-colors hover:bg-gray-50 ${filterStars === item.stars ? 'bg-brand-50' : ''}`}
              >
                <span className="w-8 text-right font-medium text-gray-700">{item.stars}</span>
                <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                <div className="flex-1">
                  <div className="h-2 rounded-full bg-gray-100">
                    <div
                      className="h-2 rounded-full bg-yellow-400 transition-all"
                      style={{ width: `${item.percentage}%` }}
                    />
                  </div>
                </div>
                <span className="w-8 text-right text-gray-500">{item.count}</span>
              </button>
            ))}
          </div>
        </div>
      </motion.div>

      {/* Filter Bar */}
      {filterStars && (
        <motion.div variants={fadeInUp} className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-gray-500" />
          <span className="text-sm text-gray-600">Showing {filterStars}-star reviews</span>
          <button
            onClick={() => setFilterStars(null)}
            className="text-sm font-medium text-brand-600 hover:text-brand-700"
          >
            Clear filter
          </button>
        </motion.div>
      )}

      {/* Reviews List */}
      <motion.div variants={fadeInUp} className="space-y-4">
        {filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-gray-300 py-16">
            <MessageSquare className="h-12 w-12 text-gray-300" />
            <p className="mt-4 font-medium text-gray-500">No reviews yet</p>
          </div>
        ) : (
          filtered.map((review) => (
            <div key={review.id} className="rounded-xl border border-gray-200 bg-white p-6">
              <div className="flex items-start justify-between">
                <div>
                  <div className="flex items-center gap-2">
                    <div className="flex items-center gap-0.5">
                      {[1, 2, 3, 4, 5].map((s) => (
                        <Star
                          key={s}
                          className={`h-4 w-4 ${s <= review.overallRating ? 'fill-yellow-400 text-yellow-400' : 'text-gray-300'}`}
                        />
                      ))}
                    </div>
                    {review.title && (
                      <span className="font-medium text-gray-900">{review.title}</span>
                    )}
                  </div>
                  <p className="mt-1 text-xs text-gray-500">
                    {review.customerName && <>{review.customerName} &middot; </>}
                    Order #{review.orderId.slice(-4)} &middot;{' '}
                    {format(new Date(review.createdAt), 'MMM d, yyyy')}
                  </p>
                </div>
                <Badge variant={review.overallRating >= 4 ? 'success' : review.overallRating >= 3 ? 'warning' : 'error'}>
                  {review.overallRating}/5
                </Badge>
              </div>

              {review.comment && (
                <p className="mt-3 text-sm text-gray-700">{review.comment}</p>
              )}

              {/* Sub-ratings */}
              <div className="mt-3 flex flex-wrap gap-3">
                {review.foodRating > 0 && (
                  <span className="text-xs text-gray-500">Food: {review.foodRating}/5</span>
                )}
                {review.valueRating && review.valueRating > 0 && (
                  <span className="text-xs text-gray-500">Value: {review.valueRating}/5</span>
                )}
                {review.deliveryRating && review.deliveryRating > 0 && (
                  <span className="text-xs text-gray-500">Delivery: {review.deliveryRating}/5</span>
                )}
              </div>

              {/* Chef Response */}
              {review.chefResponse && (
                <div className="mt-4 rounded-lg bg-gray-50 p-4">
                  <p className="text-xs font-medium text-gray-500">Your reply</p>
                  <p className="mt-1 text-sm text-gray-700">{review.chefResponse}</p>
                  {review.chefRespondedAt && (
                    <p className="mt-1 text-xs text-gray-400">
                      {format(new Date(review.chefRespondedAt), 'MMM d, yyyy')}
                    </p>
                  )}
                </div>
              )}

              {/* Reply Form */}
              {!review.chefResponse && (
                <>
                  {replyingTo === review.id ? (
                    <div className="mt-4 space-y-3">
                      <textarea
                        value={replyText}
                        onChange={(e) => setReplyText(e.target.value)}
                        placeholder="Write your reply..."
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20"
                        rows={3}
                      />
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          onClick={() =>
                            replyMutation.mutate({ reviewId: review.id, response: replyText })
                          }
                          isLoading={replyMutation.isPending}
                          disabled={!replyText.trim()}
                        >
                          Post Reply
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setReplyingTo(null);
                            setReplyText('');
                          }}
                        >
                          Cancel
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <button
                      onClick={() => setReplyingTo(review.id)}
                      className="mt-3 flex items-center gap-1.5 text-sm font-medium text-brand-600 hover:text-brand-700"
                    >
                      <MessageSquare className="h-4 w-4" />
                      Reply
                    </button>
                  )}
                </>
              )}
            </div>
          ))
        )}
      </motion.div>
    </motion.div>
  );
}
