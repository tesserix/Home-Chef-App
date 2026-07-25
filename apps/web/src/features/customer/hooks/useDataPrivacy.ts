import { useMutation, useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// DPDP Act 2023 data-subject actions plus the account lifecycle
// (deactivate / delete / restore) required by Google Play's account-deletion
// policy. Mirrors apps/mobile-customer/hooks/useDataPrivacy.ts.

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
  /** When the account is erased for good — restore is possible until then. */
  purgeAfter: string;
  notice?: string;
}

export function useExportMyData() {
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.get('/customer/me/export'),
  });
}

/**
 * Previews whether deletion would succeed, so the page can show "finish these
 * first" up front rather than after the user types their email and presses a
 * button that then fails.
 */
export function useDeletionEligibility() {
  return useQuery<DeletionEligibility>({
    queryKey: ['deletion-eligibility'],
    queryFn: () => apiClient.get<DeletionEligibility>('/customer/me/deletion-eligibility'),
    staleTime: 30_000,
  });
}

export function useDeleteAccount() {
  return useMutation<DeleteAccountResult, unknown, string>({
    mutationFn: (confirmEmail: string) =>
      apiClient.post<DeleteAccountResult>('/customer/me/delete', { confirmEmail }),
  });
}

export function useDeactivateAccount() {
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.post('/customer/me/deactivate'),
  });
}

export function useReactivateAccount() {
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.post('/customer/me/reactivate'),
  });
}
