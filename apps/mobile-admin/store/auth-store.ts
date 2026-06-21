// Re-export from mobile-shared so app code imports from a local path. This
// keeps room for future admin-specific auth state without changing imports.
export { useAuthStore } from '@homechef/mobile-shared/hooks';
export type { User, AuthResponse } from '@homechef/mobile-shared/types';
