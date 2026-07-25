import { useMutation, useQuery } from '@tanstack/react-query';

import { api } from '../lib/api';

// useAccountLifecycle — in-app account pause and deletion for drivers.
//
// Required for store submission: Apple 5.1.1(v) demands the user be able to
// START deletion inside the app. This app previously showed an alert telling
// drivers to email support, which does not satisfy that rule.
//
// Backed by /driver/me/{deactivate,reactivate,delete,deletion-eligibility}.

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
 * Previews whether deletion would succeed — a driver mid-delivery must finish
 * or hand off the parcel before the account can go.
 */
export function useDeletionEligibility() {
  return useQuery<DeletionEligibility>({
    queryKey: ['driver-deletion-eligibility'],
    queryFn: async () => {
      const res = await api.get('/v1/driver/me/deletion-eligibility');
      return res.data as DeletionEligibility;
    },
    staleTime: 30_000,
  });
}

export function useDeleteAccount() {
  return useMutation<DeleteAccountResult, unknown, string>({
    mutationFn: async (confirmEmail: string) => {
      const res = await api.post('/v1/driver/me/delete', { confirmEmail });
      return res.data as DeleteAccountResult;
    },
  });
}

/** Reversible pause — goes offline and leaves dispatch. Nothing is deleted. */
export function useDeactivateAccount() {
  return useMutation<unknown, unknown, string | undefined>({
    mutationFn: async (reason?: string) => {
      const res = await api.post('/v1/driver/me/deactivate', { reason: reason ?? '' });
      return res.data;
    },
  });
}

export function useReactivateAccount() {
  return useMutation<unknown, unknown, void>({
    mutationFn: async () => {
      const res = await api.post('/v1/driver/me/reactivate');
      return res.data;
    },
  });
}

export function useExportMyData() {
  return useMutation<unknown, unknown, void>({
    mutationFn: async () => {
      const res = await api.get('/v1/driver/me/export');
      return res.data;
    },
  });
}
