import type { SurgeFactors } from "../hooks/useDeliveryQuote";

// Plain-language reasons for a raised delivery fee. "×1.28 surge" tells a customer
// nothing; "heavy traffic and rain right now" tells them why, and that it will pass.

/** Below this a factor is rounding noise, not a condition worth naming. */
const NOTABLE = 1.05;

/**
 * Human-readable reasons behind a surged delivery fee, strongest first, so the
 * dominant condition leads the sentence.
 */
export function surgeReasons(surge?: SurgeFactors): string[] {
  if (!surge) return [];
  return (
    [
      { factor: surge.weather, label: "poor weather" },
      { factor: surge.traffic, label: "heavy traffic" },
      { factor: surge.fuel, label: "higher fuel prices" },
    ] as const
  )
    .filter((r) => r.factor > NOTABLE)
    .sort((a, b) => b.factor - a.factor)
    .map((r) => r.label);
}

/**
 * One sentence explaining a raised fee, or null when conditions are normal (so
 * the caller renders nothing rather than an empty row).
 */
export function surgeReasonText(surge?: SurgeFactors): string | null {
  const reasons = surgeReasons(surge);
  if (reasons.length === 0) return null;

  const last = reasons[reasons.length - 1];
  const joined =
    reasons.length === 1
      ? reasons[0]
      : `${reasons.slice(0, -1).join(", ")} and ${last}`;
  return `Higher than usual right now — ${joined}.`;
}
