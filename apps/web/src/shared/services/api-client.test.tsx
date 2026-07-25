import { afterEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/renderWithProviders';
import { apiClient, ACCOUNT_BLOCKED_EVENT } from '@/shared/services/api-client';

describe('test harness', () => {
  it('renders a component through the providers', () => {
    const { getByText } = renderWithProviders(<span>harness ok</span>);
    expect(getByText('harness ok')).toBeInTheDocument();
  });
});

// middleware/bff_auth.go 403s every path but /me/reactivate for a paused
// account, tagging the body with status: "account_deactivated" (or
// "account_deleted"). DataPrivacyPage listens for this event to route a
// paused user somewhere sensible instead of leaving them at an opaque error.
describe('apiClient — account-blocked detection', () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it('dispatches ACCOUNT_BLOCKED_EVENT on a 403 tagged account_deactivated', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      json: async () => ({ error: 'Account is suspended', status: 'account_deactivated' }),
    }) as unknown as typeof fetch;

    const handler = vi.fn();
    window.addEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    try {
      await expect(
        apiClient.get('/customer/me/deletion-eligibility'),
      ).rejects.toBeTruthy();

      expect(handler).toHaveBeenCalledTimes(1);
      const event = handler.mock.calls[0]?.[0] as CustomEvent<{ status?: string }>;
      expect(event.detail.status).toBe('account_deactivated');
    } finally {
      window.removeEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    }
  });

  it('does not dispatch for an ordinary 403 with no account-lifecycle tag', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      json: async () => ({ error: 'Forbidden' }),
    }) as unknown as typeof fetch;

    const handler = vi.fn();
    window.addEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    try {
      await expect(apiClient.get('/some/other/endpoint')).rejects.toBeTruthy();
      expect(handler).not.toHaveBeenCalled();
    } finally {
      window.removeEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    }
  });

  it('does not dispatch on a 401 (handled separately by auth:expired)', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ error: 'Unauthorized' }),
    }) as unknown as typeof fetch;

    const handler = vi.fn();
    window.addEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    try {
      await expect(apiClient.get('/customer/profile')).rejects.toBeTruthy();
      expect(handler).not.toHaveBeenCalled();
    } finally {
      window.removeEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    }
  });
});
