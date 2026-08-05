import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Chef compliance documents — the web twin of
// apps/mobile-vendor/app/documents/renew.tsx.
//
// The portal had no documents screen at all. A chef whose FSSAI licence was
// about to lapse could be warned by the mobile app but had no way to see or
// replace anything from the web, and an expired licence stops them trading.

export interface ChefDocument {
  id: string;
  type: string;
  fileName: string;
  fileUrl?: string;
  status: 'pending' | 'verified' | 'rejected';
  rejectionReason?: string;
  expiryDate?: string | null;
  createdAt: string;
}

/** Every document on file, with its verification state. */
export function useDocuments() {
  return useQuery<ChefDocument[]>({
    queryKey: ['chef', 'documents'],
    queryFn: () => apiClient.get<ChefDocument[]>('/chef/documents'),
    staleTime: 60_000,
  });
}

export interface ExpiringDocument {
  id: string;
  type: string;
  expiryDate: string;
  daysUntilExpiry: number;
}

/**
 * Documents expiring within the window (default 30 days).
 *
 * Deliberately excludes ALREADY-expired documents — the server filters to
 * `expiry_date > now()`. So this is the "renew before it lapses" warning, not a
 * complete compliance picture; the full list above is what shows a lapsed doc.
 */
export function useExpiringDocuments(withinDays = 30) {
  return useQuery<ExpiringDocument[]>({
    queryKey: ['chef', 'documents', 'expiring', withinDays],
    queryFn: () =>
      apiClient
        .get<{ documents: ExpiringDocument[] }>('/chef/documents/expiring', { withinDays })
        .then((r) => r?.documents ?? []),
    staleTime: 60_000,
  });
}

interface ReplaceInput {
  docId: string;
  file: File;
  expiryDate?: string;
}

/**
 * Replace a document with a fresh scan (and, where relevant, a new expiry).
 *
 * Multipart, so it deliberately does NOT go through apiClient.post — that
 * serializes JSON and would set Content-Type: application/json on a file
 * upload. It reuses the client's fetch wrapper by passing FormData as the body
 * with no explicit content type, letting the browser set the boundary.
 */
export function useReplaceDocument() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, ReplaceInput>({
    mutationFn: ({ docId, file, expiryDate }) => {
      const form = new FormData();
      form.append('file', file);
      if (expiryDate) form.append('expiryDate', expiryDate);
      return apiClient.postForm(`/chef/documents/${docId}/replace`, form);
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'documents'] });
    },
  });
}

// What onboarding asks every chef for. A chef who skipped the step arrives with
// nothing on file, so the page has to show what is still owed — an empty list
// reads as "nothing to do".
export const EXPECTED_DOCS: { type: string; why: string }[] = [
  { type: 'id_proof', why: 'Aadhaar, PAN or passport — proves who you are.' },
  { type: 'address_proof', why: 'Shows where your kitchen operates from.' },
  { type: 'fssai_license', why: 'Required by law before your menu can go live.' },
];

interface AddInput {
  type: string;
  file: File;
  expiryDate?: string;
}

/**
 * Upload a document the chef does not have on file yet.
 *
 * Distinct from useReplaceDocument only in the endpoint: there is no record to
 * replace, so the type has to travel with the file.
 */
export function useAddDocument() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, AddInput>({
    mutationFn: ({ type, file, expiryDate }) => {
      const form = new FormData();
      form.append('file', file);
      form.append('type', type);
      if (expiryDate) form.append('expiryDate', expiryDate);
      return apiClient.postForm('/chef/documents', form);
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'documents'] });
    },
  });
}

/** Human label for a document type code. */
export function documentLabel(type: string): string {
  const map: Record<string, string> = {
    fssai_license: 'FSSAI licence',
    pan_card: 'PAN card',
    aadhaar: 'Aadhaar',
    passport: 'Passport',
    bank_statement: 'Bank statement',
    profile_image: 'Profile photo',
  };
  if (map[type]) return map[type] as string;
  if (type.startsWith('kitchen_photo_')) return 'Kitchen photo';
  // Fall back to the raw code made readable rather than showing snake_case.
  return type.replace(/_/g, ' ').replace(/^\w/, (c) => c.toUpperCase());
}
