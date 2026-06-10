// Centralised, user-facing error message resolver.
//
// NEVER surface raw transport noise like axios's "Request failed with status
// code 404" to a customer. Prefer the backend's human-readable body message
// when it reads cleanly; otherwise fall back to a calm, generic line. Use this
// everywhere an error is shown to the user (Alert, toast, inline error state).

interface ApiErrorBody {
  message?: string;
  error?: string;
}

// Phrases we never want to show a customer, even if they leak into a body.
const TECHNICAL_NOISE = /status code|network error|timeout of|econnrefused|enotfound|xhr|axioserror/i;

/**
 * Resolve a thrown value into a friendly, customer-safe message.
 * @param error  The caught value (unknown).
 * @param fallback  Optional context-specific fallback line.
 */
export function friendlyErrorMessage(
  error: unknown,
  fallback = 'Something went wrong. Please try again.',
): string {
  if (typeof error === 'object' && error !== null) {
    // Axios-style error: prefer the API envelope's message/error field.
    const body = (error as { response?: { data?: ApiErrorBody } }).response?.data;
    const apiMsg = body?.message ?? body?.error;
    if (apiMsg && apiMsg.trim() && !TECHNICAL_NOISE.test(apiMsg)) {
      return apiMsg.trim();
    }

    // A plain Error whose message is human-readable (not transport noise).
    const raw = (error as { message?: unknown }).message;
    if (typeof raw === 'string' && raw.trim() && !TECHNICAL_NOISE.test(raw)) {
      return raw.trim();
    }
  }
  return fallback;
}
