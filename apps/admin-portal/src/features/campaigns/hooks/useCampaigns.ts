import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Marketing campaigns admin API (#56). Mirrors the Go admin endpoints under
// /admin/campaigns: CRUD, audience preview, schedule/send/test, and metrics.

export interface SegmentCriteria {
  roles?: string[];
  recency?: '' | 'active' | 'lapsed';
  recencyDays?: number;
  cities?: string[];
  subscription?: '' | 'active' | 'paused' | 'none';
  newWithinDays?: number;
}

export interface Campaign {
  id: string;
  name: string;
  status: 'draft' | 'scheduled' | 'queued' | 'sending' | 'sent' | 'cancelled';
  sendPush: boolean;
  sendEmail: boolean;
  pushTitle: string;
  pushBody: string;
  emailSubject: string;
  emailHtml: string;
  segment: string; // JSON SegmentCriteria
  scheduledAt?: string;
  sentAt?: string;
  recipients: number;
  createdAt: string;
}

export interface CampaignInput {
  name: string;
  sendPush: boolean;
  sendEmail: boolean;
  pushTitle: string;
  pushBody: string;
  emailSubject: string;
  emailHtml: string;
  segment: SegmentCriteria;
}

export interface SegmentPreview {
  matched: number;
  reachablePush: number;
  reachableEmail: number;
}

export interface ChannelMetrics {
  sent: number;
  failed: number;
  opened: number;
}

export interface CampaignMetrics {
  recipients: number;
  push: ChannelMetrics;
  email: ChannelMetrics;
}

interface CampaignPage {
  data: Campaign[];
}

export function useCampaigns() {
  return useQuery({
    queryKey: ['admin-campaigns'],
    queryFn: () => apiClient.get<CampaignPage>('/admin/campaigns', { page: 1, limit: 100 }),
  });
}

export function useCampaignMetrics(id: string | null) {
  return useQuery({
    queryKey: ['admin-campaign-metrics', id],
    queryFn: () => apiClient.get<CampaignMetrics>(`/admin/campaigns/${id}/metrics`),
    enabled: !!id,
    refetchInterval: 30_000,
  });
}

export function useCreateCampaign() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CampaignInput) => apiClient.post<Campaign>('/admin/campaigns', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-campaigns'] }),
  });
}

export function useUpdateCampaign() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: CampaignInput }) =>
      apiClient.put<Campaign>(`/admin/campaigns/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-campaigns'] }),
  });
}

/** Preview the matched vs reachable audience for a segment + channels. */
export function usePreviewSegment() {
  return useMutation({
    mutationFn: (body: { segment: SegmentCriteria; sendPush: boolean; sendEmail: boolean }) =>
      apiClient.post<SegmentPreview>('/admin/campaigns/preview', body),
  });
}

export function useCampaignAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, body }: { id: string; action: 'send' | 'schedule' | 'test' | 'cancel'; body?: unknown }) =>
      apiClient.post(`/admin/campaigns/${id}/${action}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-campaigns'] }),
  });
}

/** Parse a campaign's stored segment JSON (best-effort). */
export function parseSegment(segment: string): SegmentCriteria {
  try {
    return JSON.parse(segment || '{}') as SegmentCriteria;
  } catch {
    return {};
  }
}
