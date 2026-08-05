import { describe, it, expect, vi } from 'vitest';

import { validateFeedback, submitFeedback, FEEDBACK_MESSAGE_MAX } from '../feedback';

describe('validateFeedback', () => {
  it('accepts a filled-in report', () => {
    expect(validateFeedback({ title: 'Receipt is hard to read', message: 'The totals run together.' })).toBeNull();
  });

  it('asks for a title first', () => {
    expect(validateFeedback({ title: '   ', message: 'Something' })).toMatch(/title/i);
  });

  it('asks for a message', () => {
    expect(validateFeedback({ title: 'Dark mode', message: '  ' })).toMatch(/tell us|message/i);
  });

  it('rejects a message that is too short to act on', () => {
    expect(validateFeedback({ title: 'Bug', message: 'bad' })).toMatch(/bit more|short/i);
  });

  it('rejects a message past the server limit', () => {
    expect(validateFeedback({ title: 'Bug', message: 'x'.repeat(FEEDBACK_MESSAGE_MAX + 1) })).toMatch(/too long|characters/i);
  });
});

describe('submitFeedback', () => {
  it('posts the trimmed report with its app and platform context', async () => {
    const post = vi.fn().mockResolvedValue({ data: { data: { reference: 12, url: 'https://gh/12' } } });
    const res = await submitFeedback({ post } as never, {
      kind: 'idea',
      title: '  Dark mode  ',
      message: '  Please add a dark theme  ',
      app: 'vendor',
      platform: 'android',
      appVersion: '1.4.0',
    });

    expect(post).toHaveBeenCalledWith('/v1/feedback', {
      kind: 'idea',
      title: 'Dark mode',
      message: 'Please add a dark theme',
      app: 'vendor',
      platform: 'android',
      appVersion: '1.4.0',
    });
    expect(res).toEqual({ reference: 12, url: 'https://gh/12' });
  });
});
