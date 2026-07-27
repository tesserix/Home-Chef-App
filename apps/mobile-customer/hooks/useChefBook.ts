import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// ChefBook — chef-authored culinary articles, served from MongoDB via
// /chefbook/*. Deliberately separate from the short-post social feed, which
// still lives on /social/*.
//
// The response types mirror the web client's so both render the same payload
// the same way; the API is the contract between them.

export type ReactionType = 'yum' | 'love' | 'want_to_try' | 'clever';

/** Display order and labels, matching models.AllReactionTypes on the server. */
export const REACTIONS: { type: ReactionType; emoji: string; label: string }[] = [
  { type: 'yum', emoji: '😋', label: 'Yum' },
  { type: 'love', emoji: '❤️', label: 'Love' },
  { type: 'want_to_try', emoji: '🍳', label: 'Want to try' },
  { type: 'clever', emoji: '💡', label: 'Clever' },
];

export type BlockType = 'paragraph' | 'heading' | 'image' | 'quote' | 'list';

export interface ArticleBlock {
  type: BlockType;
  text?: string;
  url?: string;
  caption?: string;
  items?: string[];
}

export interface ArticleComment {
  id: string;
  userId: string;
  userName: string;
  userAvatar?: string;
  body: string;
  createdAt: string;
}

export interface Article {
  id: string;
  chefId: string;
  chefName: string;
  chefImage?: string;
  title: string;
  slug: string;
  cover?: string;
  excerpt: string;
  /** Detail view only — the feed sends excerpts, not bodies. */
  blocks?: ArticleBlock[];
  tags?: string[];
  status: 'draft' | 'published' | 'archived' | 'flagged';
  readingMinutes: number;
  reactionCounts: Record<ReactionType, number>;
  viewerReaction?: ReactionType;
  reactionsTotal: number;
  commentsCount: number;
  comments?: ArticleComment[];
  createdAt: string;
  publishedAt?: string;
}

interface FeedResponse {
  data: Article[];
  total: number;
  page: number;
  limit: number;
}

export function useChefBookFeed(params?: { chefId?: string; tag?: string }) {
  return useQuery({
    queryKey: ['chefbook', 'feed', params ?? {}],
    queryFn: async () => {
      const r = await api.get<FeedResponse>('/v1/chefbook/articles', { params });
      return r.data;
    },
  });
}

export function useArticle(slug?: string) {
  return useQuery({
    queryKey: ['chefbook', 'article', slug],
    queryFn: async () => {
      const r = await api.get<{ data: Article }>(`/v1/chefbook/articles/${slug}`);
      return r.data;
    },
    enabled: !!slug,
  });
}

/**
 * Set, switch or clear a reaction. Sending the type that's already active
 * clears it — the API reads an empty reaction as "remove mine", which is what
 * tapping the highlighted one again should mean.
 */
export function useReact(slug?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reaction }: { id: string; reaction: ReactionType | '' }) =>
      api.post(`/v1/chefbook/articles/${id}/react`, { reaction }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chefbook', 'article', slug] });
      qc.invalidateQueries({ queryKey: ['chefbook', 'feed'] });
    },
  });
}

export function useAddComment(slug?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) =>
      api.post(`/v1/chefbook/articles/${id}/comments`, { body }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chefbook', 'article', slug] }),
  });
}

export function useDeleteComment(slug?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, commentId }: { id: string; commentId: string }) =>
      api.delete(`/v1/chefbook/articles/${id}/comments/${commentId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chefbook', 'article', slug] }),
  });
}

/** The public URL for an article — what gets shared out of the app. */
export function articleUrl(slug: string): string {
  return `https://fe3dr.com/chefbook/${slug}`;
}
