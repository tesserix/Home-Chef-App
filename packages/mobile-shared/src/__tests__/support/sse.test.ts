import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { openSse } from '../../support/sse';

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

  headersReceived(status = 200) {
    this.status = status;
    this.readyState = 2;
    this.onreadystatechange?.();
  }
  push(chunk: string) {
    this.responseText += chunk;
    this.readyState = 3;
    this.onreadystatechange?.();
  }
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

describe('openSse', () => {
  it('opens on a 200 and emits complete frames only', () => {
    const onOpen = vi.fn();
    const onFrame = vi.fn();
    openSse({ url: 'https://x/sse', onFrame, onOpen });

    FakeXhr.last.headersReceived(200);
    expect(onOpen).toHaveBeenCalledTimes(1);

    FakeXhr.last.push('data: one\n\n: ping\n\ndata: tw');
    expect(onFrame.mock.calls.map((c) => c[0])).toEqual(['one']);

    FakeXhr.last.push('o\n\n');
    expect(onFrame).toHaveBeenLastCalledWith('two');
  });

  it('closes as unauthorized on a 401 without waiting for the body', () => {
    const onClose = vi.fn();
    openSse({ url: 'https://x/sse', onFrame: vi.fn(), onClose });

    FakeXhr.last.headersReceived(401);

    expect(onClose).toHaveBeenCalledExactlyOnceWith('unauthorized');
    expect(FakeXhr.last.aborted).toBe(true);
  });

  // A dropped stream fires onerror and settles to DONE; reporting both made the
  // caller count two failures and double its reconnect backoff for one drop.
  it('reports the close at most once', () => {
    const onClose = vi.fn();
    openSse({ url: 'https://x/sse', onFrame: vi.fn(), onClose });

    FakeXhr.last.fail();

    expect(onClose).toHaveBeenCalledExactlyOnceWith('error');
  });

  it('stays silent after the caller closes it', () => {
    const onClose = vi.fn();
    const handle = openSse({ url: 'https://x/sse', onFrame: vi.fn(), onClose });

    handle.close();
    FakeXhr.last.fail();

    expect(FakeXhr.last.aborted).toBe(true);
    expect(onClose).not.toHaveBeenCalled();
  });
});
