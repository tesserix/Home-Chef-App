import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { CalendarRange, Loader2 } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

interface MealPlanDay {
  date: string;
  slot: 'lunch' | 'dinner';
  variant: 'veg' | 'nonveg';
  status: string;
  dishName: string;
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
  days: MealPlanDay[];
  customer?: { firstName?: string; lastName?: string; email?: string } | null;
  chef?: { businessName?: string } | null;
}

interface MealPlansResponse {
  data: MealPlan[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
  };
}

const STATUS_OPTIONS = [
  'all',
  'pending_chef',
  'awaiting_customer',
  'confirmed',
  'active',
  'completed',
  'cancelled',
  'expired',
];

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

function statusLabel(s: string): string {
  return s.replace(/_/g, ' ');
}

function fmtDate(iso?: string): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
  });
}

function customerName(p: MealPlan): string {
  const c = p.customer;
  const full = `${c?.firstName ?? ''} ${c?.lastName ?? ''}`.trim();
  return full || c?.email || '—';
}

export default function MealPlansPage() {
  const navigate = useNavigate();
  const [statusFilter, setStatusFilter] = useState('all');
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-meal-plans', statusFilter, page],
    queryFn: () =>
      apiClient.get<MealPlansResponse>('/admin/meal-plans', {
        status: statusFilter !== 'all' ? statusFilter : undefined,
        page,
        limit: 20,
      }),
  });

  const resp = data as unknown as MealPlansResponse | undefined;
  const plans = resp?.data ?? [];
  const pagination = resp?.pagination;

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Meal Plans</h1>
        <p className="page-description">
          Tiffin meal-plan subscriptions across the platform — oversight only.
        </p>
      </div>

      {/* Status filter */}
      <div className="flex flex-wrap gap-2">
        {STATUS_OPTIONS.map((s) => (
          <button
            key={s}
            onClick={() => {
              setStatusFilter(s);
              setPage(1);
            }}
            className={`rounded-full px-3 py-1 text-sm font-medium capitalize transition-colors ${
              statusFilter === s
                ? 'bg-sidebar-primary text-primary-foreground'
                : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
            }`}
          >
            {s === 'all' ? 'All' : statusLabel(s)}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : plans.length === 0 ? (
        <div className="rounded-xl border border-border bg-card p-20 text-center">
          <CalendarRange className="mx-auto h-10 w-10 text-muted-foreground" />
          <p className="mt-4 text-muted-foreground">No meal plans found</p>
        </div>
      ) : (
        <div className="rounded-xl border border-border bg-card shadow-card">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-muted/50">
                  <Th>Plan #</Th>
                  <Th>Customer</Th>
                  <Th>Chef</Th>
                  <Th>Dates</Th>
                  <Th>Days</Th>
                  <Th>Total</Th>
                  <Th>Status</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {plans.map((p) => (
                  <tr
                    key={p.id}
                    onClick={() => navigate(`/meal-plans/${p.id}`)}
                    className="cursor-pointer transition-colors hover:bg-secondary/30"
                  >
                    <td className="px-6 py-4 text-sm font-medium text-foreground">
                      {p.mealPlanNumber}
                    </td>
                    <td className="px-6 py-4 text-sm text-foreground">
                      {customerName(p)}
                    </td>
                    <td className="px-6 py-4 text-sm text-muted-foreground">
                      {p.chef?.businessName ?? '—'}
                    </td>
                    <td className="px-6 py-4 text-sm text-muted-foreground">
                      {fmtDate(p.startDate)} – {fmtDate(p.endDate)}
                    </td>
                    <td className="px-6 py-4 text-sm text-muted-foreground">
                      {p.days?.length ?? 0}
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-foreground">
                      ₹{p.total.toFixed(0)}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${
                          statusStyles[p.status] ?? 'bg-muted text-muted-foreground'
                        }`}
                      >
                        {statusLabel(p.status)}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {pagination && pagination.totalPages > 1 ? (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Page {pagination.page} of {pagination.totalPages} · {pagination.total}{' '}
            total
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={!pagination.hasNext}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function Th({ children }: { children: React.ReactNode }) {
  return (
    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
      {children}
    </th>
  );
}
