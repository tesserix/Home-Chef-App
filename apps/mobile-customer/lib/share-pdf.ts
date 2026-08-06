/**
 * Download a PDF from a pre-signed URL into the cache and open the share sheet.
 * Throws when the download fails so the caller can show its own message.
 *
 * `expo-file-system` and `expo-sharing` are loaded lazily, INSIDE the call, and
 * never at module scope (#1038). This app ships JS over OTA, so a JS bundle that
 * touches a native module the installed binary predates would otherwise crash
 * every screen that merely *imports* this file — the receipt route rendered a
 * blank white page with no back affordance because of exactly that. Loaded here,
 * a missing module is a rejected promise the caller already handles ("Couldn't
 * share the PDF"), and the screen's non-native `expo-web-browser` path to the
 * same document keeps working.
 */
export async function shareReceiptPdf(url: string, fileName: string): Promise<void> {
  const FileSystem = await import('expo-file-system/legacy');
  const Sharing = await import('expo-sharing');

  const target = `${FileSystem.cacheDirectory}${fileName}`;
  const dl = await FileSystem.downloadAsync(url, target);
  if (dl.status !== 200) {
    throw new Error(`Server returned ${dl.status}`);
  }
  if (!(await Sharing.isAvailableAsync())) {
    throw new Error('Sharing is not available on this device');
  }
  await Sharing.shareAsync(dl.uri, {
    mimeType: 'application/pdf',
    UTI: 'com.adobe.pdf',
    dialogTitle: fileName,
  });
}
