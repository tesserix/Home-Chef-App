import { File, Paths } from 'expo-file-system';
import { api } from './api';
import * as Sharing from 'expo-sharing';
import { useAuthStore } from '../store/auth-store';
import { showAlertOutsideReact } from '@homechef/mobile-shared/ui';

// Use the same authenticated transport as the rest of the app, including token refresh.
export async function downloadAndSharePdf(
  path: string,
  localName: string,
): Promise<void> {
  try {
    const token = useAuthStore.getState().accessToken;
    if (!token) {
      showAlertOutsideReact('Sign in required', 'Sign in again to download PDFs.');
      return;
    }
    const response = await api.get<ArrayBuffer>(path, { responseType: 'arraybuffer' });
    const file = new File(Paths.cache, localName);
    file.write(new Uint8Array(response.data));
    if (await Sharing.isAvailableAsync()) {
      await Sharing.shareAsync(file.uri, {
        mimeType: 'application/pdf',
        UTI: 'com.adobe.pdf',
        dialogTitle: localName,
      });
    } else {
      showAlertOutsideReact('Saved', `Saved to ${file.uri}`);
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Download failed.';
    showAlertOutsideReact('Could not download', msg);
  }
}
