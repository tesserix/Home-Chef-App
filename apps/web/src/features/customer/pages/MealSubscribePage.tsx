import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { toast } from 'sonner';
import { Loader2 } from 'lucide-react';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Button } from '@/shared/components/ui';
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
  const [cadence, setCadence] = useState('weekly');
  const [price, setPrice] = useState<number | null>(null);

  useEffect(() => {
    if (offer?.available) {
      const firstSlot = offer.slots?.[0];
      if (slots.length === 0 && firstSlot) setSlots([firstSlot]);
      const firstCadence = offer.cadences?.[0];
      if (firstCadence && offer.cadences && !offer.cadences.includes(cadence)) setCadence(firstCadence);
    }
  }, [offer]); // eslint-disable-line react-hooks/exhaustive-deps

  const valid = slots.length > 0 && days.length > 0 && !!cadence && !!chefId;

  useEffect(() => {
    if (!valid) { setPrice(null); return; }
    preview.mutate(
      { chefId: chefId!, slots, days, variant, cadence },
      {
        onSuccess: (r) => setPrice((r as unknown as { cycleAmount: number }).cycleAmount),
        onError: () => setPrice(null),
      },
    );
  }, [slots, days, variant, cadence]); // eslint-disable-line react-hooks/exhaustive-deps

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
          <p className="mb-2 text-sm font-medium text-ink-soft">Preference</p>
          <div className="flex gap-2">
            <button type="button" className={chip(variant === 'veg')} onClick={() => setVariant('veg')}>Veg</button>
            <button type="button" className={chip(variant === 'nonveg')} onClick={() => setVariant('nonveg')}>Non-veg</button>
          </div>
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
              { chefId: chefId!, slots, days, variant, cadence },
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
