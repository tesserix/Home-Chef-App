import auth from "@react-native-firebase/auth";

export async function signInWithGoogleCredential(idToken: string, accessToken?: string) {
  const cred = auth.GoogleAuthProvider.credential(idToken, accessToken);
  return auth().signInWithCredential(cred);
}

export async function signInWithAppleCredential(idToken: string, rawNonce: string) {
  const cred = auth.AppleAuthProvider.credential(idToken, rawNonce);
  return auth().signInWithCredential(cred);
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
