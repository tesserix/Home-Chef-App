// Modular API throughout (v22+). The namespaced `auth().x()` shape still works
// but logs a deprecation warning on every call, which buried real errors.
import {
  AppleAuthProvider,
  GoogleAuthProvider,
  createUserWithEmailAndPassword,
  getAuth,
  getIdToken as getFirebaseIdToken,
  sendPasswordResetEmail as sendFirebasePasswordResetEmail,
  signInWithCredential,
  signInWithEmailAndPassword,
  signOut as firebaseSignOut,
  updateProfile,
  verifyPhoneNumber,
} from "@react-native-firebase/auth";
import type { FirebaseAuthTypes } from "@react-native-firebase/auth";
import {
  clearDevSimSession,
  devSimSignIn,
  getDevSimIdToken,
  isKeychainError,
} from "./dev-sim-auth";

export async function signInWithGoogleCredential(idToken: string, accessToken?: string) {
  const cred = GoogleAuthProvider.credential(idToken, accessToken);
  return signInWithCredential(getAuth(), cred);
}

/**
 * Apple's full name, surfaced ONLY on the very first authorization. Both parts
 * are nullable. Matches the shape of Expo's AppleAuthentication credential.fullName.
 */
export interface AppleFullName {
  givenName?: string | null;
  familyName?: string | null;
}

/**
 * Sign in with an Apple identity token. Apple returns the user's name only on
 * the FIRST authorization, so when present we backfill the Firebase user's
 * displayName (best-effort) so the name flows into subsequent ID tokens.
 * @param idToken - Apple identity token from AppleAuthentication.signInAsync
 * @param rawNonce - raw nonce (empty string when not generated)
 * @param fullName - Apple's first-authorization full name (optional, nullable parts)
 */
export async function signInWithAppleCredential(
  idToken: string,
  rawNonce: string,
  fullName?: AppleFullName | null
): Promise<FirebaseAuthTypes.UserCredential> {
  const cred = AppleAuthProvider.credential(idToken, rawNonce);
  const result = await signInWithCredential(getAuth(), cred);

  const displayName = buildDisplayName(fullName);
  if (displayName && !result.user.displayName) {
    try {
      await updateProfile(result.user, { displayName });
    } catch {
      // Best-effort: name capture is non-fatal. The user remains signed in.
    }
  }
  return result;
}

/**
 * Build a "given family" display name from Apple's full name, trimming blanks
 * and collapsing to a single space. Returns empty string when no parts exist.
 */
function buildDisplayName(fullName?: AppleFullName | null): string {
  if (!fullName) return "";
  const parts = [fullName.givenName, fullName.familyName]
    .map((p) => (p ?? "").trim())
    .filter((p) => p.length > 0);
  return parts.join(" ");
}

export async function signInWithEmail(email: string, password: string) {
  try {
    return await signInWithEmailAndPassword(getAuth(), email, password);
  } catch (err) {
    // iOS Simulator has no keychain entitlement; fall back to GIP REST in dev.
    if (__DEV__ && isKeychainError(err)) {
      await devSimSignIn(getAuth().tenantId ?? "", email, password);
      return null;
    }
    throw err;
  }
}

export async function registerWithEmail(email: string, password: string) {
  return createUserWithEmailAndPassword(getAuth(), email, password);
}

export async function startPhoneSignIn(phone: string) {
  // 60s is the auto-verify timeout the namespaced call defaulted to; modular
  // makes it a required argument.
  return verifyPhoneNumber(getAuth(), phone, 60);
}

export async function getIdToken(forceRefresh = false): Promise<string | null> {
  const u = getAuth().currentUser;
  if (u) return getFirebaseIdToken(u, forceRefresh);
  return __DEV__ ? getDevSimIdToken() : null;
}

export async function signOut(): Promise<void> {
  clearDevSimSession();
  return firebaseSignOut(getAuth());
}

export async function sendPasswordResetEmail(email: string): Promise<void> {
  return sendFirebasePasswordResetEmail(getAuth(), email);
}

/**
 * Provider IDs linked to the currently signed-in Firebase user, e.g.
 * 'password', 'google.com', 'apple.com'. Empty array when signed out.
 */
export function getLinkedProviderIds(): string[] {
  return getAuth().currentUser?.providerData.map((p) => p.providerId) ?? [];
}

/**
 * True only when the account has an email/password credential — i.e. the user
 * actually has a password to change. Google/Apple (SSO) accounts have no
 * password, so password-change UI (Change password / reset-email) must stay
 * hidden for them. A password provider linked alongside a social one still
 * counts, since such a user can genuinely change their password.
 */
export function hasPasswordProvider(): boolean {
  return getLinkedProviderIds().includes("password");
}

/**
 * Ask the Fe3dr API to email a password-reset link.
 *
 * Deliberately NOT Firebase's own sendPasswordResetEmail. Firebase mints a
 * valid token but delivers it from noreply@<project>.firebaseapp.com — an
 * unauthenticated domain Gmail files as spam — branded with the GCP project
 * name and containing a raw firebaseapp.com URL that reads as phishing.
 *
 * Our API mints the same Firebase token server-side and sends it through the
 * platform's authenticated sender with the Fe3dr template, wrapped in a
 * single-use link that expires in 15 minutes.
 *
 * `app` selects the Identity Platform tenant. It is required, and it matters:
 * accounts are tenant-scoped, so a chef's address genuinely does not exist in
 * the customer tenant. Sending the wrong one is exactly how a reset request
 * ends in a "check your inbox" message for a mail that was never sent.
 *
 * Resolves on every outcome the server treats as normal — including "no such
 * account" — because the server answers identically either way (anti-
 * enumeration). Rejects only when the request itself could not be delivered,
 * so the UI can distinguish "we've sent it if it exists" from "we're down".
 */
export async function requestPasswordReset(
  apiUrl: string,
  email: string,
  app: "customer" | "vendor" | "delivery"
): Promise<void> {
  const res = await fetch(`${apiUrl.replace(/\/$/, "")}/auth/password-reset/request`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, app }),
  });
  if (!res.ok) {
    throw new Error(
      "We couldn't send the reset email just now. Please check your connection and try again."
    );
  }
}
