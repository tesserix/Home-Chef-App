// Add-on modifier-groups editor + combo builder for the menu-item form (#52).
// Chef-facing: define choice groups (single/multi, required, price deltas) and
// optionally bundle other dishes into a combo at a fixed price. Mirrors the
// mobile-vendor ModifierComboEditor so both surfaces stay in lock-step.

import { useState } from 'react';
import { motion } from 'framer-motion';
import { Plus, Trash2, X } from 'lucide-react';
import { fadeInUp } from '@/shared/utils/animations';
import { Card } from '@/shared/components/ui/Card';
import { SimpleDialog } from '@/shared/components/ui/Dialog';
import type { ModifierGroupInput, ComboItemInput } from '@/shared/types';

interface Props {
  groups: ModifierGroupInput[];
  setGroups: (g: ModifierGroupInput[]) => void;
  isCombo: boolean;
  setIsCombo: (v: boolean) => void;
  comboItems: ComboItemInput[];
  setComboItems: (c: ComboItemInput[]) => void;
  /** The chef's other menu items, for the combo picker (excludes this item). */
  menuItems: { id: string; name: string }[];
}

const num = (s: string) => parseFloat(s.replace(/[^0-9.-]/g, '')) || 0;

/** A small accessible on/off switch styled with the brand tokens. */
function Toggle({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors ${
        checked ? 'bg-primary' : 'bg-muted-foreground/30'
      }`}
    >
      <span
        className={`inline-block h-5 w-5 transform rounded-full bg-paper shadow transition-transform ${
          checked ? 'translate-x-5' : 'translate-x-0.5'
        }`}
      />
    </button>
  );
}

export function ModifierComboEditor({
  groups,
  setGroups,
  isCombo,
  setIsCombo,
  comboItems,
  setComboItems,
  menuItems,
}: Props) {
  const [picker, setPicker] = useState(false);
  const nameById = (id: string) => menuItems.find((m) => m.id === id)?.name ?? 'Item';

  // ── Group helpers ──
  const addGroup = () =>
    setGroups([
      ...groups,
      { name: '', required: false, minSelect: 0, maxSelect: 1, options: [{ name: '', priceDelta: 0, isAvailable: true }] },
    ]);
  const patchGroup = (gi: number, patch: Partial<ModifierGroupInput>) =>
    setGroups(groups.map((g, i) => (i === gi ? { ...g, ...patch } : g)));
  const removeGroup = (gi: number) => setGroups(groups.filter((_, i) => i !== gi));
  const addOption = (gi: number) => {
    const g = groups[gi];
    if (!g) return;
    patchGroup(gi, { options: [...g.options, { name: '', priceDelta: 0, isAvailable: true }] });
  };
  const patchOption = (gi: number, oi: number, patch: Partial<ModifierGroupInput['options'][number]>) => {
    const g = groups[gi];
    if (!g) return;
    patchGroup(gi, { options: g.options.map((o, i) => (i === oi ? { ...o, ...patch } : o)) });
  };
  const removeOption = (gi: number, oi: number) => {
    const g = groups[gi];
    if (!g) return;
    patchGroup(gi, { options: g.options.filter((_, i) => i !== oi) });
  };

  // ── Combo helpers ──
  const addComboItem = (id: string) => {
    if (comboItems.some((c) => c.menuItemId === id)) return;
    setComboItems([...comboItems, { menuItemId: id, quantity: 1 }]);
    setPicker(false);
  };
  const patchComboQty = (id: string, q: number) =>
    setComboItems(comboItems.map((c) => (c.menuItemId === id ? { ...c, quantity: Math.max(1, q) } : c)));
  const removeComboItem = (id: string) => setComboItems(comboItems.filter((c) => c.menuItemId !== id));

  const inputCls =
    'h-10 w-full rounded-lg border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring';

  return (
    <>
      {/* ADD-ONS */}
      <motion.div variants={fadeInUp}>
        <Card>
          <h2 className="text-lg font-semibold text-foreground">Add-ons</h2>
          <p className="mb-4 mt-1 text-sm text-muted-foreground">
            Offer choices like “Spice level” (single) or “Extras” (multiple). Customers pick these at checkout.
          </p>

          <div className="space-y-3">
            {groups.map((g, gi) => {
              const single = g.maxSelect === 1;
              return (
                <div key={gi} className="rounded-lg border border-border p-3">
                  <div className="flex items-center gap-2">
                    <input
                      className={inputCls}
                      placeholder="Group name (e.g. Spice level)"
                      value={g.name}
                      onChange={(e) => patchGroup(gi, { name: e.target.value })}
                    />
                    <button
                      type="button"
                      onClick={() => removeGroup(gi)}
                      aria-label="Remove group"
                      className="shrink-0 text-muted-foreground hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>

                  <div className="mt-3 flex flex-wrap gap-x-6 gap-y-2">
                    <div className="flex items-center gap-2">
                      <Toggle
                        checked={g.required}
                        onChange={(v) => patchGroup(gi, { required: v })}
                        label="Required"
                      />
                      <span className="text-sm text-foreground">Required</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Toggle
                        checked={single}
                        onChange={(v) => patchGroup(gi, { maxSelect: v ? 1 : 0 })}
                        label="Single choice"
                      />
                      <span className="text-sm text-foreground">Single choice</span>
                    </div>
                  </div>

                  <div className="mt-3 space-y-2">
                    {g.options.map((o, oi) => (
                      <div key={oi} className="flex items-center gap-2">
                        <input
                          className={inputCls}
                          placeholder="Option (e.g. Extra roti)"
                          value={o.name}
                          onChange={(e) => patchOption(gi, oi, { name: e.target.value })}
                        />
                        <div className="flex h-10 w-24 shrink-0 items-center rounded-lg border border-input bg-background px-2">
                          <span className="text-sm text-muted-foreground">₹</span>
                          <input
                            className="w-full bg-transparent px-1 text-center text-sm font-medium focus:outline-none"
                            placeholder="0"
                            inputMode="decimal"
                            value={o.priceDelta ? String(o.priceDelta) : ''}
                            onChange={(e) => patchOption(gi, oi, { priceDelta: num(e.target.value) })}
                          />
                        </div>
                        <button
                          type="button"
                          onClick={() => removeOption(gi, oi)}
                          aria-label="Remove option"
                          className="shrink-0 text-muted-foreground hover:text-destructive"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      </div>
                    ))}
                    <button
                      type="button"
                      onClick={() => addOption(gi)}
                      className="text-sm font-medium text-primary hover:underline"
                    >
                      + Add option
                    </button>
                  </div>
                </div>
              );
            })}

            <button
              type="button"
              onClick={addGroup}
              className="flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
            >
              <Plus className="h-4 w-4" /> Add a group
            </button>
          </div>
        </Card>
      </motion.div>

      {/* COMBO / BUNDLE */}
      <motion.div variants={fadeInUp}>
        <Card>
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-foreground">Combo / Bundle</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Bundle several dishes together. The price above becomes the bundle price.
              </p>
            </div>
            <Toggle checked={isCombo} onChange={setIsCombo} label="This item is a combo" />
          </div>

          {isCombo && (
            <div className="mt-4 space-y-2">
              {comboItems.map((c) => (
                <div key={c.menuItemId} className="flex items-center gap-3 rounded-lg border border-border px-3 py-2">
                  <span className="flex-1 truncate text-sm text-foreground">{nameById(c.menuItemId)}</span>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => patchComboQty(c.menuItemId, c.quantity - 1)}
                      aria-label="Decrease"
                      className="flex h-7 w-7 items-center justify-center rounded-full border border-border text-foreground hover:bg-muted"
                    >
                      −
                    </button>
                    <span className="w-5 text-center text-sm font-medium tabular-nums">{c.quantity}</span>
                    <button
                      type="button"
                      onClick={() => patchComboQty(c.menuItemId, c.quantity + 1)}
                      aria-label="Increase"
                      className="flex h-7 w-7 items-center justify-center rounded-full border border-border text-foreground hover:bg-muted"
                    >
                      +
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeComboItem(c.menuItemId)}
                    aria-label="Remove"
                    className="text-muted-foreground hover:text-destructive"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))}
              <button
                type="button"
                onClick={() => setPicker(true)}
                className="flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
              >
                <Plus className="h-4 w-4" /> Add an item
              </button>
            </div>
          )}
        </Card>
      </motion.div>

      {/* Item picker dialog */}
      <SimpleDialog open={picker} onOpenChange={setPicker} title="Add to combo">
        {menuItems.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Add more dishes to your menu first, then bundle them here.
          </p>
        ) : (
          <div className="max-h-80 space-y-1 overflow-y-auto">
            {menuItems.map((m) => {
              const picked = comboItems.some((c) => c.menuItemId === m.id);
              return (
                <button
                  key={m.id}
                  type="button"
                  disabled={picked}
                  onClick={() => addComboItem(m.id)}
                  className={`flex w-full items-center justify-between rounded-lg border border-border px-3 py-2.5 text-left text-sm ${
                    picked ? 'opacity-40' : 'hover:border-primary/40'
                  }`}
                >
                  <span className="text-foreground">{m.name}</span>
                  {picked ? (
                    <span className="text-xs text-muted-foreground">Added</span>
                  ) : (
                    <Plus className="h-4 w-4 text-primary" />
                  )}
                </button>
              );
            })}
          </div>
        )}
      </SimpleDialog>
    </>
  );
}
