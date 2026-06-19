import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Tag, Loader2, Plus, BarChart3, Power, Search } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

// Admin promo-code management (#39): create platform- or chef-funded codes with
// budget + fraud caps, list/deactivate them, and view redemption analytics.

interface Promo {
  id: string;
  code: string;
  description: string;
  discountType: 'percentage' | 'fixed';
  discountValue: number;
  minOrderAmount: number;
  maxDiscount: number;
  usageLimit: number;
  usageCount: number;
  perUserLimit: number;
  validFrom: string;
  validUntil?: string;
  isActive: boolean;
  applicableTo: string;
  fundingSource: 'platform' | 'chef';
  chefId?: string;
  budgetCap: number;
  budgetSpent: number;
}

interface PromosResponse {
  data: Promo[];
}

interface Analytics {
  code: string;
  fundingSource: string;
  redemptions: number;
  totalDiscount: number;
  uniqueUsers: number;
  usageLimit: number;
  usageCount: number;
  budgetCap: number;
  budgetSpent: number;
  budgetRemaining: number;
  budgetUtilisation: number;
}

interface AdminChef {
  id: string;
  businessName: string;
}

function money(n: number): string {
  try {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(n);
  } catch {
    return `₹${n.toFixed(2)}`;
  }
}

function errMsg(e: unknown): string | undefined {
  const err = (e as { error?: unknown })?.error;
  if (typeof err === 'string') return err;
  if (err && typeof err === 'object' && 'message' in err) return String((err as { message?: unknown }).message);
  return e instanceof Error ? e.message : undefined;
}

export default function PromosPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [analyticsFor, setAnalyticsFor] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['admin-promos', search],
    queryFn: () => apiClient.get<PromosResponse>('/admin/promos', search ? { search } : undefined),
  });
  const promos = (data as unknown as PromosResponse | undefined)?.data ?? [];

  const deactivate = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/admin/promos/${id}`),
    onSuccess: () => {
      toast.success('Promo deactivated');
      queryClient.invalidateQueries({ queryKey: ['admin-promos'] });
    },
    onError: (e: unknown) => toast.error(errMsg(e) || 'Could not deactivate'),
  });

  return (
    <div className="space-y-6">
      <div className="page-header flex items-start justify-between">
        <div>
          <h1 className="page-title">Promo Codes</h1>
          <p className="page-description">
            Platform- or chef-funded discount codes with budget + fraud caps. Chef-funded discounts
            are billed to the chef at settlement.
          </p>
        </div>
        <Button variant="primary" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setShowCreate((v) => !v)}>
          {showCreate ? 'Close' : 'New promo'}
        </Button>
      </div>

      {showCreate && <CreatePromoForm onDone={() => setShowCreate(false)} />}

      <form
        onSubmit={(e) => e.preventDefault()}
        className="relative max-w-md"
      >
        <Search aria-hidden="true" className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search code or description"
          className="h-10 w-full rounded-lg border border-input bg-card pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
        />
      </form>

      {isLoading ? (
        <div className="flex items-center justify-center py-12 text-muted-foreground">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
        </div>
      ) : promos.length === 0 ? (
        <div className="rounded-lg border border-border bg-card p-10 text-center text-sm text-muted-foreground">
          No promo codes yet.
        </div>
      ) : (
        <div className="space-y-3">
          {promos.map((p) => (
            <div key={p.id} className="rounded-xl border border-border bg-card p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <Tag className="h-4 w-4 text-primary" aria-hidden="true" />
                    <span className="font-semibold text-foreground">{p.code}</span>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        p.isActive ? 'bg-success/10 text-success' : 'bg-muted text-muted-foreground'
                      }`}
                    >
                      {p.isActive ? 'Active' : 'Inactive'}
                    </span>
                    <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize text-muted-foreground">
                      {p.fundingSource}-funded
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-foreground">
                    {p.discountType === 'percentage' ? `${p.discountValue}% off` : `${money(p.discountValue)} off`}
                    {p.maxDiscount > 0 && p.discountType === 'percentage' ? ` (max ${money(p.maxDiscount)})` : ''}
                    {p.minOrderAmount > 0 ? ` · min ${money(p.minOrderAmount)}` : ''}
                    {p.applicableTo !== 'all' ? ` · ${p.applicableTo.replace('_', ' ')}` : ''}
                  </p>
                  {p.description && <p className="mt-0.5 text-xs text-muted-foreground">{p.description}</p>}
                  <p className="mt-1 text-xs text-muted-foreground">
                    Used {p.usageCount}
                    {p.usageLimit > 0 ? `/${p.usageLimit}` : ''}
                    {p.perUserLimit > 0 ? ` · ${p.perUserLimit}/user` : ''}
                    {p.budgetCap > 0 ? ` · budget ${money(p.budgetSpent)}/${money(p.budgetCap)}` : ''}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    leftIcon={<BarChart3 className="h-4 w-4" />}
                    onClick={() => setAnalyticsFor(analyticsFor === p.id ? null : p.id)}
                  >
                    Analytics
                  </Button>
                  {p.isActive && (
                    <Button
                      variant="ghost"
                      size="sm"
                      leftIcon={<Power className="h-4 w-4" />}
                      isLoading={deactivate.isPending && deactivate.variables === p.id}
                      onClick={() => deactivate.mutate(p.id)}
                    >
                      Deactivate
                    </Button>
                  )}
                </div>
              </div>
              {analyticsFor === p.id && <PromoAnalytics promoId={p.id} />}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function PromoAnalytics({ promoId }: { promoId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ['admin-promo-analytics', promoId],
    queryFn: () => apiClient.get<Analytics>(`/admin/promos/${promoId}/analytics`),
  });
  const a = data as unknown as Analytics | undefined;

  if (isLoading) {
    return (
      <div className="mt-3 flex items-center border-t border-border pt-3 text-sm text-muted-foreground">
        <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Loading analytics…
      </div>
    );
  }
  if (!a) return null;

  const cards: { label: string; value: string }[] = [
    { label: 'Redemptions', value: String(a.redemptions) },
    { label: 'Unique users', value: String(a.uniqueUsers) },
    { label: 'Discount given', value: money(a.totalDiscount) },
    {
      label: 'Budget',
      value: a.budgetCap > 0 ? `${money(a.budgetSpent)} / ${money(a.budgetCap)} (${a.budgetUtilisation}%)` : '—',
    },
  ];
  return (
    <div className="mt-3 grid grid-cols-2 gap-3 border-t border-border pt-3 sm:grid-cols-4">
      {cards.map((c) => (
        <div key={c.label} className="rounded-lg bg-muted/50 p-3">
          <p className="text-xs text-muted-foreground">{c.label}</p>
          <p className="mt-0.5 text-sm font-semibold tabular-nums text-foreground">{c.value}</p>
        </div>
      ))}
    </div>
  );
}

function CreatePromoForm({ onDone }: { onDone: () => void }) {
  const queryClient = useQueryClient();
  const [code, setCode] = useState('');
  const [description, setDescription] = useState('');
  const [discountType, setDiscountType] = useState<'percentage' | 'fixed'>('percentage');
  const [discountValue, setDiscountValue] = useState('');
  const [minOrderAmount, setMinOrderAmount] = useState('');
  const [maxDiscount, setMaxDiscount] = useState('');
  const [usageLimit, setUsageLimit] = useState('');
  const [perUserLimit, setPerUserLimit] = useState('');
  const [budgetCap, setBudgetCap] = useState('');
  const [applicableTo, setApplicableTo] = useState('all');
  const [fundingSource, setFundingSource] = useState<'platform' | 'chef'>('platform');
  const [chefId, setChefId] = useState('');
  const [validUntil, setValidUntil] = useState('');

  const { data: chefData } = useQuery({
    queryKey: ['admin-chefs-for-promo'],
    queryFn: () => apiClient.get<{ data: AdminChef[] }>('/admin/chefs', { limit: '100' }),
    enabled: fundingSource === 'chef',
  });
  const chefs = (chefData as unknown as { data: AdminChef[] } | undefined)?.data ?? [];

  const create = useMutation({
    mutationFn: (body: Record<string, unknown>) => apiClient.post('/admin/promos', body),
    onSuccess: () => {
      toast.success('Promo created');
      queryClient.invalidateQueries({ queryKey: ['admin-promos'] });
      onDone();
    },
    onError: (e: unknown) => toast.error(errMsg(e) || 'Could not create promo'),
  });

  const submit = () => {
    const value = Number(discountValue);
    if (!code.trim() || !(value > 0)) {
      toast.error('Code and a positive discount value are required');
      return;
    }
    if (fundingSource === 'chef' && !chefId) {
      toast.error('Pick a chef for a chef-funded promo');
      return;
    }
    create.mutate({
      code: code.trim().toUpperCase(),
      description: description.trim(),
      discountType,
      discountValue: value,
      minOrderAmount: Number(minOrderAmount) || 0,
      maxDiscount: Number(maxDiscount) || 0,
      usageLimit: Number(usageLimit) || 0,
      perUserLimit: Number(perUserLimit) || 0,
      budgetCap: Number(budgetCap) || 0,
      applicableTo,
      fundingSource,
      chefId: fundingSource === 'chef' ? chefId : undefined,
      validFrom: new Date().toISOString(),
      validUntil: validUntil ? new Date(validUntil).toISOString() : undefined,
    });
  };

  const field = 'h-9 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring';
  const label = 'mb-1 block text-xs font-medium text-muted-foreground';

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <h3 className="mb-4 text-sm font-semibold text-foreground">New promo code</h3>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div className="sm:col-span-2 lg:col-span-1">
          <label className={label}>Code</label>
          <input className={`${field} uppercase`} value={code} onChange={(e) => setCode(e.target.value.toUpperCase())} placeholder="SUMMER20" />
        </div>
        <div className="sm:col-span-2">
          <label className={label}>Description</label>
          <input className={field} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="20% off summer orders" />
        </div>

        <div>
          <label className={label}>Discount type</label>
          <select className={field} value={discountType} onChange={(e) => setDiscountType(e.target.value as 'percentage' | 'fixed')}>
            <option value="percentage">Percentage (%)</option>
            <option value="fixed">Fixed (₹)</option>
          </select>
        </div>
        <div>
          <label className={label}>{discountType === 'percentage' ? 'Percent off' : 'Amount off (₹)'}</label>
          <input className={field} type="number" min={0} value={discountValue} onChange={(e) => setDiscountValue(e.target.value)} />
        </div>
        <div>
          <label className={label}>Max discount (₹, % only)</label>
          <input className={field} type="number" min={0} value={maxDiscount} onChange={(e) => setMaxDiscount(e.target.value)} placeholder="0 = none" />
        </div>

        <div>
          <label className={label}>Min order (₹)</label>
          <input className={field} type="number" min={0} value={minOrderAmount} onChange={(e) => setMinOrderAmount(e.target.value)} placeholder="0" />
        </div>
        <div>
          <label className={label}>Eligibility</label>
          <select className={field} value={applicableTo} onChange={(e) => setApplicableTo(e.target.value)}>
            <option value="all">Everyone</option>
            <option value="new_users">New users (first order)</option>
            <option value="returning_users">Returning users</option>
          </select>
        </div>
        <div>
          <label className={label}>Expires</label>
          <input className={field} type="date" value={validUntil} onChange={(e) => setValidUntil(e.target.value)} />
        </div>

        <div>
          <label className={label}>Total uses (0 = ∞)</label>
          <input className={field} type="number" min={0} value={usageLimit} onChange={(e) => setUsageLimit(e.target.value)} placeholder="0" />
        </div>
        <div>
          <label className={label}>Per-user uses (0 = ∞)</label>
          <input className={field} type="number" min={0} value={perUserLimit} onChange={(e) => setPerUserLimit(e.target.value)} placeholder="0" />
        </div>
        <div>
          <label className={label}>Budget cap (₹, 0 = ∞)</label>
          <input className={field} type="number" min={0} value={budgetCap} onChange={(e) => setBudgetCap(e.target.value)} placeholder="0" />
        </div>

        <div>
          <label className={label}>Funding</label>
          <select className={field} value={fundingSource} onChange={(e) => setFundingSource(e.target.value as 'platform' | 'chef')}>
            <option value="platform">Platform-funded</option>
            <option value="chef">Chef-funded</option>
          </select>
        </div>
        {fundingSource === 'chef' && (
          <div className="sm:col-span-2">
            <label className={label}>Chef (bears the discount)</label>
            <select className={field} value={chefId} onChange={(e) => setChefId(e.target.value)}>
              <option value="">Select a chef…</option>
              {chefs.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.businessName}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      <div className="mt-5 flex justify-end gap-3">
        <Button variant="ghost" onClick={onDone}>
          Cancel
        </Button>
        <Button variant="primary" isLoading={create.isPending} onClick={submit}>
          Create promo
        </Button>
      </div>
    </div>
  );
}
