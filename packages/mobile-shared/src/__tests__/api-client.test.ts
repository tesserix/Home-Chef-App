// api-client.test.ts — RED phase: these tests FAIL until client.ts is implemented

import { describe, it, expect, vi, beforeEach } from 'vitest';
import axios from 'axios';

vi.mock('axios', () => {
  const mockAxiosInstance = {
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
    post: vi.fn(),
  };
  return {
    default: {
      create: vi.fn(() => mockAxiosInstance),
      post: vi.fn(),
    },
  };
});

vi.mock('expo-secure-store', () => ({
  getItemAsync: vi.fn(),
  setItemAsync: vi.fn(),
  deleteItemAsync: vi.fn(),
}));

import { createApiClient } from '../api/client';

describe('createApiClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates an axios instance with provided baseURL', () => {
    createApiClient({
      baseURL: 'https://api.test.com/api/v1',
      getToken: () => null,
    });
    expect(axios.create).toHaveBeenCalledWith(
      expect.objectContaining({ baseURL: 'https://api.test.com/api/v1' })
    );
  });

  it('registers request and response interceptors', () => {
    const client = createApiClient({
      baseURL: 'https://api.test.com/api/v1',
      getToken: () => null,
    });
    // The interceptors are registered on the instance
    expect(client.interceptors.request.use).toHaveBeenCalled();
    expect(client.interceptors.response.use).toHaveBeenCalled();
  });

  it('injects Bearer token header when getToken returns a non-null string', () => {
    const mockConfig = { headers: {} as Record<string, string> };
    let requestInterceptor: ((config: typeof mockConfig) => typeof mockConfig) | null = null;

    vi.mocked(axios.create).mockReturnValueOnce({
      interceptors: {
        request: {
          use: vi.fn((fn) => {
            requestInterceptor = fn;
          }),
        },
        response: { use: vi.fn() },
      },
    } as unknown as ReturnType<typeof axios.create>);

    createApiClient({
      baseURL: 'https://api.test.com/api/v1',
      getToken: () => 'my-token',
    });

    const result = requestInterceptor!(mockConfig);
    expect(result.headers.Authorization).toBe('Bearer my-token');
  });

  it('does NOT inject Authorization header when getToken returns null', () => {
    const mockConfig = { headers: {} as Record<string, string> };
    let requestInterceptor: ((config: typeof mockConfig) => typeof mockConfig) | null = null;

    vi.mocked(axios.create).mockReturnValueOnce({
      interceptors: {
        request: {
          use: vi.fn((fn) => {
            requestInterceptor = fn;
          }),
        },
        response: { use: vi.fn() },
      },
    } as unknown as ReturnType<typeof axios.create>);

    createApiClient({
      baseURL: 'https://api.test.com/api/v1',
      getToken: () => null,
    });

    const result = requestInterceptor!(mockConfig);
    expect(result.headers.Authorization).toBeUndefined();
  });
});
