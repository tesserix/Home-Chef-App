import { describe, expect, it } from 'vitest';
import type { AxiosInstance } from 'axios';

import { apiVersionPrefix } from './version-prefix';

const client = (baseURL?: string) =>
  ({ defaults: { baseURL } }) as AxiosInstance;

describe('apiVersionPrefix', () => {
  it('adds the version when the client stops at /api (customer)', () => {
    expect(apiVersionPrefix(client('https://fe3dr.com/api'))).toBe('/v1');
  });

  it('adds nothing when the client already carries it (vendor, delivery)', () => {
    expect(apiVersionPrefix(client('https://vendors.fe3dr.com/api/v1'))).toBe('');
    expect(apiVersionPrefix(client('https://delivery.fe3dr.com/api/v1'))).toBe('');
  });

  it('ignores a trailing slash', () => {
    expect(apiVersionPrefix(client('http://localhost:8090/api/v1/'))).toBe('');
    expect(apiVersionPrefix(client('http://localhost:8080/api/'))).toBe('/v1');
  });

  it('handles a future version and a missing base URL', () => {
    expect(apiVersionPrefix(client('https://fe3dr.com/api/v2'))).toBe('');
    expect(apiVersionPrefix(client(undefined))).toBe('/v1');
  });
});
