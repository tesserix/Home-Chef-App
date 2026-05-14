// apps/web/src/lib/firebase.ts
import { initializeApp } from "firebase/app";
import { getAuth } from "firebase/auth";

const apiKey = import.meta.env.VITE_GIP_API_KEY;
const projectId = import.meta.env.VITE_GIP_PROJECT_ID;
const authDomain = import.meta.env.VITE_GIP_AUTH_DOMAIN;
const tenantId = import.meta.env.VITE_GIP_TENANT_ID;

if (!apiKey || !projectId || !authDomain || !tenantId) {
  console.error(
    "[firebase] Missing one or more required env vars:",
    { apiKey: !!apiKey, projectId: !!projectId, authDomain: !!authDomain, tenantId: !!tenantId }
  );
}

export const firebaseApp = initializeApp({ apiKey, authDomain, projectId });
export const firebaseAuth = getAuth(firebaseApp);

// Pin this app to its GIP tenant pool.
firebaseAuth.tenantId = tenantId ?? null;
