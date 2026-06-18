import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

interface MealPlanDay {
  id: string;
  date: string;
  slot: 'lunch' | 'dinner';
  variant: 'veg' | 'nonveg';
  status: string;
  dishName?: string;
  price: number;
}

interface MealPlan {
  id: string;
  mealPlanNumber: string;
  status: string;
  startDate: string;
  endDate: string;
  subtotal: number;
  total: number;
  currency?: string;
  days: MealPlanDay[];
  customer?: { firstName?: string; lastName?: string; email?: string } | null;
  chef?: { businessName?: string } | null;
}

interface MealPlanResponse {
  mealPlan: MealPlan;
}

const statusStyles: Record<string, string> = {
  pending_chef: 'bg-warning/10 text-warning',
  awaiting_customer: 'bg-info/10 text-info',
  chef_modified: 'bg-info/10 text-info',
  confirmed: 'bg-success/10 text-success',
  chef_accepted_full: 'bg-success/10 text-success',
  active: 'bg-success/10 text-success',
  completed: 'bg-muted text-muted-foreground',
  cancelled: 'bg-destructive/10 text-destructive',
  expired: 'bg-destructive/10 text-destructive',
};

const declinedDay = new Set(['declined', 'skipped', 'cancelled', 'refunded']);

function statusLabel(s: string): string {
  return s.replace(/_/g, ' ');
}

function fmtDay(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

export default function MealPlanDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-meal-plan', id],
    queryFn: () => apiClient.get<MealPlanResponse>(`/admin/meal-plans/${id}`),
    enabled: !!id,
  });

  const resp = data as unknown as MealPlanResponse | undefined;
  const plan = resp?.mealPlan;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!plan) {
    return (
      <div className="py-20 text-center">
        <p className="text-muted-foreground">Meal plan not found</p>
        <Button
          variant="link"
          size="sm"
          className="mt-4"
          onClick={() => navigate('/meal-plans')}
        >
          Back to Meal Plans
        </Button>
      </div>
    );
  }

  const customer = plan.customer;
  const customerName =
    `${customer?.firstName ?? ''} ${customer?.lastName ?? ''}`.trim() ||
    customer?.email ||
    '—';

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => navigate('/meal-plans')}
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div className="flex-1">
          <h1 className="font-display text-2xl font-semibold">
            {plan.mealPlanNumber}
          </h1>
          <p className="text-sm text-muted-foreground">
            {customerName} · {plan.chef?.businessName ?? '—'}
          </p>
        </div>
        <span
          className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium capitalize ${
            statusStyles[plan.status] ?? 'bg-muted text-muted-foreground'
          }`}
        >
          {statusLabel(plan.status)}
        </span>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Days */}
        <div className="space-y-6 lg:col-span-2">
          <div className="rounded-xl border border-border bg-card shadow-card">
            <div className="border-b border-border px-6 py-4">
              <h2 className="font-medium text-foreground">
                Days ({plan.days?.length ?? 0})
              </h2>
            </div>
            <div className="divide-y divide-border">
              {(plan.days ?? []).map((d) => {
                const declined = declinedDay.has(d.status);
                return (
                  <div
                    key={d.id}
                    className="flex items-center justify-between px-6 py-3"
                  >
                    <div className="flex items-center gap-3">
                      <span
                        className={`inline-block h-2.5 w-2.5 rounded-sm ${
                          d.variant === 'veg' ? 'bg-success' : 'bg-destructive'
                        }`}
                        aria-hidden
                      />
                      <div>
                        <p
                          className={`text-sm font-medium ${
                            declined
                              ? 'text-muted-foreground line-through'
                              : 'text-foreground'
                          }`}
                        >
                          {fmtDay(d.date)}
                        </p>
                        <p className="text-xs capitalize text-muted-foreground">
                          {d.slot} · {d.variant === 'veg' ? 'Veg' : 'Non-veg'} ·{' '}
                          {d.dishName ?? '—'}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-medium text-foreground">
                        ₹{(d.price ?? 0).toFixed(0)}
                      </p>
                      <p className="text-xs capitalize text-muted-foreground">
                        {statusLabel(d.status)}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* Summary sidebar */}
        <div className="space-y-6">
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="mb-4 font-medium text-foreground">Summary</h2>
            <dl className="space-y-3 text-sm">
              <Row label="Period">
                {fmtDay(plan.startDate)} – {fmtDay(plan.endDate)}
              </Row>
              <Row label="Subtotal">₹{plan.subtotal.toFixed(0)}</Row>
              <Row label="Total">
                <span className="font-semibold">₹{plan.total.toFixed(0)}</span>
              </Row>
            </dl>
          </div>

          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="mb-4 font-medium text-foreground">Parties</h2>
            <dl className="space-y-3 text-sm">
              <Row label="Customer">{customerName}</Row>
              {customer?.email ? <Row label="Email">{customer.email}</Row> : null}
              <Row label="Chef">{plan.chef?.businessName ?? '—'}</Row>
            </dl>
          </div>
        </div>
      </div>
    </div>
  );
}

function Row({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="text-right text-foreground">{children}</dd>
    </div>
  );
}
