import { useMemo, useState } from 'react';
import { toast } from 'sonner';
import { addDays, format } from 'date-fns';
import { Plus, Trash2 } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import {
  useDailyMenu,
  useSaveDailyMenu,
  type DailyMenuItem,
  type MealSlot,
  type MealVariant,
} from '../hooks/useDailyMenu';

// Per-date tiffin menu — the web twin of
// apps/mobile-vendor/app/meal-plans/daily-menu.tsx.
//
// Different from the weekly menu: the weekly one is the repeating template, this
// is what is actually served on a given date. A chef publishing per-date menus
// from the phone previously could not see or change any of it on the portal.

const HORIZON_DAYS = 14;
const SLOTS: MealSlot[] = ['lunch', 'dinner'];

function emptyItem(slot: MealSlot): DailyMenuItem {
  return { slot, variant: 'veg', name: '', description: '', price: 0 };
}

export function DailyMenuPage() {
  const today = useMemo(() => new Date(), []);
  const from = format(today, 'yyyy-MM-dd');
  const to = format(addDays(today, HORIZON_DAYS), 'yyyy-MM-dd');

  const { data: days = [], isLoading } = useDailyMenu(from, to);
  const [selected, setSelected] = useState(from);

  // The date list is generated rather than taken from the response: a date with
  // no menu yet has no row on the server, and those are precisely the dates a
  // chef needs to be able to click on.
  const dates = useMemo(
    () => Array.from({ length: HORIZON_DAYS + 1 }, (_, i) => format(addDays(today, i), 'yyyy-MM-dd')),
    [today],
  );

  const day = days.find((d) => d.date === selected);

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Daily menu</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Different dishes each day, plus any combo or thali. Customers can pre-book up to{' '}
        {HORIZON_DAYS} days ahead.
      </p>

      {/* Date strip */}
      <div className="mt-6 flex gap-2 overflow-x-auto pb-2">
        {dates.map((d) => {
          const entry = days.find((x) => x.date === d);
          const isSel = d === selected;
          return (
            <button
              key={d}
              type="button"
              onClick={() => setSelected(d)}
              aria-pressed={isSel}
              className={`flex min-w-[4.5rem] shrink-0 flex-col items-center rounded-lg border px-3 py-2 transition-colors ${
                isSel ? 'border-herb bg-herb-tint' : 'border-mist hover:bg-paper'
              }`}
            >
              <span className="text-xs text-ink-muted">{format(new Date(d), 'EEE')}</span>
              <span className="text-sm font-semibold text-ink tabular-nums">
                {format(new Date(d), 'd MMM')}
              </span>
              {/* A dot rather than a label: the strip has to stay scannable, and
                  "published vs draft" is detail for the selected day below. */}
              {entry && entry.items.length > 0 && (
                <span
                  className={`mt-1 h-1.5 w-1.5 rounded-full ${
                    entry.isPublished ? 'bg-herb' : 'bg-amber'
                  }`}
                  aria-hidden="true"
                />
              )}
            </button>
          );
        })}
      </div>

      {isLoading ? (
        <Skeleton className="mt-6 h-64 w-full" />
      ) : (
        <DayEditor key={selected} date={selected} initial={day} />
      )}
    </div>
  );
}

function DayEditor({
  date,
  initial,
}: {
  date: string;
  initial?: { isPublished: boolean; items: DailyMenuItem[] };
}) {
  const save = useSaveDailyMenu();
  const [items, setItems] = useState<DailyMenuItem[]>(initial?.items ?? []);
  const [isPublished, setIsPublished] = useState(initial?.isPublished ?? false);

  function update(i: number, patch: Partial<DailyMenuItem>) {
    setItems((cur) => cur.map((it, idx) => (idx === i ? { ...it, ...patch } : it)));
  }

  function submit(publish: boolean) {
    // A published day with no dishes would show customers an empty menu they can
    // still try to book against, so it is refused here rather than server-side.
    if (publish && items.length === 0) {
      toast.error('Add at least one dish before publishing this day.');
      return;
    }
    const cleaned = items.filter((i) => i.name.trim() !== '');
    save.mutate(
      { date, isPublished: publish, items: cleaned },
      {
        onSuccess: () => {
          setIsPublished(publish);
          toast.success(publish ? 'Menu published.' : 'Draft saved.');
        },
        onError: () => toast.error('Could not save this day. Please try again.'),
      },
    );
  }

  return (
    <Card className="mt-6 p-5">
      <div className="flex items-center justify-between gap-3">
        <h2 className="font-semibold text-foreground">
          {format(new Date(date), 'EEEE d MMMM')}
        </h2>
        <Badge variant={isPublished ? 'success' : 'warning'}>
          {isPublished ? 'Published' : 'Draft'}
        </Badge>
      </div>

      {SLOTS.map((slot) => {
        const slotItems = items
          .map((it, idx) => ({ it, idx }))
          .filter(({ it }) => it.slot === slot);
        return (
          <section key={slot} className="mt-5">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                {slot}
              </h3>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setItems((cur) => [...cur, emptyItem(slot)])}
              >
                <Plus className="mr-1 h-4 w-4" aria-hidden="true" />
                Add dish
              </Button>
            </div>

            {slotItems.length === 0 ? (
              <p className="mt-2 text-sm text-ink-muted">Nothing set for {slot} yet.</p>
            ) : (
              <div className="mt-2 flex flex-col gap-3">
                {slotItems.map(({ it, idx }) => (
                  <div key={idx} className="rounded-lg border border-mist p-3">
                    <div className="flex gap-2">
                      <input
                        value={it.name}
                        onChange={(e) => update(idx, { name: e.target.value })}
                        placeholder="Dish name"
                        aria-label={`${slot} dish name`}
                        className="input-base flex-1"
                      />
                      <input
                        type="number"
                        min={0}
                        inputMode="numeric"
                        value={it.price}
                        onChange={(e) => update(idx, { price: Number(e.target.value) })}
                        aria-label={`${slot} dish price in rupees`}
                        className="input-base w-24 tabular-nums"
                      />
                      <button
                        type="button"
                        onClick={() => setItems((cur) => cur.filter((_, i2) => i2 !== idx))}
                        aria-label={`Remove ${it.name || 'dish'}`}
                        className="rounded-lg px-2 text-ink-muted transition-colors hover:bg-paprika-tint hover:text-paprika"
                      >
                        <Trash2 className="h-4 w-4" aria-hidden="true" />
                      </button>
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      {(['veg', 'nonveg'] as MealVariant[]).map((v) => (
                        <button
                          key={v}
                          type="button"
                          onClick={() => update(idx, { variant: v })}
                          aria-pressed={it.variant === v}
                          className={`rounded-lg border px-2.5 py-1 text-xs transition-colors ${
                            it.variant === v
                              ? 'border-herb bg-herb-tint text-herb'
                              : 'border-mist text-ink-soft hover:bg-paper'
                          }`}
                        >
                          {v === 'veg' ? 'Veg' : 'Non-veg'}
                        </button>
                      ))}
                      <label className="flex items-center gap-1.5 text-xs text-ink-soft">
                        <input
                          type="checkbox"
                          checked={!!it.isCombo}
                          onChange={(e) => update(idx, { isCombo: e.target.checked })}
                          className="h-3.5 w-3.5"
                        />
                        Combo / thali
                      </label>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>
        );
      })}

      <div className="mt-6 flex gap-2">
        <Button onClick={() => submit(true)} isLoading={save.isPending}>
          Publish
        </Button>
        <Button variant="secondary" onClick={() => submit(false)} disabled={save.isPending}>
          Save draft
        </Button>
      </div>
    </Card>
  );
}

export default DailyMenuPage;
