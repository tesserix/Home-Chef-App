import auth from "@react-native-firebase/auth";

/**
 * Pin the Firebase auth client to a specific GIP tenant pool.
 * Must be called once during app bootstrap, before any sign-in attempt.
 */
export function configureFirebaseAuth(tenantId: string): void {
  auth().tenantId = tenantId;
}

export type FirebaseAuth = ReturnType<typeof auth>;
export type FirebaseUser = NonNullable<FirebaseAuth["currentUser"]>;
