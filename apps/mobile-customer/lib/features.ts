/**
 * Client-side feature flags for surfaces that are DEFERRED for the v1 launch.
 *
 * Each mirrors a backend flag that ships OFF for v1 (MEAL_PLAN_ESCROW_ENABLED /
 * GROUP_ORDERS_ENABLED / CATERING_DEPOSIT_ENABLED — all default false). We hide
 * the customer entry points so the app never leads into a money flow that isn't
 * live yet. Flip a flag to `true` AND enable the matching backend flag when the
 * feature actually ships.
 *
 * These are intentionally simple compile-time constants (not a server-read
 * capability) — the features are fully deferred, so no runtime toggle is needed
 * yet. Promote to a server flag if per-tenant control is ever required.
 */

/** Tiffin meal-plans ("plan a week") + daily tiffin subscription (escrow, UPI Autopay). */
export const TIFFIN_ENABLED = false;

/** Group / office orders — shared cart, split payment. */
export const GROUP_ORDERS_ENABLED = false;

/** Catering deposit / advance-order flow. */
export const CATERING_ENABLED = false;
