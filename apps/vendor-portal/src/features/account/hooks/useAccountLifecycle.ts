import { useMutation, useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Account pause and deletion for chefs — the web twin of
// apps/mobile-vendor/hooks/useAccountLifecycle.ts.
//
// The portal had no way to pause or delete an account at all: a chef who wanted
// to stop trading had to email support. Mobile got this for store-submission
// reasons (Apple 5.1.1(v)), but the underlying need is the same on web, and the
// endpoints already exist.
//
// Backed by /chef/me/{deactivate,reactivate,delete,deletion-eligibility}.

/** One reason the account cannot be deleted yet. Codes are stable. */
export interface DeletionBlocker {
  code:
    | 'active_orders'
    | 'mealplan_escrow'
    | 'wallet_balance'
    | 'pending_payout'
    | 'active_delivery';
  label: string;
  count?: number;
  amount?: number;
}

export interface DeletionEligibility {
  deletable: boolean;
  blockers: DeletionBlocker[];
  retentionDays: number;
}

export interface DeleteAccountResult {
  status: 'deleted' | 'already_deleted';
  deletedAt: string;
  purgeAfter: string;
  notice?: string;
}

/**
 * Previews whether deletion would succeed.
 *
 * Chefs are the role most likely to be blocked — unfulfilled orders and
 * unreleased payouts both hold real money — so the screen shows this BEFORE
 * asking anyone to type their email, rather than letting them commit to a
 * destructive action and then refusing it.
 */
export function useDeletionEligibility() {
  return useQuery<DeletionEligibility>({
    queryKey: ['chef', 'deletion-eligibility'],
    queryFn: () => apiClient.get<DeletionEligibility>('/chef/me/deletion-eligibility'),
    staleTime: 30_000,
  });
}

export function useDeleteAccount() {
  return useMutation<DeleteAccountResult, unknown, string>({
    mutationFn: (confirmEmail: string) =>
      apiClient.post<DeleteAccountResult>('/chef/me/delete', { confirmEmail }),
  });
}

/**
 * Reversible pause. Closes the kitchen, hides it from customers and turns off
 * schedule-driven auto-open so it cannot quietly reopen. Nothing is deleted and
 * approval is not reset.
 */
export function useDeactivateAccount() {
  return useMutation<unknown, unknown, string | undefined>({
    mutationFn: (reason?: string) =>
      apiClient.post('/chef/me/deactivate', { reason: reason ?? '' }),
  });
}

export function useReactivateAccount() {
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.post('/chef/me/reactivate'),
  });
}
