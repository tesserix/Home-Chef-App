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
export const TIFFIN_ENABLED: boolean = true;

/**
 * Group / office orders — shared cart, split payment. Off: the split-pay half
 * isn't live, so the entry point stays hidden rather than leading customers
 * into a checkout that can't complete (#875).
 *
 * This gates *starting* a group, not joining one — invite deep-links
 * (`homechef-customer://group/<token>`) and the group hub stay reachable so
 * groups created before this flip still work.
 */
export const GROUP_ORDERS_ENABLED: boolean = false;

/** Catering deposit / advance-order flow. */
export const CATERING_ENABLED: boolean = false;

/**
 * Store-credit wallet — the balance view. On for all customers: refunds from
 * cancelled / undelivered meal-plan days (and any goodwill credit) land in the
 * wallet, so the customer must be able to SEE what they're owed. Spending that
 * balance at checkout is a separate gate (WALLET_CHECKOUT_ENABLED / the API's
 * WALLET_CHECKOUT_ENABLED) that must move in lockstep with the server.
 */
export const WALLET_ENABLED: boolean = true;

/** Loyalty / rewards program. */
export const REWARDS_ENABLED: boolean = true;

/** Referral / refer-&-earn program (v2-deferred). */
export const REFERRAL_ENABLED: boolean = true;

/**
 * Social feed / community — ChefBook. Wired to /api/v1/social; verified
 * returning 200 against prod before switching on. Keep in lockstep with
 * apps/web/src/shared/config/features.ts.
 */
export const SOCIAL_ENABLED: boolean = true;

/**
 * In-app messaging / "Message support about this order" (#53). Still off, but
 * the old note here ("MongoDB is NOT provisioned") is out of date: the
 * homechef-mongodb replica set is healthy and the API has MONGODB_URI set.
 * Re-verify what actually blocks messaging before flipping this.
 */
export const MESSAGING_ENABLED: boolean = false;
