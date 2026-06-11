import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from '@tanstack/react-query';
import { api } from '../lib/api';

// Support ticketing — mirrors the Home Chef API at /api/v1/support/tickets
// (apps/api/handlers/support.go). The vendor base URL already includes
// `/api/v1`, so paths here are relative to that ("/support/tickets").
//
// Two shapes come off the wire:
//   - list  (GET /support/tickets)      → { data: SupportTicket[], pagination }
//     where each row is the RAW gorm struct (no messages, no reporterName).
//   - detail (GET /support/tickets/:id) → SupportTicketResponse (bare object)
//     a DTO that adds `reporterName` + `messages[]` and strips PII.

export type TicketCategory =
  | 'order_issue'
  | 'payment_issue'
  | 'account_issue'
  | 'chef_complaint'
  | 'delivery_complaint'
  | 'technical'
  | 'other';

export type TicketPriority = 'low' | 'medium' | 'high' | 'urgent';

export type TicketStatus =
  | 'open'
  | 'in_progress'
  | 'waiting_on_customer'
  | 'waiting_on_chef'
  | 'resolved'
  | 'closed';

// List row — the raw SupportTicket JSON. Fields beyond these exist on the
// struct (resolution, deletedAt) but the UI never reads them.
export interface SupportTicket {
  id: string;
  ticketNumber: string;
  reporterId: string;
  reporterRole: string;
  assignedToId?: string;
  orderId?: string;
  category: TicketCategory;
  priority: TicketPriority;
  status: TicketStatus;
  subject: string;
  description: string;
  resolvedAt?: string;
  closedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SupportMessage {
  id: string;
  ticketId: string;
  senderId: string;
  senderRole: string;
  senderName?: string;
  content: string;
  isInternal: boolean;
  createdAt: string;
}

// Detail DTO — superset of the list row with reporterName + messages.
export interface SupportTicketDetail {
  id: string;
  ticketNumber?: string;
  subject: string;
  description?: string;
  category?: TicketCategory;
  priority?: TicketPriority;
  status: TicketStatus;
  reporterId: string;
  reporterName?: string;
  reporterRole?: string;
  orderId?: string;
  assignedToId?: string;
  createdAt: string;
  updatedAt: string;
  resolvedAt?: string;
  closedAt?: string;
  messages?: SupportMessage[];
}

interface TicketListResponse {
  data: SupportTicket[];
  pagination?: {
    page: number;
    limit: number;
    total: number;
  };
}

export interface CreateTicketInput {
  category: TicketCategory;
  subject: string;
  description: string;
  priority?: TicketPriority;
  orderId?: string;
}

const SUPPORT_KEY = ['chef', 'support'] as const;

export function useSupportTickets(): UseQueryResult<SupportTicket[]> {
  return useQuery({
    queryKey: [...SUPPORT_KEY, 'list'],
    queryFn: () =>
      api
        .get<TicketListResponse>('/support/tickets')
        .then((r) => r.data?.data ?? []),
    staleTime: 30_000,
  });
}

export function useTicket(id: string): UseQueryResult<SupportTicketDetail> {
  return useQuery({
    queryKey: [...SUPPORT_KEY, 'detail', id],
    queryFn: () =>
      api
        .get<SupportTicketDetail>(`/support/tickets/${id}`)
        .then((r) => r.data),
    enabled: id.length > 0,
    staleTime: 15_000,
  });
}

export function useCreateTicket() {
  const qc = useQueryClient();
  return useMutation<SupportTicket, Error, CreateTicketInput>({
    mutationFn: (input) =>
      api.post<SupportTicket>('/support/tickets', input).then((r) => r.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [...SUPPORT_KEY, 'list'] });
    },
  });
}

export function useAddMessage(ticketId: string) {
  const qc = useQueryClient();
  return useMutation<SupportMessage, Error, string>({
    mutationFn: (content) =>
      api
        .post<SupportMessage>(`/support/tickets/${ticketId}/messages`, {
          content,
        })
        .then((r) => r.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [...SUPPORT_KEY, 'detail', ticketId] });
      qc.invalidateQueries({ queryKey: [...SUPPORT_KEY, 'list'] });
    },
  });
}

export function useCloseTicket(ticketId: string) {
  const qc = useQueryClient();
  return useMutation<void, Error, void>({
    mutationFn: () =>
      api.put(`/support/tickets/${ticketId}/close`).then(() => undefined),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [...SUPPORT_KEY, 'detail', ticketId] });
      qc.invalidateQueries({ queryKey: [...SUPPORT_KEY, 'list'] });
    },
  });
}

// ---------- Shared presentation helpers ----------

export const CATEGORY_LABEL: Record<TicketCategory, string> = {
  order_issue: 'An order',
  payment_issue: 'Payments & payouts',
  account_issue: 'Account & verification',
  chef_complaint: 'Complaint',
  delivery_complaint: 'Delivery',
  technical: 'App problem',
  other: 'Something else',
};

export const STATUS_LABEL: Record<TicketStatus, string> = {
  open: 'Open',
  in_progress: 'In progress',
  waiting_on_customer: 'In progress',
  waiting_on_chef: 'Action needed',
  resolved: 'Resolved',
  closed: 'Closed',
};
