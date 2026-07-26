// Social feed hooks — endpoints confirmed from apps/api/handlers/social.go
// GET  /v1/social/feed          → paginated PostResponse list
// POST /v1/social/posts/:id/like → toggle like (returns { liked, likesCount })

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface SocialPost {
  id: string;
  chefId: string;
  /** The chef's USER id — what /v1/blocks takes. Absent on older API builds. */
  chefUserId?: string;
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

/**
 * The wire shape of a post, as models.PostResponse actually serialises it.
 *
 * This is NOT SocialPost. The API nests the author under `chef` and has never
 * had a top-level `chefName`; the feed hook used to cast the response straight
 * to SocialPost, so `post.chefName` was always undefined and the feed screen's
 * `post.chefName.charAt(0)` threw as soon as a single post existed. The screen
 * only looked healthy because an empty feed short-circuits to <EmptyState>.
 */
interface PostWire {
  id: string;
  chefId: string;
  chef?: {
    id?: string;
    businessName?: string;
    profileImage?: string;
    verified?: boolean;
    userId?: string;
  };
  content: string;
  images?: string[];
  hashtags?: string[];
  likesCount: number;
  commentsCount: number;
  isLiked: boolean;
  createdAt: string;
}

/** Flatten the wire shape into what the screens render. */
function toSocialPost(p: PostWire): SocialPost {
  return {
    id: p.id,
    chefId: p.chefId,
    chefUserId: p.chef?.userId,
    // Falls back rather than rendering "undefined" — a post whose Chef relation
    // failed to preload should still be readable.
    chefName: p.chef?.businessName?.trim() || 'Home chef',
    chefAvatar: p.chef?.profileImage || undefined,
    content: p.content,
    images: p.images,
    hashtags: p.hashtags,
    likesCount: p.likesCount ?? 0,
    commentsCount: p.commentsCount ?? 0,
    isLiked: p.isLiked ?? false,
    createdAt: p.createdAt,
  };
}

export function useSocialFeed(params: SocialFeedParams = {}) {
  return useQuery<SocialFeedResponse>({
    queryKey: ['social-feed', params],
    queryFn: () =>
      api.get('/v1/social/feed', { params }).then((r) => {
        const body = r.data as {
          data?: PostWire[];
          total?: number;
          page?: number;
          limit?: number;
        };
        return {
          data: (body.data ?? []).map(toSocialPost),
          total: body.total ?? 0,
          page: body.page ?? 1,
          limit: body.limit ?? 20,
        };
      }),
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
