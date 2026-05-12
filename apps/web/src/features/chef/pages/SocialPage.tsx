import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import {
  Plus,
  Image as ImageIcon,
  Heart,
  MessageCircle,
  Trash2,
  Edit2,
  Loader2,
  X,
  Upload,
  Hash,
  AlertCircle,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import type { SocialPost, MenuItem, PaginatedResponse } from '@/shared/types';

export default function ChefSocialPage() {
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingPost, setEditingPost] = useState<SocialPost | null>(null);
  const queryClient = useQueryClient();

  const { data: posts, isLoading } = useQuery({
    queryKey: ['chef-social-posts'],
    queryFn: () => apiClient.get<PaginatedResponse<SocialPost>>('/chef/social/posts'),
  });

  const deleteMutation = useMutation({
    mutationFn: (postId: string) => apiClient.delete(`/chef/social/posts/${postId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-social-posts'] });
      toast.success('Post deleted');
    },
  });

  const handleDelete = (post: SocialPost) => {
    if (confirm('Are you sure you want to delete this post?')) {
      deleteMutation.mutate(post.id);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink">Social Feed</h1>
          <p className="mt-1 text-ink-soft">
            Share your culinary creations with customers
          </p>
        </div>
        <button onClick={() => setShowCreateModal(true)} className="btn-primary">
          <Plus className="h-5 w-5" />
          Create Post
        </button>
      </div>

      {/* Content Moderation Notice */}
      <div className="flex items-start gap-3 rounded-xl bg-info/10 p-4">
        <AlertCircle className="h-5 w-5 flex-shrink-0 text-info" />
        <div className="text-sm text-info">
          <p className="font-medium">Content Guidelines</p>
          <p className="mt-1">
            To maintain platform safety, posts are automatically scanned for personal contact
            information. Phone numbers, emails, and social media handles will be filtered.
            All customer communication should go through the platform.
          </p>
        </div>
      </div>

      {/* Posts Grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-herb" />
        </div>
      ) : (posts?.data ?? []).length === 0 ? (
        <div className="rounded-xl bg-bone p-12 text-center shadow-sm">
          <ImageIcon className="mx-auto h-12 w-12 text-ink-muted" />
          <h3 className="mt-4 font-medium text-ink">No posts yet</h3>
          <p className="mt-2 text-ink-soft">
            Share photos of your dishes to attract more customers
          </p>
          <button onClick={() => setShowCreateModal(true)} className="btn-primary mt-4">
            Create Your First Post
          </button>
        </div>
      ) : (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {(posts?.data ?? []).map((post) => (
            <PostCard
              key={post.id}
              post={post}
              onEdit={() => {
                setEditingPost(post);
                setShowCreateModal(true);
              }}
              onDelete={() => handleDelete(post)}
            />
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      {showCreateModal && (
        <CreatePostModal
          post={editingPost}
          onClose={() => {
            setShowCreateModal(false);
            setEditingPost(null);
          }}
        />
      )}
    </div>
  );
}

function PostCard({
  post,
  onEdit,
  onDelete,
}: {
  post: SocialPost;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="rounded-xl bg-bone shadow-sm overflow-hidden">
      {/* Image */}
      {post.images.length > 0 && (
        <div className="aspect-square relative">
          <img
            src={post.images[0]}
            alt="Post"
            className="h-full w-full object-cover" loading="lazy" decoding="async"
          />
          {post.images.length > 1 && (
            <div className="absolute bottom-2 right-2 rounded-full bg-ink/50 px-2 py-0.5 text-xs text-paper">
              +{post.images.length - 1} more
            </div>
          )}
        </div>
      )}

      <div className="p-4">
        {/* Content */}
        <p className="text-ink line-clamp-3">{post.content}</p>

        {/* Hashtags */}
        {post.hashtags.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {post.hashtags.map((tag) => (
              <span key={tag} className="text-sm text-herb">
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Stats */}
        <div className="mt-4 flex items-center gap-4 text-sm text-ink-muted">
          <span className="flex items-center gap-1">
            <Heart className="h-4 w-4" />
            {post.likesCount}
          </span>
          <span className="flex items-center gap-1">
            <MessageCircle className="h-4 w-4" />
            {post.commentsCount}
          </span>
        </div>

        {/* Date */}
        <p className="mt-2 text-xs text-ink-muted">
          {new Date(post.createdAt).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            hour: 'numeric',
            minute: '2-digit',
          })}
        </p>

        {/* Actions */}
        <div className="mt-4 flex gap-2 border-t pt-4">
          <button
            onClick={onEdit}
            className="flex-1 rounded-lg bg-mist py-2 text-sm font-medium text-ink-soft hover:bg-mist"
          >
            <Edit2 className="mr-1 inline h-4 w-4" />
            Edit
          </button>
          <button
            onClick={onDelete}
            className="flex-1 rounded-lg bg-paprika-tint py-2 text-sm font-medium text-paprika hover:bg-paprika"
          >
            <Trash2 className="mr-1 inline h-4 w-4" />
            Delete
          </button>
        </div>
      </div>
    </div>
  );
}

function CreatePostModal({
  post,
  onClose,
}: {
  post: SocialPost | null;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [images, setImages] = useState<string[]>(post?.images || []);
  const [hashtags, setHashtags] = useState<string[]>(post?.hashtags || []);
  const [newHashtag, setNewHashtag] = useState('');

  const { data: menuItems } = useQuery({
    queryKey: ['chef-menu-items'],
    queryFn: () => apiClient.get<{ items: MenuItem[] }>('/chef/menu'),
  });

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    defaultValues: {
      content: post?.content || '',
      taggedMenuItems: post?.taggedMenuItems || [],
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: { content: string; images: string[]; hashtags: string[]; taggedMenuItems: string[] }) =>
      post
        ? apiClient.put(`/chef/social/posts/${post.id}`, data)
        : apiClient.post('/chef/social/posts', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-social-posts'] });
      toast.success(post ? 'Post updated' : 'Post created');
      onClose();
    },
    onError: () => {
      toast.error('Failed to save post');
    },
  });

  const addHashtag = () => {
    const tag = newHashtag.startsWith('#') ? newHashtag : `#${newHashtag}`;
    if (tag.length > 1 && !hashtags.includes(tag)) {
      setHashtags([...hashtags, tag]);
      setNewHashtag('');
    }
  };

  const onSubmit = (data: { content: string; taggedMenuItems: string[] }) => {
    createMutation.mutate({
      content: data.content,
      images,
      hashtags,
      taggedMenuItems: data.taggedMenuItems,
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-ink/50 p-4 pt-20">
      <div className="w-full max-w-lg rounded-xl bg-bone shadow-xl">
        <div className="flex items-center justify-between border-b p-4">
          <h2 className="text-lg font-semibold text-ink">
            {post ? 'Edit Post' : 'Create Post'}
          </h2>
          <button
            onClick={onClose}
            aria-label="Close"
            className="rounded-full p-1 text-ink-muted hover:text-ink-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb"
          >
            <X className="h-6 w-6" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="p-4 space-y-4">
          {/* Image Upload */}
          <div>
            <label className="block text-sm font-medium text-ink-soft mb-2">Photos</label>
            <div className="flex gap-2 flex-wrap">
              {images.map((img, i) => (
                <div key={i} className="relative h-20 w-20">
                  <img src={img} alt={`Upload preview ${i + 1}`} loading="lazy" decoding="async" className="h-full w-full rounded-lg object-cover" />
                  <button
                    type="button"
                    onClick={() => setImages(images.filter((_, idx) => idx !== i))}
                    aria-label={`Remove image ${i + 1}`}
                    className="absolute -right-1 -top-1 rounded-full bg-paprika p-0.5 text-paper focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-paprika focus-visible:ring-offset-1"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}
              {images.length < 5 && (
                <button
                  type="button"
                  className="flex h-20 w-20 items-center justify-center rounded-lg border-2 border-dashed border-mist-strong hover:border-herb"
                >
                  <Upload className="h-6 w-6 text-ink-muted" />
                </button>
              )}
            </div>
            <p className="mt-1 text-xs text-ink-muted">Add up to 5 photos</p>
          </div>

          {/* Content */}
          <div>
            <label className="block text-sm font-medium text-ink-soft mb-2">Caption</label>
            <textarea
              {...register('content', { required: 'Caption is required' })}
              rows={4}
              placeholder="Share something about this dish..."
              className="input-base"
            />
            {errors.content && (
              <p className="mt-1 text-xs text-paprika">{errors.content.message}</p>
            )}
          </div>

          {/* Hashtags */}
          <div>
            <label className="block text-sm font-medium text-ink-soft mb-2">
              <Hash className="mr-1 inline h-4 w-4" />
              Hashtags
            </label>
            <div className="flex flex-wrap gap-2 mb-2">
              {hashtags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-3 py-1 text-sm text-herb"
                >
                  {tag}
                  <button type="button" onClick={() => setHashtags(hashtags.filter((t) => t !== tag))}>
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                type="text"
                value={newHashtag}
                onChange={(e) => setNewHashtag(e.target.value.replace(/\s/g, ''))}
                onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), addHashtag())}
                placeholder="#homemade"
                className="input-base flex-1"
              />
              <button type="button" onClick={addHashtag} className="btn-outline">
                Add
              </button>
            </div>
          </div>

          {/* Tag Menu Items */}
          <div>
            <label className="block text-sm font-medium text-ink-soft mb-2">
              Tag Menu Items (Optional)
            </label>
            <select {...register('taggedMenuItems')} multiple className="input-base h-24">
              {menuItems?.items.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            <p className="mt-1 text-xs text-ink-muted">
              Hold Ctrl/Cmd to select multiple items
            </p>
          </div>

          {/* Submit */}
          <div className="flex justify-end gap-3 pt-4 border-t">
            <button type="button" onClick={onClose} className="btn-outline">
              Cancel
            </button>
            <button
              type="submit"
              disabled={createMutation.isPending}
              className="btn-primary"
            >
              {createMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : post ? (
                'Update Post'
              ) : (
                'Publish Post'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
