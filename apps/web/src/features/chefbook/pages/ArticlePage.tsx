import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { toast } from 'sonner';
import { Loader2, Clock, Share2, Trash2 } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui';
import {
  useArticle,
  useReact,
  useAddComment,
  useDeleteComment,
  shareTargets,
  REACTIONS,
  type ArticleBlock,
  type ReactionType,
} from '../api/useChefBook';

// The article reader. Body blocks are rendered natively rather than as HTML —
// the API sends typed blocks, so there is no markup to sanitise here.

function Block({ block }: { block: ArticleBlock }) {
  switch (block.type) {
    case 'heading':
      return <h2 className="mt-8 font-display text-xl font-semibold text-ink">{block.text}</h2>;
    case 'quote':
      return (
        <blockquote className="mt-6 border-l-2 border-herb pl-4 text-ink-soft italic">
          {block.text}
        </blockquote>
      );
    case 'image':
      return (
        <figure className="mt-6">
          <img src={block.url} alt={block.caption ?? ''} className="w-full rounded-lg" loading="lazy" decoding="async" />
          {block.caption && (
            <figcaption className="mt-2 text-center text-sm text-ink-muted">{block.caption}</figcaption>
          )}
        </figure>
      );
    case 'list':
      return (
        <ul className="mt-4 list-disc space-y-1 pl-5 text-ink">
          {(block.items ?? []).map((it, i) => (
            <li key={i}>{it}</li>
          ))}
        </ul>
      );
    default:
      return <p className="mt-4 leading-relaxed text-ink">{block.text}</p>;
  }
}

export default function ArticlePage() {
  const { slug } = useParams<{ slug: string }>();
  const { data, isLoading, isError } = useArticle(slug);
  const { user, isAuthenticated } = useAuth();
  const react = useReact(slug);
  const addComment = useAddComment(slug);
  const removeComment = useDeleteComment(slug);
  const [comment, setComment] = useState('');
  const [shareOpen, setShareOpen] = useState(false);

  const article = data?.data;

  if (isLoading) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }
  if (isError || !article) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-16 text-center">
        <p className="text-ink-soft">That article isn&apos;t available.</p>
        <Link to="/chefbook" className="mt-3 inline-block text-herb hover:underline">
          Back to ChefBook
        </Link>
      </div>
    );
  }

  const url = typeof window !== 'undefined' ? window.location.href : '';

  const onReact = (type: ReactionType) => {
    if (!isAuthenticated) {
      toast.error('Sign in to react');
      return;
    }
    // Tapping the active reaction clears it — the server reads '' as "remove".
    const next = article.viewerReaction === type ? '' : type;
    react.mutate({ id: article.id, reaction: next });
  };

  const onShare = async () => {
    // Prefer the OS share sheet where it exists; it's the only route to
    // Instagram and the like, which have no web share URL.
    if (navigator.share) {
      try {
        await navigator.share({ title: article.title, text: article.excerpt, url });
        return;
      } catch {
        // The user dismissed the sheet — fall through to the link list rather
        // than reporting an error they caused deliberately.
      }
    }
    setShareOpen((v) => !v);
  };

  const submitComment = () => {
    const body = comment.trim();
    if (!body) return;
    addComment.mutate(
      { id: article.id, body },
      {
        onSuccess: () => {
          setComment('');
          toast.success('Comment posted');
        },
        onError: (e: unknown) =>
          toast.error(e instanceof Error ? e.message : 'Could not post your comment'),
      },
    );
  };

  return (
    <article className="mx-auto max-w-2xl px-4 py-10">
      <Link to="/chefbook" className="text-sm text-herb hover:underline">
        ← ChefBook
      </Link>

      <h1 className="mt-4 font-display text-3xl font-semibold text-ink">{article.title}</h1>

      <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-ink-soft">
        <Link to={`/chefs/${article.chefId}`} className="flex items-center gap-2 hover:underline">
          {article.chefImage ? (
            <img src={article.chefImage} alt="" className="h-8 w-8 rounded-full object-cover" loading="lazy" />
          ) : (
            <div aria-hidden="true" className="h-8 w-8 rounded-full bg-herb-tint" />
          )}
          {article.chefName}
        </Link>
        <span className="flex items-center gap-1 text-ink-muted">
          <Clock aria-hidden="true" className="h-4 w-4" />
          {article.readingMinutes} min read
        </span>
      </div>

      {article.cover && (
        <img src={article.cover} alt="" className="mt-6 w-full rounded-xl" loading="lazy" decoding="async" />
      )}

      <div className="mt-6">
        {(article.blocks ?? []).map((b, i) => (
          <Block key={i} block={b} />
        ))}
      </div>

      {article.tags && article.tags.length > 0 && (
        <div className="mt-8 flex flex-wrap gap-2">
          {article.tags.map((t) => (
            <Link
              key={t}
              to={`/chefbook?tag=${encodeURIComponent(t)}`}
              className="rounded-full border border-mist px-3 py-1 text-sm text-ink-soft hover:border-ink-soft hover:text-ink"
            >
              #{t}
            </Link>
          ))}
        </div>
      )}

      {/* Reactions */}
      <div className="mt-8 border-t border-mist pt-6">
        <div className="flex flex-wrap items-center gap-2">
          {REACTIONS.map((r) => {
            const active = article.viewerReaction === r.type;
            const count = article.reactionCounts?.[r.type] ?? 0;
            return (
              <button
                key={r.type}
                type="button"
                onClick={() => onReact(r.type)}
                aria-pressed={active}
                disabled={react.isPending}
                className={`flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm transition-colors disabled:opacity-60 ${
                  active
                    ? 'border-herb bg-herb-tint text-herb'
                    : 'border-mist bg-paper text-ink-soft hover:border-ink-soft hover:text-ink'
                }`}
              >
                <span aria-hidden="true">{r.emoji}</span>
                <span>{r.label}</span>
                {count > 0 && <span className="tabular-nums">{count}</span>}
              </button>
            );
          })}

          <div className="relative ml-auto">
            <Button variant="ghost" size="sm" onClick={onShare} aria-expanded={shareOpen}>
              <Share2 aria-hidden="true" className="h-4 w-4" />
              Share
            </Button>
            {shareOpen && (
              <div className="absolute right-0 z-20 mt-2 w-44 rounded-lg border border-mist bg-bone py-1 shadow-3">
                {shareTargets(url, article.title).map((s) => (
                  <a
                    key={s.name}
                    href={s.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block px-3 py-2 text-sm text-ink-soft hover:bg-paper hover:text-ink"
                    onClick={() => setShareOpen(false)}
                  >
                    {s.name}
                  </a>
                ))}
                <button
                  type="button"
                  className="block w-full px-3 py-2 text-left text-sm text-ink-soft hover:bg-paper hover:text-ink"
                  onClick={() => {
                    void navigator.clipboard?.writeText(url);
                    toast.success('Link copied');
                    setShareOpen(false);
                  }}
                >
                  Copy link
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Comments */}
      <section className="mt-8 border-t border-mist pt-6">
        <h2 className="font-medium text-ink">
          Comments {article.commentsCount > 0 && <span className="tabular-nums text-ink-muted">({article.commentsCount})</span>}
        </h2>

        {isAuthenticated ? (
          <div className="mt-3">
            <textarea
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={3}
              maxLength={1000}
              placeholder="Ask the chef something, or say how yours turned out."
              className="w-full rounded-lg border border-mist bg-paper p-3 text-sm text-ink placeholder:text-ink-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
            />
            <div className="mt-2 flex justify-end">
              <Button
                variant="primary"
                size="sm"
                onClick={submitComment}
                disabled={!comment.trim() || addComment.isPending}
                isLoading={addComment.isPending}
              >
                Post
              </Button>
            </div>
          </div>
        ) : (
          <p className="mt-3 text-sm text-ink-soft">
            <Link to="/login" className="text-herb hover:underline">
              Sign in
            </Link>{' '}
            to join the conversation.
          </p>
        )}

        <ul className="mt-4 space-y-4">
          {(article.comments ?? []).map((c) => (
            <li key={c.id} className="flex gap-3">
              {c.userAvatar ? (
                <img src={c.userAvatar} alt="" className="h-8 w-8 shrink-0 rounded-full object-cover" loading="lazy" />
              ) : (
                <div aria-hidden="true" className="h-8 w-8 shrink-0 rounded-full bg-mist" />
              )}
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-ink">{c.userName || 'Someone'}</p>
                <p className="text-sm text-ink-soft">{c.body}</p>
              </div>
              {user?.id === c.userId && (
                <button
                  type="button"
                  aria-label="Delete your comment"
                  onClick={() => removeComment.mutate({ id: article.id, commentId: c.id })}
                  className="shrink-0 text-ink-muted hover:text-paprika"
                >
                  <Trash2 aria-hidden="true" className="h-4 w-4" />
                </button>
              )}
            </li>
          ))}
        </ul>
        {(article.comments ?? []).length === 0 && (
          <p className="mt-4 text-sm text-ink-muted">No comments yet.</p>
        )}
      </section>
    </article>
  );
}
