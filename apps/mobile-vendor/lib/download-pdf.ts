import * as FileSystem from 'expo-file-system/legacy';
import * as Sharing from 'expo-sharing';
import * as SecureStore from 'expo-secure-store';
import { Alert } from 'react-native';

/**
 * Download an authenticated PDF from the API and open the platform share
 * sheet. The access token is read from the secure store and sent as a Bearer
 * header (it can't be embedded in the URL for FileSystem.downloadAsync).
 *
 * @param path - API path beginning with '/', e.g. '/chef/statements/{id}/statement.pdf'
 * @param localName - cache filename, e.g. 'statement-2026-06-01.pdf'
 */
export async function downloadAndSharePdf(
  path: string,
  localName: string,
): Promise<void> {
  try {
    const token = await SecureStore.getItemAsync('access_token');
    if (!token) {
      Alert.alert('Sign in required', 'Sign in again to download PDFs.');
      return;
    }
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? '';
    const url = `${apiBase}${path}`;
    const target = `${FileSystem.cacheDirectory}${localName}`;
    const dl = await FileSystem.downloadAsync(url, target, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (dl.status !== 200) {
      Alert.alert('Could not download', `Server returned ${dl.status}.`);
      return;
    }
    if (await Sharing.isAvailableAsync()) {
      await Sharing.shareAsync(dl.uri, {
        mimeType: 'application/pdf',
        dialogTitle: localName,
      });
    } else {
      Alert.alert('Saved', `Saved to ${dl.uri}`);
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Download failed.';
    Alert.alert('Could not download', msg);
  }
}
