/**
 * IST calendar-day helpers for meal-subscription skip eligibility.
 *
 * The server (`apps/api/handlers/meal_subscription.go` Skip handler) decides
 * whether a day is skippable by comparing IST calendar dates: a day is
 * skippable only if it is strictly after today in Asia/Kolkata (`d.After(todayIST)`).
 * These helpers must match that rule bit-for-bit so the client never offers a
 * Skip control the server will reject. See GitHub #696.
 *
 * Fixed UTC+5:30 offset arithmetic only -- India has no DST, so this is safe
 * and avoids relying on device-local `Date` accessors or `Intl`/`timeZone`
 * support (both of which depend on the device's own timezone/locale, which is
 * exactly the bug this fixes).
 */

const IST_OFFSET_MS = 5.5 * 60 * 60 * 1000;

/**
 * istCalendarDate converts an absolute instant to its IST calendar date
 * (`YYYY-MM-DD`), independent of the device's local timezone.
 */
export function istCalendarDate(instant: Date): string {
  const wall = new Date(instant.getTime() + IST_OFFSET_MS);
  const year = wall.getUTCFullYear();
  const month = String(wall.getUTCMonth() + 1).padStart(2, '0');
  const day = String(wall.getUTCDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * isSkippableMealDay mirrors the server's `d.After(todayIST)` check: a
 * fulfillment date is skippable only if its IST calendar day is strictly
 * later than the IST calendar day of `now`. Never true for today or the past.
 */
export function isSkippableMealDay(dateISO: string, now: Date = new Date()): boolean {
  return istCalendarDate(new Date(dateISO)) > istCalendarDate(now);
}

/**
 * isPastMealDay is true only when a fulfillment's IST calendar day is
 * strictly before the IST calendar day of `now`. Distinct from the inverse of
 * isSkippableMealDay: today is neither skippable nor past -- it should still
 * be listed (with no Skip control), which is exactly what this predicate lets
 * callers express (`!isPastMealDay(...)` keeps today in a listing filter,
 * while `isSkippableMealDay(...)` alone gates whether Skip renders).
 */
export function isPastMealDay(dateISO: string, now: Date = new Date()): boolean {
  return istCalendarDate(new Date(dateISO)) < istCalendarDate(now);
}
