// Reconnect-delay helper for real-time sockets (order-status, order-tracking,
// vendor live-updates). A socket that gives up permanently after a handful of
// failures leaves the app dead for the rest of the session with no recovery
// short of a restart (#892). The fix is indefinite retry with a capped
// exponential backoff so a dropped connection keeps trying — cheaply once the
// delay reaches its ceiling — while giving connectivity blips (elevator,
// tunnel, carrier handover) a fast first retry.
import { backoffMs } from '../support/outbox';

/**
 * Delay in ms before the next reconnect attempt, given the number of
 * consecutive failures so far. Reuses the outbox resend curve (1s, 2s, 4s,
 * 8s, 16s, capped at 30s) rather than inventing a new one — the shape is
 * already proven for support-chat resends and the same bounds (fast enough
 * to recover quickly, capped low enough to never become a battery-draining
 * hot loop) apply to socket reconnects.
 *
 * Never returns 0 or a negative value: `consecutiveFailures <= 0` is treated
 * as the first attempt.
 */
export function socketReconnectDelayMs(consecutiveFailures: number): number {
  return backoffMs(consecutiveFailures);
}

/** Proportion of the base delay that jitter may subtract (#982). */
const JITTER_FRACTION = 0.3;

/**
 * The same curve with decorrelating jitter. Without it every client that lost
 * the socket to one server-side event retries in the same millisecond, and the
 * synchronised herd is itself capable of knocking the endpoint back over — the
 * dropped connections stay in lockstep because the backoff is deterministic.
 *
 * Jitter only ever subtracts (up to 30%), so the ceiling still holds and the
 * delay can never collapse toward a hot loop.
 *
 * `random` is injectable so the spread can be asserted at its bounds rather
 * than probabilistically.
 */
export function socketReconnectDelayWithJitterMs(
  consecutiveFailures: number,
  random: () => number = Math.random,
): number {
  const base = socketReconnectDelayMs(consecutiveFailures);
  return Math.round(base * (1 - JITTER_FRACTION * random()));
}
