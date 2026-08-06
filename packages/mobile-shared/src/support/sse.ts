// Minimal server-sent-events reader for React Native.
//
// RN has no EventSource, and its fetch cannot stream a response body, so this
// drives XMLHttpRequest and parses the incrementally-growing responseText.
// The stream comes from the HomeChef API (not otto directly): the app is
// already authenticated to that origin, so there is no ticket to hold and no
// cross-origin upgrade to negotiate — which is exactly why this works where
// the WebSocket does not.
export interface SseHandle {
  close: () => void;
}

export interface SseOptions {
  url: string;
  /** Authorization header value, e.g. `Bearer <token>`. */
  authorization?: string;
  /** Called once per `data:` frame with the raw payload string. */
  onFrame: (data: string) => void;
  /** Called when the stream opens (HEADERS_RECEIVED with 200). */
  onOpen?: () => void;
  /** Called when the stream ends or errors; the caller decides on retry. */
  onClose?: (reason: "error" | "end" | "unauthorized") => void;
}

/**
 * Opens an SSE stream. Returns a handle whose close() aborts it.
 * Frames are delimited by a blank line; only `data:` lines are surfaced
 * (otto's heartbeat is a `: ping` comment and is ignored by construction).
 */
export function openSse(opts: SseOptions): SseHandle {
  const xhr = new XMLHttpRequest();
  let consumed = 0;
  let closedByUs = false;
  let opened = false;
  // A dropped stream fires onerror AND settles to DONE. Reporting both made the
  // caller count two failures for one drop and double its reconnect backoff.
  let reported = false;
  const report = (reason: "error" | "end" | "unauthorized") => {
    if (closedByUs || reported) return;
    reported = true;
    opts.onClose?.(reason);
  };

  const emitFrom = (text: string) => {
    // Only parse what is newly arrived, and only up to the last complete
    // frame — a partial tail stays buffered until its blank line lands.
    const fresh = text.slice(consumed);
    const lastBreak = fresh.lastIndexOf("\n\n");
    if (lastBreak === -1) return;
    const ready = fresh.slice(0, lastBreak);
    consumed += lastBreak + 2;

    for (const block of ready.split("\n\n")) {
      const data = block
        .split("\n")
        .filter((l) => l.startsWith("data:"))
        .map((l) => l.slice(5).trimStart())
        .join("\n");
      if (data) opts.onFrame(data);
    }
  };

  xhr.open("GET", opts.url, true);
  xhr.setRequestHeader("Accept", "text/event-stream");
  xhr.setRequestHeader("Cache-Control", "no-cache");
  if (opts.authorization) xhr.setRequestHeader("Authorization", opts.authorization);

  xhr.onreadystatechange = () => {
    if (closedByUs) return;
    if (xhr.readyState === 2 /* HEADERS_RECEIVED */) {
      if (xhr.status === 401 || xhr.status === 403) {
        report("unauthorized");
        closedByUs = true;
        xhr.abort();
        return;
      }
      if (xhr.status === 200) {
        opened = true;
        opts.onOpen?.();
      }
      return;
    }
    if (xhr.readyState === 3 /* LOADING */) {
      emitFrom(xhr.responseText ?? "");
      return;
    }
    if (xhr.readyState === 4 /* DONE */) {
      emitFrom(xhr.responseText ?? "");
      report(opened ? "end" : "error");
    }
  };
  xhr.onerror = () => report("error");
  xhr.send();

  return {
    close: () => {
      if (closedByUs) return;
      closedByUs = true;
      try {
        xhr.abort();
      } catch {
        // already finished
      }
    },
  };
}
