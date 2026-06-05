import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// ---- API contract types -------------------------------------------------------
// Shape returned by GET /chef/documents/expiring?withinDays=30.

export interface ExpiringDocument {
  id: string;
  type: string;
  expiryDate: string;
  daysUntilExpiry: number;
}

export interface ExpiringDocumentsResponse {
  documents: ExpiringDocument[];
}

// ---- Helpers -----------------------------------------------------------------

/** Return a human-readable description for a document type. */
export function describeDocumentType(type: string): string {
  const map: Record<string, string> = {
    fssai_license: 'FSSAI license',
    id_proof: 'ID proof',
    address_proof: 'Address proof',
    pan_card: 'PAN card',
    gst_certificate: 'GST certificate',
  };
  return map[type] ?? type.replace(/_/g, ' ');
}

// ---- Hook --------------------------------------------------------------------

/**
 * Fetch documents expiring within the next 30 days.
 * Polls every 30 minutes (not mission-critical, but should stay fresh).
 */
export function useExpiringDocuments() {
  return useQuery<ExpiringDocumentsResponse>({
    queryKey: ['chef', 'documents', 'expiring'],
    queryFn: () =>
      api
        .get<ExpiringDocumentsResponse>('/chef/documents/expiring?withinDays=30')
        .then((r) => r.data),
    staleTime: 30 * 60_000, // 30 minutes
  });
}
