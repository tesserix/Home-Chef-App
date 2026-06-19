import { useEffect, useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import { CalendarDays, Copy, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import {
  useWeeklyMenu,
  useSaveWeeklyMenu,
  weeklyMenuHole,
  type MealSlot,
  type MealVariant,
  type WeeklyMenuItem,
} from '../hooks/useWeeklyMenu';

// Weekly-menu editor (#1) — the fixed dishes a chef offers per day × slot ×
// veg/nonveg. Grid by day; replace-all save; publish requires a complete grid
// (mirrors the API validator via weeklyMenuHole). Web parity with the mobile
// vendor editor.

const DAYS: { dow: number; long: string }[] = [
  { dow: 1, long: 'Monday' },
  { dow: 2, long: 'Tuesday' },
  { dow: 3, long: 'Wednesday' },
  { dow: 4, long: 'Thursday' },
  { dow: 5, long: 'Friday' },
  { dow: 6, long: 'Saturday' },
  { dow: 0, long: 'Sunday' },
];
const SLOTS: { slot: MealSlot; label: string }[] = [
  { slot: 'lunch', label: 'Lunch' },
  { slot: 'dinner', label: 'Dinner' },
];
const VARIANTS: { variant: MealVariant; label: string; dot: string }[] = [
  { variant: 'veg', label: 'Veg', dot: 'bg-herb' },
  { variant: 'nonveg', label: 'Non-veg', dot: 'bg-paprika' },
];

interface Cell {
  name: string;
  price: string;
}
const cellKey = (dow: number, slot: MealSlot, variant: MealVariant) => `${dow}-${slot}-${variant}`;

export default function WeeklyMenuPage() {
  const { data, isLoading } = useWeeklyMenu();
  const save = useSaveWeeklyMenu();

  const [selectedDow, setSelectedDow] = useState(1);
  const [cells, setCells] = useState<Record<string, Cell>>({});
  const [published, setPublished] = useState(false);
  const [hydrated, setHydrated] = useState(false);

  // Hydrate once; later refetches don't clobber in-progress edits.
  useEffect(() => {
    if (hydrated || !data) return;
    const next: Record<string, Cell> = {};
    for (const it of data.items ?? []) {
      next[cellKey(it.dayOfWeek, it.slot, it.variant)] = {
        name: it.name ?? '',
        price: it.price ? String(it.price) : '',
      };
    }
    setCells(next);
    setPublished(Boolean(data.isPublished));
    setHydrated(true);
  }, [data, hydrated]);

  const filledDays = useMemo(() => {
    const set = new Set<number>();
    for (const [key, c] of Object.entries(cells)) {
      if (c?.name.trim()) set.add(Number(key.split('-')[0]));
    }
    return set;
  }, [cells]);

  function setCell(dow: number, slot: MealSlot, variant: MealVariant, patch: Partial<Cell>) {
    const key = cellKey(dow, slot, variant);
    setCells((prev) => ({ ...prev, [key]: { ...(prev[key] ?? { name: '', price: '' }), ...patch } }));
  }

  // Copy convenience (#1): replicate the selected day's cells to every other day.
  function copyDayToAll() {
    setCells((prev) => {
      const next = { ...prev };
      for (const slot of SLOTS) {
        for (const v of VARIANTS) {
          const src = prev[cellKey(selectedDow, slot.slot, v.variant)];
          for (const d of DAYS) {
            if (d.dow === selectedDow) continue;
            next[cellKey(d.dow, slot.slot, v.variant)] = src ? { ...src } : { name: '', price: '' };
          }
        }
      }
      return next;
    });
    toast.success('Copied this day to the whole week');
  }

  function buildItems(): WeeklyMenuItem[] {
    const items: WeeklyMenuItem[] = [];
    for (const [key, c] of Object.entries(cells)) {
      if (!c?.name.trim()) continue;
      const [dowStr, slot, variant] = key.split('-');
      items.push({
        dayOfWeek: Number(dowStr),
        slot: slot as MealSlot,
        variant: variant as MealVariant,
        name: c.name.trim(),
        price: Number.parseFloat(c.price) || 0,
      });
    }
    return items;
  }

  function onSave(nextPublished: boolean) {
    const items = buildItems();
    if (nextPublished) {
      const hole = weeklyMenuHole(items);
      if (hole) {
        toast.error(hole);
        return;
      }
    }
    save.mutate(
      { isPublished: nextPublished, items },
      {
        onSuccess: () => {
          setPublished(nextPublished);
          toast.success(
            nextPublished
              ? 'Your weekly menu is live — customers can pre-book it.'
              : 'Saved as a draft (not visible to customers yet).',
          );
        },
        onError: () => toast.error('Could not save. Please try again.'),
      },
    );
  }

  if (isLoading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" aria-hidden="true" />
      </div>
    );
  }

  return (
    <motion.div variants={staggerContainer} initial="hidden" animate="visible" className="mx-auto max-w-4xl space-y-6 p-4 sm:p-6">
      <motion.div variants={fadeInUp} className="flex items-center gap-3">
        <CalendarDays className="h-6 w-6 text-herb" aria-hidden="true" />
        <div>
          <h1 className="page-title">Weekly menu</h1>
          <p className="page-description">
            The fixed dishes customers pre-book — one per day, per slot. {published ? 'Published.' : 'Draft.'}
          </p>
        </div>
      </motion.div>

      {/* Day tabs */}
      <motion.div variants={fadeInUp} className="flex flex-wrap gap-2">
        {DAYS.map((d) => {
          const active = d.dow === selectedDow;
          return (
            <button
              key={d.dow}
              type="button"
              onClick={() => setSelectedDow(d.dow)}
              className={`rounded-full px-3 py-1.5 text-sm font-medium transition-colors ${
                active ? 'bg-herb text-paper' : 'bg-mist text-ink-soft hover:bg-bone'
              }`}
            >
              {d.long.slice(0, 3)}
              {filledDays.has(d.dow) ? <span className="ml-1 text-xs">•</span> : null}
            </button>
          );
        })}
      </motion.div>

      {/* Cells for the selected day */}
      <motion.div variants={fadeInUp}>
        <Card>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-ink">{DAYS.find((d) => d.dow === selectedDow)?.long}</h2>
            <Button variant="ghost" size="sm" onClick={copyDayToAll} leftIcon={<Copy className="h-4 w-4" />}>
              Copy to all days
            </Button>
          </div>
          <div className="space-y-4">
            {SLOTS.map((slot) => (
              <div key={slot.slot}>
                <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-ink-muted">{slot.label}</p>
                <div className="grid gap-3 sm:grid-cols-2">
                  {VARIANTS.map((v) => {
                    const key = cellKey(selectedDow, slot.slot, v.variant);
                    const cell = cells[key] ?? { name: '', price: '' };
                    return (
                      <div key={v.variant} className="rounded-lg border border-mist p-3">
                        <div className="mb-2 flex items-center gap-1.5">
                          <span className={`h-2 w-2 rounded-sm ${v.dot}`} aria-hidden="true" />
                          <span className="text-xs font-medium text-ink-soft">{v.label}</span>
                        </div>
                        <input
                          className="h-9 w-full rounded-lg border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus:border-herb focus:outline-none focus:ring-2 focus:ring-ring"
                          placeholder="Dish name"
                          value={cell.name}
                          onChange={(e) => setCell(selectedDow, slot.slot, v.variant, { name: e.target.value })}
                        />
                        <div className="mt-2 flex h-9 items-center rounded-lg border border-input bg-background px-2">
                          <span className="text-sm text-muted-foreground">₹</span>
                          <input
                            className="w-full bg-transparent px-1 text-sm focus:outline-none"
                            placeholder="0"
                            inputMode="decimal"
                            value={cell.price}
                            onChange={(e) => setCell(selectedDow, slot.slot, v.variant, { price: e.target.value.replace(/[^0-9.]/g, '') })}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </Card>
      </motion.div>

      {/* Actions */}
      <motion.div variants={fadeInUp} className="flex flex-wrap items-center justify-end gap-3">
        <Button variant="outline" disabled={save.isPending} onClick={() => onSave(false)}>
          Save draft
        </Button>
        <Button variant="primary" isLoading={save.isPending} disabled={save.isPending} onClick={() => onSave(true)}>
          {published ? 'Save & keep live' : 'Publish'}
        </Button>
      </motion.div>
    </motion.div>
  );
}
