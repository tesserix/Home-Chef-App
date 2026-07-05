import { useMutation } from '@tanstack/react-query';

import { api } from '../lib/api';

// useDataPrivacy — DPDP Act 2023 data-subject actions for the customer:
//   - exportMyData: pulls the full personal-data bundle (Right to Access)
//   - deleteAccount: confirm-email-gated soft-delete (Right to Erasure)
// Both hit the backend endpoints added in the customer DPDP handler,
// registered under the v1 customer group: /api/v1/customer/me/{export,delete}.
// The api client's baseURL already ends in /api, so hooks must supply the
// /v1 prefix — omitting it (as this hook originally did) 404s the request.

export interface DeleteAccountResult {
  status: 'deleted' | 'already_deleted';
  deletedAt: string;
  retainUntil: string;
  notice?: string;
}

export function useExportMyData() {
  return useMutation<unknown, unknown, void>({
    mutationFn: async () => {
      const res = await api.get('/v1/customer/me/export');
      return res.data;
    },
  });
}

export function useDeleteAccount() {
  return useMutation<DeleteAccountResult, unknown, string>({
    mutationFn: async (confirmEmail: string) => {
      const res = await api.post('/v1/customer/me/delete', { confirmEmail });
      return res.data as DeleteAccountResult;
    },
  });
}
