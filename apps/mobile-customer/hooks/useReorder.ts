import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { SelectedModifier } from '../types/customer';

// useReorder (#238) — fetches a re-validated "reorder preview" for a past order.
// The server matches each line against the live menu, resolves current add-on
// option IDs from the stored name snapshot, and flags lines that are now
// unavailable or whose add-ons changed. The screen then fills the cart with the
// available lines. SelectedModifier matches the server's `modifiers` shape 1:1.

export interface ReorderItem {
  menuItemId: string;
  name: string;
  quantity: number;
  notes?: string;
  imageUrl?: string;
  modifiers: SelectedModifier[];
  unitPrice: number;
  available: boolean;
  reason?: string;
  needsReview?: boolean;
}

export interface ReorderResponse {
  chefId: string;
  chefName: string;
  chefAccepting: boolean;
  items: ReorderItem[];
}

export function useReorder() {
  return useMutation<ReorderResponse, Error, string>({
    // The endpoint returns the preview object directly (not wrapped in `data`).
    mutationFn: (orderId: string) =>
      api.post(`/v1/orders/${orderId}/reorder`).then((r) => r.data as ReorderResponse),
  });
}
