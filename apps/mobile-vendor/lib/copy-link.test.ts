import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// Promote imported expo-clipboard at module scope. The installed binary has no
// ExpoClipboard native module, so merely opening the screen threw "Cannot find
// native module 'ExpoClipboard'" and redboxed the whole route — the same failure
// mode as the receipt screen in #1038, and the same fix: load it inside the call.

const mockSetStringAsync = jest.fn<(v: string) => Promise<void>>();

jest.mock('expo-clipboard', () => ({ setStringAsync: mockSetStringAsync }));

import { copyLink } from './copy-link';

beforeEach(() => {
  mockSetStringAsync.mockReset();
  mockSetStringAsync.mockResolvedValue(undefined);
});

describe('copyLink', () => {
  it('copies the link and reports success', async () => {
    await expect(copyLink('https://fe3dr.com/c/anita')).resolves.toBe(true);
    expect(mockSetStringAsync).toHaveBeenCalledWith('https://fe3dr.com/c/anita');
  });

  // The whole point: a binary without the native module must not take the screen
  // down. The caller shows "Could not copy" and the link stays on screen to read.
  it('reports failure instead of throwing when the native module is missing', async () => {
    mockSetStringAsync.mockRejectedValue(new Error("Cannot find native module 'ExpoClipboard'"));

    await expect(copyLink('https://fe3dr.com/c/anita')).resolves.toBe(false);
  });

  it('does nothing for an empty link', async () => {
    await expect(copyLink('')).resolves.toBe(false);
    expect(mockSetStringAsync).not.toHaveBeenCalled();
  });
});
