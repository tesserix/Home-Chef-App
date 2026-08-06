// What to do when a live stream dies, given the status it died with.
//
// Every realtime hook used to treat all failures alike: increment, back off,
// redial. That is right for a dropped connection and wrong for a refusal — an
// order with no driver assigned answers 400 no_active_delivery on every attempt,
// so the backoff curve bottoms out at its floor and dials an answer that cannot
// change until the kitchen dispatches (6 such rejections in a two-hour window).

import { socketReconnectDelayWithJitterMs } from '../utils/socket-backoff';

/** How often to re-offer a stream the server refused. */
const REFUSED_RETRY_MS = 30_000;

export interface StreamRetryPlan {
  /** ms to wait before redialling. */
  delayMs: number;
  /** True when the caller should fall back now rather than after a budget. */
  degrade: boolean;
}

/**
 * `status` is the HTTP status from `openEventStream`'s `onError` — 0 when the
 * transport never reached the server.
 *
 * A 4xx is the server's considered answer, so we degrade to the REST fallback
 * immediately and keep only a slow heartbeat open in case the condition clears
 * (a driver is assigned, a token is refreshed). Anything else is treated as
 * transient and retried on the usual jittered curve.
 */
export function streamRetryPlan(
  status: number,
  consecutiveFailures: number,
): StreamRetryPlan {
  if (status >= 400 && status < 500) {
    return { delayMs: REFUSED_RETRY_MS, degrade: true };
  }
  return {
    delayMs: socketReconnectDelayWithJitterMs(consecutiveFailures),
    degrade: false,
  };
}
