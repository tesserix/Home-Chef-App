import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { act } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/renderWithProviders';
import { apiClient, ACCOUNT_BLOCKED_EVENT } from '@/shared/services/api-client';
import DataPrivacyPage from './DataPrivacyPage';

// "Pause my account" previously fired the deactivate mutation straight from
// the button's onClick with no confirmation, and its own success toast said
// "sign in again to resume" — which is factually wrong (the server keeps
// rejecting every request but /me/reactivate). Both are covered here,
// alongside the reactivate path that makes the pause actually reversible.

vi.mock('@/app/providers/AuthProvider', () => ({
  useAuth: () => ({
    user: { email: 'customer@demo.com' },
  }),
}));

function mockEligibility(
  overrides: Partial<{
    deletable: boolean;
    blockers: { code: string; label: string }[];
    retentionDays: number;
  }> = {},
) {
  return vi.spyOn(apiClient, 'get').mockResolvedValue({
    deletable: true,
    blockers: [],
    retentionDays: 180,
    ...overrides,
  });
}

describe('DataPrivacyPage — pause my account', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('does not pause on a single click — a confirmation dialog appears first', async () => {
    mockEligibility();
    const post = vi.spyOn(apiClient, 'post');
    const user = userEvent.setup();
    renderWithProviders(<DataPrivacyPage />);

    await user.click(await screen.findByRole('button', { name: 'Pause my account' }));

    expect(post).not.toHaveBeenCalled();
    expect(
      screen.getByRole('dialog', { name: 'Pause your account?' }),
    ).toBeInTheDocument();
  });

  it('does not fire the mutation when the dialog is dismissed', async () => {
    mockEligibility();
    const post = vi.spyOn(apiClient, 'post');
    const user = userEvent.setup();
    renderWithProviders(<DataPrivacyPage />);

    await user.click(await screen.findByRole('button', { name: 'Pause my account' }));
    const dialog = screen.getByRole('dialog', { name: 'Pause your account?' });
    await user.click(within(dialog).getByRole('button', { name: 'Keep my account active' }));

    expect(post).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('pauses only after the dialog is confirmed, with an accurate toast and a reachable reactivate control', async () => {
    mockEligibility();
    vi.spyOn(apiClient, 'post').mockResolvedValue({});
    const user = userEvent.setup();
    renderWithProviders(<DataPrivacyPage />);

    await user.click(await screen.findByRole('button', { name: 'Pause my account' }));
    const dialog = screen.getByRole('dialog', { name: 'Pause your account?' });
    await user.click(within(dialog).getByRole('button', { name: 'Pause my account' }));

    // Deactivation replaces the page with the paused state — including a
    // working way back, which is the whole point of this fix.
    await screen.findByText('Your account is paused');
    expect(screen.getByText('Your account is paused')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Turn my account back on' }),
    ).toBeInTheDocument();

    // The old copy ("Sign in again to resume") was factually wrong — signing
    // in again does not reactivate the account.
    expect(
      screen.queryByText(/sign in again to resume/i),
    ).not.toBeInTheDocument();
  });

  it('reactivates via the useReactivateAccount hook when the account is already paused on load', async () => {
    mockEligibility();
    const post = vi.spyOn(apiClient, 'post').mockResolvedValue({});
    renderWithProviders(<DataPrivacyPage />);

    // Simulate the api-client's account-blocked event, which fires when any
    // request 403s with status=account_deactivated — e.g. the eligibility
    // fetch this page makes on mount, for a user who paused earlier and just
    // reloaded.
    act(() => {
      window.dispatchEvent(
        new CustomEvent(ACCOUNT_BLOCKED_EVENT, {
          detail: { status: 'account_deactivated' },
        }),
      );
    });

    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'Turn my account back on' }),
    );

    await waitFor(() => expect(post).toHaveBeenCalledWith('/customer/me/reactivate'));
    await screen.findByText('Your data');
  });
});
