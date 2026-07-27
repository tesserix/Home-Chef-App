import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// ChefBook authoring — the chef's half of /chef/chefbook/articles. Reading
// happens in the customer app and on the web; this app only writes.
//
// Paths carry no /v1 prefix here because this app's EXPO_PUBLIC_API_URL
// already includes it — matching every other hook in this directory.

export type BlockType = 'paragraph' | 'heading' | 'image' | 'quote' | 'list';

export interface ArticleBlock {
  type: BlockType;
  text?: string;
  url?: string;
  caption?: string;
  items?: string[];
}

export interface Article {
  id: string;
  title: string;
  slug: string;
  cover?: string;
  excerpt: string;
  tags?: string[];
  status: 'draft' | 'published' | 'archived' | 'flagged';
  readingMinutes: number;
  reactionsTotal: number;
  commentsCount: number;
  createdAt: string;
  publishedAt?: string;
}

export interface ArticleInput {
  title: string;
  cover?: string;
  blocks: ArticleBlock[];
  tags?: string[];
  status: 'draft' | 'published';
}

interface ListResponse {
  data: Article[];
  total: number;
  page: number;
  limit: number;
}

export function useMyArticles() {
  return useQuery({
    queryKey: ['chefbook', 'mine'],
    queryFn: async () => {
      const r = await api.get<ListResponse>('/chef/chefbook/articles');
      return r.data;
    },
  });
}

export function useSaveArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ArticleInput) => api.post('/chef/chefbook/articles', input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chefbook', 'mine'] }),
  });
}

export function useDeleteArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/chef/chefbook/articles/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chefbook', 'mine'] }),
  });
}

/**
 * The API refuses content with no culinary signal (422, code `off_topic`) and
 * says what to write instead. Surface its wording rather than a generic
 * failure, which would leave the chef guessing why a save didn't take.
 */
export function chefBookErrorMessage(err: unknown): string {
  const e = err as { response?: { data?: { error?: string } }; message?: string };
  return e?.response?.data?.error || e?.message || 'Could not save your article';
}
