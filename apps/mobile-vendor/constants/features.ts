// Build-time feature visibility for the vendor app.
//
// These hide a surface from the UI WITHOUT deleting the feature. The screens,
// hooks and API routes all stay intact and keep compiling, so re-enabling is a
// one-line change here rather than a revert archaeology exercise.
//
// Not a runtime/remote flag: this is deliberate. A remote toggle would need the
// flag plumbed through the API and a fetch before first paint, which is a lot of
// machinery for "don't show this yet". Flipping one of these needs a new build —
// which is the same cost as shipping the feature itself, so nothing is lost.

/**
 * Catering — chef-side event requests, quotes and bookings (#55).
 *
 * Hidden from the vendor UI: the More-tab row is filtered out and /catering
 * redirects, so the screen is unreachable in the app. The code, the
 * useCateringVendor hooks and the /chef/catering API are all untouched and
 * still work — this only controls whether a chef can SEE it.
 *
 * Set to true to bring it back; nothing else needs changing.
 */
export const CATERING_ENABLED = false;
