import type { User } from '@/shared/types/auth';

// Mock-only auth types (production uses BFF session-based auth)
interface AuthResponse {
  user: User;
  token: string;
  refreshToken: string;
  expiresIn: number;
}

interface TokenRefreshResponse {
  token: string;
  refreshToken: string;
  expiresIn: number;
}
import type { Chef, MenuItem, Order, PaginatedResponse, SocialPost, CateringRequest, CateringQuote } from '@/shared/types';
import { mockChefs, mockMenuItems, mockOrders, mockUsers, mockSocialPosts, mockCateringRequests, mockCateringQuotes } from './data';

// Simulated network delay
const MOCK_DELAY = 300;

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

interface RequestOptions {
  params?: Record<string, string | number | boolean | undefined>;
  body?: string;
}

class MockService {
  private currentUser: User | null = null;

  async request<T>(method: string, endpoint: string, options: RequestOptions = {}): Promise<T> {
    await delay(MOCK_DELAY);

    const { params } = options;
    const body = options.body ? JSON.parse(options.body) : undefined;

    // Route handling
    const routes: Record<string, () => unknown> = {
      // Auth
      'POST /auth/login': () => this.login(body),
      'POST /auth/register': () => this.register(body),
      'POST /auth/social': () => this.socialLogin(body),
      'POST /auth/refresh': () => this.refreshToken(body),
      'GET /users/me': () => this.getCurrentUser(),
      'PUT /users/me': () => this.updateUser(body),

      // Chefs
      'GET /chefs': () => this.getChefs(params),
      'GET /chefs/:id': () => this.getChefById(endpoint.split('/')[2]!),
      'GET /chefs/:id/menu': () => this.getChefMenu(endpoint.split('/')[2]!),

      // Orders
      'POST /orders': () => this.createOrder(body),
      'GET /orders': () => this.getOrders(params),
      'GET /orders/:id': () => this.getOrderById(endpoint.split('/')[2]!),

      // Social Feed
      'GET /feed': () => this.getFeed(params),
      'POST /feed': () => this.createPost(body),
      'POST /feed/:id/like': () => this.likePost(endpoint.split('/')[2]!),

      // Catering
      'GET /catering/requests': () => this.getCateringRequests(params),
      'POST /catering/requests': () => this.createCateringRequest(body),
      'GET /catering/quotes': () => this.getCateringQuotes(params),
    };

    // Find matching route
    const routeKey = `${method} ${this.normalizeEndpoint(endpoint)}`;
    const handler = routes[routeKey];

    if (handler) {
      return handler() as T;
    }

    // Try pattern matching for dynamic routes
    for (const [pattern, handlerFn] of Object.entries(routes)) {
      if (this.matchRoute(pattern, `${method} ${endpoint}`)) {
        return handlerFn() as T;
      }
    }

    throw { success: false, error: { code: 'NOT_FOUND', message: `Route not found: ${method} ${endpoint}` } };
  }

  private normalizeEndpoint(endpoint: string): string {
    // Replace IDs with :id placeholder for matching
    return endpoint.replace(/\/[a-f0-9-]{36}/g, '/:id').replace(/\/[a-z]+-\d+/g, '/:id');
  }

  private matchRoute(pattern: string, actual: string): boolean {
    const patternParts = pattern.split(' ')[1]?.split('/') ?? [];
    const actualParts = actual.split(' ')[1]?.split('/') ?? [];

    if (patternParts.length !== actualParts.length) return false;
    if (pattern.split(' ')[0] !== actual.split(' ')[0]) return false;

    return patternParts.every((part, i) => part?.startsWith(':') || part === actualParts[i]);
  }

  // Auth handlers
  private login(body: { email: string; password: string }): AuthResponse {
    const user = mockUsers.find((u) => u.email === body.email);
    if (!user || body.password !== 'password123') {
      throw { success: false, error: { code: 'AUTH_INVALID_CREDENTIALS', message: 'Invalid credentials' } };
    }
    this.currentUser = user;
    return {
      user,
      token: `mock-token-${user.id}`,
      refreshToken: `mock-refresh-${user.id}`,
      expiresIn: 3600,
    };
  }

  private register(body: { email: string; password: string; firstName: string; lastName: string; role?: string }): AuthResponse {
    const newUser: User = {
      id: `user-${Date.now()}`,
      email: body.email,
      firstName: body.firstName,
      lastName: body.lastName,
      role: (body.role as User['role']) || 'customer',
      emailVerified: false,
      phoneVerified: false,
      isActive: true,
      createdAt: new Date().toISOString(),
    };
    this.currentUser = newUser;
    return {
      user: newUser,
      token: `mock-token-${newUser.id}`,
      refreshToken: `mock-refresh-${newUser.id}`,
      expiresIn: 3600,
    };
  }

  private socialLogin(_body: { provider: string; token: string }): AuthResponse {
    // Simulate social login - return first customer
    const user = mockUsers.find((u) => u.role === 'customer') ?? mockUsers[0]!;
    this.currentUser = user;
    return {
      user,
      token: `mock-token-${user.id}`,
      refreshToken: `mock-refresh-${user.id}`,
      expiresIn: 3600,
    };
  }

  private refreshToken(_body: { refreshToken: string }): TokenRefreshResponse {
    return {
      token: `mock-token-refreshed-${Date.now()}`,
      refreshToken: `mock-refresh-${Date.now()}`,
      expiresIn: 3600,
    };
  }

  private getCurrentUser(): User {
    if (!this.currentUser) {
      // Return a default user for demo
      this.currentUser = mockUsers[0]!;
    }
    return this.currentUser;
  }

  private updateUser(body: Partial<User>): User {
    if (this.currentUser) {
      this.currentUser = { ...this.currentUser, ...body };
    }
    return this.currentUser!;
  }

  // Chef handlers
  private getChefs(params?: Record<string, unknown>): PaginatedResponse<Chef> {
    let chefs = [...mockChefs];

    // Apply filters
    if (params?.cuisine) {
      chefs = chefs.filter((c) => c.cuisines.some((cuisine) => cuisine.toLowerCase().includes(String(params.cuisine).toLowerCase())));
    }
    if (params?.search) {
      const search = String(params.search).toLowerCase();
      chefs = chefs.filter(
        (c) =>
          c.businessName.toLowerCase().includes(search) ||
          c.description.toLowerCase().includes(search) ||
          c.cuisines.some((cuisine) => cuisine.toLowerCase().includes(search))
      );
    }
    if (params?.rating) {
      chefs = chefs.filter((c) => c.rating >= Number(params.rating));
    }
    if (params?.isOpen) {
      chefs = chefs.filter((c) => c.isOnline && c.acceptingOrders);
    }

    // Sort
    if (params?.sort === 'rating') {
      chefs.sort((a, b) => b.rating - a.rating);
    } else if (params?.sort === 'orders') {
      chefs.sort((a, b) => b.totalOrders - a.totalOrders);
    }

    // Pagination
    const page = Number(params?.page) || 1;
    const limit = Number(params?.limit) || 20;
    const start = (page - 1) * limit;
    const paginatedChefs = chefs.slice(start, start + limit);

    return {
      data: paginatedChefs,
      pagination: {
        page,
        limit,
        total: chefs.length,
        totalPages: Math.ceil(chefs.length / limit),
        hasNext: start + limit < chefs.length,
        hasPrev: page > 1,
      },
    };
  }

  private getChefById(id: string): Chef {
    const chef = mockChefs.find((c) => c.id === id);
    if (!chef) {
      throw { success: false, error: { code: 'CHEF_NOT_FOUND', message: 'Chef not found' } };
    }
    return chef;
  }

  private getChefMenu(chefId: string): MenuItem[] {
    return mockMenuItems.filter((item) => item.chefId === chefId);
  }

  // Order handlers
  private createOrder(body: unknown): Order {
    const newOrder: Order = {
      id: `order-${Date.now()}`,
      orderNumber: `HC${Date.now()}`,
      ...(body as Partial<Order>),
      status: 'pending',
      paymentStatus: 'pending',
      createdAt: new Date().toISOString(),
    } as Order;
    return newOrder;
  }

  private getOrders(params?: Record<string, unknown>): PaginatedResponse<Order> {
    let orders = [...mockOrders];

    if (params?.status) {
      const statuses = String(params.status).split(',');
      orders = orders.filter((o) => statuses.includes(o.status));
    }

    const page = Number(params?.page) || 1;
    const limit = Number(params?.limit) || 20;
    const start = (page - 1) * limit;

    return {
      data: orders.slice(start, start + limit),
      pagination: {
        page,
        limit,
        total: orders.length,
        totalPages: Math.ceil(orders.length / limit),
        hasNext: start + limit < orders.length,
        hasPrev: page > 1,
      },
    };
  }

  private getOrderById(id: string): Order {
    const order = mockOrders.find((o) => o.id === id);
    if (!order) {
      throw { success: false, error: { code: 'ORDER_NOT_FOUND', message: 'Order not found' } };
    }
    return order;
  }

  // Social Feed handlers
  private getFeed(params?: Record<string, unknown>): PaginatedResponse<SocialPost> {
    const page = Number(params?.page) || 1;
    const limit = Number(params?.limit) || 20;
    const start = (page - 1) * limit;

    return {
      data: mockSocialPosts.slice(start, start + limit),
      pagination: {
        page,
        limit,
        total: mockSocialPosts.length,
        totalPages: Math.ceil(mockSocialPosts.length / limit),
        hasNext: start + limit < mockSocialPosts.length,
        hasPrev: page > 1,
      },
    };
  }

  private createPost(body: unknown): SocialPost {
    const newPost: SocialPost = {
      id: `post-${Date.now()}`,
      ...(body as Partial<SocialPost>),
      likesCount: 0,
      commentsCount: 0,
      savesCount: 0,
      createdAt: new Date().toISOString(),
    } as SocialPost;
    return newPost;
  }

  private likePost(_id: string): { liked: boolean } {
    return { liked: true };
  }

  // Catering handlers
  private getCateringRequests(_params?: Record<string, unknown>): PaginatedResponse<CateringRequest> {
    return {
      data: mockCateringRequests,
      pagination: {
        page: 1,
        limit: 20,
        total: mockCateringRequests.length,
        totalPages: 1,
        hasNext: false,
        hasPrev: false,
      },
    };
  }

  private createCateringRequest(body: unknown): CateringRequest {
    const newRequest: CateringRequest = {
      id: `catering-${Date.now()}`,
      ...(body as Partial<CateringRequest>),
      status: 'pending',
      quotesCount: 0,
      createdAt: new Date().toISOString(),
    } as CateringRequest;
    return newRequest;
  }

  private getCateringQuotes(_params?: Record<string, unknown>): PaginatedResponse<CateringQuote> {
    return {
      data: mockCateringQuotes,
      pagination: {
        page: 1,
        limit: 20,
        total: mockCateringQuotes.length,
        totalPages: 1,
        hasNext: false,
        hasPrev: false,
      },
    };
  }
}

export const mockService = new MockService();
