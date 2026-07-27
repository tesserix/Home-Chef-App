import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// ChefBook — chef-authored culinary articles. Backed by MongoDB via
// /chefbook/*; deliberately separate from the short-post social feed, which
// still lives on /social/*.

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
  /** Only present on the detail view — the feed sends excerpts only. */
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

export function useChefBookFeed(params?: { chefId?: string; tag?: string; q?: string }) {
  return useQuery({
    queryKey: ['chefbook', 'feed', params ?? {}],
    queryFn: () =>
      apiClient.get<FeedResponse>('/chefbook/articles', params as Record<string, string>),
  });
}

export function useArticle(slug?: string) {
  return useQuery({
    queryKey: ['chefbook', 'article', slug],
    queryFn: () => apiClient.get<{ data: Article }>(`/chefbook/articles/${slug}`),
    enabled: !!slug,
  });
}

/**
 * Set, switch or clear a reaction. Sending the type that's already active
 * clears it — the server treats an empty reaction as "remove mine", which is
 * what tapping the highlighted one again should mean.
 */
export function useReact(slug?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reaction }: { id: string; reaction: ReactionType | '' }) =>
      apiClient.post(`/chefbook/articles/${id}/react`, { reaction }),
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
      apiClient.post(`/chefbook/articles/${id}/comments`, { body }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chefbook', 'article', slug] }),
  });
}

export function useDeleteComment(slug?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, commentId }: { id: string; commentId: string }) =>
      apiClient.delete(`/chefbook/articles/${id}/comments/${commentId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chefbook', 'article', slug] }),
  });
}

// ---- Chef authoring ---------------------------------------------------------

export interface ArticleInput {
  title: string;
  cover?: string;
  blocks: ArticleBlock[];
  tags?: string[];
  status: 'draft' | 'published';
}

export function useMyArticles() {
  return useQuery({
    queryKey: ['chefbook', 'mine'],
    queryFn: () => apiClient.get<FeedResponse>('/chef/chefbook/articles'),
  });
}

export function useSaveArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id?: string; input: ArticleInput }) =>
      id
        ? apiClient.put(`/chef/chefbook/articles/${id}`, input)
        : apiClient.post<{ data: Article }>('/chef/chefbook/articles', input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chefbook', 'mine'] });
      qc.invalidateQueries({ queryKey: ['chefbook', 'feed'] });
    },
  });
}

export function useDeleteArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiClient.delete(`/chef/chefbook/articles/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chefbook', 'mine'] }),
  });
}

/**
 * Share targets for an article. Native share where the browser offers it
 * (mobile, and Safari on desktop), falling back to per-network links.
 *
 * Instagram has no web share URL — it only accepts shares from its own app —
 * so it is deliberately absent rather than represented by a link that would
 * silently do nothing.
 */
export function shareTargets(url: string, title: string) {
  const u = encodeURIComponent(url);
  const t = encodeURIComponent(title);
  return [
    { name: 'WhatsApp', href: `https://wa.me/?text=${t}%20${u}` },
    { name: 'Facebook', href: `https://www.facebook.com/sharer/sharer.php?u=${u}` },
    { name: 'X', href: `https://twitter.com/intent/tweet?url=${u}&text=${t}` },
    { name: 'Pinterest', href: `https://pinterest.com/pin/create/button/?url=${u}&description=${t}` },
  ];
}
