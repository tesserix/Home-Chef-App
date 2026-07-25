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
//
// Deletion previously only swapped in a static "Account deleted" panel and
// left the auth store, Firebase session and BFF cookie intact, so
// ProtectedRoute kept passing and browser-Back landed the "deleted" user
// right back inside the app — covered in the second describe block below.

const mockLogout = vi.fn();

vi.mock('@/app/providers/AuthProvider', () => ({
  useAuth: () => ({
    user: { email: 'customer@demo.com' },
    logout: mockLogout,
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
    mockLogout.mockClear();
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

describe('DataPrivacyPage — delete my account', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    mockLogout.mockClear();
  });

  it('terminates the session after a successful deletion', async () => {
    mockEligibility({ deletable: true });
    vi.spyOn(apiClient, 'post').mockResolvedValue({
      status: 'deleted',
      deletedAt: '2026-07-25T00:00:00Z',
      purgeAfter: '2027-01-21T00:00:00Z',
    });
    const user = userEvent.setup();
    renderWithProviders(<DataPrivacyPage />);

    const emailInput = await screen.findByLabelText(/type customer@demo.com to confirm/i);
    await user.type(emailInput, 'customer@demo.com');
    await user.click(screen.getByRole('button', { name: 'Delete my account' }));

    // logout() clears the auth store, signs out of Firebase and clears the
    // BFF session cookie — without it, ProtectedRoute keeps passing and
    // browser-Back lands the "deleted" user right back inside the app.
    await waitFor(() => expect(mockLogout).toHaveBeenCalledTimes(1));
  });

  it('does not delete while blocked, and never calls logout in that case', async () => {
    mockEligibility({
      deletable: false,
      blockers: [{ code: 'wallet_balance', label: 'Wallet credit left' }],
    });
    const post = vi.spyOn(apiClient, 'post');
    renderWithProviders(<DataPrivacyPage />);

    await screen.findByText('Wallet credit left');
    expect(screen.getByRole('button', { name: 'Delete my account' })).toBeDisabled();
    expect(post).not.toHaveBeenCalled();
    expect(mockLogout).not.toHaveBeenCalled();
  });
});
