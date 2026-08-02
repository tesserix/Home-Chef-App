import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from './api';

/** Fields the backend OCR (Cloud Vision) lifts off a document image. */
export interface OcrResult {
  fssaiNumber?: string;
  expiryDate?: string; // ISO YYYY-MM-DD
  panNumber?: string; // ABCDE1234F, detected on ID uploads
  billDate?: string; // ISO YYYY-MM-DD — utility-bill issue date (address proof)
}

/** True when a bill date falls outside the allowed window: the current
 *  calendar month or the three before it. Mirrors the server guardrail —
 *  the server is authoritative; this exists for a friendlier early error. */
export function billDateTooOld(billDateISO: string): boolean {
  const bill = new Date(`${billDateISO}T00:00:00`);
  if (Number.isNaN(bill.getTime())) return false;
  const windowStart = new Date();
  windowStart.setHours(0, 0, 0, 0);
  windowStart.setDate(1);
  windowStart.setMonth(windowStart.getMonth() - 3);
  return bill < windowStart;
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
