// Sharing a kitchen's public page to the networks chefs actually use.
//
// Every target is a plain URL opened with Linking, so this adds no native
// module and ships in an OTA update. The two that cannot work that way fall
// through to the OS share sheet:
//
//   - Instagram has no public share URL for the feed or stories; posting needs
//     their SDK or a share extension. The sheet lists Instagram anyway.
//   - "Copy link" would need a clipboard native module. The sheet already
//     offers Copy on both platforms, so it routes there too.

import { Linking, Share } from 'react-native';

export type SocialNetwork =
  | 'whatsapp'
  | 'instagram'
  | 'facebook'
  | 'x'
  | 'linkedin'
  | 'more';

export interface ShareTarget {
  id: SocialNetwork;
  label: string;
  /** Shown under the label so a chef knows what tapping it does. */
  hint: string;
}

export const SHARE_TARGETS: ShareTarget[] = [
  { id: 'whatsapp', label: 'WhatsApp', hint: 'Send to a chat or your status' },
  { id: 'instagram', label: 'Instagram', hint: 'Opens the share sheet' },
  { id: 'facebook', label: 'Facebook', hint: 'Post to your page or timeline' },
  { id: 'x', label: 'X', hint: 'Post to your followers' },
  { id: 'linkedin', label: 'LinkedIn', hint: 'Share with your network' },
  { id: 'more', label: 'More', hint: 'Copy the link, or any other app' },
];

/** The public page a customer lands on. */
export function chefPublicUrl(slug: string, baseUrl = 'https://fe3dr.com'): string {
  return `${baseUrl}/chef/${slug}`;
}

/** The message that goes out with the link. */
export function shareMessage(businessName: string, url: string): string {
  return `Order home-cooked food from ${businessName} on Fe3dr — ${url}`;
}

/**
 * One thing to share.
 *
 * `title` is the short line for networks that take text and URL separately;
 * `message` is the full line for WhatsApp and the OS sheet, and already
 * contains the URL.
 */
export interface ShareContent {
  title: string;
  message: string;
  url: string;
}

/** Sharing the kitchen itself. */
export function kitchenShare(businessName: string, url: string): ShareContent {
  return {
    title: `Order home-cooked food from ${businessName} on Fe3dr`,
    message: shareMessage(businessName, url),
    url,
  };
}

/**
 * Sharing one ChefBook post.
 *
 * The URL is the kitchen's page, not a per-article one: fe3dr.com is a static
 * export, so an article page would only exist after the next site build and a
 * chef sharing a post they published minutes ago would send people to a 404.
 * The title carries the post; the link carries them to the kitchen.
 */
export function articleShare(
  businessName: string,
  articleTitle: string,
  url: string,
): ShareContent {
  const title = `${articleTitle} — from ${businessName} on Fe3dr`;
  return { title, message: `${title} — ${url}`, url };
}

/** Opens the OS share sheet. Resolves false only if the sheet itself failed. */
async function openSheet(message: string, url: string): Promise<boolean> {
  try {
    // iOS uses `url` for the rich preview and `message` for the text; Android
    // reads `message` only, so the URL is already inside it.
    await Share.share({ message, url });
    return true;
  } catch {
    return false;
  }
}

/**
 * Opens `url` if anything can handle it, else falls back to the share sheet.
 *
 * canOpenURL is the guard rather than a try/catch: on iOS, openURL for an app
 * that isn't installed can silently do nothing, which reads to the chef as a
 * dead button.
 */
async function openOrFallback(url: string, message: string, shareUrl: string): Promise<boolean> {
  try {
    if (await Linking.canOpenURL(url)) {
      await Linking.openURL(url);
      return true;
    }
  } catch {
    // Fall through to the sheet.
  }
  return openSheet(message, shareUrl);
}

/**
 * Shares one piece of content to one network.
 *
 * Returns false when nothing could be opened, so the caller can say so rather
 * than leaving the chef looking at an unchanged screen.
 */
export async function shareToNetwork(
  network: SocialNetwork,
  content: ShareContent,
): Promise<boolean> {
  const { title, message, url } = content;
  const encodedUrl = encodeURIComponent(url);
  const encodedMessage = encodeURIComponent(message);

  switch (network) {
    case 'whatsapp':
      return openOrFallback(`whatsapp://send?text=${encodedMessage}`, message, url);
    case 'facebook':
      return openOrFallback(
        `https://www.facebook.com/sharer/sharer.php?u=${encodedUrl}`,
        message,
        url,
      );
    case 'x':
      return openOrFallback(
        `https://twitter.com/intent/tweet?text=${encodeURIComponent(title)}&url=${encodedUrl}`,
        message,
        url,
      );
    case 'linkedin':
      return openOrFallback(
        `https://www.linkedin.com/sharing/share-offsite/?url=${encodedUrl}`,
        message,
        url,
      );
    case 'instagram':
    case 'more':
      return openSheet(message, url);
  }
}
