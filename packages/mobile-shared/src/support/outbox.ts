// Durable send-outbox for the support chat. Guarantees a customer's message is
// never lost mid-way: persisted to device storage the instant the user hits
// send, shown optimistically, re-sent with backoff until otto confirms it —
// surviving network drops, backgrounding, and cold starts.
//
// Delivery is at-least-once (otto has no client idempotency key), so we retry
// only on network/5xx errors, never on a 4xx — duplicates are rare, a lost
// message impossible. Pure logic here is unit-tested; the RN wiring lives in
// useSupportChat.
import type { SupportMessage } from "./types";

// KVStorage is the minimal async key/value contract (expo-secure-store
// satisfies it). Injected by the app so this module stays platform-agnostic.
export interface KVStorage {
  getItem(key: string): Promise<string | null>;
  setItem(key: string, value: string): Promise<void>;
  removeItem(key: string): Promise<void>;
}

export type OutboxStatus = "queued" | "sending" | "failed";

export interface OutboxItem {
  clientMsgId: string;
  conversationId: string;
  body: string;
  createdAt: string;
  attempts: number;
  status: OutboxStatus;
}

const KEY_PREFIX = "otto_outbox:";

export function outboxKey(conversationId: string): string {
  return `${KEY_PREFIX}${conversationId}`;
}

/** Capped exponential backoff for re-send attempts. */
export function backoffMs(attempts: number): number {
  return Math.min(30_000, 1_000 * 2 ** Math.max(0, attempts - 1));
}

/** Renders a queued/sending/failed outbox item as an optimistic message. */
export function outboxItemToMessage(item: OutboxItem): SupportMessage & { pending: true; failed: boolean } {
  return {
    id: item.clientMsgId,
    conversation_id: item.conversationId,
    sender_type: "customer",
    sender_name: "",
    body: item.body,
    created_at: item.createdAt,
    pending: true,
    failed: item.status === "failed",
  };
}

export function addItem(items: OutboxItem[], item: OutboxItem): OutboxItem[] {
  if (items.some((i) => i.clientMsgId === item.clientMsgId)) return items;
  return [...items, item];
}

export function removeItem(items: OutboxItem[], clientMsgId: string): OutboxItem[] {
  return items.filter((i) => i.clientMsgId !== clientMsgId);
}

export function markItem(items: OutboxItem[], clientMsgId: string, patch: Partial<OutboxItem>): OutboxItem[] {
  return items.map((i) => (i.clientMsgId === clientMsgId ? { ...i, ...patch } : i));
}

export async function loadOutbox(storage: KVStorage, conversationId: string): Promise<OutboxItem[]> {
  try {
    const raw = await storage.getItem(outboxKey(conversationId));
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as OutboxItem[]) : [];
  } catch {
    return [];
  }
}

export async function saveOutbox(storage: KVStorage, conversationId: string, items: OutboxItem[]): Promise<void> {
  if (items.length === 0) {
    await storage.removeItem(outboxKey(conversationId));
    return;
  }
  await storage.setItem(outboxKey(conversationId), JSON.stringify(items));
}

/** Whether a failed send should be retried. 4xx (except 429) is terminal. */
export function isRetryable(status: number | null): boolean {
  if (status === null) return true; // transport error → retry
  if (status === 429) return true;
  return status < 400 || status >= 500;
}
