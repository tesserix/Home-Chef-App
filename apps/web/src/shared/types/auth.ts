export type UserRole = 'customer' | 'chef' | 'delivery' | 'admin' | 'super_admin' | 'fleet_manager';

export interface User {
  id: string;
  email: string;
  phone?: string;
  firstName: string;
  lastName: string;
  avatar?: string;
  role: UserRole;
  emailVerified: boolean;
  phoneVerified: boolean;
  isActive: boolean;
  preferences?: UserPreferences;
  createdAt: string;
}

export interface UserPreferences {
  dietary?: string[];
  notifications?: {
    push: boolean;
    email: boolean;
    sms: boolean;
  };
}

/** Session user returned by the BFF /auth/session endpoint */
export interface SessionUser {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  name?: string;
  phone?: string;
  avatar?: string;
  roles?: string[];
  tenantId?: string;
  tenantSlug?: string;
}

/** Response from GET /auth/session */
export interface SessionResponse {
  authenticated: boolean;
  user?: SessionUser;
  expiresAt?: number;
  csrfToken?: string;
  error?: string;
}

/** Supported social login providers (Keycloak IDP aliases) */
export type SocialProvider = 'google' | 'facebook';

// Permission types for RBAC
export type Permission =
  | 'users:read'
  | 'users:write'
  | 'users:delete'
  | 'chefs:read'
  | 'chefs:write'
  | 'chefs:verify'
  | 'orders:read:own'
  | 'orders:read:all'
  | 'orders:write'
  | 'orders:cancel'
  | 'orders:refund'
  | 'menu:read'
  | 'menu:write'
  | 'analytics:read'
  | 'settings:read'
  | 'settings:write'
  | 'social:post'
  | 'social:moderate'
  | 'catering:request'
  | 'catering:quote'
  | 'delivery:read'
  | 'delivery:write';

// Role-Permission mapping
export const ROLE_PERMISSIONS: Record<UserRole, Permission[]> = {
  customer: [
    'orders:read:own',
    'orders:write',
    'orders:cancel',
    'catering:request',
  ],
  chef: [
    'orders:read:own',
    'menu:read',
    'menu:write',
    'analytics:read',
    'social:post',
    'catering:quote',
  ],
  delivery: [
    'orders:read:own',
    'delivery:read',
    'delivery:write',
  ],
  fleet_manager: [
    'orders:read:all',
    'delivery:read',
    'delivery:write',
    'analytics:read',
  ],
  admin: [
    'users:read',
    'users:write',
    'chefs:read',
    'chefs:write',
    'chefs:verify',
    'orders:read:all',
    'orders:refund',
    'analytics:read',
    'settings:read',
    'social:moderate',
  ],
  super_admin: [
    'users:read',
    'users:write',
    'users:delete',
    'chefs:read',
    'chefs:write',
    'chefs:verify',
    'orders:read:all',
    'orders:refund',
    'analytics:read',
    'settings:read',
    'settings:write',
    'social:moderate',
  ],
};

export function hasPermission(role: UserRole, permission: Permission): boolean {
  return ROLE_PERMISSIONS[role]?.includes(permission) ?? false;
}
