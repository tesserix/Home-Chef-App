// expo-apple-authentication and expo-crypto are loaded LAZILY inside the
// functions below — never at module top level. This file is re-exported from the
// auth barrel (./index.ts) that every app imports at startup, and both are
// iOS-only NATIVE modules. A top-level import made app STARTUP crash on BOTH
// platforms with "Cannot find native module 'ExpoCrypto'" whenever the native
// build didn't link them — e.g. a build predating this dependency, or Android,
// which never offers Sign in with Apple. Deferring the load to an actual Apple
// sign-in tap keeps startup free of these native modules on every platform.
import type { FirebaseAuthTypes } from '@react-native-firebase/auth';

import { signInWithAppleCredential } from './sign-in';

// apple.ts — Sign in with Apple, with the nonce Apple and Firebase both expect.
//
// Every app was calling signInWithAppleCredential(token, '', name): no nonce at
// all. Apple's identity token then carries no binding to the request that
// produced it, so a token captured once can be replayed to sign in as that user
// later, or into a different app. It "works" without one, which is exactly why
// six copies of it accumulated.
//
// The protocol is:
//
//   1. generate a random RAW nonce
//   2. send SHA-256(raw) to Apple  — it embeds that in the identity token
//   3. send the RAW nonce to Firebase, which hashes it and compares
//
// Firebase rejects the exchange if they don't match, so a replayed token fails
// against a fresh nonce.
//
// This lives in the shared package because it was duplicated across all four
// apps; a security detail with six copies is a security detail with six chances
// to drift.

/** Bytes of entropy in the raw nonce. 32 is what Apple's own sample uses. */
const NONCE_BYTES = 32;

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * Run the full Apple sign-in exchange and return the Firebase credential.
 *
 * Throws if the user cancels (Apple raises ERR_REQUEST_CANCELED) or if Apple
 * returns no identity token. Callers handle those the same way they handle any
 * other sign-in failure.
 */
export async function signInWithApple(): Promise<FirebaseAuthTypes.UserCredential> {
  // Native modules — loaded here, not at module top level (see header comment),
  // so app startup never touches them. This runs only when the user taps
  // Sign in with Apple, which the UI offers on iOS only.
  const AppleAuthentication = await import('expo-apple-authentication');
  const Crypto = await import('expo-crypto');

  // A CSPRNG, not Math.random: the whole value of a nonce is that an attacker
  // cannot predict the next one.
  const rawNonce = toHex(await Crypto.getRandomBytesAsync(NONCE_BYTES));
  const hashedNonce = await Crypto.digestStringAsync(
    Crypto.CryptoDigestAlgorithm.SHA256,
    rawNonce,
  );

  const credential = await AppleAuthentication.signInAsync({
    requestedScopes: [
      AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
      AppleAuthentication.AppleAuthenticationScope.EMAIL,
    ],
    // Apple embeds this hash in the identity token's `nonce` claim.
    nonce: hashedNonce,
  });

  if (!credential.identityToken) {
    throw new Error('Apple sign-in failed: no identity token');
  }

  // Firebase hashes the raw nonce and compares it against that claim.
  //
  // fullName is only ever populated on the FIRST authorization for a given
  // Apple ID — on every later sign-in Apple returns null for it. That is why the
  // name is captured here and persisted onto the Firebase profile rather than
  // being re-read on subsequent logins.
  return signInWithAppleCredential(credential.identityToken, rawNonce, credential.fullName);
}

/** True when the device can offer Sign in with Apple (iOS 13+ hardware). */
export async function isAppleSignInAvailable(): Promise<boolean> {
  try {
    // Lazy load (see header): on Android or a build without the native module the
    // import throws, and this correctly reports Apple sign-in as unavailable
    // instead of crashing.
    const AppleAuthentication = await import('expo-apple-authentication');
    return await AppleAuthentication.isAvailableAsync();
  } catch {
    return false;
  }
}
