import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Support tickets — the web twin of apps/mobile-vendor/app/support/*.
//
// The portal had no help surface at all: a chef hitting a problem on the web had
// nowhere to report it and no way to read a reply. The endpoints have existed
// throughout and the mobile app has used them.
//
// Backed by /support/tickets{,/:id,/:id/messages,/:id/close}.

export type TicketCategory =
  | 'order_issue'
  | 'payment_issue'
  | 'account_issue'
  | 'chef_complaint'
  | 'delivery_complaint'
  | 'technical'
  | 'other';

export type TicketStatus =
  | 'open'
  | 'in_progress'
  | 'waiting_on_customer'
  | 'waiting_on_chef'
  | 'resolved'
  | 'closed';

/** Mirrors models.SupportTicket. */
export interface SupportTicket {
  id: string;
  ticketNumber: string;
  category: TicketCategory;
  priority: string;
  status: TicketStatus;
  subject: string;
  description: string;
  resolution?: string;
  orderId?: string;
  resolvedAt?: string;
  closedAt?: string;
  createdAt: string;
  updatedAt: string;
  messages?: TicketMessage[];
}

export interface TicketMessage {
  id: string;
  content: string;
  createdAt: string;
  senderId?: string;
  sender?: { id: string; firstName?: string; lastName?: string; role?: string };
}

/** Categories a CHEF would plausibly raise, in the order they'd look for them. */
export const CHEF_TICKET_CATEGORIES: { value: TicketCategory; label: string }[] = [
  { value: 'order_issue', label: 'A problem with an order' },
  { value: 'payment_issue', label: 'Payments or payouts' },
  { value: 'account_issue', label: 'My account or verification' },
  { value: 'delivery_complaint', label: 'Delivery or the rider' },
  { value: 'technical', label: 'Something is broken' },
  { value: 'other', label: 'Something else' },
];

export function useMyTickets() {
  return useQuery<SupportTicket[]>({
    queryKey: ['support', 'tickets'],
    queryFn: () =>
      apiClient.get<SupportTicket[] | { data: SupportTicket[] }>('/support/tickets').then((r) =>
        // The endpoint has returned both a bare array and a {data} envelope
        // depending on pagination; accept either rather than rendering nothing.
        Array.isArray(r) ? r : (r?.data ?? []),
      ),
    staleTime: 30_000,
  });
}

export function useTicket(id: string | undefined) {
  return useQuery<SupportTicket>({
    queryKey: ['support', 'ticket', id],
    queryFn: () => apiClient.get<SupportTicket>(`/support/tickets/${id}`),
    enabled: Boolean(id),
  });
}

interface CreateTicketInput {
  category: TicketCategory;
  subject: string;
  description: string;
  orderId?: string;
}

export function useCreateTicket() {
  const qc = useQueryClient();
  return useMutation<SupportTicket, unknown, CreateTicketInput>({
    mutationFn: (input) => apiClient.post<SupportTicket>('/support/tickets', input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['support', 'tickets'] });
    },
  });
}

export function useAddTicketMessage(ticketId: string) {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, string>({
    mutationFn: (content: string) =>
      apiClient.post(`/support/tickets/${ticketId}/messages`, { content }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['support', 'ticket', ticketId] });
    },
  });
}

export function useCloseTicket(ticketId: string) {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.put(`/support/tickets/${ticketId}/close`),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['support', 'ticket', ticketId] });
      void qc.invalidateQueries({ queryKey: ['support', 'tickets'] });
    },
  });
}

/** Presentation for a ticket status. `waiting_on_chef` is the chef's move. */
export const TICKET_STATUS_META: Record<
  TicketStatus,
  { label: string; variant: 'success' | 'warning' | 'info' | 'secondary' }
> = {
  open: { label: 'Open', variant: 'info' },
  in_progress: { label: 'In progress', variant: 'info' },
  waiting_on_customer: { label: 'Waiting on customer', variant: 'warning' },
  waiting_on_chef: { label: 'Waiting on you', variant: 'warning' },
  resolved: { label: 'Resolved', variant: 'success' },
  closed: { label: 'Closed', variant: 'secondary' },
};
