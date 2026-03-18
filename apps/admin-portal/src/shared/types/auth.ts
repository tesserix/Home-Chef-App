export type UserRole = 'customer' | 'chef' | 'delivery' | 'admin' | 'super_admin' | 'fleet_manager';

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

export type Permission =
  | 'users:read'
  | 'users:write'
  | 'users:delete'
  | 'chefs:read'
  | 'chefs:write'
  | 'chefs:verify'
  | 'orders:read:all'
  | 'orders:refund'
  | 'analytics:read'
  | 'settings:read'
  | 'settings:write'
  | 'social:moderate';

export const ADMIN_PERMISSIONS: Permission[] = [
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
];

export const SUPER_ADMIN_PERMISSIONS: Permission[] = [
  ...ADMIN_PERMISSIONS,
  'users:delete',
  'settings:write',
];

export function hasPermission(roles: string[], permission: Permission): boolean {
  if (roles.includes('super_admin')) {
    return SUPER_ADMIN_PERMISSIONS.includes(permission);
  }
  if (roles.includes('admin')) {
    return ADMIN_PERMISSIONS.includes(permission);
  }
  return false;
}
