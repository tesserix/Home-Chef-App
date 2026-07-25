import { describe, expect, it } from "vitest";

import { mergeMessage, mergeMessages, parseOttoEvent } from "../../support/events";
import { backoffMs, isRetryable } from "../../support/outbox";
import type { SupportMessage } from "../../support/types";

const msg = (id: string, at: string): SupportMessage => ({
  id,
  sender_type: "customer",
  sender_name: "",
  body: id,
  created_at: at,
});

describe("parseOttoEvent", () => {
  it("parses a message.created envelope", () => {
    const ev = parseOttoEvent(
      JSON.stringify({ type: "otto.message.created", payload: { message: msg("m1", "2026-01-01T00:00:00Z") } }),
    );
    expect(ev.kind).toBe("message");
  });
  it("maps conversation.closed", () => {
    expect(parseOttoEvent(JSON.stringify({ type: "otto.conversation.closed" })).kind).toBe("conversation_closed");
  });
  it("collapses garbage to unknown", () => {
    expect(parseOttoEvent("not json").kind).toBe("unknown");
    expect(parseOttoEvent(JSON.stringify({ type: "nope" })).kind).toBe("unknown");
  });
});

describe("mergeMessage", () => {
  it("dedupes by id and sorts by created_at", () => {
    let out = mergeMessages([], [msg("b", "2026-01-02T00:00:00Z"), msg("a", "2026-01-01T00:00:00Z")]);
    out = mergeMessage(out, msg("a", "2026-01-01T00:00:00Z")); // dup
    expect(out.map((m) => m.id)).toEqual(["a", "b"]);
  });
});

describe("outbox retry policy", () => {
  it("retries transport + 5xx + 429, not other 4xx", () => {
    expect(isRetryable(null)).toBe(true);
    expect(isRetryable(500)).toBe(true);
    expect(isRetryable(429)).toBe(true);
    expect(isRetryable(400)).toBe(false);
    expect(isRetryable(422)).toBe(false);
  });
  it("caps backoff at 30s", () => {
    expect(backoffMs(1)).toBe(1000);
    expect(backoffMs(99)).toBe(30_000);
  });
});
