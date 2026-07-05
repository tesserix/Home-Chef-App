import { useMutation } from '@tanstack/react-query';

import { api } from '../lib/api';

// useDataPrivacy — DPDP Act 2023 data-subject actions for the customer:
//   - exportMyData: pulls the full personal-data bundle (Right to Access)
//   - deleteAccount: confirm-email-gated soft-delete (Right to Erasure)
// Both hit the backend endpoints added in the customer DPDP handler
// (/customer/me/export, /customer/me/delete).

export interface DeleteAccountResult {
  status: 'deleted' | 'already_deleted';
  deletedAt: string;
  retainUntil: string;
  notice?: string;
}

export function useExportMyData() {
  return useMutation<unknown, unknown, void>({
    mutationFn: async () => {
      const res = await api.get('/customer/me/export');
      return res.data;
    },
  });
}

export function useDeleteAccount() {
  return useMutation<DeleteAccountResult, unknown, string>({
    mutationFn: async (confirmEmail: string) => {
      const res = await api.post('/customer/me/delete', { confirmEmail });
      return res.data as DeleteAccountResult;
    },
  });
}
