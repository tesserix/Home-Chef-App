/** Narrows a picker list by a typed query, preserving the list's own order. */
export function filterOptions<T>(
  options: readonly T[],
  query: string,
  label: (option: T) => string = String,
): T[] {
  const q = query.trim().toLowerCase();
  if (!q) return [...options];
  return options.filter((o) => label(o).toLowerCase().includes(q));
}
