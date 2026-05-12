import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { ApiError } from '@/shared/types';
import {
  ArrowLeft,
  Percent,
  Truck,
  Clock,
  MapPin,
  Plus,
  Trash2,
  X,
} from 'lucide-react';

interface PlatformPolicy {
  serviceFeePercent: number;
  taxPercent: number;
  baseDeliveryFee: number;
  perKmDeliveryFee: number;
  chefPayoutPercent: number;
  driverPayoutPercent: number;
  timezone: string;
  openingTime: string;
  closingTime: string;
  operatingDays: number[];
  closedMessage: string;
}

interface DeliveryZone {
  id: string;
  name: string;
  city: string;
  state: string;
  country: string;
  tier: string;
  minLatitude: number;
  maxLatitude: number;
  minLongitude: number;
  maxLongitude: number;
  baseFare: number;
  perKmRate: number;
  minimumFare: number;
  isActive: boolean;
}

const DAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

export default function PlatformSettingsPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to="/settings"
          className="rounded-lg p-2 text-muted-foreground hover:bg-secondary hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="page-title">Platform</h1>
          <p className="page-description">
            Commission rates, delivery fees, operating hours, and service areas
          </p>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <CommissionSection />
        <DeliveryFeesSection />
        <OperatingHoursSection />
        <ServiceAreasSection />
      </div>
    </div>
  );
}

// ============================================================
// Shared feedback banner
// ============================================================

function useFeedback() {
  const [state, setState] = useState<{ kind: 'success' | 'error'; message: string } | null>(null);
  return { state, setState } as const;
}

function FeedbackBanner({
  value,
  onDismiss,
}: {
  value: { kind: 'success' | 'error'; message: string } | null;
  onDismiss: () => void;
}) {
  if (!value) return null;
  return (
    <div
      className={`mt-4 flex items-start justify-between gap-2 rounded-lg border px-3 py-2 text-xs ${
        value.kind === 'success'
          ? 'border-herb/30 bg-herb-tint text-herb'
          : 'border-paprika/30 bg-paprika-tint text-paprika'
      }`}
    >
      <span>{value.message}</span>
      <button
        onClick={onDismiss}
        aria-label="Dismiss"
        className="shrink-0 opacity-60 hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb rounded p-0.5"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

// ============================================================
// Commission + Tax
// ============================================================

function CommissionSection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data } = useQuery({
    queryKey: ['platform-policy'],
    queryFn: () => apiClient.get<PlatformPolicy>('/admin/platform/policy'),
  });

  const [draft, setDraft] = useState<Partial<PlatformPolicy>>({});
  const current = { ...data, ...draft } as PlatformPolicy;

  const save = useMutation({
    mutationFn: (body: Partial<PlatformPolicy>) =>
      apiClient.put<PlatformPolicy>('/admin/platform/policy', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['platform-policy'] });
      setDraft({});
      feedback.setState({ kind: 'success', message: 'Commission rates updated' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to save' });
    },
  });

  return (
    <Card
      icon={<Percent className="h-5 w-5 text-primary" />}
      title="Commission rates"
      description="Service fee, tax, and payout percentages"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />
      {!data ? (
        <p className="py-4 text-sm text-muted-foreground">Loading...</p>
      ) : (
        <div className="mt-4 space-y-3">
          <NumberField
            label="Service fee (%)"
            value={current.serviceFeePercent}
            onChange={(v) => setDraft((d) => ({ ...d, serviceFeePercent: v }))}
            min={0}
            max={50}
            step={0.5}
          />
          <NumberField
            label="Tax (%)"
            value={current.taxPercent}
            onChange={(v) => setDraft((d) => ({ ...d, taxPercent: v }))}
            min={0}
            max={50}
            step={0.5}
          />
          <NumberField
            label="Chef payout (% of subtotal)"
            value={current.chefPayoutPercent}
            onChange={(v) => setDraft((d) => ({ ...d, chefPayoutPercent: v }))}
            min={0}
            max={100}
            step={1}
          />
          <NumberField
            label="Driver payout (% of delivery fee)"
            value={current.driverPayoutPercent}
            onChange={(v) => setDraft((d) => ({ ...d, driverPayoutPercent: v }))}
            min={0}
            max={100}
            step={1}
          />
          <SaveRow
            dirty={Object.keys(draft).length > 0}
            saving={save.isPending}
            onCancel={() => setDraft({})}
            onSave={() => save.mutate(draft)}
          />
        </div>
      )}
    </Card>
  );
}

// ============================================================
// Delivery fees
// ============================================================

function DeliveryFeesSection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data } = useQuery({
    queryKey: ['platform-policy'],
    queryFn: () => apiClient.get<PlatformPolicy>('/admin/platform/policy'),
  });

  const [draft, setDraft] = useState<Partial<PlatformPolicy>>({});
  const current = { ...data, ...draft } as PlatformPolicy;

  const save = useMutation({
    mutationFn: (body: Partial<PlatformPolicy>) =>
      apiClient.put<PlatformPolicy>('/admin/platform/policy', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['platform-policy'] });
      setDraft({});
      feedback.setState({ kind: 'success', message: 'Delivery fees updated' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to save' });
    },
  });

  return (
    <Card
      icon={<Truck className="h-5 w-5 text-primary" />}
      title="Delivery fees"
      description="Base fee + per-km rate applied at checkout"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />
      {!data ? (
        <p className="py-4 text-sm text-muted-foreground">Loading...</p>
      ) : (
        <div className="mt-4 space-y-3">
          <NumberField
            label="Base delivery fee"
            value={current.baseDeliveryFee}
            onChange={(v) => setDraft((d) => ({ ...d, baseDeliveryFee: v }))}
            min={0}
            step={0.1}
          />
          <NumberField
            label="Per-km rate"
            value={current.perKmDeliveryFee}
            onChange={(v) => setDraft((d) => ({ ...d, perKmDeliveryFee: v }))}
            min={0}
            step={0.1}
          />
          <SaveRow
            dirty={Object.keys(draft).length > 0}
            saving={save.isPending}
            onCancel={() => setDraft({})}
            onSave={() => save.mutate(draft)}
          />
        </div>
      )}
    </Card>
  );
}

// ============================================================
// Operating hours
// ============================================================

function OperatingHoursSection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data } = useQuery({
    queryKey: ['platform-policy'],
    queryFn: () => apiClient.get<PlatformPolicy>('/admin/platform/policy'),
  });

  const [draft, setDraft] = useState<Partial<PlatformPolicy>>({});
  const current = { ...data, ...draft } as PlatformPolicy;
  const days = new Set(current.operatingDays ?? []);

  const save = useMutation({
    mutationFn: (body: Partial<PlatformPolicy>) =>
      apiClient.put<PlatformPolicy>('/admin/platform/policy', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['platform-policy'] });
      setDraft({});
      feedback.setState({ kind: 'success', message: 'Operating hours updated' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to save' });
    },
  });

  const toggleDay = (day: number) => {
    const next = new Set(days);
    if (next.has(day)) next.delete(day);
    else next.add(day);
    setDraft((d) => ({ ...d, operatingDays: Array.from(next).sort((a, b) => a - b) }));
  };

  return (
    <Card
      icon={<Clock className="h-5 w-5 text-primary" />}
      title="Operating hours"
      description="When the platform accepts new orders"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />
      {!data ? (
        <p className="py-4 text-sm text-muted-foreground">Loading...</p>
      ) : (
        <div className="mt-4 space-y-3">
          <TextField
            label="Timezone (IANA)"
            value={current.timezone ?? ''}
            onChange={(v) => setDraft((d) => ({ ...d, timezone: v }))}
            placeholder="Asia/Kolkata"
          />
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="Opening time (HH:MM)"
              value={current.openingTime ?? ''}
              onChange={(v) => setDraft((d) => ({ ...d, openingTime: v }))}
              placeholder="09:00"
            />
            <TextField
              label="Closing time (HH:MM)"
              value={current.closingTime ?? ''}
              onChange={(v) => setDraft((d) => ({ ...d, closingTime: v }))}
              placeholder="23:00"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">
              Operating days
            </label>
            <div className="flex flex-wrap gap-1.5">
              {DAY_LABELS.map((label, i) => (
                <button
                  key={i}
                  onClick={() => toggleDay(i)}
                  className={`rounded-md border px-2.5 py-1 text-xs font-medium ${
                    days.has(i)
                      ? 'border-primary bg-primary text-primary-foreground'
                      : 'border-border bg-bone text-foreground hover:bg-secondary'
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              No days selected = every day.
            </p>
          </div>
          <TextField
            label="Closed message"
            value={current.closedMessage ?? ''}
            onChange={(v) => setDraft((d) => ({ ...d, closedMessage: v }))}
            placeholder="We're currently closed."
          />
          <SaveRow
            dirty={Object.keys(draft).length > 0}
            saving={save.isPending}
            onCancel={() => setDraft({})}
            onSave={() => save.mutate(draft)}
          />
        </div>
      )}
    </Card>
  );
}

// ============================================================
// Service areas (delivery zones)
// ============================================================

function ServiceAreasSection() {
  const qc = useQueryClient();
  const feedback = useFeedback();
  const { data, isLoading } = useQuery({
    queryKey: ['delivery-zones'],
    queryFn: () => apiClient.get<{ zones: DeliveryZone[]; total: number }>('/admin/delivery/zones'),
  });

  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<Partial<DeliveryZone>>({
    name: '',
    city: '',
    country: 'IN',
    tier: 'standard',
    minLatitude: 0,
    maxLatitude: 0,
    minLongitude: 0,
    maxLongitude: 0,
    baseFare: 0,
    perKmRate: 0,
  });

  const create = useMutation({
    mutationFn: (body: Partial<DeliveryZone>) =>
      apiClient.post<{ zone: DeliveryZone }>('/admin/delivery/zones', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['delivery-zones'] });
      setCreating(false);
      feedback.setState({ kind: 'success', message: 'Zone created' });
    },
    onError: (err) => {
      const e = err as Partial<ApiError>;
      feedback.setState({ kind: 'error', message: e?.error?.message ?? 'Failed to create' });
    },
  });
  const remove = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/admin/delivery/zones/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['delivery-zones'] });
      feedback.setState({ kind: 'success', message: 'Zone deleted' });
    },
  });

  return (
    <Card
      icon={<MapPin className="h-5 w-5 text-primary" />}
      title="Service areas"
      description="Delivery zones — orders outside a zone are rejected"
    >
      <FeedbackBanner value={feedback.state} onDismiss={() => feedback.setState(null)} />
      {creating ? (
        <div className="mt-4 space-y-3 rounded-lg border border-border bg-secondary/20 p-3">
          <TextField
            label="Name"
            value={form.name ?? ''}
            onChange={(v) => setForm({ ...form, name: v })}
            placeholder="Central Bangalore"
          />
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="City"
              value={form.city ?? ''}
              onChange={(v) => setForm({ ...form, city: v })}
            />
            <TextField
              label="State"
              value={form.state ?? ''}
              onChange={(v) => setForm({ ...form, state: v })}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <NumberField
              label="Min latitude"
              value={form.minLatitude ?? 0}
              onChange={(v) => setForm({ ...form, minLatitude: v })}
              step={0.001}
            />
            <NumberField
              label="Max latitude"
              value={form.maxLatitude ?? 0}
              onChange={(v) => setForm({ ...form, maxLatitude: v })}
              step={0.001}
            />
            <NumberField
              label="Min longitude"
              value={form.minLongitude ?? 0}
              onChange={(v) => setForm({ ...form, minLongitude: v })}
              step={0.001}
            />
            <NumberField
              label="Max longitude"
              value={form.maxLongitude ?? 0}
              onChange={(v) => setForm({ ...form, maxLongitude: v })}
              step={0.001}
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setCreating(false)}
              className="flex-1 rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary"
            >
              Cancel
            </button>
            <button
              onClick={() => create.mutate(form)}
              disabled={!form.name || !form.city || create.isPending}
              className="flex-1 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {create.isPending ? 'Creating...' : 'Create zone'}
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setCreating(true)}
          className="mt-4 inline-flex items-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-2 text-sm text-muted-foreground hover:border-primary hover:text-primary"
        >
          <Plus className="h-4 w-4" /> New zone
        </button>
      )}

      <div className="mt-4 space-y-2">
        {isLoading ? (
          <p className="py-2 text-sm text-muted-foreground">Loading...</p>
        ) : (data?.zones ?? []).length === 0 ? (
          <p className="py-2 text-xs text-muted-foreground">
            No zones yet — delivery is available everywhere until a zone is created.
          </p>
        ) : (
          (data?.zones ?? []).map((z) => (
            <div
              key={z.id}
              className="flex items-center justify-between gap-3 rounded-lg border border-border bg-secondary/30 px-3 py-2 text-xs"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-foreground">{z.name}</p>
                <p className="text-muted-foreground">
                  {z.city}
                  {z.state ? `, ${z.state}` : ''} · {z.country} · {z.tier}
                  {!z.isActive && ' · inactive'}
                </p>
              </div>
              <button
                onClick={() => remove.mutate(z.id)}
                className="shrink-0 text-paprika hover:text-paprika"
                title="Delete"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))
        )}
      </div>
    </Card>
  );
}

// ============================================================
// Primitives (shared with SecuritySettingsPage shape)
// ============================================================

function Card({
  icon,
  title,
  description,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          {icon}
        </div>
        <div>
          <h3 className="font-semibold text-foreground">{title}</h3>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
      </div>
      {children}
    </div>
  );
}

function NumberField({
  label,
  value,
  onChange,
  min,
  max,
  step,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  min?: number;
  max?: number;
  step?: number;
}) {
  return (
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
      <input
        type="number"
        value={value}
        min={min}
        max={max}
        step={step ?? 1}
        onChange={(e) => {
          const raw = Number(e.target.value);
          if (Number.isNaN(raw)) return;
          let clamped = raw;
          if (min !== undefined) clamped = Math.max(min, clamped);
          if (max !== undefined) clamped = Math.min(max, clamped);
          onChange(clamped);
        }}
        className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
      />
    </div>
  );
}

function TextField({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}) {
  return (
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
      />
    </div>
  );
}

function SaveRow({
  dirty,
  saving,
  onCancel,
  onSave,
}: {
  dirty: boolean;
  saving: boolean;
  onCancel: () => void;
  onSave: () => void;
}) {
  return (
    <div className="flex justify-end gap-2 pt-2">
      <button
        onClick={onCancel}
        disabled={!dirty || saving}
        className="rounded-lg border border-border px-3 py-1.5 text-sm text-foreground hover:bg-secondary disabled:opacity-50"
      >
        Cancel
      </button>
      <button
        onClick={onSave}
        disabled={!dirty || saving}
        className="rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {saving ? 'Saving...' : 'Save'}
      </button>
    </div>
  );
}
