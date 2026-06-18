import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { ChefHat, Check } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';

// Bulk subscription prep view (#50) — web parity with the vendor mobile app.
// "Tomorrow you owe N lunches / M dinners" rolled up by dish from confirmed
// meal-plan subscriptions, with a packing list and mark-prepared.

interface ManifestLine {
  slot: 'lunch' | 'dinner';
  variant: 'veg' | 'nonveg';
  dishName: string;
  total: number;
  prepared: number;
}
interface PackingRow {
  dayId: string;
  slot: 'lunch' | 'dinner';
  variant: 'veg' | 'nonveg';
  dishName: string;
  status: string;
  planNumber: string;
  customerName: string;
}
interface PrepManifest {
  date: string;
  manifest: ManifestLine[];
  packingList: PackingRow[];
  totals: { lunch: number; dinner: number; total: number; prepared: number };
}

function ymd(offset: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offset);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
const DAYS = [
  { label: 'Today', value: ymd(0) },
  { label: 'Tomorrow', value: ymd(1) },
  { label: 'In 2 days', value: ymd(2) },
];

function VariantDot({ variant }: { variant: 'veg' | 'nonveg' }) {
  return (
    <span
      className={`inline-block h-2.5 w-2.5 rounded-full ${variant === 'veg' ? 'bg-herb' : 'bg-paprika'}`}
      aria-hidden="true"
    />
  );
}

export default function PrepPage() {
  const qc = useQueryClient();
  const [date, setDate] = useState(DAYS[1]!.value);

  const { data, isLoading } = useQuery({
    queryKey: ['chef-prep', date],
    queryFn: () => apiClient.get<PrepManifest>(`/chef/prep?date=${date}`),
    refetchInterval: 30_000,
  });

  const mark = useMutation({
    mutationFn: (body: { date?: string; slot?: string; variant?: string; dishName?: string; dayIds?: string[] }) =>
      apiClient.post('/chef/prep/mark', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef-prep'] });
      toast.success('Marked prepared');
    },
    onError: () => toast.error('Could not update'),
  });

  const manifest = data?.manifest ?? [];
  const packing = data?.packingList ?? [];
  const totals = data?.totals;
  const lunch = manifest.filter((m) => m.slot === 'lunch');
  const dinner = manifest.filter((m) => m.slot === 'dinner');

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 font-display text-2xl font-semibold text-ink">
          <ChefHat className="h-6 w-6 text-herb" /> Prep
        </h1>
        <p className="text-sm text-ink-muted">
          What you owe from confirmed subscriptions, by dish — plus a packing list.
        </p>
      </div>

      {/* Day selector */}
      <div className="inline-flex rounded-lg bg-bone p-1">
        {DAYS.map((d) => (
          <button
            key={d.value}
            type="button"
            onClick={() => setDate(d.value)}
            className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors ${
              date === d.value ? 'bg-paper text-ink shadow-1' : 'text-ink-muted'
            }`}
          >
            {d.label}
          </button>
        ))}
      </div>

      {isLoading ? (
        <p className="text-ink-muted">Loading…</p>
      ) : !totals || totals.total === 0 ? (
        <div className="rounded-xl border border-mist bg-bone p-8 text-center text-ink-muted">
          No confirmed subscription meals for this day yet.
        </div>
      ) : (
        <>
          <div className="rounded-xl bg-herb-tint p-4">
            <p className="text-ink">
              You owe <span className="font-semibold text-herb">{totals.lunch} lunch</span> ·{' '}
              <span className="font-semibold text-herb">{totals.dinner} dinner</span>
            </p>
            <p className="mt-1 text-sm text-ink-soft tabular-nums">
              {totals.prepared}/{totals.total} prepared
            </p>
          </div>

          <ManifestSection
            title="Lunch"
            lines={lunch}
            onMark={(l) => mark.mutate({ date, slot: l.slot, variant: l.variant, dishName: l.dishName })}
          />
          <ManifestSection
            title="Dinner"
            lines={dinner}
            onMark={(l) => mark.mutate({ date, slot: l.slot, variant: l.variant, dishName: l.dishName })}
          />

          {/* Packing list */}
          <div className="rounded-xl border border-mist bg-bone p-6">
            <h2 className="mb-3 text-lg font-semibold text-ink">Packing list</h2>
            <div className="divide-y divide-mist">
              {packing.map((row) => (
                <div key={row.dayId} className="flex items-center gap-3 py-3">
                  <VariantDot variant={row.variant} />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-ink">{row.dishName || 'Dish'}</p>
                    <p className="truncate text-xs text-ink-muted">
                      {row.customerName || 'Customer'} · {row.slot} · {row.planNumber}
                    </p>
                  </div>
                  {row.status === 'prepared' ? (
                    <span className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-3 py-1 text-xs font-medium text-herb">
                      <Check className="h-3 w-3" /> Prepared
                    </span>
                  ) : (
                    <Button variant="outline" size="sm" onClick={() => mark.mutate({ dayIds: [row.dayId] })}>
                      Mark
                    </Button>
                  )}
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function ManifestSection({
  title,
  lines,
  onMark,
}: {
  title: string;
  lines: ManifestLine[];
  onMark: (line: ManifestLine) => void;
}) {
  if (lines.length === 0) return null;
  return (
    <div className="rounded-xl border border-mist bg-bone p-6">
      <h2 className="mb-3 text-lg font-semibold text-ink">{title}</h2>
      <div className="divide-y divide-mist">
        {lines.map((line) => {
          const done = line.prepared >= line.total;
          return (
            <div key={`${line.variant}-${line.dishName}`} className="flex items-center gap-3 py-3">
              <VariantDot variant={line.variant} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-ink">{line.dishName || 'Dish'}</p>
                <p className="text-xs text-ink-muted">
                  {line.total} owed{line.prepared > 0 ? ` · ${line.prepared} prepared` : ''}
                </p>
              </div>
              <span className="text-base font-semibold tabular-nums text-ink">×{line.total}</span>
              {done ? (
                <span className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-3 py-1 text-xs font-medium text-herb">
                  <Check className="h-3 w-3" /> Done
                </span>
              ) : (
                <Button variant="outline" size="sm" onClick={() => onMark(line)}>
                  Mark prepared
                </Button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
