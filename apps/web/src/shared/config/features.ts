/**
 * Client-side feature flags for surfaces DEFERRED for the v1 launch.
 *
 * This is the web mirror of `apps/mobile-customer/lib/features.ts`. The two
 * files MUST be kept in lockstep: the web had no flags at all, so it happily
 * advertised deferred surfaces in its navigation while the mobile app
 * deliberately hid them — customers on web were being led into flows the
 * product has deferred.
 *
 * Each flag mirrors a backend flag / feature that isn't live for v1. Hiding the
 * entry point is the point: the routes still exist, so anyone holding a deep
 * link still lands somewhere sensible, but nothing in the UI leads there.
 *
 * Flip a flag to `true` here AND in the mobile file (and enable any matching
 * backend flag) when the feature actually ships.
 *
 * Typed as `boolean` rather than the literal so conditional branches
 * type-check cleanly instead of narrowing to `never`.
 */

/** Tiffin meal-plans ("plan a week") + daily tiffin subscription. */
export const TIFFIN_ENABLED: boolean = true;

/** Group / office orders — shared cart, split payment. */
export const GROUP_ORDERS_ENABLED: boolean = true;

/** Catering deposit / advance-order flow. Deferred for v1. */
export const CATERING_ENABLED: boolean = false;

/** Store-credit wallet — the balance view. */
export const WALLET_ENABLED: boolean = true;

/** Loyalty / rewards program. */
export const REWARDS_ENABLED: boolean = true;

/** Referral / refer-&-earn program. */
export const REFERRAL_ENABLED: boolean = true;

/**
 * In-app messaging. Still off, but NOT for the reason previously recorded
 * here: MongoDB *is* provisioned — homechef-mongodb is a healthy 3-node
 * replica set and the API has MONGODB_URI configured. Whatever remains before
 * messaging can ship needs re-checking rather than assuming the old note.
 */
export const MESSAGING_ENABLED: boolean = false;
