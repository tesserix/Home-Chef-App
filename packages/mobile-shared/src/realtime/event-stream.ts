// Server-Sent Events over XHR — the real-time transport that actually works on
// React Native (#982).
//
// React Native's WebSocket does its own TLS through SocketRocket/CFStream
// rather than NSURLSession, and against our edge that handshake fails outright:
// close code 1006, `OSStatus -9836` (errSSLProtocol). The socket never
// completes TLS, so no HTTP request is ever made and no amount of auth,
// backoff or URL fixing helps. Everything that goes through NSURLSession —
// axios, fetch, and this XHR — reaches the same host over the same TLS without
// trouble, which is what makes SSE the dependable choice on native.
//
// Same NATS-backed stream, same frames as the socket; only the pipe differs.
// The vendor app has carried this as its WebSocket fallback since #892 and it
// is the reason chefs have had live updates at all.

/** A live stream handle. `close` is idempotent. */
export interface EventStreamHandle {
  close: () => void;
}

/**
 * Opens an SSE stream and calls `onFrame` with each `data:` payload.
 *
 * XHR rather than EventSource: RN has no EventSource, and EventSource cannot
 * set an Authorization header anyway.
 */
export function openEventStream(
  url: string,
  token: string,
  onFrame: (raw: string) => void,
  onError?: () => void,
): EventStreamHandle {
  const xhr = new XMLHttpRequest();
  // How much of responseText has already been turned into frames. XHR keeps the
  // whole response in memory and grows it, so we parse only the tail each time.
  let consumed = 0;
  let closed = false;

  const drain = () => {
    const text = xhr.responseText ?? '';
    // Only complete frames (terminated by a blank line) are safe to parse — the
    // tail may be half a frame still in flight.
    const lastBreak = text.lastIndexOf('\n\n');
    if (lastBreak < consumed) return;
    const chunk = text.slice(consumed, lastBreak);
    consumed = lastBreak + 2;
    for (const frame of chunk.split('\n\n')) {
      for (const line of frame.split('\n')) {
        // `:` lines are comments (our heartbeat) — ignore them.
        if (line.startsWith('data:')) onFrame(line.slice(5).trim());
      }
    }
  };

  xhr.open('GET', url);
  xhr.setRequestHeader('Authorization', `Bearer ${token}`);
  xhr.setRequestHeader('Accept', 'text/event-stream');
  xhr.onreadystatechange = () => {
    // 3 = LOADING: body is arriving. Parsing here rather than on completion is
    // the whole point — this response never completes.
    if (xhr.readyState >= 3) drain();
    // 4 = DONE: the stream ended. A stream that ends is a failure by
    // definition, so let the caller reconnect — unless we closed it ourselves.
    if (xhr.readyState === 4 && !closed) onError?.();
  };
  xhr.onerror = () => {
    if (!closed) onError?.();
  };
  xhr.send();

  return {
    close: () => {
      closed = true;
      xhr.abort();
    },
  };
}
