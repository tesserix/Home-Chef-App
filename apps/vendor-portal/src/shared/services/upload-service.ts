// File uploads go through the BFF proxy which handles session auth
// and forwards the request with proper Authorization header to the API
const BFF_URL = (() => { const env = import.meta.env.VITE_BFF_URL; if (env) return env; if (typeof window !== "undefined" && window.location.hostname !== "localhost") { return `${window.location.origin}/bff`; } return "/bff"; })();

// Max file sizes
const MAX_DOC_SIZE = 5 * 1024 * 1024; // 5MB for documents (PAN, Aadhaar, FSSAI, etc.)
const MAX_IMAGE_SIZE = 5 * 1024 * 1024; // 5MB for photos

// Allowed file types
const ALLOWED_DOC_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'application/pdf'];
const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'];

function validateFile(file: File, allowedTypes: string[], maxSize: number): string | null {
  if (!allowedTypes.includes(file.type)) {
    const extensions = allowedTypes.map(t => (t.split('/')[1] ?? t).toUpperCase()).join(', ');
    return `Invalid file type. Allowed: ${extensions}`;
  }
  if (file.size > maxSize) {
    const sizeMB = Math.round(maxSize / (1024 * 1024));
    return `File too large. Maximum ${sizeMB} MB.`;
  }
  return null;
}

export async function uploadDocument(
  file: File,
  type: string
): Promise<{
  id: string;
  type: string;
  fileName: string;
  fileUrl?: string;
  status: string;
}> {
  // Determine allowed types based on document category
  const isImage = type.startsWith('kitchen_photo') || type === 'profile_image';
  const allowedTypes = isImage ? ALLOWED_IMAGE_TYPES : ALLOWED_DOC_TYPES;
  const maxSize = isImage ? MAX_IMAGE_SIZE : MAX_DOC_SIZE;

  const error = validateFile(file, allowedTypes, maxSize);
  if (error) throw new Error(error);

  const formData = new FormData();
  formData.append('file', file);
  formData.append('type', type);

  const res = await fetch(`${BFF_URL}/api/v1/chef/documents`, {
    method: 'POST',
    body: formData,
    credentials: 'include',
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(err.error || err.message || 'Upload failed');
  }

  return res.json();
}

export async function uploadMenuItemImage(
  itemId: string,
  file: File
): Promise<{
  id: string;
  menuItemId: string;
  url: string;
  isPrimary: boolean;
  sortOrder: number;
}> {
  const error = validateFile(file, ALLOWED_IMAGE_TYPES, MAX_IMAGE_SIZE);
  if (error) throw new Error(error);

  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch(`${BFF_URL}/api/v1/chef/menu/items/${itemId}/images`, {
    method: 'POST',
    body: formData,
    credentials: 'include',
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(err.error || err.message || 'Upload failed');
  }

  return res.json();
}

export async function deleteMenuItemImage(
  itemId: string,
  imageId: string
): Promise<void> {
  const { useAuthStore } = await import('@/app/store/auth-store');
  const csrfToken = useAuthStore.getState().csrfToken;

  const res = await fetch(`${BFF_URL}/api/v1/chef/menu/items/${itemId}/images/${imageId}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      ...(csrfToken ? { 'x-csrf-token': csrfToken } : {}),
    },
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Delete failed' }));
    throw new Error(err.error || err.message || 'Delete failed');
  }
}

/** Lifecycle photo kinds the chef attaches to an order. Mirrors the vendor
 *  app's `OrderPhotoKind` and the API's `kind` form field. */
export type OrderPhotoKind = 'ready' | 'handover';

/**
 * Uploads a required lifecycle photo (food-ready or proof-of-handover) for an
 * order and returns its public URL. Callers must await this BEFORE the matching
 * status update, so a failed upload leaves the order where it was and the chef
 * simply retries — the same ordering the vendor app uses.
 */
export async function uploadOrderPhoto(
  orderId: string,
  kind: OrderPhotoKind,
  file: File
): Promise<{ kind: OrderPhotoKind; url: string }> {
  const error = validateFile(file, ALLOWED_IMAGE_TYPES, MAX_IMAGE_SIZE);
  if (error) throw new Error(error);

  const formData = new FormData();
  formData.append('file', file);
  formData.append('kind', kind);

  const res = await fetch(`${BFF_URL}/api/v1/chef/orders/${orderId}/photos`, {
    method: 'POST',
    body: formData,
    credentials: 'include',
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(err.error || err.message || 'Upload failed');
  }

  return res.json();
}

export async function uploadBannerImage(file: File): Promise<string> {
  const error = validateFile(file, ALLOWED_IMAGE_TYPES, MAX_IMAGE_SIZE);
  if (error) throw new Error(error);

  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch(`${BFF_URL}/api/v1/chef/banner-image`, {
    method: 'POST',
    body: formData,
    credentials: 'include',
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(err.error || err.message || 'Upload failed');
  }

  const data = await res.json();
  return data.url;
}

export async function uploadProfileImage(file: File): Promise<string> {
  const error = validateFile(file, ALLOWED_IMAGE_TYPES, MAX_IMAGE_SIZE);
  if (error) throw new Error(error);

  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch(`${BFF_URL}/api/v1/chef/profile-image`, {
    method: 'POST',
    body: formData,
    credentials: 'include',
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(err.error || err.message || 'Upload failed');
  }

  const data = await res.json();
  return data.url;
}
