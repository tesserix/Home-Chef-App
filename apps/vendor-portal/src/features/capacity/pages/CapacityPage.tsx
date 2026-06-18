import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Gauge, Clock } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import type { MenuItem } from '@/shared/types';

// Chef capacity & cutoff controls (#48) — web parity with the vendor mobile app.
// Per-meal order cutoffs + auto-sold-out, and per-dish daily caps with today's
// remaining/sold. Mirrors apps/mobile-vendor/app/capacity.tsx.

interface CapacitySettings {
  chefId: string;
  cutoffEnabled: boolean;
  lunchCutoff: string; // "HH:MM" IST, "" = none
  dinnerCutoff: string;
  autoSoldOut: boolean;
}

const HHMM = /^([01]\d|2[0-3]):[0-5]\d$/;

export default function CapacityPage() {
  const queryClient = useQueryClient();
  const { data: settings, isLoading } = useQuery({
    queryKey: ['chef-capacity-settings'],
    queryFn: () => apiClient.get<CapacitySettings>('/chef/capacity-settings'),
  });
  const { data: items } = useQuery({
    queryKey: ['chef-menu'],
    queryFn: () => apiClient.get<MenuItem[]>('/chef/menu'),
  });

  const [cutoffEnabled, setCutoffEnabled] = useState(false);
  const [lunch, setLunch] = useState('');
  const [dinner, setDinner] = useState('');
  const [autoSoldOut, setAutoSoldOut] = useState(true);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    if (hydrated || !settings) return;
    setCutoffEnabled(settings.cutoffEnabled);
    setLunch(settings.lunchCutoff ?? '');
    setDinner(settings.dinnerCutoff ?? '');
    setAutoSoldOut(settings.autoSoldOut);
    setHydrated(true);
  }, [settings, hydrated]);

  const saveSettings = useMutation({
    mutationFn: (body: Partial<Omit<CapacitySettings, 'chefId'>>) =>
      apiClient.put('/chef/capacity-settings', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-capacity-settings'] });
      toast.success('Capacity settings saved');
    },
    onError: () => toast.error('Could not save settings'),
  });

  const setItemCap = useMutation({
    mutationFn: (vars: { itemId: string; dailyCapacity: number | null }) =>
      apiClient.put(`/chef/menu/items/${vars.itemId}/capacity`, {
        dailyCapacity: vars.dailyCapacity,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success('Daily cap updated');
    },
    onError: () => toast.error('Could not update cap'),
  });

  function onSaveSettings() {
    for (const [label, v] of [['Lunch', lunch], ['Dinner', dinner]] as const) {
      if (v !== '' && !HHMM.test(v)) {
        toast.error(`${label} cutoff must be HH:MM (24h), e.g. 10:00`);
        return;
      }
    }
    saveSettings.mutate({ cutoffEnabled, lunchCutoff: lunch, dinnerCutoff: dinner, autoSoldOut });
  }

  if (isLoading) {
    return <div className="p-8 text-ink-muted">Loading…</div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 font-display text-2xl font-semibold text-ink">
          <Gauge className="h-6 w-6 text-herb" /> Capacity &amp; cutoffs
        </h1>
        <p className="text-sm text-ink-muted">
          Cap how many of each dish you'll make per day and close ordering after your meal cutoffs.
        </p>
      </div>

      {/* Cutoffs + auto sold-out */}
      <div className="rounded-xl border border-mist bg-bone p-6 space-y-5">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
          <Clock className="h-5 w-5 text-herb" /> Order cutoffs
        </h2>
        <ToggleRow
          label="Enable cutoffs"
          description="Auto-close ordering once the day's cutoffs pass."
          checked={cutoffEnabled}
          onChange={() => setCutoffEnabled((v) => !v)}
        />
        {cutoffEnabled && (
          <div className="grid grid-cols-2 gap-4">
            <CutoffInput label="Lunch cutoff" value={lunch} onChange={setLunch} />
            <CutoffInput label="Dinner cutoff" value={dinner} onChange={setDinner} />
          </div>
        )}
        <ToggleRow
          label="Auto sold-out"
          description="Hide a dish automatically when its daily cap is reached."
          checked={autoSoldOut}
          onChange={() => setAutoSoldOut((v) => !v)}
        />
        <Button isLoading={saveSettings.isPending} onClick={onSaveSettings}>
          Save settings
        </Button>
      </div>

      {/* Per-dish daily caps */}
      <div className="rounded-xl border border-mist bg-bone p-6">
        <h2 className="mb-4 text-lg font-semibold text-ink">Daily caps per dish</h2>
        {!items || items.length === 0 ? (
          <p className="text-sm text-ink-muted">Add menu items first to set daily caps.</p>
        ) : (
          <div className="divide-y divide-mist">
            {items.map((it) => (
              <CapRow key={it.id} item={it} onSave={(cap) => setItemCap.mutate({ itemId: it.id, dailyCapacity: cap })} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function CapRow({ item, onSave }: { item: MenuItem; onSave: (cap: number | null) => void }) {
  const [draft, setDraft] = useState(
    item.dailyCapacity != null && item.dailyCapacity > 0 ? String(item.dailyCapacity) : '',
  );
  const capped = item.dailyCapacity != null && item.dailyCapacity > 0;
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-ink">{item.name}</p>
        <p className="text-xs text-ink-muted">
          {capped
            ? item.soldOut
              ? `Sold out (cap ${item.dailyCapacity})`
              : `${item.remainingToday ?? item.dailyCapacity} left today · cap ${item.dailyCapacity}`
            : 'Unlimited'}
        </p>
      </div>
      <input
        type="number"
        min={0}
        inputMode="numeric"
        placeholder="∞"
        value={draft}
        onChange={(e) => setDraft(e.target.value.replace(/[^0-9]/g, ''))}
        className="w-20 rounded-lg border border-mist-strong px-3 py-2 text-sm text-ink focus:border-herb focus:outline-none"
        aria-label={`Daily cap for ${item.name}`}
      />
      <Button
        variant="outline"
        size="sm"
        onClick={() => onSave(draft.trim() === '' ? null : Math.max(0, parseInt(draft, 10) || 0))}
      >
        Set
      </Button>
    </div>
  );
}

function CutoffInput({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (s: string) => void;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium text-ink">{label}</span>
      <input
        type="text"
        placeholder="HH:MM"
        value={value}
        onChange={(e) => onChange(e.target.value.replace(/[^0-9:]/g, '').slice(0, 5))}
        className="w-full rounded-lg border border-mist-strong px-3 py-2 text-sm text-ink focus:border-herb focus:outline-none"
      />
    </label>
  );
}

function ToggleRow({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-medium text-ink">{label}</p>
        <p className="text-xs text-ink-muted">{description}</p>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        onClick={onChange}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${
          checked ? 'bg-herb' : 'bg-mist'
        }`}
      >
        <span
          className={`inline-block h-4 w-4 rounded-full bg-bone transition-transform ${
            checked ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
    </div>
  );
}
