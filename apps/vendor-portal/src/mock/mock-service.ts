import type { User } from '@/shared/types/auth';
import {
  mockUsers,
  mockChefs,
  mockCategories,
  mockMenuItems,
  mockOrders,
  mockDashboardStats,
  mockEarnings,
  mockSettings,
} from './data';

interface MockRequestOptions {
  params?: Record<string, string | number | boolean | undefined>;
  body?: string;
}

class MockService {
  private currentUser: User | null = null;
  private delay = 300;

  private async simulateDelay(): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, this.delay));
  }

  async request<T>(method: string, endpoint: string, options?: MockRequestOptions): Promise<T> {
    await this.simulateDelay();

    const body = options?.body ? JSON.parse(options.body) : undefined;

    // ── Auth Routes ─────────────────────────────────────────
    if (endpoint === '/auth/login' && method === 'POST') {
      const user = mockUsers.find((u) => u.email === body?.email);
      if (!user) throw { success: false, error: { code: 'INVALID_CREDENTIALS', message: 'Invalid email or password' } };
      if (user.role !== 'chef') throw { success: false, error: { code: 'ACCESS_DENIED', message: 'This portal is only for vendor accounts.' } };
      this.currentUser = user;
      return {
        user,
        token: 'mock-jwt-token-vendor',
        refreshToken: 'mock-refresh-token-vendor',
        expiresIn: 3600,
      } as unknown as T;
    }

    if (endpoint === '/auth/register' && method === 'POST') {
      const newUser: User = {
        id: `chef-${Date.now()}`,
        email: body?.email || '',
        phone: body?.phone,
        firstName: body?.firstName || '',
        lastName: body?.lastName || '',
        role: 'chef',
        emailVerified: false,
        phoneVerified: false,
        isActive: true,
        createdAt: new Date().toISOString(),
      };
      this.currentUser = newUser;
      return {
        user: newUser,
        token: 'mock-jwt-token-vendor',
        refreshToken: 'mock-refresh-token-vendor',
        expiresIn: 3600,
      } as unknown as T;
    }

    if (endpoint === '/auth/refresh' && method === 'POST') {
      return {
        token: 'mock-jwt-token-refreshed-vendor',
        refreshToken: 'mock-refresh-token-refreshed-vendor',
        expiresIn: 3600,
      } as unknown as T;
    }

    if (endpoint === '/auth/me' && method === 'GET') {
      return (this.currentUser || mockUsers[0]) as unknown as T;
    }

    if (endpoint === '/auth/forgot-password' && method === 'POST') {
      return undefined as unknown as T;
    }

    // ── Dashboard ───────────────────────────────────────────
    if (endpoint === '/chef/dashboard/stats' && method === 'GET') {
      return mockDashboardStats as unknown as T;
    }

    // ── Menu ────────────────────────────────────────────────
    if (endpoint === '/chef/menu/categories' && method === 'GET') {
      return mockCategories as unknown as T;
    }

    if (endpoint === '/chef/menu' && method === 'GET') {
      return mockMenuItems as unknown as T;
    }

    if (endpoint.match(/\/chef\/menu\/items\/(.+)/) && method === 'GET') {
      const id = endpoint.split('/').pop();
      const item = mockMenuItems.find((i) => i.id === id);
      return (item || mockMenuItems[0]) as unknown as T;
    }

    if (endpoint === '/chef/menu/items' && method === 'POST') {
      const newItem = { id: `item-${Date.now()}`, chefId: 'chef-1', ...body };
      return newItem as unknown as T;
    }

    if (endpoint.match(/\/chef\/menu\/items\/(.+)/) && method === 'PUT') {
      return { ...body } as unknown as T;
    }

    if (endpoint.match(/\/chef\/menu\/items\/(.+)/) && method === 'DELETE') {
      return undefined as unknown as T;
    }

    // ── Orders ──────────────────────────────────────────────
    if (endpoint === '/chef/orders' && method === 'GET') {
      const status = options?.params?.status as string | undefined;
      if (status) {
        const statuses = status.split(',');
        return mockOrders.filter((o) => statuses.includes(o.status)) as unknown as T;
      }
      return mockOrders as unknown as T;
    }

    if (endpoint.match(/\/chef\/orders\/(.+)\/status/) && method === 'PUT') {
      return { status: body?.status } as unknown as T;
    }

    // ── Earnings ────────────────────────────────────────────
    if (endpoint === '/chef/earnings' && method === 'GET') {
      return mockEarnings as unknown as T;
    }

    if (endpoint === '/chef/earnings/payouts' && method === 'GET') {
      return mockEarnings.recentPayouts as unknown as T;
    }

    // ── Profile ─────────────────────────────────────────────
    if (endpoint === '/chef/profile' && method === 'GET') {
      return mockChefs[0] as unknown as T;
    }

    if (endpoint === '/chef/profile' && method === 'PUT') {
      return { ...mockChefs[0], ...body } as unknown as T;
    }

    // ── Settings ────────────────────────────────────────────
    if (endpoint === '/chef/settings' && method === 'GET') {
      return mockSettings as unknown as T;
    }

    if (endpoint === '/chef/settings' && method === 'PUT') {
      return { ...mockSettings, ...body } as unknown as T;
    }

    // ── Fallback ────────────────────────────────────────────
    console.warn(`[MockService] Unhandled: ${method} ${endpoint}`);
    return {} as unknown as T;
  }
}

export const mockService = new MockService();
