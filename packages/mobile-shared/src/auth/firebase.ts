import auth from "@react-native-firebase/auth";

/**
 * Pin the Firebase auth client to a specific GIP tenant pool.
 * Must be called once during app bootstrap, before any sign-in attempt.
 *
 * In @react-native-firebase/auth v22+, `tenantId` is a read-only getter —
 * direct property assignment throws a Proxy TypeError. Use the async
 * `setTenantId()` method instead. The native side propagates the tenant
 * via the await; the JS-side `_tenantId` field is set synchronously inside
 * setTenantId so subsequent `auth().tenantId` reads return immediately.
 */
export async function configureFirebaseAuth(tenantId: string): Promise<void> {
  await auth().setTenantId(tenantId);
}

export type FirebaseAuth = ReturnType<typeof auth>;
export type FirebaseUser = NonNullable<FirebaseAuth["currentUser"]>;
