import { useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Button } from '@/shared/components/ui';
import type { MealSlot, MealVariant, WeeklyMenu, WeeklyMenuItem } from '@/shared/types';
import {
  apiErrorMessage,
  useCreateMealPlan,
  type BookDayInput,
} from '@/features/customer/hooks/useMealPlans';

// Plan your week — the web twin of apps/mobile-customer/app/book-meal-plan.tsx.
//
// Renders the chef's published weekly menu against real calendar dates and
// submits one advance request (POST /meal-plans). The chef then confirms which
// days they can cook before anything is charged.

const SLOTS: MealSlot[] = ['lunch', 'dinner'];

/**
 * Bookable dates, mirroring the server's rule.
 *
 * mealPlanLeadTime is 12h and a booking date is read as IST midnight, so a day is
 * only offered once its midnight is beyond now+12h — which is why "today" never
 * appears and, late in the evening, tomorrow drops off too. Computing this here
 * keeps the picker from offering days the API would reject with "past the booking
 * cutoff".
 */
function bookableDates(days = 14): Date[] {
  const cutoff = Date.now() + 12 * 60 * 60 * 1000;
  const out: Date[] = [];
  for (let i = 0; i < days + 2 && out.length < days; i++) {
    const d = new Date();
    d.setHours(0, 0, 0, 0);
    d.setDate(d.getDate() + i);
    if (d.getTime() >= cutoff) out.push(d);
  }
  return out;
}

/** YYYY-MM-DD in local time — never toISOString(), which would shift the date in IST. */
function toDateKey(d: Date): string {
  const m = `${d.getMonth() + 1}`.padStart(2, '0');
  const day = `${d.getDate()}`.padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}

type Selection = Record<string, { variant: MealVariant; price: number; name: string }>;

const keyOf = (date: string, slot: MealSlot) => `${date}|${slot}`;

export default function BookMealPlanPage() {
  const { chefId } = useParams<{ chefId: string }>();
  const navigate = useNavigate();
  const fp = useFormatPrice();
  const create = useCreateMealPlan();
  const [selected, setSelected] = useState<Selection>({});

  const { data: weeklyMenu, isLoading } = useQuery({
    queryKey: ['chef', chefId, 'weekly-menu'],
    queryFn: () => apiClient.get<WeeklyMenu>(`/chefs/${chefId}/weekly-menu`),
    enabled: !!chefId,
  });

  const dates = useMemo(() => bookableDates(), []);

  /** Published cells indexed by weekday|slot|variant, matching the server's lookup. */
  const cells = useMemo(() => {
    const map = new Map<string, WeeklyMenuItem>();
    if (!weeklyMenu?.isPublished) return map;
    for (const it of weeklyMenu.items ?? []) {
      map.set(`${it.dayOfWeek}|${it.slot}|${it.variant}`, it);
    }
    return map;
  }, [weeklyMenu]);

  const picked = Object.entries(selected);
  const subtotal = picked.reduce((sum, [, v]) => sum + v.price, 0);

  const toggle = (date: string, slot: MealSlot, item: WeeklyMenuItem) => {
    const k = keyOf(date, slot);
    setSelected((prev) => {
      const next = { ...prev };
      // Re-tapping the chosen variant clears the slot; picking the other swaps it.
      if (next[k]?.variant === item.variant) delete next[k];
      else next[k] = { variant: item.variant, price: item.price, name: item.name };
      return next;
    });
  };

  const submit = async () => {
    if (!chefId || picked.length === 0) return;
    const days: BookDayInput[] = picked.map(([k, v]) => {
      const [date, slot] = k.split('|');
      return { date: date!, slot: slot as MealSlot, variant: v.variant };
    });
    try {
      await create.mutateAsync({ chefId, days });
      toast.success('Request sent — your chef will confirm the days they can cook.');
      navigate('/meal-plans');
    } catch (err) {
      // Show the server's reason verbatim. The common failure here is a 409
      // duplicate_plan ("you already have a plan with this chef for these
      // dates") — a generic "try again" tells the customer to repeat something
      // that cannot succeed, and hides which dates actually clash.
      toast.error(apiErrorMessage(err) || 'Could not send your request. Please try again.');
    }
  };

  if (isLoading) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  if (!weeklyMenu?.isPublished || (weeklyMenu.items?.length ?? 0) === 0) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-16 text-center">
        <p className="font-medium text-ink">This chef hasn&apos;t published a weekly menu yet.</p>
        <Button asChild variant="outline" className="mt-4">
          <Link to={`/chefs/${chefId}`}>Back to the kitchen</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8 pb-32">
      <Link
        to={`/chefs/${chefId}`}
        className="inline-flex items-center gap-1.5 text-sm text-ink-soft hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to the kitchen
      </Link>

      <h1 className="mt-4 text-2xl font-semibold tracking-tight text-ink">Plan your week</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Pick the meals you want over the next two weeks. Choose veg or non-veg for each — send your
        plan and the chef confirms the days they can cook.
      </p>

      <div className="mt-6 flex flex-col gap-6">
        {dates.map((date) => {
          const dateKey = toDateKey(date);
          const weekday = date.getDay();
          const slotsWithFood = SLOTS.filter((slot) =>
            (['veg', 'nonveg'] as MealVariant[]).some((v) => cells.has(`${weekday}|${slot}|${v}`)),
          );
          if (slotsWithFood.length === 0) return null;

          return (
            <section key={dateKey}>
              <h2 className="font-semibold text-ink">
                {date.toLocaleDateString('en-IN', {
                  weekday: 'short',
                  day: 'numeric',
                  month: 'short',
                })}
              </h2>
              <div className="mt-2 divide-y divide-mist rounded-lg border border-mist bg-bone">
                {slotsWithFood.map((slot) => {
                  const options = (['nonveg', 'veg'] as MealVariant[])
                    .map((v) => cells.get(`${weekday}|${slot}|${v}`))
                    .filter((x): x is WeeklyMenuItem => !!x);
                  const chosen = selected[keyOf(dateKey, slot)];
                  return (
                    <div key={slot}>
                      <div className="flex items-center justify-between px-4 pt-3">
                        <span className="text-xs uppercase tracking-wide text-ink-muted">
                          {slot}
                        </span>
                        <span className="text-xs text-ink-muted">pick one</span>
                      </div>
                      {options.map((item) => {
                        const on = chosen?.variant === item.variant;
                        return (
                          <button
                            key={item.variant}
                            type="button"
                            onClick={() => toggle(dateKey, slot, item)}
                            aria-pressed={on}
                            className={`flex w-full items-center gap-3 px-4 py-3 text-left transition-colors ${
                              on ? 'bg-herb/10' : 'hover:bg-mist/50'
                            }`}
                          >
                            <span
                              className={`h-3 w-3 shrink-0 rounded-sm border-2 ${
                                item.variant === 'veg'
                                  ? 'border-emerald-600 bg-emerald-600'
                                  : 'border-destructive bg-destructive'
                              }`}
                              aria-hidden="true"
                            />
                            <span className="min-w-0 flex-1 font-medium text-ink">{item.name}</span>
                            <span className="tabular-nums text-ink">{fp(item.price)}</span>
                            <span
                              className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2 ${
                                on ? 'border-herb bg-herb text-white' : 'border-mist'
                              }`}
                              aria-hidden="true"
                            >
                              {on ? '✓' : ''}
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  );
                })}
              </div>
            </section>
          );
        })}
      </div>

      <div className="fixed inset-x-0 bottom-0 border-t border-mist bg-paper/95 p-4 backdrop-blur">
        <div className="mx-auto flex max-w-2xl items-center justify-between gap-4">
          <div>
            <p className="text-sm text-ink">
              {picked.length} {picked.length === 1 ? 'meal' : 'meals'} · food subtotal
            </p>
            <p className="text-xs text-ink-muted">GST &amp; delivery shown before you pay</p>
          </div>
          <span className="text-xl font-semibold tabular-nums text-ink">{fp(subtotal)}</span>
        </div>
        <div className="mx-auto mt-3 max-w-2xl">
          <Button
            className="w-full"
            size="lg"
            disabled={picked.length === 0 || create.isPending}
            onClick={() => void submit()}
          >
            {picked.length === 0
              ? 'Select meals to continue'
              : create.isPending
                ? 'Sending…'
                : `Request ${picked.length} ${picked.length === 1 ? 'meal' : 'meals'}`}
          </Button>
        </div>
      </div>
    </div>
  );
}
