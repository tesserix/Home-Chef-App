import * as FileSystem from 'expo-file-system/legacy';
import * as Sharing from 'expo-sharing';

/**
 * Download a PDF from a pre-signed URL into the cache and open the share sheet.
 * Throws when the download fails so the caller can show its own message.
 */
export async function shareReceiptPdf(url: string, fileName: string): Promise<void> {
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
