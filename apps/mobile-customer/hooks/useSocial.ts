// Social feed hooks — endpoints confirmed from apps/api/handlers/social.go
// GET  /v1/social/feed          → paginated PostResponse list
// POST /v1/social/posts/:id/like → toggle like (returns { liked, likesCount })

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface SocialPost {
  id: string;
  chefId: string;
  chefName: string;
  chefAvatar?: string;
  content: string;
  images?: string[];
  hashtags?: string[];
  likesCount: number;
  commentsCount: number;
  isLiked: boolean;
  createdAt: string;
}

export interface SocialFeedResponse {
  data: SocialPost[];
  total: number;
  page: number;
  limit: number;
}

export interface SocialFeedParams {
  page?: number;
  limit?: number;
  hashtag?: string;
}

export function useSocialFeed(params: SocialFeedParams = {}) {
  return useQuery<SocialFeedResponse>({
    queryKey: ['social-feed', params],
    queryFn: () =>
      api
        .get('/v1/social/feed', { params })
        .then((r) => r.data as SocialFeedResponse),
    staleTime: 1000 * 60, // 1 minute — social feed changes frequently
  });
}

export interface LikePostResponse {
  liked: boolean;
  likesCount: number;
}

export function useLikePost() {
  const queryClient = useQueryClient();

  return useMutation<LikePostResponse, Error, string>({
    mutationFn: (postId: string) =>
      api
        .post(`/v1/social/posts/${postId}/like`)
        .then((r) => r.data as LikePostResponse),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['social-feed'] });
    },
  });
}
