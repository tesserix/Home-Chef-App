import { useMemo, useState } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Loader2, Star } from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
import type { Order, Review } from '@/shared/types';

/** A single clickable 1–5 star row. */
function StarRow({
  label,
  value,
  onChange,
  required,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  required?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <span className="text-sm text-foreground">
        {label}
        {required && <span className="ml-0.5 text-destructive">*</span>}
      </span>
      <div className="flex items-center gap-1" role="radiogroup" aria-label={label}>
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            type="button"
            role="radio"
            aria-checked={value === n}
            aria-label={`${n} star${n > 1 ? 's' : ''}`}
            onClick={() => onChange(n)}
            className="rounded-md p-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
          >
            <Star
              className={
                n <= value ? 'h-7 w-7 fill-herb text-herb' : 'h-7 w-7 text-muted-foreground/40'
              }
              aria-hidden="true"
            />
          </button>
        ))}
      </div>
    </div>
  );
}

export default function ReviewPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: order, isLoading } = useQuery({
    queryKey: ['order', id],
    queryFn: () => apiClient.get<Order>(`/orders/${id}`),
    enabled: !!id,
  });

  const [overall, setOverall] = useState(0);
  const [food, setFood] = useState(0);
  const [delivery, setDelivery] = useState(0);
  const [value, setValue] = useState(0);
  const [packaging, setPackaging] = useState(0);
  const [hygiene, setHygiene] = useState(0);
  const [dishRatings, setDishRatings] = useState<Record<string, number>>({});
  const [title, setTitle] = useState('');
  const [comment, setComment] = useState('');

  // One rating row per distinct dish in the order (#145).
  const dishes = useMemo(() => {
    const seen = new Map<string, string>();
    for (const it of order?.items ?? []) {
      if (!seen.has(it.menuItemId)) seen.set(it.menuItemId, it.name);
    }
    return Array.from(seen, ([menuItemId, name]) => ({ menuItemId, name }));
  }, [order]);

  const submit = useMutation({
    mutationFn: async () => {
      const fd = new FormData();
      fd.append('orderId', id!);
      fd.append('overallRating', String(overall));
      if (food) fd.append('foodRating', String(food));
      if (delivery) fd.append('deliveryRating', String(delivery));
      if (value) fd.append('valueRating', String(value));
      if (packaging) fd.append('packagingRating', String(packaging));
      if (hygiene) fd.append('hygieneRating', String(hygiene));
      if (title.trim()) fd.append('title', title.trim());
      if (comment.trim()) fd.append('comment', comment.trim());
      const dishPayload = Object.entries(dishRatings)
        .filter(([, r]) => r >= 1 && r <= 5)
        .map(([menuItemId, rating]) => ({ menuItemId, rating }));
      if (dishPayload.length) fd.append('dishRatings', JSON.stringify(dishPayload));
      return apiClient.upload<Review>('/reviews', fd);
    },
    onSuccess: () => {
      toast.success('Thanks for your review!');
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      if (order?.chefId) {
        queryClient.invalidateQueries({ queryKey: ['chef-reviews', order.chefId] });
        queryClient.invalidateQueries({ queryKey: ['chef', order.chefId] });
      }
      navigate(`/orders/${id}`);
    },
    onError: (e: unknown) => {
      toast.error(e instanceof Error ? e.message : 'Could not submit your review');
    },
  });

  if (isLoading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" aria-hidden="true" />
      </div>
    );
  }

  if (!order) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-12 text-center">
        <p className="text-muted-foreground">We couldn’t find that order.</p>
        <Button asChild variant="outline" className="mt-4">
          <Link to="/orders">Back to orders</Link>
        </Button>
      </div>
    );
  }

  if (order.status !== 'delivered') {
    return (
      <div className="mx-auto max-w-2xl px-4 py-12 text-center">
        <p className="text-muted-foreground">You can review this order once it’s delivered.</p>
        <Button asChild variant="outline" className="mt-4">
          <Link to={`/orders/${id}`}>Back to order</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <Link
        to={`/orders/${id}`}
        className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to order
      </Link>

      <h1 className="text-2xl font-semibold text-foreground">Leave a review</h1>
      <p className="mt-1 text-sm text-muted-foreground">
        Tell others about your order from {order.chef?.businessName ?? 'this chef'}.
      </p>

      <form
        className="mt-8 space-y-6"
        onSubmit={(e) => {
          e.preventDefault();
          if (overall < 1) {
            toast.error('Please give an overall rating');
            return;
          }
          submit.mutate();
        }}
      >
        <section className="rounded-2xl border border-border p-5">
          <h2 className="mb-2 text-sm font-medium text-foreground">Your ratings</h2>
          <div className="divide-y divide-border">
            <StarRow label="Overall" value={overall} onChange={setOverall} required />
            <StarRow label="Food quality" value={food} onChange={setFood} />
            <StarRow label="Delivery" value={delivery} onChange={setDelivery} />
            <StarRow label="Value for money" value={value} onChange={setValue} />
            <StarRow label="Packaging" value={packaging} onChange={setPackaging} />
            <StarRow label="Hygiene" value={hygiene} onChange={setHygiene} />
          </div>
        </section>

        {dishes.length > 0 && (
          <section className="rounded-2xl border border-border p-5">
            <h2 className="mb-2 text-sm font-medium text-foreground">Rate the dishes</h2>
            <div className="divide-y divide-border">
              {dishes.map((d) => (
                <StarRow
                  key={d.menuItemId}
                  label={d.name}
                  value={dishRatings[d.menuItemId] ?? 0}
                  onChange={(v) => setDishRatings((prev) => ({ ...prev, [d.menuItemId]: v }))}
                />
              ))}
            </div>
          </section>
        )}

        <section className="space-y-3 rounded-2xl border border-border p-5">
          <div>
            <label htmlFor="review-title" className="mb-1 block text-sm font-medium text-foreground">
              Title <span className="text-muted-foreground">(optional)</span>
            </label>
            <input
              id="review-title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={120}
              placeholder="Sum it up in a few words"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb"
            />
          </div>
          <div>
            <label htmlFor="review-comment" className="mb-1 block text-sm font-medium text-foreground">
              Comment <span className="text-muted-foreground">(optional)</span>
            </label>
            <textarea
              id="review-comment"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={4}
              maxLength={1000}
              placeholder="What did you enjoy? Anything the chef could improve?"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb"
            />
          </div>
        </section>

        <div className="flex justify-end gap-3">
          <Button asChild variant="outline" type="button">
            <Link to={`/orders/${id}`}>Cancel</Link>
          </Button>
          <Button type="submit" disabled={submit.isPending || overall < 1}>
            {submit.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />}
            Submit review
          </Button>
        </div>
      </form>
    </div>
  );
}
