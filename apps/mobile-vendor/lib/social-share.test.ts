import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// Importing the real 'react-native' pulls NativeIntentAndroid, which throws
// under the node test environment. These are logic tests, so the two surfaces
// the module actually touches are stubbed outright.
jest.mock('react-native', () => ({
  Linking: { canOpenURL: jest.fn(), openURL: jest.fn() },
  Share: { share: jest.fn() },
}));

// eslint-disable-next-line import/first -- must follow the mock above
import { Linking, Share } from 'react-native';

import {
  chefPublicUrl,
  shareMessage,
  shareToNetwork,
  SHARE_TARGETS,
} from './social-share';

// Why this exists: every one of these targets is a hand-built URL, and a wrong
// one fails silently — the app opens, shows the wrong thing or nothing, and the
// chef assumes sharing is broken. The fallback matters just as much: a chef
// without WhatsApp installed must still get the OS sheet rather than a dead tap.

const canOpenURL = Linking.canOpenURL as unknown as jest.Mock;
const openURL = Linking.openURL as unknown as jest.Mock;
const share = Share.share as unknown as jest.Mock;

beforeEach(() => {
  canOpenURL.mockReset().mockResolvedValue(true as never);
  openURL.mockReset().mockResolvedValue(undefined as never);
  share.mockReset().mockResolvedValue({ action: 'sharedAction' } as never);
});

describe('chefPublicUrl', () => {
  it('points at the public chef page', () => {
    expect(chefPublicUrl('ammas-kitchen')).toBe('https://fe3dr.com/chef/ammas-kitchen');
  });
});

describe('shareMessage', () => {
  it('names the kitchen and carries the link', () => {
    const msg = shareMessage("Amma's Kitchen", 'https://fe3dr.com/chef/ammas-kitchen');
    expect(msg).toContain("Amma's Kitchen");
    expect(msg).toContain('https://fe3dr.com/chef/ammas-kitchen');
  });
});

describe('shareToNetwork', () => {
  const url = 'https://fe3dr.com/chef/ammas-kitchen';
  const name = 'Ammas Kitchen';

  it('opens WhatsApp with the message pre-filled', async () => {
    await shareToNetwork('whatsapp', name, url);
    const opened = String(openURL.mock.calls[0]?.[0]);
    expect(opened.startsWith('whatsapp://send?text=')).toBe(true);
    expect(decodeURIComponent(opened)).toContain(url);
  });

  it('sends the URL to the Facebook sharer', async () => {
    await shareToNetwork('facebook', name, url);
    const opened = String(openURL.mock.calls[0]?.[0]);
    expect(opened).toContain('facebook.com/sharer/sharer.php');
    expect(decodeURIComponent(opened)).toContain(url);
  });

  it('posts to X with both text and url', async () => {
    await shareToNetwork('x', name, url);
    const opened = decodeURIComponent(String(openURL.mock.calls[0]?.[0]));
    expect(opened).toContain('twitter.com/intent/tweet');
    expect(opened).toContain(name);
    expect(opened).toContain(url);
  });

  it('sends the URL to LinkedIn', async () => {
    await shareToNetwork('linkedin', name, url);
    const opened = decodeURIComponent(String(openURL.mock.calls[0]?.[0]));
    expect(opened).toContain('linkedin.com/sharing/share-offsite');
    expect(opened).toContain(url);
  });

  // Instagram has no public share URL, so it must not pretend to deep-link.
  it('routes Instagram to the OS share sheet', async () => {
    await shareToNetwork('instagram', name, url);
    expect(openURL).not.toHaveBeenCalled();
    expect(share).toHaveBeenCalledTimes(1);
  });

  it('routes More to the OS share sheet', async () => {
    await shareToNetwork('more', name, url);
    expect(share).toHaveBeenCalledTimes(1);
  });

  // The whole point of the canOpenURL guard: without it, iOS silently does
  // nothing for an app that isn't installed and the button reads as broken.
  it('falls back to the share sheet when the app is not installed', async () => {
    canOpenURL.mockResolvedValue(false as never);
    const ok = await shareToNetwork('whatsapp', name, url);
    expect(openURL).not.toHaveBeenCalled();
    expect(share).toHaveBeenCalledTimes(1);
    expect(ok).toBe(true);
  });

  it('reports failure when even the sheet fails, so the caller can say so', async () => {
    canOpenURL.mockResolvedValue(false as never);
    share.mockRejectedValue(new Error('no sheet') as never);
    expect(await shareToNetwork('whatsapp', name, url)).toBe(false);
  });

  it('offers every network the screen lists', () => {
    expect(SHARE_TARGETS.map((t) => t.id)).toEqual([
      'whatsapp',
      'instagram',
      'facebook',
      'x',
      'linkedin',
      'more',
    ]);
  });
});
