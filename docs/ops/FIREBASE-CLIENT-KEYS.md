# Firebase client identifier policy

The tracked `apps/mobile-*/google-services.json` files and `EXPO_PUBLIC_*`
Firebase values in `apps/mobile-*/eas.json` contain Firebase client
configuration. Firebase API keys in a mobile bundle identify the Firebase
project; they are not server credentials and cannot be kept secret from someone
who installs the app. The repository secret scan therefore classifies these as
public client identifiers, not leaked authentication secrets.

This classification does **not** make an unrestricted key safe. Authorization
must come from verified Firebase Authentication tokens, Firebase Security Rules,
App Check, and backend object-level authorization. A client key must never grant
privileged API access by itself.

## Required controls

- Keep service-account JSON, private keys, OAuth client secrets, KMS material,
  and Admin SDK credentials out of every mobile config and frontend bundle.
- Use a dedicated Google Cloud API key for each released app/environment where
  practical. Do not reuse a server-side Maps, Routes, Weather, or other billable
  API key in Firebase client configuration.
- Apply API restrictions in Google Cloud to only the Firebase APIs the app
  actually calls. For Firebase Authentication this includes Identity Toolkit;
  confirm the complete allowlist against the app's enabled Firebase products.
- Apply Android application restrictions for the exact package name and release
  signing-certificate fingerprint. If an iOS key is used, restrict it to the
  exact bundle identifier. Preserve separate development and release signing
  identities.
- Enable Firebase App Check for supported products and enforce it after metrics
  show legitimate release builds are presenting valid attestations.
- Keep Firebase Auth authorized domains minimal and review them after every
  portal or domain retirement.
- Treat Firestore, Realtime Database, and Storage rules as default-deny. Test
  cross-user and unauthenticated access in the Firebase emulator before release.
- Set quotas and billing alerts for every client-accessible billable API.

## Release verification

For each of the customer, vendor, delivery, and admin apps:

1. Confirm the app's configured Android package or iOS bundle identifier appears
   in its Firebase project registration.
2. Confirm the release signing fingerprint is present in the Android app
   registration and the key's application restrictions.
3. Confirm a release-equivalent build can sign in, refresh a token, register for
   push, and access only its own test data.
4. Attempt the same Firebase requests with an unregistered package or invalid
   App Check token and verify denial where enforcement is supported.
5. Record the restriction review date and owner in the release evidence; never
   record the key value in tickets, logs, or screenshots.

Rotating or restricting live keys is an operational change. It requires an
explicitly approved project and a staged mobile release; merging documentation
or application code does not authorize that rotation.
