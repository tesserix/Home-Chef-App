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

/** Tiffin meal-plans — "plan a week", pre-booked and paid per plan (escrow).
 *  No longer covers the RECURRING daily tiffin subscription: that UI was removed
 *  in #1035, so this flag now gates the pre-book flow only. */
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
 * In-app messaging / "Message support about this order" (#53). Still off, but
 * the old note here ("MongoDB is NOT provisioned") is out of date: the
 * homechef-mongodb replica set is healthy and the API has MONGODB_URI set.
 * Re-verify what actually blocks messaging before flipping this.
 */
export const MESSAGING_ENABLED: boolean = false;

/**
 * The chefs map (`app/chefs-map.tsx`) — kitchens plotted on a Google map.
 * Off because no Maps SDK key exists for this project, not because the screen
 * is unfinished.
 *
 * GCP `tesseracthub-480811` has exactly one maps-adjacent key,
 * `homechef-routes-api`, held in the secret `prod-homechef-google-maps-api-key`
 * and restricted to `routes.googleapis.com`. It is a SERVER key and is NOT
 * usable here — shipping it in a public binary hands anyone who unzips the APK
 * a billable Routes API key. `maps-android-backend.googleapis.com` is not an
 * enabled service, so no valid key can be minted today either. With no
 * `GOOGLE_MAPS_API_KEY` in any EAS environment, `withMapsKey`
 * (`lib/maps-config.js`) no-ops and the Android manifest carries no
 * `com.google.android.geo.API_KEY` — inflating a `MapView` then throws
 * `RuntimeException: API key not found`.
 *
 * Unblock: enable `maps-android-backend.googleapis.com`, mint a NEW key
 * restricted to Maps SDK for Android + the `com.tesserix.homechef.customer`
 * SHA-1s, publish it as `GOOGLE_MAPS_API_KEY` via `eas secret:create`, then
 * flip this to `true`. See `docs/store-release/README.md`.
 *
 * Scope: this gates the chefs map only. `DeliveryMap` in order tracking is a
 * separate surface behind its own `showMap` gate and is untouched by this flag.
 */
export const CHEFS_MAP_ENABLED: boolean = false;
