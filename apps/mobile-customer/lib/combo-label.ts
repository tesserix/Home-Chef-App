// combo-label.ts — the daily-menu bundle is called a "Thali" in India and a
// "Combo" internationally (#406). The API/model term is neutral (`isCombo`); we
// localize the DISPLAY label by the user's country. HomeChef is India-first
// (INR/IST), so India — or an unknown country — shows "Thali"; an explicitly
// non-IN country shows "Combo".

/** Display label for a daily-menu combo/bundle, localized by country code. */
export function comboLabel(country?: string | null): string {
  if (!country) return 'Thali';
  return country.trim().toUpperCase() === 'IN' ? 'Thali' : 'Combo';
}
