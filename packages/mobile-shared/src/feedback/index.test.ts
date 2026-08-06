import { describe, expect, it, vi } from 'vitest';
import type { AxiosInstance } from 'axios';

import { submitFeedback } from './index';

function fakeApi(baseURL: string) {
  const post = vi.fn().mockResolvedValue({ data: { data: { reference: 7, url: 'u' } } });
  return { post, defaults: { baseURL } } as unknown as AxiosInstance & { post: typeof post };
}

const input = {
  kind: 'idea',
  title: 'www',
  message: 'Something I wish the app could do.',
  app: 'vendor',
} as const;

describe('submitFeedback', () => {
  it('posts to /v1/feedback when the client stops at /api', async () => {
    const api = fakeApi('https://fe3dr.com/api');
    await submitFeedback(api, { ...input, app: 'customer' });
    expect(api.post.mock.calls[0]![0]).toBe('/v1/feedback');
  });

  // The vendor client already ends in /api/v1, so a hardcoded '/v1/feedback'
  // resolved to /api/v1/v1/feedback and 404'd on every send.
  it('does not repeat the version when the client already carries it', async () => {
    const api = fakeApi('https://vendors.fe3dr.com/api/v1');
    await submitFeedback(api, input);
    expect(api.post.mock.calls[0]![0]).toBe('/feedback');
  });
});
