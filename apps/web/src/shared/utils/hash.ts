/**
 * Map any string (UUID, slug, numeric ID) to a stable index in [0, modulo).
 *
 * Why: `parseInt(uuid)` returns NaN, and `array[NaN]` is `undefined`, so
 * picking a placeholder image with `parseInt(id) % len` silently breaks
 * for every non-numeric id. djb2 is overkill for this and that's the point —
 * deterministic, branch-free, no allocations.
 */
export function hashStringToIndex(input: string, modulo: number): number {
  if (modulo <= 0) return 0;
  if (!input) return 0;
  let h = 5381;
  for (let i = 0; i < input.length; i++) {
    h = ((h << 5) + h + input.charCodeAt(i)) | 0;
  }
  return Math.abs(h) % modulo;
}
