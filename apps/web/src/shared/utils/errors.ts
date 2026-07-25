// Centralised, user-facing error message resolver for errors thrown by
// apiClient. Mirrors apps/mobile-customer/lib/errors.ts's friendlyErrorMessage
// — same purpose, adapted to this app's error shape.
//
// apiClient throws the parsed error body with the HTTP status attached (see
// api-client.ts). Handlers on the Go side mostly return a flat
// `{ error: "some message" }`, but a few endpoints (and the ApiError type)
// use a nested `{ error: { message: "some message" } }` — LoyaltyPage.tsx
// already reads both shapes inline; this gives every call site the same
// logic instead of repeating it.
//
// Use this anywhere a caught apiClient error is shown to the customer —
// especially on money flows, where a blanket "something went wrong" hides a
// server outcome the customer needs to know (e.g. "cancelled, but the refund
// could not be issued").

interface FlatApiErrorBody {
  error?: unknown;
}

interface NestedApiErrorBody {
  error?: { message?: unknown };
}

/**
 * Resolve a thrown value (from apiClient) into a friendly, customer-safe
 * message.
 *
 * Deliberately does NOT fall back to a bare `.message` — that field isn't
 * server-provided. For an apiClient error it's absent (the server payload
 * lives under `error` / `error.message`, handled below); for a fetch-level
 * failure (offline, DNS, CORS) it's a `TypeError` whose `.message` is the
 * literal browser string `"Failed to fetch"`, which is not something a
 * customer should ever see. Anything without a real server error payload
 * falls through to the caller's fallback instead.
 * @param error  The caught value (unknown).
 * @param fallback  Shown when the body carries nothing usable.
 */
export function friendlyErrorMessage(
  error: unknown,
  fallback = 'Something went wrong. Please try again.',
): string {
  if (typeof error === 'object' && error !== null) {
    const flat = (error as FlatApiErrorBody).error;
    if (typeof flat === 'string' && flat.trim()) {
      return flat.trim();
    }

    const nested = (error as NestedApiErrorBody).error?.message;
    if (typeof nested === 'string' && nested.trim()) {
      return nested.trim();
    }
  }
  return fallback;
}
