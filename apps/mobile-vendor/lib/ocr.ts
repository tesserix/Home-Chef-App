import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from './api';

/** Fields the backend OCR (Cloud Vision) lifts off a document image. */
export interface OcrResult {
  fssaiNumber?: string;
  expiryDate?: string; // ISO YYYY-MM-DD
}

/**
 * OCR a picked document image to pre-fill the FSSAI number + expiry. The
 * chef always confirms/edits the result. Best-effort: the backend soft-fails
 * to an empty result, and callers must never block the upload on this.
 *
 * @param uri - local file URI from the image picker
 * @param mimeType - the picked file's mime type (call only for `image/*`)
 */
export async function ocrDocument(uri: string, mimeType: string): Promise<OcrResult> {
  const form = new FormData();
  const filename = uri.split('/').pop() ?? 'doc.jpg';
  form.append('file', { uri, name: filename, type: mimeType } as unknown as Blob);
  const res = await api.post<OcrResult>('/chef/documents/ocr', form, multipartConfig());
  return res.data ?? {};
}
