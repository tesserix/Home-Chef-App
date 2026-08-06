import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { openEventStream } from '../realtime/event-stream';

// A stand-in for RN's XHR that lets a test drive readyState, responseText and
// the error callback by hand.
class FakeXhr {
  static last: FakeXhr;
  readyState = 0;
  status = 200;
  responseText = '';
  headers: Record<string, string> = {};
  aborted = false;
  onreadystatechange: (() => void) | null = null;
  onerror: (() => void) | null = null;

  constructor() {
    FakeXhr.last = this;
  }
  open() {}
  setRequestHeader(k: string, v: string) {
    this.headers[k] = v;
  }
  send() {}
  abort() {
    this.aborted = true;
  }

  /** Append body text and fire a LOADING tick, as a real stream does. */
  push(chunk: string) {
    this.responseText += chunk;
    this.readyState = 3;
    this.onreadystatechange?.();
  }
  /** The transport dropped: RN fires onerror AND settles readyState to DONE. */
  fail() {
    this.readyState = 4;
    this.onerror?.();
    this.onreadystatechange?.();
  }
}

beforeEach(() => {
  (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest = FakeXhr;
});
afterEach(() => {
  delete (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest;
});

describe('openEventStream', () => {
  it('emits one frame per complete data block and buffers a partial tail', () => {
    const onFrame = vi.fn();
    openEventStream('https://x/sse', 'tok', onFrame);

    FakeXhr.last.push('data: {"a":1}\n\ndata: {"a":2}\n\ndata: {"a":3');
    expect(onFrame.mock.calls.map((c) => c[0])).toEqual(['{"a":1}', '{"a":2}']);

    FakeXhr.last.push('}\n\n');
    expect(onFrame).toHaveBeenCalledTimes(3);
    expect(onFrame).toHaveBeenLastCalledWith('{"a":3}');
  });

  it('ignores heartbeat comment lines', () => {
    const onFrame = vi.fn();
    openEventStream('https://x/sse', 'tok', onFrame);

    FakeXhr.last.push(': ping\n\n');
    expect(onFrame).not.toHaveBeenCalled();
  });

  it('sends the bearer header only when a token is given', () => {
    openEventStream('https://x/sse', 'tok', vi.fn());
    expect(FakeXhr.last.headers.Authorization).toBe('Bearer tok');

    openEventStream('https://x/sse', '', vi.fn());
    expect(FakeXhr.last.headers.Authorization).toBeUndefined();
  });

  // A dropped stream fires onerror and settles to DONE. Reporting both told the
  // caller it had failed twice: two reconnect timers, one of which was never
  // cleared, so every drop leaked a live stream that kept delivering frames.
  it('reports a failure at most once', () => {
    const onError = vi.fn();
    openEventStream('https://x/sse', 'tok', vi.fn(), onError);

    FakeXhr.last.fail();
    expect(onError).toHaveBeenCalledTimes(1);
  });

  // A stream the server refused (400 no_active_delivery, 404, 401) will refuse
  // the next attempt for the same reason. The caller can only tell that from a
  // transport drop if it is given the status.
  it('reports the HTTP status the server rejected with', () => {
    const onError = vi.fn();
    openEventStream('https://x/sse', 'tok', vi.fn(), onError);

    FakeXhr.last.status = 400;
    FakeXhr.last.fail();

    expect(onError).toHaveBeenCalledWith(400);
  });

  it('reports status 0 when the transport never reached the server', () => {
    const onError = vi.fn();
    openEventStream('https://x/sse', 'tok', vi.fn(), onError);

    FakeXhr.last.status = 0;
    FakeXhr.last.fail();

    expect(onError).toHaveBeenCalledWith(0);
  });

  it('stays silent after the caller closes it', () => {
    const onError = vi.fn();
    const handle = openEventStream('https://x/sse', 'tok', vi.fn(), onError);

    handle.close();
    FakeXhr.last.fail();

    expect(FakeXhr.last.aborted).toBe(true);
    expect(onError).not.toHaveBeenCalled();
  });
});
