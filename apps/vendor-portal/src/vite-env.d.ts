/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_MOCK_MODE: string;
  readonly VITE_GIP_API_KEY: string;
  readonly VITE_GIP_PROJECT_ID: string;
  readonly VITE_GIP_AUTH_DOMAIN: string;
  readonly VITE_GIP_TENANT_ID: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
