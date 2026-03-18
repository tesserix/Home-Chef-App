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

export interface SessionUser {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  name?: string;
  roles?: string[];
  tenantId?: string;
  tenantSlug?: string;
}

export interface SessionResponse {
  authenticated: boolean;
  user?: SessionUser;
  expiresAt?: number;
  csrfToken?: string;
  error?: string;
}

export type SocialProvider = 'google' | 'facebook';

export type Permission =
  | 'delivery:read'
  | 'delivery:write'
  | 'orders:read:own';

export const ROLE_PERMISSIONS: Record<string, Permission[]> = {
  delivery: ['orders:read:own', 'delivery:read', 'delivery:write'],
};

export function hasPermission(role: string, permission: Permission): boolean {
  return ROLE_PERMISSIONS[role]?.includes(permission) ?? false;
}
