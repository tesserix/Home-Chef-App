import { useMemo, useState } from 'react';
import { ChevronDown, Pencil, Check } from 'lucide-react';
import {
  optionsForVariant,
  prefillFrom,
  useMenuItemOptions,
  type MenuItemOption,
  type PrefillFromMenuItem,
} from '../hooks/useMenuItemOptions';

// Pick a dish from the menu you already built, instead of typing it again.
//
// Two modes on purpose. PICK is the default because the answer is nearly always
// already on the chef's menu, and picking carries the price, portion, serves,
// tags and allergens across for free. TYPE stays available because a tiffin
// often includes something the chef never sells à la carte — refusing that would
// force junk items onto the public menu just to fill a plan cell.

interface DishPickerProps {
  variant: 'veg' | 'nonveg';
  /** Current cell values, so the control reflects what is already saved. */
  value: { name: string; menuItemId?: string | null };
  onPick: (prefill: PrefillFromMenuItem) => void;
  /** Free-typed name — clears menuItemId, because it is no longer that item. */
  onTypeName: (name: string) => void;
  label?: string;
  placeholder?: string;
}

export function DishPicker({
  variant,
  value,
  onPick,
  onTypeName,
  label,
  placeholder = 'Choose a dish',
}: DishPickerProps) {
  const { data: all = [], isLoading } = useMenuItemOptions();
  const options = useMemo(() => optionsForVariant(all, variant), [all, variant]);

  // Start in whichever mode matches the saved cell: a linked item opens as a
  // pick, a hand-typed name opens as text, so nothing the chef saved is
  // silently reinterpreted.
  const [typing, setTyping] = useState(() => !value.menuItemId && !!value.name);
  const [open, setOpen] = useState(false);

  const picked = options.find((o) => o.id === value.menuItemId);

  if (typing) {
    return (
      <div>
        {label && <span className="mb-1 block text-xs font-medium text-ink-soft">{label}</span>}
        <input
          value={value.name}
          onChange={(e) => onTypeName(e.target.value)}
          placeholder="Dish name"
          className="input-base"
          autoFocus
        />
        {options.length > 0 && (
          <button
            type="button"
            onClick={() => setTyping(false)}
            className="mt-1 text-xs font-medium text-herb hover:underline"
          >
            Choose from my menu instead
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="relative">
      {label && <span className="mb-1 block text-xs font-medium text-ink-soft">{label}</span>}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="flex min-h-11 w-full items-center justify-between gap-2 rounded-lg border border-mist bg-bone px-3 py-2 text-left transition-colors hover:bg-paper"
      >
        <span className="min-w-0 flex-1">
          {value.name ? (
            <>
              <span className="block truncate text-sm font-medium text-foreground">
                {value.name}
              </span>
              {picked && (
                <span className="block truncate text-xs text-ink-muted tabular-nums">
                  ₹{picked.price}
                  {picked.portionSize ? ` · ${picked.portionSize}` : ''}
                  {picked.serves > 1 ? ` · serves ${picked.serves}` : ''}
                </span>
              )}
              {/* A name with no link is something the chef typed. Saying so beats
                  showing an identical-looking row that behaves differently. */}
              {!picked && value.name && (
                <span className="block text-xs text-ink-muted">Custom dish</span>
              )}
            </>
          ) : (
            <span className="text-sm text-ink-muted">
              {isLoading ? 'Loading your menu…' : placeholder}
            </span>
          )}
        </span>
        <ChevronDown className="h-4 w-4 shrink-0 text-ink-muted" aria-hidden="true" />
      </button>

      {open && (
        <>
          {/* Click-away catcher. Cheaper and more reliable here than a document
              listener, since this control is rendered many times per page. */}
          <button
            type="button"
            aria-hidden="true"
            tabIndex={-1}
            className="fixed inset-0 z-10 cursor-default"
            onClick={() => setOpen(false)}
          />
          <div
            role="listbox"
            className="absolute left-0 right-0 top-full z-20 mt-1 max-h-64 overflow-auto rounded-lg border border-mist bg-bone py-1 shadow-3"
          >
            {options.length === 0 ? (
              <p className="px-3 py-3 text-sm text-ink-soft">
                Nothing on your menu yet — type the dish instead.
              </p>
            ) : (
              options.map((o) => (
                <OptionRow
                  key={o.id}
                  option={o}
                  selected={o.id === value.menuItemId}
                  onSelect={() => {
                    onPick(prefillFrom(o));
                    setOpen(false);
                  }}
                />
              ))
            )}
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                setTyping(true);
              }}
              className="mt-1 flex w-full items-center gap-2 border-t border-mist px-3 py-2.5 text-left text-sm font-medium text-herb transition-colors hover:bg-paper"
            >
              <Pencil className="h-4 w-4" aria-hidden="true" />
              Type a dish that isn&apos;t on my menu
            </button>
          </div>
        </>
      )}
    </div>
  );
}

function OptionRow({
  option,
  selected,
  onSelect,
}: {
  option: MenuItemOption;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      role="option"
      aria-selected={selected}
      onClick={onSelect}
      className={`flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-paper ${
        selected ? 'bg-herb-tint' : ''
      }`}
    >
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium text-foreground">{option.name}</span>
        <span className="block truncate text-xs text-ink-muted tabular-nums">
          ₹{option.price}
          {option.portionSize ? ` · ${option.portionSize}` : ''}
          {option.serves > 1 ? ` · serves ${option.serves}` : ''}
          {/* An unavailable dish is still pickable: a plan is a schedule, and a
              chef may well turn it back on before the day it is served. */}
          {option.isAvailable === false ? ' · currently off menu' : ''}
        </span>
      </span>
      {selected && <Check className="h-4 w-4 shrink-0 text-herb" aria-hidden="true" />}
    </button>
  );
}
