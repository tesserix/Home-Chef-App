import { useState } from 'react';
import { Plus, X } from 'lucide-react';
import { useMenuItemOptions } from '../hooks/useMenuItemOptions';

// What's in the thali.
//
// Components are stored as NAMES, not ids — the API field is []string, and a
// thali legitimately contains things that are not menu items ("2 rotis",
// "salad", "pickle"). So this picks from the menu where it can and accepts
// typing where it can't, and both end up as the same plain list. That keeps the
// customer-facing "Includes: …" line honest without forcing a chef to create a
// menu item for pickle.

interface ComboComponentsProps {
  value: string[];
  onChange: (next: string[]) => void;
}

export function ComboComponents({ value, onChange }: ComboComponentsProps) {
  const { data: options = [] } = useMenuItemOptions();
  const [custom, setCustom] = useState('');
  const [showPicker, setShowPicker] = useState(false);

  // Case-insensitive, so "Rice" and "rice" can't both end up in one thali.
  function add(name: string) {
    const trimmed = name.trim();
    if (!trimmed) return;
    if (value.some((v) => v.toLowerCase() === trimmed.toLowerCase())) return;
    onChange([...value, trimmed]);
  }

  const unused = options.filter(
    (o) => !value.some((v) => v.toLowerCase() === o.name.toLowerCase()),
  );

  return (
    <div>
      <span className="mb-1.5 block text-xs font-medium text-ink-soft">What&apos;s included</span>

      {value.length > 0 && (
        <ul className="mb-2 flex flex-wrap gap-1.5">
          {value.map((c) => (
            <li key={c}>
              <span className="inline-flex items-center gap-1 rounded-lg bg-mist px-2.5 py-1 text-xs text-foreground">
                {c}
                <button
                  type="button"
                  onClick={() => onChange(value.filter((v) => v !== c))}
                  aria-label={`Remove ${c}`}
                  className="rounded p-0.5 text-ink-muted transition-colors hover:text-paprika"
                >
                  <X className="h-3 w-3" aria-hidden="true" />
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      <div className="flex flex-wrap items-center gap-2">
        {unused.length > 0 && (
          <div className="relative">
            <button
              type="button"
              onClick={() => setShowPicker((s) => !s)}
              aria-expanded={showPicker}
              className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-mist px-2.5 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-paper"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden="true" />
              From my menu
            </button>
            {showPicker && (
              <>
                <button
                  type="button"
                  aria-hidden="true"
                  tabIndex={-1}
                  className="fixed inset-0 z-10 cursor-default"
                  onClick={() => setShowPicker(false)}
                />
                <div className="absolute left-0 top-full z-20 mt-1 max-h-52 w-56 overflow-auto rounded-lg border border-mist bg-bone py-1 shadow-3">
                  {unused.map((o) => (
                    <button
                      key={o.id}
                      type="button"
                      onClick={() => {
                        add(o.name);
                        setShowPicker(false);
                      }}
                      className="block w-full truncate px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-paper"
                    >
                      {o.name}
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>
        )}

        <input
          value={custom}
          onChange={(e) => setCustom(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              add(custom);
              setCustom('');
            }
          }}
          placeholder="or type — e.g. 2 rotis, salad"
          aria-label="Add an item that isn't on your menu"
          className="input-base h-9 flex-1 min-w-[12rem] text-sm"
        />
      </div>
      <p className="mt-1.5 text-xs text-ink-muted">
        Customers see this as &ldquo;Includes: {value.length ? value.join(', ') : '…'}&rdquo;
      </p>
    </div>
  );
}

/**
 * The one-line answer to "what am I actually selling here?".
 *
 * A bare "140" in a box says nothing — not the currency, not how much food, not
 * what it works out to per head. This states the whole thing in the order a chef
 * reasons about it: total, portion, how many it feeds, and the per-person figure
 * DERIVED rather than typed, so the two can never disagree.
 */
export function cellSummary(price: number, portionSize: string, serves: number): string {
  const bits = [`₹${Math.round(price)}`];
  if (portionSize.trim()) bits.push(portionSize.trim());
  bits.push(serves > 1 ? `feeds ${serves}` : 'single portion');
  let out = bits.join(' · ');
  // Only worth showing when it differs from the total — "₹140 per person" under
  // "₹140" is noise.
  if (serves > 1 && price > 0) out += `  →  ₹${Math.round(price / serves)} per person`;
  return out;
}

export function PriceSummary({
  price,
  portionSize,
  serves,
}: {
  price: number;
  portionSize: string;
  serves: number;
}) {
  if (!price) return null;
  return (
    <p className="mt-2 text-xs text-ink-soft tabular-nums">
      {cellSummary(price, portionSize, serves)}
    </p>
  );
}

interface PortionFieldsProps {
  portionSize: string;
  serves: number;
  onChange: (patch: { portionSize?: string; serves?: number }) => void;
}

/**
 * How much food, and for how many.
 *
 * Both matter for a tiffin: a price with no portion tells a customer nothing
 * about value, and "serves" is what makes a per-person figure meaningful.
 */
export function PortionFields({ portionSize, serves, onChange }: PortionFieldsProps) {
  const perPerson = serves > 1;
  return (
    <div className="flex flex-wrap gap-2">
      <div className="min-w-[8rem] flex-1">
        <label className="mb-1 block text-xs font-medium text-ink-soft">Portion</label>
        <input
          value={portionSize}
          onChange={(e) => onChange({ portionSize: e.target.value })}
          placeholder="e.g. 500 ml, 2 rotis"
          className="input-base h-10 text-sm"
        />
      </div>
      <div className="w-28">
        <label className="mb-1 block text-xs font-medium text-ink-soft">Serves</label>
        <input
          type="number"
          min={1}
          max={50}
          inputMode="numeric"
          value={serves}
          onChange={(e) => onChange({ serves: Math.max(1, Number(e.target.value) || 1) })}
          className="input-base h-10 text-sm tabular-nums"
        />
      </div>
      {perPerson && (
        <p className="w-full text-xs text-ink-muted">
          Customers see this as feeding {serves} people.
        </p>
      )}
    </div>
  );
}
