// Test-controllable stand-in: tests assign __nextAuthResult (or a factory via
// __authSessionHandler) to script the browser round trip.
export type AuthSessionResult =
  | { type: "success"; url: string }
  | { type: "cancel" }
  | { type: "dismiss" }
  | { type: "locked" };

export let __lastAuthRequest: { url: string; redirectUri: string } | null = null;
export let __nextAuthResult: AuthSessionResult = { type: "cancel" };
export let __authSessionHandler:
  | ((url: string, redirectUri: string) => AuthSessionResult)
  | null = null;

export function __setNextAuthResult(r: AuthSessionResult): void {
  __nextAuthResult = r;
}
export function __setAuthSessionHandler(
  h: ((url: string, redirectUri: string) => AuthSessionResult) | null
): void {
  __authSessionHandler = h;
}

export async function openAuthSessionAsync(
  url: string,
  redirectUri: string
): Promise<AuthSessionResult> {
  __lastAuthRequest = { url, redirectUri };
  return __authSessionHandler ? __authSessionHandler(url, redirectUri) : __nextAuthResult;
}
