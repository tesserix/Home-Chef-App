import auth, { FirebaseAuthTypes } from "@react-native-firebase/auth";

export async function signInWithGoogleCredential(idToken: string, accessToken?: string) {
  const cred = auth.GoogleAuthProvider.credential(idToken, accessToken);
  return auth().signInWithCredential(cred);
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
  const cred = auth.AppleAuthProvider.credential(idToken, rawNonce);
  const result = await auth().signInWithCredential(cred);

  const displayName = buildDisplayName(fullName);
  if (displayName && !result.user.displayName) {
    try {
      await result.user.updateProfile({ displayName });
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
  return auth().signInWithEmailAndPassword(email, password);
}

export async function registerWithEmail(email: string, password: string) {
  return auth().createUserWithEmailAndPassword(email, password);
}

export async function startPhoneSignIn(phone: string) {
  return auth().verifyPhoneNumber(phone);
}

export async function getIdToken(forceRefresh = false): Promise<string | null> {
  const u = auth().currentUser;
  return u ? u.getIdToken(forceRefresh) : null;
}

export async function signOut(): Promise<void> {
  return auth().signOut();
}

export async function sendPasswordResetEmail(email: string): Promise<void> {
  return auth().sendPasswordResetEmail(email);
}
