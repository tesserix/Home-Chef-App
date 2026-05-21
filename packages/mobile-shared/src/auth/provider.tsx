import React, { createContext, useContext, useEffect, useState, ReactNode } from "react";
import auth from "@react-native-firebase/auth";
import { configureFirebaseAuth } from "./firebase";
import {
  autoLogin,
  clearStoredSession,
  fetchSessionUser,
  logoutBFF,
} from "./bff-session";
import { getIdToken, signOut as fbSignOut } from "./sign-in";

export interface AuthUser {
  id: string;
  email: string;
  role: string;
  pool: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  loading: boolean;
  /**
   * Call after a Firebase signInWith* succeeds to exchange the id_token
   * for a BFF session. Updates the context user.
   */
  completeSignIn: () => Promise<void>;
  signOut: () => Promise<void>;
}

const Ctx = createContext<AuthContextValue | undefined>(undefined);

export interface AuthProviderProps {
  children: ReactNode;
  bffUrl: string;
  tenantId: string;
}

export function AuthProvider({ children, bffUrl, tenantId }: AuthProviderProps) {
  if (!bffUrl) {
    throw new Error(
      "AuthProvider: bffUrl prop is empty. Set EXPO_PUBLIC_BFF_URL in your .env.local or EAS build profile."
    );
  }
  if (!tenantId) {
    throw new Error(
      "AuthProvider: tenantId prop is empty. Set EXPO_PUBLIC_GIP_TENANT_ID in your .env.local or EAS build profile."
    );
  }

  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    configureFirebaseAuth(tenantId);
    const unsub = auth().onAuthStateChanged(async (fb) => {
      if (!fb) {
        await clearStoredSession();
        setUser(null);
        setLoading(false);
        return;
      }
      const s = await fetchSessionUser(bffUrl);
      if (s) {
        setUser({ id: s.user_id, email: s.email, role: s.role, pool: s.pool });
      } else {
        setUser(null);
      }
      setLoading(false);
    });
    return () => unsub();
  }, [bffUrl, tenantId]);

  const completeSignIn = async () => {
    const idToken = await getIdToken();
    if (!idToken) throw new Error("no_id_token_after_sign_in");
    const body = await autoLogin(bffUrl, idToken, tenantId);
    setUser({
      id: body.user.id,
      email: body.user.email,
      role: body.user.role,
      pool: body.user.pool,
    });
  };

  const signOut = async () => {
    try {
      await logoutBFF(bffUrl);
    } catch {
      // best-effort
    }
    try {
      await fbSignOut();
    } catch {
      // best-effort
    }
    setUser(null);
  };

  return <Ctx.Provider value={{ user, loading, completeSignIn, signOut }}>{children}</Ctx.Provider>;
}

export function useAuth(): AuthContextValue {
  const c = useContext(Ctx);
  if (!c) throw new Error("useAuth must be used inside <AuthProvider>");
  return c;
}
