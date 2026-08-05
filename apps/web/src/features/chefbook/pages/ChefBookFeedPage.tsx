import { Link, useSearchParams } from 'react-router';
import { Loader2, Clock, MessageCircle } from 'lucide-react';
import { useChefBookFeed, REACTIONS, type Article } from '../api/useChefBook';

// ChefBook feed — chef-written culinary articles. Deliberately reads as a
// magazine rather than a social timeline: cover, title, excerpt, byline.

function ReactionSummary({ article }: { article: Article }) {
  if (article.reactionsTotal === 0) return null;
  // Show only the reactions this article actually received, so an unloved
  // article doesn't display a row of zeros.
  const present = REACTIONS.filter((r) => (article.reactionCounts?.[r.type] ?? 0) > 0);
  return (
    <span className="flex items-center gap-1 text-ink-muted">
      <span aria-hidden="true">{present.map((r) => r.emoji).join('')}</span>
      <span className="tabular-nums">{article.reactionsTotal}</span>
      <span className="sr-only">reactions</span>
    </span>
  );
}

function ArticleCard({ article }: { article: Article }) {
  return (
    <article className="overflow-hidden rounded-xl border border-mist bg-bone transition-shadow hover:shadow-2">
      <Link to={`/chefbook/${article.slug}`} className="block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40">
        {article.cover && (
          <img
            src={article.cover}
            alt=""
            className="aspect-[16/9] w-full object-cover"
            loading="lazy"
            decoding="async"
          />
        )}
        <div className="p-4">
          <div className="flex items-center gap-2">
            {article.chefImage ? (
              <img src={article.chefImage} alt="" className="h-6 w-6 rounded-full object-cover" loading="lazy" />
            ) : (
              <div aria-hidden="true" className="h-6 w-6 rounded-full bg-herb-tint" />
            )}
            <span className="truncate text-sm text-ink-soft">{article.chefName}</span>
          </div>

          <h2 className="mt-2 font-display text-lg font-semibold text-ink">{article.title}</h2>
          <p className="mt-1 line-clamp-2 text-sm text-ink-soft">{article.excerpt}</p>

          <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-muted">
            <span className="flex items-center gap-1">
              <Clock aria-hidden="true" className="h-3.5 w-3.5" />
              {article.readingMinutes} min read
            </span>
            {article.commentsCount > 0 && (
              <span className="flex items-center gap-1">
                <MessageCircle aria-hidden="true" className="h-3.5 w-3.5" />
                <span className="tabular-nums">{article.commentsCount}</span>
              </span>
            )}
            <ReactionSummary article={article} />
          </div>
        </div>
      </Link>
    </article>
  );
}

export default function ChefBookFeedPage() {
  const [params] = useSearchParams();
  const tag = params.get('tag') ?? undefined;
  const { data, isLoading, isError } = useChefBookFeed(tag ? { tag } : undefined);
  const articles = data?.data ?? [];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink">ChefBook</h1>
        <p className="mt-1 text-ink-soft">
          Recipes, methods and kitchen notes, written by the chefs who cook them.
        </p>
        {tag && (
          <p className="mt-2 text-sm text-ink-muted">
            Showing <span className="font-medium text-ink">#{tag}</span> ·{' '}
            <Link to="/chefbook" className="text-herb hover:underline">
              clear
            </Link>
          </p>
        )}
      </header>

      {isLoading ? (
        <div className="flex min-h-[30vh] items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-herb" />
        </div>
      ) : isError ? (
        <p className="mt-8 rounded-lg border border-mist bg-paper p-4 text-sm text-ink-soft">
          ChefBook couldn&apos;t load just now. Please try again shortly.
        </p>
      ) : articles.length === 0 ? (
        <p className="mt-8 rounded-lg border border-mist bg-paper p-6 text-center text-sm text-ink-soft">
          No articles yet. When chefs start writing, their recipes and kitchen notes land here.
        </p>
      ) : (
        <div className="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {articles.map((a) => (
            <ArticleCard key={a.id} article={a} />
          ))}
        </div>
      )}
    </div>
  );
}
