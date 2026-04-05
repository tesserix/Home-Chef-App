// Re-export from mobile-shared so app code imports from local path
// This allows future app-specific auth state extensions without changing import paths
export { useAuthStore } from '@homechef/mobile-shared/hooks';
export type { User, AuthResponse } from '@homechef/mobile-shared/types';
