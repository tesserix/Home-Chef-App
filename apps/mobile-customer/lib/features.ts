/**
 * Client-side feature flags for surfaces that are DEFERRED for the v1 launch.
 *
 * Each mirrors a backend flag / feature that isn't live for v1. We hide the
 * customer entry points so the app never leads into a flow that isn't ready.
 * Flip a flag to `true` (and enable any matching backend flag) when the feature
 * actually ships.
 *
 * Typed as `boolean` (not the literal `false`) so conditional branches type-check
 * cleanly. Compile-time constants — promote to a server-read capability if
 * per-tenant control is ever needed.
 */

/** Tiffin meal-plans ("plan a week") + daily tiffin subscription (escrow, UPI Autopay). */
export const TIFFIN_ENABLED: boolean = false;

/** Group / office orders — shared cart, split payment. */
export const GROUP_ORDERS_ENABLED: boolean = false;

/** Catering deposit / advance-order flow. */
export const CATERING_ENABLED: boolean = false;

/** Store-credit wallet — view exists, but spending is off and balances are 0 at launch. */
export const WALLET_ENABLED: boolean = false;

/** Loyalty / rewards program (v2-deferred). */
export const REWARDS_ENABLED: boolean = false;

/** Referral / refer-&-earn program (v2-deferred). */
export const REFERRAL_ENABLED: boolean = false;

/** Social feed / community (stub — no real data yet, v2-deferred). */
export const SOCIAL_ENABLED: boolean = false;
