// Shared support-chat types + Zod schemas for the mobile support kit. Shapes
// mirror otto's storefront responses relayed by the HomeChef API proxy;
// schemas are lenient (passthrough) on the conversation so a future otto field
// addition doesn't break parsing.
import { z } from "zod";

// "assistant" is slm-router's AI reply. Omitting it made every bot message
// fail schema validation, so the answer sat in otto while the thread looked
// empty — the app rendered nothing and stayed on "Connecting…".
export const SupportSenderTypeSchema = z.enum(["customer", "staff", "system", "assistant"]);
export type SupportSenderType = z.infer<typeof SupportSenderTypeSchema>;

export const SupportStatusSchema = z.enum(["pending", "active", "closed"]);
export type SupportStatus = z.infer<typeof SupportStatusSchema>;

export const SupportMessageSchema = z.object({
  id: z.string(),
  conversation_id: z.string().optional(),
  sender_type: SupportSenderTypeSchema,
  sender_name: z.string().optional().default(""),
  body: z.string(),
  created_at: z.string(),
});
export type SupportMessage = z.infer<typeof SupportMessageSchema>;

export const SupportConversationSchema = z
  .object({
    id: z.string(),
    case_id: z.string().optional().default(""),
    status: SupportStatusSchema,
    subject: z.string().optional().default(""),
    created_at: z.string().optional(),
    closed_at: z.string().optional(),
    // True once the thread is handed to a person; the UI stops offering a
    // handoff that is already under way.
    needs_human: z.boolean().optional().default(false),
  })
  .passthrough();
export type SupportConversation = z.infer<typeof SupportConversationSchema>;

export const QueueStateSchema = z.object({
  status: SupportStatusSchema,
  position: z.number().default(0),
  total_pending: z.number().default(0),
  estimated_wait_seconds: z.number().default(0),
  all_busy: z.boolean().default(false),
});
export type QueueState = z.infer<typeof QueueStateSchema>;

export const WsTicketSchema = z.object({
  ticket: z.string(),
  ws_url: z.string(),
});
export type WsTicket = z.infer<typeof WsTicketSchema>;

/** An intake reason offered before the first message is sent. */
export interface IntakeReason {
  value: string;
  label: string;
  /** When false, the free-text "what's going on" field is optional. */
  requiresStatus?: boolean;
}

/** Input for opening a new support conversation. */
export interface CreateConversationInput {
  message: string;
  reason: string;
  statusInfo: string;
  subject?: string;
  name?: string;
  email?: string;
  /** Required by otto only for order/account reasons — never used by HomeChef. */
  dob?: string;
}

/** Post-case survey payload. */
export interface FeedbackInput {
  call_rating: number;
  query_resolved: boolean;
  staff_rating: number;
  comments?: string;
}

/** Maps the friendly input onto otto's create wire shape. */
export function toCreateBody(input: CreateConversationInput): Record<string, unknown> {
  return {
    message: input.message,
    reason: input.reason,
    status_info: input.statusInfo,
    subject: input.subject ?? "",
    name: input.name ?? "",
    email: input.email ?? "",
    dob: input.dob ?? "",
  };
}
