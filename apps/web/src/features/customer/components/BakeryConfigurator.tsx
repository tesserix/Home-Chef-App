// Cake configurator (#1065) — the web twin of the app's BakerySheet. The
// customer picks a size, the baker's option kinds (shape, flavour, egg, sugar),
// optionally a message and an occasion, and sees the live price.
//
// Pricing runs through the same rules the server enforces
// (@homechef/mobile-shared/bakery), so the price shown here is the price
// charged. A rejection from those rules becomes the reason the CTA is off.

import { useMemo, useState } from "react";
import { Check, Minus, Plus } from "lucide-react";
import {
  BAKERY_OCCASIONS,
  BAKERY_OPTION_KINDS,
  BAKERY_OPTION_KIND_LABELS,
  bakerySummary,
  priceBakeryLine,
  weightChoices,
} from "@homechef/mobile-shared/bakery";
import type { BakeryLineInput, MenuItem } from "@/shared/types";
import { Button, SimpleDialog } from "@/shared/components/ui";
import { useFormatPrice } from "@/shared/utils/format-price";

interface BakeryConfiguratorProps {
  item: MenuItem;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (
    config: BakeryLineInput,
    summary: string,
    unitPrice: number,
    quantity: number,
  ) => void;
}

export function BakeryConfigurator({
  item,
  open,
  onOpenChange,
  onConfirm,
}: BakeryConfiguratorProps) {
  const spec = item.bakery ?? null;
  const fp = useFormatPrice();
  const sizes = useMemo(() => (spec ? weightChoices(spec) : []), [spec]);

  const [weightKg, setWeightKg] = useState<number>(0);
  const [picked, setPicked] = useState<Record<string, string>>(() =>
    defaultPicks(item),
  );
  const [message, setMessage] = useState("");
  const [occasion, setOccasion] = useState("");
  const [qty, setQty] = useState(1);

  const size = weightKg || sizes[0] || 0;
  const optionIds = useMemo(
    () => Object.values(picked).filter(Boolean),
    [picked],
  );

  const config: BakeryLineInput = useMemo(
    () => ({
      weightKg: size || undefined,
      optionIds,
      messageOnCake: message.trim() || undefined,
      occasion: occasion || undefined,
    }),
    [size, optionIds, message, occasion],
  );

  const priced = useMemo(() => {
    if (!spec) return null;
    try {
      const { price, snapshot } = priceBakeryLine(spec, item.price, config);
      return {
        price,
        summary: bakerySummary(snapshot),
        serves: snapshot.serves,
        error: null as string | null,
      };
    } catch (err) {
      return {
        price: 0,
        summary: "",
        serves: 0,
        error: err instanceof Error ? err.message : "Choose your options",
      };
    }
  }, [spec, item.price, config]);

  if (!spec || !priced) return null;
  const valid = priced.error === null;

  return (
    <SimpleDialog
      open={open}
      onOpenChange={onOpenChange}
      size="md"
      title={item.name}
    >
      <div className="space-y-5">
        {/* Size — the price driver, so it comes first. */}
        {sizes.length > 0 && (
          <div>
            <div className="mb-2 flex items-center justify-between">
              <span className="font-medium text-ink">Size</span>
              <span className="text-xs text-ink-muted">
                {priced.serves > 0
                  ? `Serves about ${priced.serves}`
                  : `${fp(spec.pricePerKg)} per kg`}
              </span>
            </div>
            <div className="flex flex-wrap gap-2">
              {sizes.map((w) => (
                <button
                  key={w}
                  type="button"
                  role="radio"
                  aria-checked={size === w}
                  aria-label={`${w} kilograms`}
                  onClick={() => setWeightKg(w)}
                  className={`min-h-[44px] rounded-lg border px-4 text-sm tabular-nums text-ink ${
                    size === w ? "border-herb bg-herb-tint" : "border-mist"
                  }`}
                >
                  {w} kg
                </button>
              ))}
            </div>
          </div>
        )}

        {/* One required pick per kind the baker offers. */}
        {BAKERY_OPTION_KINDS.map((kind) => {
          const options = spec.options.filter(
            (o) => o.kind === kind && o.isAvailable,
          );
          if (options.length === 0) return null;
          return (
            <div key={kind}>
              <div className="mb-2 flex items-center justify-between">
                <span className="font-medium text-ink">
                  {BAKERY_OPTION_KIND_LABELS[kind]}
                </span>
                <span className="text-xs text-ink-muted">Required</span>
              </div>
              <div className="space-y-1.5">
                {options.map((o) => {
                  const on = picked[kind] === o.id;
                  const delta =
                    o.priceMode === "per_kg"
                      ? o.priceDelta * (size || 1)
                      : o.priceDelta;
                  const notes = [
                    ...(o.dietaryTags ?? []),
                    ...(o.allergens ?? []).map((a) => `contains ${a}`),
                  ];
                  return (
                    <button
                      key={o.id}
                      type="button"
                      role="radio"
                      aria-checked={on}
                      onClick={() =>
                        setPicked((prev) => ({ ...prev, [kind]: o.id }))
                      }
                      className={`flex w-full items-center gap-3 rounded-lg border px-3 py-2 text-left text-sm ${
                        on ? "border-herb bg-herb-tint" : "border-mist"
                      }`}
                    >
                      <span
                        className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full border ${
                          on ? "border-herb bg-herb" : "border-mist"
                        }`}
                      >
                        {on && (
                          <Check
                            className="h-3 w-3 text-paper"
                            aria-hidden="true"
                          />
                        )}
                      </span>
                      <span className="flex-1">
                        <span className="block text-ink">{o.name}</span>
                        {notes.length > 0 && (
                          <span className="block text-xs text-ink-muted">
                            {notes.join(" · ")}
                          </span>
                        )}
                      </span>
                      {delta !== 0 && (
                        <span className="text-ink-muted tabular-nums">
                          {delta > 0 ? "+" : ""}
                          {fp(delta)}
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          );
        })}

        {/* Message on the bake */}
        {spec.allowMessage && (
          <div>
            <div className="mb-2 flex items-center justify-between">
              <label htmlFor="bakery-message" className="font-medium text-ink">
                Message on it
              </label>
              <span className="text-xs text-ink-muted tabular-nums">
                {message.length}/{spec.maxMessageChars || 40}
              </span>
            </div>
            <input
              id="bakery-message"
              type="text"
              value={message}
              maxLength={spec.maxMessageChars || 40}
              placeholder="Happy Birthday Aarav"
              onChange={(e) => setMessage(e.target.value)}
              className="w-full rounded-lg border border-mist px-3 py-2 text-sm text-ink"
            />
          </div>
        )}

        {/* Occasion — what the bake is for, so the baker can pack for it. */}
        {(spec.occasions?.length ?? 0) > 0 && (
          <div>
            <span className="mb-2 block font-medium text-ink">
              What&apos;s the occasion?
            </span>
            <div className="flex flex-wrap gap-2">
              {BAKERY_OCCASIONS.filter((o) =>
                spec.occasions?.includes(o.value),
              ).map((o) => {
                const on = occasion === o.value;
                return (
                  <button
                    key={o.value}
                    type="button"
                    aria-pressed={on}
                    onClick={() => setOccasion(on ? "" : o.value)}
                    className={`min-h-[40px] rounded-full border px-4 text-sm text-ink ${
                      on ? "border-herb bg-herb-tint" : "border-mist"
                    }`}
                  >
                    {o.label}
                  </button>
                );
              })}
            </div>
          </div>
        )}

        {spec.leadTimeHours > 0 && (
          <p className="text-xs text-ink-muted">
            This bake needs {spec.leadTimeHours} hours&apos; notice — you&apos;ll
            pick a date and time at checkout.
          </p>
        )}

        <div className="flex items-center justify-between">
          <span className="font-medium text-ink">Quantity</span>
          <div className="flex items-center rounded-lg border border-mist">
            <button
              type="button"
              aria-label="Decrease quantity"
              onClick={() => setQty((q) => Math.max(1, q - 1))}
              className="p-2 hover:bg-mist"
            >
              <Minus className="h-4 w-4" aria-hidden="true" />
            </button>
            <span className="w-8 text-center tabular-nums">{qty}</span>
            <button
              type="button"
              aria-label="Increase quantity"
              onClick={() => setQty((q) => q + 1)}
              className="p-2 hover:bg-mist"
            >
              <Plus className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        </div>

        <div>
          <Button
            variant="primary"
            fullWidth
            disabled={!valid}
            onClick={() => {
              if (!valid) return;
              onConfirm(config, priced.summary, priced.price, qty);
            }}
          >
            Add {qty} · {fp(priced.price * qty)}
          </Button>
          <p className="mt-1 text-center text-xs text-ink-muted">
            {priced.error ?? priced.summary}
          </p>
        </div>
      </div>
    </SimpleDialog>
  );
}

// Pre-select the baker's defaults so a customer who wants the standard cake can
// add it in one click.
function defaultPicks(item: MenuItem): Record<string, string> {
  const out: Record<string, string> = {};
  for (const o of item.bakery?.options ?? []) {
    if (o.isDefault && o.isAvailable && !out[o.kind]) out[o.kind] = o.id;
  }
  return out;
}
