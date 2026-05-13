import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Heart,
  MessageCircle,
  Bookmark,
  Share2,
  MoreHorizontal,
  ChefHat,
  Clock,
  Loader2,
  Search,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useAuth } from '@/app/providers/AuthProvider';
import type { SocialPost, PaginatedResponse } from '@/shared/types';

const TRENDING_HASHTAGS = [
  '#homemade',
  '#southindian',
  '#weekendspecial',
  '#healthyeating',
  '#desserts',
  '#biryani',
];

export default function SocialFeedPage() {
  const { isAuthenticated } = useAuth();
  const queryClient = useQueryClient();
  const [selectedHashtag, setSelectedHashtag] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['social-feed', selectedHashtag],
    queryFn: () =>
      apiClient.get<PaginatedResponse<SocialPost>>('/social/posts', {
        hashtag: selectedHashtag || undefined,
      }),
  });

  const likeMutation = useMutation({
    mutationFn: (postId: string) => apiClient.post(`/social/posts/${postId}/like`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['social-feed'] });
    },
  });

  const saveMutation = useMutation({
    mutationFn: (postId: string) => apiClient.post(`/social/posts/${postId}/save`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['social-feed'] });
    },
  });

  const handleLike = (postId: string) => {
    if (!isAuthenticated) {
      toast.error('Please log in to like posts');
      return;
    }
    likeMutation.mutate(postId);
  };

  const handleSave = (postId: string) => {
    if (!isAuthenticated) {
      toast.error('Please log in to save posts');
      return;
    }
    saveMutation.mutate(postId);
  };

  const posts = data?.data || [];

  return (
    <div className="min-h-screen bg-paper">
      <div className="container-app py-8">
        <div className="flex flex-col gap-8 lg:flex-row">
          {/* Main Feed */}
          <div className="flex-1 max-w-2xl">
            {/* Header */}
            <div className="mb-6">
              <h1 className="font-display text-2xl font-semibold text-ink">Chef's Feed</h1>
              <p className="mt-1 text-ink-soft">
                Discover culinary creations from our talented home chefs
              </p>
            </div>

            {/* Search */}
            <div className="mb-6 relative">
              <label htmlFor="social-feed-search" className="sr-only">Search social feed</label>
              <Search aria-hidden="true" className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-ink-muted" />
              <input
                id="social-feed-search"
                type="search"
                aria-label="Search posts, chefs, or dishes"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search posts, chefs, or dishes..."
                className="input-base pl-12"
              />
            </div>

            {/* Trending Tags */}
            <div className="mb-6 flex flex-wrap gap-2">
              <button type="button"
                onClick={() => setSelectedHashtag(null)}
                className={`rounded-full px-3 py-1.5 text-sm font-medium transition-colors ${
                  selectedHashtag === null
                    ? 'bg-herb text-paper'
                    : 'bg-bone text-ink-soft hover:bg-mist'
                }`}
              >
                All
              </button>
              {TRENDING_HASHTAGS.map((tag) => (
                <button type="button"
                  key={tag}
                  onClick={() => setSelectedHashtag(tag)}
                  className={`rounded-full px-3 py-1.5 text-sm font-medium transition-colors ${
                    selectedHashtag === tag
                      ? 'bg-herb text-paper'
                      : 'bg-bone text-ink-soft hover:bg-mist'
                  }`}
                >
                  {tag}
                </button>
              ))}
            </div>

            {/* Posts */}
            {isLoading ? (
              <div className="flex items-center justify-center py-20">
                <Loader2 className="h-8 w-8 animate-spin text-herb"  aria-hidden="true" />
              </div>
            ) : posts.length === 0 ? (
              <div className="rounded-xl bg-bone p-12 text-center shadow-1">
                <ChefHat className="mx-auto h-12 w-12 text-ink-muted"  aria-hidden="true" />
                <h3 className="mt-4 text-lg font-semibold text-ink">No posts yet</h3>
                <p className="mt-2 text-ink-soft">
                  Check back later for delicious content from our chefs
                </p>
              </div>
            ) : (
              <div className="space-y-6">
                {posts.map((post) => (
                  <PostCard
                    key={post.id}
                    post={post}
                    onLike={() => handleLike(post.id)}
                    onSave={() => handleSave(post.id)}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Sidebar */}
          <div className="lg:w-80">
            {/* Trending Chefs */}
            <div className="rounded-xl bg-bone p-6 shadow-1">
              <h3 className="font-semibold text-ink">Trending Chefs</h3>
              <div className="mt-4 space-y-4">
                {[
                  { name: 'Priya\'s Kitchen', cuisine: 'South Indian', followers: '2.5k' },
                  { name: 'Chef Mario\'s', cuisine: 'Italian', followers: '1.8k' },
                  { name: 'Sakura Home', cuisine: 'Japanese', followers: '1.2k' },
                ].map((chef) => (
                  <div key={chef.name} className="flex items-center gap-3">
                    <div className="h-10 w-10 rounded-full bg-herb-tint flex items-center justify-center">
                      <ChefHat className="h-5 w-5 text-herb"  aria-hidden="true" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-ink truncate">{chef.name}</p>
                      <p className="text-sm text-ink-muted">{chef.cuisine}</p>
                    </div>
                    <span className="text-sm text-ink-muted">{chef.followers}</span>
                  </div>
                ))}
              </div>
              <Link
                to="/chefs"
                className="mt-4 block text-center text-sm text-herb hover:text-herb"
              >
                View all chefs
              </Link>
            </div>

            {/* Popular Tags */}
            <div className="mt-6 rounded-xl bg-bone p-6 shadow-1">
              <h3 className="font-semibold text-ink">Popular This Week</h3>
              <div className="mt-4 space-y-2">
                {[
                  { tag: '#biryani', posts: 234 },
                  { tag: '#homemade', posts: 189 },
                  { tag: '#sundaybrunch', posts: 156 },
                  { tag: '#desserts', posts: 134 },
                  { tag: '#healthyfood', posts: 98 },
                ].map((item) => (
                  <button type="button"
                    key={item.tag}
                    onClick={() => setSelectedHashtag(item.tag)}
                    className="flex w-full items-center justify-between rounded-lg p-2 text-left hover:bg-paper"
                  >
                    <span className="text-herb">{item.tag}</span>
                    <span className="text-sm text-ink-muted">{item.posts} posts</span>
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function PostCard({
  post,
  onLike,
  onSave,
}: {
  post: SocialPost;
  onLike: () => void;
  onSave: () => void;
}) {
  const [showComments, setShowComments] = useState(false);
  const [imageIndex, setImageIndex] = useState(0);

  return (
    <div className="rounded-xl bg-bone shadow-1 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between p-4">
        <Link to={`/chefs/${post.chefId}`} className="flex items-center gap-3">
          <img
            src={post.chef?.profileImage || '/placeholder-avatar.png'}
            alt={post.chef?.businessName}
            className="h-10 w-10 rounded-full object-cover"
           loading="lazy" decoding="async"/>
          <div>
            <p className="font-medium text-ink">{post.chef?.businessName}</p>
            <div className="flex items-center gap-1 text-sm text-ink-muted">
              <Clock className="h-3 w-3"  aria-hidden="true" />
              {formatTimeAgo(post.createdAt)}
            </div>
          </div>
        </Link>
        <button type="button" className="p-2 text-ink-muted hover:text-ink-soft">
          <MoreHorizontal className="h-5 w-5"  aria-hidden="true" />
        </button>
      </div>

      {/* Images */}
      {post.images.length > 0 && (
        <div className="relative aspect-square">
          <img
            src={post.images[imageIndex]}
            alt="Post"
            className="h-full w-full object-cover"
           loading="lazy" decoding="async"/>
          {post.images.length > 1 && (
            <>
              <div className="absolute bottom-4 left-1/2 flex -translate-x-1/2 gap-1.5">
                {post.images.map((_, i) => (
                  <button type="button"
                    key={i}
                    onClick={() => setImageIndex(i)}
                    className={`h-2 w-2 rounded-full transition-colors ${
                      i === imageIndex ? 'bg-bone' : 'bg-bone/50'
                    }`}
                  />
                ))}
              </div>
              <button type="button"
                onClick={() => setImageIndex((i) => (i > 0 ? i - 1 : post.images.length - 1))}
                className="absolute left-2 top-1/2 -translate-y-1/2 rounded-full bg-ink/50 p-2 text-paper"
              >
                ←
              </button>
              <button type="button"
                onClick={() => setImageIndex((i) => (i < post.images.length - 1 ? i + 1 : 0))}
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded-full bg-ink/50 p-2 text-paper"
              >
                →
              </button>
            </>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between p-4">
        <div className="flex items-center gap-4">
          <button type="button"
            onClick={onLike}
            className={`flex items-center gap-1.5 ${
              post.isLiked ? 'text-paprika' : 'text-ink-soft hover:text-paprika'
            }`}
          >
            <Heart className={`h-6 w-6 ${post.isLiked ? 'fill-current' : ''}`}  aria-hidden="true" />
            <span className="text-sm font-medium">{post.likesCount}</span>
          </button>
          <button type="button"
            onClick={() => setShowComments(!showComments)}
            className="flex items-center gap-1.5 text-ink-soft hover:text-herb"
          >
            <MessageCircle className="h-6 w-6" />
            <span className="text-sm font-medium">{post.commentsCount}</span>
          </button>
          <button type="button" className="text-ink-soft hover:text-herb">
            <Share2 className="h-6 w-6"  aria-hidden="true" />
          </button>
        </div>
        <button type="button"
          onClick={onSave}
          className={`${post.isSaved ? 'text-herb' : 'text-ink-soft hover:text-herb'}`}
        >
          <Bookmark className={`h-6 w-6 ${post.isSaved ? 'fill-current' : ''}`}  aria-hidden="true" />
        </button>
      </div>

      {/* Content */}
      <div className="px-4 pb-4">
        <p className="text-ink">
          <Link to={`/chefs/${post.chefId}`} className="font-medium hover:underline">
            {post.chef?.businessName}
          </Link>{' '}
          {post.content}
        </p>

        {/* Hashtags */}
        {post.hashtags.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-2">
            {post.hashtags.map((tag) => (
              <span key={tag} className="text-herb hover:underline cursor-pointer">
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Tagged Items */}
        {post.taggedMenuItems && post.taggedMenuItems.length > 0 && (
          <div className="mt-3 rounded-lg bg-paper p-3">
            <p className="text-sm text-ink-muted">Featured in this post:</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {post.taggedMenuItems.map((itemId) => (
                <span
                  key={itemId}
                  className="rounded-full bg-herb-tint px-3 py-1 text-sm text-herb"
                >
                  Order now
                </span>
              ))}
            </div>
          </div>
        )}

        {/* View Comments */}
        {post.commentsCount > 0 && !showComments && (
          <button type="button"
            onClick={() => setShowComments(true)}
            className="mt-2 text-sm text-ink-muted hover:text-ink-soft"
          >
            View all {post.commentsCount} comments
          </button>
        )}
      </div>

      {/* Comments Section */}
      {showComments && (
        <div className="border-t px-4 py-3">
          <div className="flex gap-3">
            <input
              type="text"
              placeholder="Add a comment..."
              className="flex-1 border-none bg-transparent text-sm outline-none placeholder:text-ink-muted"
            />
            <button type="button" className="text-sm font-medium text-herb hover:text-herb">
              Post
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function formatTimeAgo(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 60) return 'Just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d`;
  return date.toLocaleDateString();
}
