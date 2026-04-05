// auth-api.test.ts — RED phase: tests for auth API functions

import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { AxiosInstance } from 'axios';
import {
  loginWithEmail,
  registerUser,
  oauthLogin,
  forgotPassword,
  resetPassword,
  logoutUser,
} from '../api/auth';
import type { AuthResponse } from '../types/user';

const mockAuthResponse: AuthResponse = {
  user: {
    id: 'user-1',
    email: 'test@test.com',
    firstName: 'Test',
    lastName: 'User',
    phone: '123456789',
    role: 'customer',
    avatar: null,
    fcmToken: null,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
  accessToken: 'access-token-123',
  refreshToken: 'refresh-token-456',
};

const makeClient = (postImpl: ReturnType<typeof vi.fn>) =>
  ({ post: postImpl, put: vi.fn() } as unknown as AxiosInstance);

describe('auth API functions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loginWithEmail calls POST /auth/login and returns AuthResponse', async () => {
    const postMock = vi.fn().mockResolvedValueOnce({ data: mockAuthResponse });
    const client = makeClient(postMock);

    const result = await loginWithEmail(client, { email: 'test@test.com', password: 'pass' });

    expect(postMock).toHaveBeenCalledWith('/auth/login', {
      email: 'test@test.com',
      password: 'pass',
    });
    expect(result).toEqual(mockAuthResponse);
  });

  it('loginWithEmail throws on 401 with server error message', async () => {
    const error = Object.assign(new Error('Unauthorized'), {
      response: { status: 401, data: { message: 'Invalid credentials' } },
    });
    const postMock = vi.fn().mockRejectedValueOnce(error);
    const client = makeClient(postMock);

    await expect(loginWithEmail(client, { email: 'bad@test.com', password: 'wrong' })).rejects.toThrow();
  });

  it('oauthLogin calls POST /auth/oauth with provider and token fields', async () => {
    const postMock = vi.fn().mockResolvedValueOnce({ data: mockAuthResponse });
    const client = makeClient(postMock);

    await oauthLogin(client, { provider: 'google', token: 'id-token-xyz' });

    expect(postMock).toHaveBeenCalledWith('/auth/oauth', {
      provider: 'google',
      token: 'id-token-xyz',
    });
  });

  it('registerUser calls POST /auth/register', async () => {
    const postMock = vi.fn().mockResolvedValueOnce({ data: mockAuthResponse });
    const client = makeClient(postMock);

    await registerUser(client, {
      email: 'new@test.com',
      password: 'pass',
      firstName: 'New',
      lastName: 'User',
    });

    expect(postMock).toHaveBeenCalledWith('/auth/register', expect.objectContaining({
      email: 'new@test.com',
    }));
  });

  it('forgotPassword calls POST /auth/forgot-password', async () => {
    const postMock = vi.fn().mockResolvedValueOnce({ data: { message: 'sent' } });
    const client = makeClient(postMock);

    await forgotPassword(client, { email: 'test@test.com' });

    expect(postMock).toHaveBeenCalledWith('/auth/forgot-password', { email: 'test@test.com' });
  });

  it('resetPassword calls POST /auth/reset-password', async () => {
    const postMock = vi.fn().mockResolvedValueOnce({ data: { message: 'reset' } });
    const client = makeClient(postMock);

    await resetPassword(client, { token: 'reset-token', password: 'newpass' });

    expect(postMock).toHaveBeenCalledWith('/auth/reset-password', {
      token: 'reset-token',
      password: 'newpass',
    });
  });

  it('logoutUser calls POST /auth/logout', async () => {
    const postMock = vi.fn().mockResolvedValueOnce({ data: {} });
    const client = makeClient(postMock);

    await logoutUser(client, 'refresh-token-to-revoke');

    expect(postMock).toHaveBeenCalledWith('/auth/logout', {
      refreshToken: 'refresh-token-to-revoke',
    });
  });
});
