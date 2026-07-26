// useModeration.ts — reporting content and blocking users.
//
// Backed by POST /v1/reports, GET|POST /v1/blocks and DELETE /v1/blocks/:userId.
//
// Required by App Review guideline 1.2: an app carrying user-generated content
// must let users report objectionable content and block abusive users. The
// customer app's UGC surfaces are the chef social feed (posts + comments), chef
// reviews, and order messaging.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ReportReason, ReportTargetType } from '@homechef/mobile-shared/ui';
import { api } from '../lib/api';

export interface BlockedAccount {
  userId: string;
  name: string;
  reason?: string;
  createdAt: string;
}

/**
 * File a report.
 *
 * The server answers identically for a first report and a duplicate, so the
 * caller never has to special-case a double tap.
 */
export function useReportContent() {
  return useMutation({
    mutationFn: async (input: {
      targetType: ReportTargetType;
      targetId: string;
      reason: ReportReason;
      details?: string;
    }) => {
      const res = await api.post('/v1/reports', {
        targetType: input.targetType,
        targetId: input.targetId,
        reason: input.reason,
        details: input.details ?? '',
      });
      return res.data as { status: string; reportId: string; message: string };
    },
  });
}

/**
 * Block a user.
 *
 * Invalidates the feed and review queries on success, because blocked authors
 * are filtered server-side — without this the blocked chef's posts stay on
 * screen until the cache expires, which reads as "blocking did nothing".
 */
export function useBlockUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { userId: string; reason?: ReportReason }) => {
      const res = await api.post('/v1/blocks', {
        userId: input.userId,
        reason: input.reason ?? 'other',
      });
      return res.data as { status: string };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['social-feed'] });
      queryClient.invalidateQueries({ queryKey: ['chef-reviews'] });
      queryClient.invalidateQueries({ queryKey: ['blocked-accounts'] });
    },
  });
}

/** Reverse a block. Apple expects blocking to be undoable. */
export function useUnblockUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (userId: string) => {
      const res = await api.delete(`/v1/blocks/${userId}`);
      return res.data as { status: string };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['social-feed'] });
      queryClient.invalidateQueries({ queryKey: ['chef-reviews'] });
      queryClient.invalidateQueries({ queryKey: ['blocked-accounts'] });
    },
  });
}

/** Everyone the signed-in customer has blocked, for the manage-blocks screen. */
export function useBlockedAccounts() {
  return useQuery({
    queryKey: ['blocked-accounts'],
    queryFn: async () => {
      const res = await api.get('/v1/blocks');
      return (res.data?.blocks ?? []) as BlockedAccount[];
    },
  });
}
