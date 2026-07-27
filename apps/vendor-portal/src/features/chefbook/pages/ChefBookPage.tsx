import { useState } from 'react';
import { toast } from 'sonner';
import { Loader2, Plus, Trash2, GripVertical, BookOpen } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';
import {
  useMyArticles,
  useSaveArticle,
  useDeleteArticle,
  type ArticleBlock,
  type BlockType,
} from '../api/useChefBook';

// ChefBook authoring. The editor composes typed blocks directly rather than
// wrapping a rich text field: the API stores typed blocks, so this avoids a
// lossy HTML round-trip and what the chef arranges is exactly what readers get
// on web and mobile.

const BLOCK_LABELS: Record<BlockType, string> = {
  paragraph: 'Paragraph',
  heading: 'Heading',
  image: 'Image',
  quote: 'Quote',
  list: 'List',
};

function emptyBlock(type: BlockType): ArticleBlock {
  return type === 'list' ? { type, items: [''] } : { type, text: '', url: '' };
}

export default function ChefBookPage() {
  const { data, isLoading } = useMyArticles();
  const save = useSaveArticle();
  const remove = useDeleteArticle();

  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState('');
  const [cover, setCover] = useState('');
  const [tags, setTags] = useState('');
  const [blocks, setBlocks] = useState<ArticleBlock[]>([emptyBlock('paragraph')]);

  const articles = data?.data ?? [];

  const reset = () => {
    setEditing(false);
    setTitle('');
    setCover('');
    setTags('');
    setBlocks([emptyBlock('paragraph')]);
  };

  const setBlock = (i: number, patch: Partial<ArticleBlock>) =>
    setBlocks((prev) => prev.map((b, idx) => (idx === i ? { ...b, ...patch } : b)));

  const submit = (status: 'draft' | 'published') => {
    if (!title.trim()) {
      toast.error('Give your article a title');
      return;
    }
    // Drop blocks the chef added but never filled in, rather than sending
    // empties the API would reject as a whole-article failure.
    const cleaned = blocks
      .map((b) =>
        b.type === 'list'
          ? { ...b, items: (b.items ?? []).map((i) => i.trim()).filter(Boolean) }
          : b,
      )
      .filter((b) =>
        b.type === 'image'
          ? !!b.url?.trim()
          : b.type === 'list'
            ? (b.items ?? []).length > 0
            : !!b.text?.trim(),
      );

    if (cleaned.length === 0) {
      toast.error('Add some content before saving');
      return;
    }

    save.mutate(
      {
        input: {
          title: title.trim(),
          cover: cover.trim(),
          blocks: cleaned,
          tags: tags.split(',').map((t) => t.trim()).filter(Boolean),
          status,
        },
      },
      {
        onSuccess: () => {
          toast.success(status === 'published' ? 'Article published' : 'Draft saved');
          reset();
        },
        onError: (e: unknown) => {
          // The API refuses content with no culinary signal. Surface its own
          // wording — it says what to write instead, which a generic message
          // can't.
          toast.error(e instanceof Error ? e.message : 'Could not save your article');
        },
      },
    );
  };

  if (isLoading) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink">ChefBook</h1>
          <p className="mt-1 text-sm text-ink-soft">
            Write about your food — recipes, methods, the story behind a dish. Customers read these
            on the web and in the app.
          </p>
        </div>
        {!editing && (
          <Button variant="primary" onClick={() => setEditing(true)}>
            <Plus aria-hidden="true" className="h-4 w-4" />
            Write
          </Button>
        )}
      </header>

      {editing && (
        <section className="mt-6 rounded-xl border border-mist bg-bone p-4">
          <label className="block text-sm font-medium text-ink-soft" htmlFor="cb-title">
            Title
          </label>
          <input
            id="cb-title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Dalma, three ways"
            className="mt-1 w-full rounded-lg border border-mist bg-paper p-2.5 text-ink placeholder:text-ink-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
          />

          <label className="mt-4 block text-sm font-medium text-ink-soft" htmlFor="cb-cover">
            Cover image URL <span className="font-normal text-ink-muted">(optional)</span>
          </label>
          <input
            id="cb-cover"
            value={cover}
            onChange={(e) => setCover(e.target.value)}
            placeholder="Leave blank to use your first image"
            className="mt-1 w-full rounded-lg border border-mist bg-paper p-2.5 text-ink placeholder:text-ink-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
          />

          <div className="mt-5 space-y-3">
            {blocks.map((b, i) => (
              <div key={i} className="rounded-lg border border-mist bg-paper p-3">
                <div className="flex items-center gap-2">
                  <GripVertical aria-hidden="true" className="h-4 w-4 text-ink-muted" />
                  <select
                    value={b.type}
                    onChange={(e) => setBlock(i, emptyBlock(e.target.value as BlockType))}
                    aria-label={`Block ${i + 1} type`}
                    className="rounded-md border border-mist bg-bone px-2 py-1 text-sm text-ink"
                  >
                    {(Object.keys(BLOCK_LABELS) as BlockType[]).map((t) => (
                      <option key={t} value={t}>
                        {BLOCK_LABELS[t]}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    aria-label={`Remove block ${i + 1}`}
                    onClick={() => setBlocks((prev) => prev.filter((_, idx) => idx !== i))}
                    disabled={blocks.length === 1}
                    className="ml-auto text-ink-muted hover:text-paprika disabled:opacity-40"
                  >
                    <Trash2 aria-hidden="true" className="h-4 w-4" />
                  </button>
                </div>

                {b.type === 'image' ? (
                  <>
                    <input
                      value={b.url ?? ''}
                      onChange={(e) => setBlock(i, { url: e.target.value })}
                      placeholder="Image URL"
                      className="mt-2 w-full rounded-md border border-mist bg-bone p-2 text-sm text-ink"
                    />
                    <input
                      value={b.caption ?? ''}
                      onChange={(e) => setBlock(i, { caption: e.target.value })}
                      placeholder="Caption (optional)"
                      className="mt-2 w-full rounded-md border border-mist bg-bone p-2 text-sm text-ink"
                    />
                  </>
                ) : b.type === 'list' ? (
                  <div className="mt-2 space-y-2">
                    {(b.items ?? []).map((item, j) => (
                      <input
                        key={j}
                        value={item}
                        onChange={(e) =>
                          setBlock(i, {
                            items: (b.items ?? []).map((it, idx) =>
                              idx === j ? e.target.value : it,
                            ),
                          })
                        }
                        placeholder={`Item ${j + 1}`}
                        className="w-full rounded-md border border-mist bg-bone p-2 text-sm text-ink"
                      />
                    ))}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setBlock(i, { items: [...(b.items ?? []), ''] })}
                    >
                      <Plus aria-hidden="true" className="h-4 w-4" />
                      Add item
                    </Button>
                  </div>
                ) : (
                  <textarea
                    value={b.text ?? ''}
                    onChange={(e) => setBlock(i, { text: e.target.value })}
                    rows={b.type === 'heading' ? 1 : 4}
                    placeholder={b.type === 'heading' ? 'Section heading' : 'Write here…'}
                    className="mt-2 w-full rounded-md border border-mist bg-bone p-2 text-sm text-ink"
                  />
                )}
              </div>
            ))}
          </div>

          <Button
            variant="ghost"
            size="sm"
            className="mt-3"
            onClick={() => setBlocks((prev) => [...prev, emptyBlock('paragraph')])}
          >
            <Plus aria-hidden="true" className="h-4 w-4" />
            Add block
          </Button>

          <label className="mt-5 block text-sm font-medium text-ink-soft" htmlFor="cb-tags">
            Tags <span className="font-normal text-ink-muted">(comma separated)</span>
          </label>
          <input
            id="cb-tags"
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            placeholder="odia, dal, weeknight"
            className="mt-1 w-full rounded-lg border border-mist bg-paper p-2.5 text-ink placeholder:text-ink-muted"
          />

          <p className="mt-4 text-xs text-ink-muted">
            ChefBook is for food and cooking — articles that aren&apos;t about either are declined.
          </p>

          <div className="mt-4 flex flex-wrap justify-end gap-2">
            <Button variant="ghost" onClick={reset} disabled={save.isPending}>
              Cancel
            </Button>
            <Button variant="outline" onClick={() => submit('draft')} disabled={save.isPending}>
              Save draft
            </Button>
            <Button variant="primary" onClick={() => submit('published')} isLoading={save.isPending}>
              Publish
            </Button>
          </div>
        </section>
      )}

      <section className="mt-8">
        {articles.length === 0 ? (
          <div className="rounded-xl border border-mist bg-paper p-8 text-center">
            <BookOpen aria-hidden="true" className="mx-auto h-8 w-8 text-ink-muted" />
            <p className="mt-3 text-sm text-ink-soft">
              You haven&apos;t written anything yet. Your first recipe is a good place to start.
            </p>
          </div>
        ) : (
          <ul className="space-y-3">
            {articles.map((a) => (
              <li
                key={a.id}
                className="flex items-center gap-3 rounded-lg border border-mist bg-bone p-3"
              >
                {a.cover && (
                  <img
                    src={a.cover}
                    alt=""
                    className="h-14 w-20 shrink-0 rounded object-cover"
                    loading="lazy"
                  />
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium text-ink">{a.title}</p>
                  <p className="text-xs text-ink-muted">
                    {a.status === 'published' ? 'Published' : 'Draft'} · {a.readingMinutes} min ·{' '}
                    <span className="tabular-nums">{a.reactionsTotal}</span> reactions ·{' '}
                    <span className="tabular-nums">{a.commentsCount}</span> comments
                  </p>
                </div>
                <button
                  type="button"
                  aria-label={`Delete ${a.title}`}
                  onClick={() =>
                    remove.mutate(a.id, { onSuccess: () => toast.success('Article deleted') })
                  }
                  className="shrink-0 text-ink-muted hover:text-paprika"
                >
                  <Trash2 aria-hidden="true" className="h-4 w-4" />
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
