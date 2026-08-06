// Sharing a receipt shares the PDF the API issues — never a re-typed text copy,
// which could disagree with the tax document it claims to be.
//
// The native modules are required lazily: a static import of expo-sharing threw
// "Cannot find native module 'ExpoSharing'" while the receipt screen was still
// evaluating its imports, so a build older than the dependency lost the whole
// screen rather than just the share button (#1038).

/** Thrown when this build or device cannot open a share sheet — open the URL instead. */
export const SHARING_UNAVAILABLE = 'sharing-unavailable';

export class ShareUnavailableError extends Error {
  readonly code = SHARING_UNAVAILABLE;
  constructor(message = 'Sharing is not available on this device') {
    super(message);
  }
}

export function isShareUnavailable(error: unknown): boolean {
  return (error as { code?: string } | null)?.code === SHARING_UNAVAILABLE;
}

// A build whose native side predates expo-file-system / expo-sharing throws this
// rather than failing the operation — it means "this build cannot", not "this
// document is broken", so the caller opens the PDF in the browser instead.
export function isMissingNativeModule(error: unknown): boolean {
  return /cannot find native module/i.test(String((error as { message?: unknown })?.message ?? ''));
}

/**
 * Download a PDF from a pre-signed URL into the cache and open the share sheet.
 * Throws when the download fails so the caller can show its own message, and a
 * ShareUnavailableError when there is no share sheet to open.
 */
export async function shareReceiptPdf(url: string, fileName: string): Promise<void> {
  let FileSystem: typeof import('expo-file-system/legacy');
  let Sharing: typeof import('expo-sharing');
  try {
    FileSystem = require('expo-file-system/legacy');
    Sharing = require('expo-sharing');
    if (!(await Sharing.isAvailableAsync())) throw new ShareUnavailableError();
  } catch (e) {
    if (isShareUnavailable(e) || isMissingNativeModule(e)) throw new ShareUnavailableError();
    throw e;
  }

  const target = `${FileSystem.cacheDirectory}${fileName}`;
  let dl: { status: number; uri: string };
  try {
    dl = await FileSystem.downloadAsync(url, target);
  } catch (e) {
    if (isMissingNativeModule(e)) throw new ShareUnavailableError();
    throw e;
  }
  if (dl.status !== 200) {
    throw new Error(`Server returned ${dl.status}`);
  }

  await Sharing.shareAsync(dl.uri, {
    mimeType: 'application/pdf',
    UTI: 'com.adobe.pdf',
    dialogTitle: fileName,
  });
}
