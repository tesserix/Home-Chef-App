/**
 * Menu-item form validation reporting.
 *
 * The form already computed per-field errors and rendered them inline, but a
 * failed save gave no other signal: on a long form the offending field is
 * usually scrolled out of view, so "Add item" simply appeared to do nothing.
 * These helpers turn that silence into a message and a scroll target.
 */

/** Errors keyed by form field. A cleared field holds undefined. */
export type MenuItemErrors = Partial<Record<string, string | undefined>>;

/**
 * Fields in the order they appear on the form. Reporting follows this order
 * rather than object key order so the chef is always sent to the topmost
 * problem, which is also the one they will hit first when scrolling up.
 */
const FIELD_ORDER = ['name', 'description', 'price', 'categoryId'] as const;

function populated(errors: MenuItemErrors): string[] {
  return FIELD_ORDER.filter((f) => {
    const v = errors[f];
    return typeof v === 'string' && v.trim() !== '';
  });
}

/** The topmost field with an error, or null when the form is valid. */
export function firstInvalidField(errors: MenuItemErrors): string | null {
  return populated(errors)[0] ?? null;
}

/**
 * A short message for the save-failed toast, or null when the form is valid.
 *
 * One problem reads as itself. Several lead with the topmost problem and say
 * how many there are, so the chef knows to keep scrolling rather than fixing
 * one field and tapping save again.
 */
export function validationSummary(errors: MenuItemErrors): string | null {
  const fields = populated(errors);
  if (fields.length === 0) return null;

  const first = errors[fields[0]!]!;
  if (fields.length === 1) return first;
  return `${first} (${fields.length} fields need attention)`;
}
