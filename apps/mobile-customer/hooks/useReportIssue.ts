import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Report an order issue (#37) → instant/assisted refund to the wallet. Mirrors
// the review flow's multipart upload. The refund amount is decided server-side.

export type IssueReason = 'missing_item' | 'quality_issue' | 'wrong_item' | 'damaged' | 'other';

export interface ReportIssueInput {
  orderId: string;
  reason: IssueReason;
  description?: string;
  affectedItemIds?: string[];
  // Up to a few evidence photos (camera or library). The backend accepts
  // repeated `photo` fields.
  photoUris?: string[];
}

export interface ReportIssueResult {
  issueId: string;
  status: 'pending' | 'auto_refunded' | 'resolved' | 'rejected';
  refundAmount: number;
  message: string;
}

export function useReportIssue() {
  const queryClient = useQueryClient();
  return useMutation<ReportIssueResult, Error, ReportIssueInput>({
    mutationFn: async (input) => {
      const fd = new FormData();
      fd.append('reason', input.reason);
      if (input.description?.trim()) fd.append('description', input.description.trim());
      for (const id of input.affectedItemIds ?? []) fd.append('affectedItemIds', id);
      // Repeated `photo` fields — React Native FormData file shape. The backend
      // caps the count and ignores extras.
      (input.photoUris ?? []).forEach((uri, i) => {
        const name = uri.split('/').pop() ?? `issue-${i}.jpg`;
        const type = name.toLowerCase().endsWith('.png') ? 'image/png' : 'image/jpeg';
        fd.append('photo', { uri, name, type } as unknown as Blob);
      });
      const r = await api.post(`/v1/orders/${input.orderId}/report-issue`, fd, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      return r.data as ReportIssueResult;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['orders'] });
      void queryClient.invalidateQueries({ queryKey: ['wallet'] });
      void queryClient.invalidateQueries({ queryKey: ['wallet-transactions'] });
    },
  });
}
