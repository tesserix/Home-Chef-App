import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Loader2 } from 'lucide-react';
import { useFormatPrice } from '@/shared/utils/format-price';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
import type { WeeklyMenu, WeeklyMenuItem } from '@/shared/types';
import {
  useMealChefOffer,
  usePreviewMealPrice,
  useSubscribeMeal,
} from '@/features/customer/hooks/useMealSubscription';

// Configure + subscribe to a chef's daily tiffin (#283, web). Live price preview;
// subscribe sets it up (the Razorpay UPI-Autopay mandate is the billing phase).

const DAYS = [
  { v: 1, l: 'Mon' }, { v: 2, l: 'Tue' }, { v: 3, l: 'Wed' }, { v: 4, l: 'Thu' },
  { v: 5, l: 'Fri' }, { v: 6, l: 'Sat' }, { v: 0, l: 'Sun' },
];

export default function MealSubscribePage() {
  const { id: chefId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const fp = useFormatPrice();
  const { data: offerRes, isLoading } = useMealChefOffer(chefId);
  const offer = offerRes as unknown as { available: boolean; slots?: string[]; cadences?: string[]; deliveryFee?: number } | undefined;
  const preview = usePreviewMealPrice();
  const subscribe = useSubscribeMeal();

  const [slots, setSlots] = useState<string[]>([]);
  const [days, setDays] = useState<number[]>([1, 2, 3, 4, 5]);
  const [variant, setVariant] = useState<'veg' | 'nonveg'>('veg');
  // Per-day overrides. Absent day => the plan default above applies, which is
  // exactly how the API resolves it (MealSubscription.VariantForDay).
  const [dayVariants, setDayVariants] = useState<Record<number, 'veg' | 'nonveg'>>({});
  const [cadence, setCadence] = useState('weekly');
  const [price, setPrice] = useState<number | null>(null);

  // The chef's published weekly menu, so the configurator can show the actual
  // dishes for the selected days rather than asking for a blind commitment.
  const { data: weeklyMenu } = useQuery({
    queryKey: ['chef', chefId, 'weekly-menu'],
    queryFn: () => apiClient.get<WeeklyMenu>(`/chefs/${chefId}/weekly-menu`),
    enabled: !!chefId,
  });
  const weeklyCells: WeeklyMenuItem[] = weeklyMenu?.isPublished ? (weeklyMenu.items ?? []) : [];

  useEffect(() => {
    if (offer?.available) {
      const firstSlot = offer.slots?.[0];
      if (slots.length === 0 && firstSlot) setSlots([firstSlot]);
      const firstCadence = offer.cadences?.[0];
      if (firstCadence && offer.cadences && !offer.cadences.includes(cadence)) setCadence(firstCadence);
    }
  }, [offer]); // eslint-disable-line react-hooks/exhaustive-deps

  const variantFor = (day: number): 'veg' | 'nonveg' => dayVariants[day] ?? variant;
  // Only send genuine overrides — a day matching the plan default is noise.
  const dayVariantsPayload = Object.fromEntries(
    days
      .filter((d) => dayVariants[d] && dayVariants[d] !== variant)
      .map((d) => [String(d), dayVariants[d] as string]),
  );

  const valid = slots.length > 0 && days.length > 0 && !!cadence && !!chefId;

  useEffect(() => {
    if (!valid) { setPrice(null); return; }
    preview.mutate(
      { chefId: chefId!, slots, days, variant, dayVariants: dayVariantsPayload, cadence },
      {
        onSuccess: (r) => setPrice((r as unknown as { cycleAmount: number }).cycleAmount),
        onError: () => setPrice(null),
      },
    );
  }, [slots, days, variant, dayVariants, cadence]); // eslint-disable-line react-hooks/exhaustive-deps

  const toggle = (list: number[] | string[], v: never, set: (x: never[]) => void) =>
    set((list.includes(v) ? (list as never[]).filter((x) => x !== v) : [...(list as never[]), v]) as never[]);

  const chip = (active: boolean) =>
    `rounded-full border px-4 py-1.5 text-sm transition-colors ${active ? 'border-herb bg-herb-tint text-herb' : 'border-mist bg-paper text-ink-soft hover:border-ink-soft'}`;

  if (isLoading) {
    return <div className="flex min-h-[50vh] items-center justify-center"><Loader2 className="h-8 w-8 animate-spin text-herb" /></div>;
  }
  if (!offer?.available) {
    return <div className="mx-auto max-w-2xl px-4 py-16 text-center text-ink-soft">This chef doesn’t offer a tiffin subscription yet.</div>;
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-10">
      <h1 className="font-display text-2xl font-semibold text-ink">Daily tiffin subscription</h1>

      <div className="mt-6 space-y-5">
        <div>
          <p className="mb-2 text-sm font-medium text-ink-soft">Meals</p>
          <div className="flex flex-wrap gap-2">
            {(offer.slots ?? ['lunch', 'dinner']).map((s) => (
              <button key={s} type="button" className={chip(slots.includes(s))} onClick={() => toggle(slots as string[], s as never, setSlots as never)}>
                {s === 'lunch' ? 'Lunch' : 'Dinner'}
              </button>
            ))}
          </div>
        </div>

        <div>
          <p className="mb-2 text-sm font-medium text-ink-soft">Days</p>
          <div className="flex flex-wrap gap-2">
            {DAYS.map((d) => (
              <button key={d.v} type="button" className={chip(days.includes(d.v))} onClick={() => toggle(days, d.v as never, setDays as never)}>
                {d.l}
              </button>
            ))}
          </div>
        </div>

        <div>
          <p className="mb-2 text-sm font-medium text-ink-soft">Preference (applies to every day unless you change a day below)</p>
          <div className="flex gap-2">
            <button type="button" className={chip(variant === 'veg')} onClick={() => setVariant('veg')}>Veg</button>
            <button type="button" className={chip(variant === 'nonveg')} onClick={() => setVariant('nonveg')}>Non-veg</button>
          </div>
        </div>

        {/* What you'll actually receive.
            The configurator previously asked for days, slots and a preference
            without ever showing the food — you committed to a recurring plan
            without seeing a single dish. This reflects the chef's published
            weekly menu back against the exact selection above, so the choice is
            made against real meals. */}
        <div>
          <p className="mb-2 text-sm font-medium text-ink-soft">What you&apos;ll get</p>
          {weeklyCells.length === 0 ? (
            <p className="rounded-lg border border-mist bg-paper p-3 text-sm text-ink-soft">
              This chef hasn&apos;t published a weekly menu yet, so the daily dishes
              aren&apos;t listed. Your subscription still runs — the chef cooks that
              day&apos;s meal.
            </p>
          ) : days.length === 0 || slots.length === 0 ? (
            <p className="rounded-lg border border-mist bg-paper p-3 text-sm text-ink-soft">
              Pick at least one day and one meal to see the dishes.
            </p>
          ) : (
            <ul className="space-y-2">
              {DAYS.filter((d) => days.includes(d.v)).map((d) => {
                const dayVariant = variantFor(d.v);
                const forDay = weeklyCells.filter(
                  (c) => c.dayOfWeek === d.v && slots.includes(c.slot) && c.variant === dayVariant,
                );
                return (
                  <li key={d.v} className="rounded-lg border border-mist bg-paper p-3">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-semibold text-ink">{d.l}</p>
                      <div className="flex shrink-0 gap-1" role="group" aria-label={`${d.l} preference`}>
                        {(['veg', 'nonveg'] as const).map((v) => (
                          <button
                            key={v}
                            type="button"
                            aria-pressed={dayVariant === v}
                            onClick={() =>
                              setDayVariants((prev) => ({ ...prev, [d.v]: v }))
                            }
                            className={`rounded-full border px-2.5 py-0.5 text-xs transition-colors ${
                              dayVariant === v
                                ? 'border-herb bg-herb-tint text-herb'
                                : 'border-mist bg-paper text-ink-soft hover:border-ink-soft'
                            }`}
                          >
                            {v === 'veg' ? 'Veg' : 'Non-veg'}
                          </button>
                        ))}
                      </div>
                    </div>
                    {forDay.length === 0 ? (
                      <p className="mt-1 text-sm text-ink-soft">
                        No {dayVariant === 'veg' ? 'veg' : 'non-veg'} dish listed for this day.
                      </p>
                    ) : (
                      <ul className="mt-1 space-y-1">
                        {forDay.map((c, i) => (
                          <li key={c.id ?? `${d.v}-${c.slot}-${i}`} className="flex items-baseline justify-between gap-3">
                            <span className="text-sm text-ink">
                              <span className="text-ink-soft">
                                {c.slot === 'lunch' ? 'Lunch' : 'Dinner'}
                              </span>
                              {' · '}
                              {c.name}
                            </span>
                            {c.price > 0 ? (
                              <span className="shrink-0 text-sm tabular-nums text-ink-soft">{fp(c.price)}</span>
                            ) : null}
                          </li>
                        ))}
                      </ul>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <div>
          <p className="mb-2 text-sm font-medium text-ink-soft">Plan</p>
          <div className="flex gap-2">
            {(offer.cadences ?? ['weekly', 'monthly']).map((c) => (
              <button key={c} type="button" className={chip(cadence === c)} onClick={() => setCadence(c)}>
                {c === 'weekly' ? 'Weekly' : 'Monthly'}
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center justify-between rounded-xl bg-bone p-4">
          <span className="text-sm text-ink-soft">{cadence === 'monthly' ? 'Per month' : 'Per week'}</span>
          <span className="font-display text-xl font-semibold text-ink">{price == null ? '—' : fp(price)}</span>
        </div>

        <Button
          variant="primary"
          fullWidth
          isLoading={subscribe.isPending}
          disabled={!valid || subscribe.isPending}
          onClick={() =>
            subscribe.mutate(
              { chefId: chefId!, slots, days, variant, dayVariants: dayVariantsPayload, cadence },
              {
                onSuccess: () => { toast.success('Subscription created'); navigate('/subscriptions'); },
                onError: (e: unknown) => toast.error(e instanceof Error ? e.message : 'Could not subscribe'),
              },
            )
          }
        >
          Subscribe{price != null ? ` · ${fp(price)}` : ''}
        </Button>
      </div>
    </div>
  );
}
