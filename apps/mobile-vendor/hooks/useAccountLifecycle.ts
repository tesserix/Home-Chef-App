import { useMutation, useQuery } from '@tanstack/react-query';

import { api } from '../lib/api';

// useAccountLifecycle — in-app account pause and deletion for chefs.
//
// Required for store submission: Apple 5.1.1(v) demands that an app offering
// account creation lets the user START deletion inside the app. The vendor app
// previously showed an alert telling chefs to email support, which does not
// satisfy that rule.
//
// Backed by /chef/me/{deactivate,reactivate,delete,deletion-eligibility}. The
// api client's baseURL ends in /api, so the /v1 prefix belongs here.

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
 * Previews whether deletion would succeed. Chefs are the most likely role to be
 * blocked — unfulfilled orders and unreleased payouts both hold real money — so
 * the screen shows this before asking anyone to type their email.
 */
export function useDeletionEligibility() {
  return useQuery<DeletionEligibility>({
    queryKey: ['chef-deletion-eligibility'],
    queryFn: async () => {
      const res = await api.get('/v1/chef/me/deletion-eligibility');
      return res.data as DeletionEligibility;
    },
    staleTime: 30_000,
  });
}

export function useDeleteAccount() {
  return useMutation<DeleteAccountResult, unknown, string>({
    mutationFn: async (confirmEmail: string) => {
      const res = await api.post('/v1/chef/me/delete', { confirmEmail });
      return res.data as DeleteAccountResult;
    },
  });
}

/**
 * Reversible pause. Closes the kitchen, hides it from customers and turns off
 * schedule-driven auto-open so it cannot quietly reopen. Nothing is deleted and
 * approval is not reset.
 */
export function useDeactivateAccount() {
  return useMutation<unknown, unknown, string | undefined>({
    mutationFn: async (reason?: string) => {
      const res = await api.post('/v1/chef/me/deactivate', { reason: reason ?? '' });
      return res.data;
    },
  });
}

export function useReactivateAccount() {
  return useMutation<unknown, unknown, void>({
    mutationFn: async () => {
      const res = await api.post('/v1/chef/me/reactivate');
      return res.data;
    },
  });
}

export function useExportMyData() {
  return useMutation<unknown, unknown, void>({
    mutationFn: async () => {
      const res = await api.get('/v1/chef/me/export');
      return res.data;
    },
  });
}
