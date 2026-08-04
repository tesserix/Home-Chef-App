import { describe, it, expect } from 'vitest';
import { wsOriginFrom, wsEndpointUrl, fetchWSTicket } from './ws-ticket';
import { socketReconnectDelayWithJitterMs, socketReconnectDelayMs } from '../utils/socket-backoff';

describe('wsOriginFrom', () => {
  // The two apps configure the base URL differently and the /ws routes are
  // siblings of /api, not children — both suffixes must come off or the URL
  // lands back on the BFF path that cannot carry an upgrade.
  it('strips the /api suffix (customer app)', () => {
    expect(wsOriginFrom('https://fe3dr.com/api')).toBe('wss://fe3dr.com');
  });

  it('strips the /api/v1 suffix (vendor app)', () => {
    expect(wsOriginFrom('https://fe3dr.com/api/v1')).toBe('wss://fe3dr.com');
  });

  it('strips a trailing slash before the suffix', () => {
    expect(wsOriginFrom('https://fe3dr.com/api/')).toBe('wss://fe3dr.com');
  });

  it('maps http to ws for local development', () => {
    expect(wsOriginFrom('http://localhost:8080/api')).toBe('ws://localhost:8080');
  });

  it('leaves a host with no api suffix alone', () => {
    expect(wsOriginFrom('https://fe3dr.com')).toBe('wss://fe3dr.com');
  });
});

describe('wsEndpointUrl', () => {
  it('builds a top-level /ws URL carrying the ticket', () => {
    expect(wsEndpointUrl('https://fe3dr.com/api', 'notifications', 'tk123')).toBe(
      'wss://fe3dr.com/ws/notifications?ticket=tk123',
    );
  });

  it('never produces the /api/v1 path the upgrade cannot use', () => {
    const url = wsEndpointUrl('https://fe3dr.com/api', 'orders/abc/track', 'tk');
    expect(url).toBe('wss://fe3dr.com/ws/orders/abc/track?ticket=tk');
    expect(url).not.toContain('/api/');
  });

  it('tolerates a leading slash on the path', () => {
    expect(wsEndpointUrl('https://fe3dr.com/api', '/notifications', 'tk')).toBe(
      'wss://fe3dr.com/ws/notifications?ticket=tk',
    );
  });

  // A ticket is base64url + '.', but encoding it keeps the URL correct even if
  // the token format ever changes.
  it('url-encodes the ticket', () => {
    expect(wsEndpointUrl('https://fe3dr.com/api', 'notifications', 'a+b/c=')).toContain(
      'ticket=a%2Bb%2Fc%3D',
    );
  });
});

describe('fetchWSTicket', () => {
  const asApi = (impl: unknown) => impl as Parameters<typeof fetchWSTicket>[0];

  it('returns the minted ticket', async () => {
    const api = asApi({
      defaults: { baseURL: 'https://fe3dr.com/api' },
      post: async () => ({ data: { ticket: 'minted' } }),
    });
    await expect(fetchWSTicket(api)).resolves.toBe('minted');
  });

  it('uses the /v1 prefix when the base URL lacks a version', async () => {
    let called = '';
    const api = asApi({
      defaults: { baseURL: 'https://fe3dr.com/api' },
      post: async (p: string) => {
        called = p;
        return { data: { ticket: 't' } };
      },
    });
    await fetchWSTicket(api);
    expect(called).toBe('/v1/realtime/ws-ticket');
  });

  it('does not double the version when the base URL already has one', async () => {
    let called = '';
    const api = asApi({
      defaults: { baseURL: 'https://fe3dr.com/api/v1' },
      post: async (p: string) => {
        called = p;
        return { data: { ticket: 't' } };
      },
    });
    await fetchWSTicket(api);
    expect(called).toBe('/realtime/ws-ticket');
  });

  // A signed-out user or a network blip must look like an ordinary failure the
  // caller can back off on — not an exception escaping into a render.
  it('returns null instead of throwing when the mint fails', async () => {
    const api = asApi({
      defaults: { baseURL: 'https://fe3dr.com/api' },
      post: async () => {
        throw new Error('401');
      },
    });
    await expect(fetchWSTicket(api)).resolves.toBeNull();
  });

  it('returns null when the response carries no ticket', async () => {
    const api = asApi({
      defaults: { baseURL: 'https://fe3dr.com/api' },
      post: async () => ({ data: {} }),
    });
    await expect(fetchWSTicket(api)).resolves.toBeNull();
  });
});

describe('socketReconnectDelayWithJitterMs', () => {
  // Jitter must only ever subtract, so the ceiling that keeps this off a hot
  // loop still holds.
  it('never exceeds the un-jittered delay', () => {
    for (let f = 1; f <= 10; f += 1) {
      expect(socketReconnectDelayWithJitterMs(f, () => 0)).toBe(socketReconnectDelayMs(f));
      expect(socketReconnectDelayWithJitterMs(f, () => 1)).toBeLessThan(
        socketReconnectDelayMs(f),
      );
    }
  });

  it('subtracts at most 30% at full jitter', () => {
    const base = socketReconnectDelayMs(8); // at the 30s cap
    expect(socketReconnectDelayWithJitterMs(8, () => 1)).toBe(Math.round(base * 0.7));
  });

  it('stays positive so a reconnect is never scheduled at zero delay', () => {
    for (let f = 0; f <= 10; f += 1) {
      expect(socketReconnectDelayWithJitterMs(f, () => 1)).toBeGreaterThan(0);
    }
  });

  // The whole point: two clients failing at the same instant must not retry in
  // the same millisecond.
  it('spreads concurrent clients across a range', () => {
    const delays = new Set(
      Array.from({ length: 50 }, (_, i) => socketReconnectDelayWithJitterMs(8, () => i / 50)),
    );
    expect(delays.size).toBeGreaterThan(10);
  });
});
